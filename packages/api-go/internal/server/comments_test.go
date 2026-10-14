package server_test

import (
	"net/http"
	"strings"
	"testing"
)

type comment struct {
	ID       string `json:"id"`
	Page     int    `json:"page"`
	Body     string `json:"body"`
	ParentID string `json:"parent_id"`
	Replies  []comment
}

type commentsResponse struct {
	FurthestPage int `json:"furthest_page"`
	Threads      []struct {
		AnchorKey string    `json:"anchor_key"`
		Page      int       `json:"page"`
		Comments  []comment `json:"comments"`
	} `json:"threads"`
	LockedCount int `json:"locked_count"`
}

// commentsSetup is a room with two readers, reading one book.
func commentsSetup(t *testing.T) (a *api, alice, bob user, r room, book string) {
	a = newAPI(t)
	alice, bob = a.user("alice"), a.user("bob")
	book = a.book("Dune")
	r = a.newRoom(alice, "Club")
	a.joinRoom(bob, r)
	return
}

func (a *api) comment(as user, r room, book string, body map[string]any) comment {
	a.t.Helper()
	var c comment
	a.must(http.StatusCreated, "POST", "/room/"+r.ID+"/book/"+book+"/comments", &as, body, &c)
	return c
}

func (a *api) comments(as user, r room, book string) commentsResponse {
	a.t.Helper()
	var resp commentsResponse
	a.must(http.StatusOK, "GET", "/room/"+r.ID+"/book/"+book+"/comments", &as, nil, &resp)
	return resp
}

// The core promise of comments: nothing written about a page you haven't
// reached reaches you, beyond a count.
func TestCommentsAheadOfTheReaderAreLocked(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)

	a.putProgress(bob, book, 10, 300)
	a.comment(alice, r, book, map[string]any{"page": 50, "anchor_text": "the spice must flow", "body": "huge twist"})

	got := a.comments(bob, r, book)
	if len(got.Threads) != 0 {
		t.Fatalf("bob at page 10 can see %d threads written on page 50", len(got.Threads))
	}
	if got.LockedCount != 1 {
		t.Errorf("locked_count = %d, want 1", got.LockedCount)
	}

	// Reading past the page unlocks it, and the save says so.
	p := a.putProgress(bob, book, 60, 300)
	if p.NewlyUnlocked != 1 {
		t.Errorf("newly_unlocked = %d, want 1", p.NewlyUnlocked)
	}
	got = a.comments(bob, r, book)
	if len(got.Threads) != 1 || got.Threads[0].Comments[0].Body != "huge twist" || got.LockedCount != 0 {
		t.Errorf("after reaching page 60: %+v", got)
	}

	// Going back to re-read doesn't lock it again.
	a.putProgress(bob, book, 5, 300)
	if got := a.comments(bob, r, book); len(got.Threads) != 1 {
		t.Error("flipping back re-locked a comment bob had already seen")
	}
}

func TestOwnCommentsAreNeverLocked(t *testing.T) {
	a, alice, _, r, book := commentsSetup(t)
	a.comment(alice, r, book, map[string]any{"page": 200, "body": "note to self"})

	got := a.comments(alice, r, book)
	if len(got.Threads) != 1 || got.LockedCount != 0 {
		t.Errorf("author's own page-200 comment: %+v", got)
	}
	// Writing on a page proves you reached it.
	if got.FurthestPage != 200 {
		t.Errorf("furthest_page = %d, want 200", got.FurthestPage)
	}
}

func TestRepliesFollowTheirThread(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(bob, book, 100, 300)

	root := a.comment(alice, r, book, map[string]any{"page": 50, "anchor_text": "Fear is the mind-killer", "body": "root"})
	// A reply inherits the root's page whatever the body claims.
	reply := a.comment(bob, r, book, map[string]any{"page": 1, "parent_id": root.ID, "body": "reply"})
	if reply.Page != 50 || reply.ParentID != root.ID {
		t.Errorf("reply = %+v, want page 50 under %s", reply, root.ID)
	}

	// One level only.
	a.must(http.StatusBadRequest, "POST", "/room/"+r.ID+"/book/"+book+"/comments", &alice,
		map[string]any{"parent_id": reply.ID, "body": "nested"}, nil)

	got := a.comments(alice, r, book)
	if len(got.Threads) != 1 || len(got.Threads[0].Comments) != 1 || len(got.Threads[0].Comments[0].Replies) != 1 {
		t.Errorf("threads = %+v", got.Threads)
	}
}

func TestSamePassageSharesAThread(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(bob, book, 100, 300)
	a.putProgress(alice, book, 100, 300)

	a.comment(alice, r, book, map[string]any{"page": 9, "anchor_text": "A beginning is the time", "body": "one"})
	a.comment(bob, r, book, map[string]any{"page": 9, "anchor_text": "  a BEGINNING is the  time ", "body": "two"})

	got := a.comments(alice, r, book)
	if len(got.Threads) != 1 || len(got.Threads[0].Comments) != 2 {
		t.Errorf("want one thread with both comments, got %+v", got.Threads)
	}
}

func TestCommentRetryIsIdempotent(t *testing.T) {
	a, alice, _, r, book := commentsSetup(t)
	body := map[string]any{"page": 3, "body": "hello", "client_id": "c-123"}
	first := a.comment(alice, r, book, body)
	second := a.comment(alice, r, book, body)
	if first.ID != second.ID {
		t.Errorf("retry created a second comment: %s vs %s", first.ID, second.ID)
	}
	if n := len(a.comments(alice, r, book).Threads); n != 1 {
		t.Errorf("threads = %d, want 1", n)
	}
}

func TestCommentValidationAndMembership(t *testing.T) {
	a, alice, _, r, book := commentsSetup(t)
	outsider := a.user("outsider")
	path := "/room/" + r.ID + "/book/" + book + "/comments"

	a.must(http.StatusForbidden, "GET", path, &outsider, nil, nil)
	a.must(http.StatusForbidden, "POST", path, &outsider, map[string]any{"body": "hi"}, nil)

	for name, body := range map[string]map[string]any{
		"empty":         {"body": "   "},
		"too long":      {"body": strings.Repeat("x", 2001)},
		"negative page": {"body": "x", "page": -1},
	} {
		if got := a.do("POST", path, &alice, body, nil); got != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, got)
		}
	}
}
