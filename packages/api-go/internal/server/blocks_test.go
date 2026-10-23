package server_test

import (
	"net/http"
	"testing"
)

type blockedReader struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func (a *api) block(as user, who user) {
	a.t.Helper()
	a.must(http.StatusNoContent, "POST", "/users/me/blocks", &as, map[string]string{"user_id": who.ID}, nil)
}

// report-block R6.2, R6.3: blocking is per reader, idempotent, and listed.
func TestBlockRoutes(t *testing.T) {
	a := newAPI(t)
	bob, alice, carol := a.user("bob"), a.user("alice"), a.user("carol")

	a.must(http.StatusBadRequest, "POST", "/users/me/blocks", &bob, map[string]string{"user_id": bob.ID}, nil)
	a.must(http.StatusBadRequest, "POST", "/users/me/blocks", &bob,
		map[string]string{"user_id": "00000000-0000-0000-0000-000000000000"}, nil)
	a.must(http.StatusNotFound, "POST", "/users/me/blocks", &bob,
		map[string]string{"user_id": "6f1c3a56-0000-4000-8000-000000000000"}, nil)

	a.block(bob, alice)
	a.block(bob, alice)
	a.exec(`UPDATE user_blocks SET created_at = created_at - interval '1 hour'`)
	a.block(bob, carol)

	var list []blockedReader
	a.must(http.StatusOK, "GET", "/users/me/blocks", &bob, nil, &list)
	if len(list) != 2 || list[0].Username != "carol" || list[1].Username != "alice" {
		t.Errorf("blocked %+v, want carol then alice", list)
	}
	a.must(http.StatusOK, "GET", "/users/me/blocks", &alice, nil, &list)
	if len(list) != 0 {
		t.Errorf("alice's list %+v, want empty: blocks are one-way", list)
	}

	a.must(http.StatusNoContent, "DELETE", "/users/me/blocks/"+alice.ID, &bob, nil, nil)
	a.must(http.StatusNoContent, "DELETE", "/users/me/blocks/"+alice.ID, &bob, nil, nil)
	a.must(http.StatusOK, "GET", "/users/me/blocks", &bob, nil, &list)
	if len(list) != 1 || list[0].Username != "carol" {
		t.Errorf("after unblocking alice: %+v", list)
	}
}

// report-block R5.3, R5.7: the blocker gets nothing of the blocked reader's
// comments or counts; the blocked reader sees no change.
func TestBlockedCommentsAreLeftOut(t *testing.T) {
	a := newAPI(t)
	bob, alice, carol := a.user("bob"), a.user("alice"), a.user("carol")
	book := a.book("Dune")
	r := a.newRoom(carol, "Club")
	a.joinRoom(alice, r)
	a.joinRoom(bob, r)
	for _, u := range []user{alice, bob, carol} {
		a.putProgress(u, book, 100, 300)
	}

	aliceRoot := a.comment(alice, r, book, map[string]any{"page": 5, "anchor_text": "one", "body": "alice starts"})
	a.comment(carol, r, book, map[string]any{"page": 5, "parent_id": aliceRoot.ID, "body": "carol answers alice"})
	carolRoot := a.comment(carol, r, book, map[string]any{"page": 6, "anchor_text": "two", "body": "carol starts"})
	a.comment(alice, r, book, map[string]any{"page": 6, "parent_id": carolRoot.ID, "body": "alice answers carol"})
	// Past bob's page: alice's thread (with carol's reply) and one of carol's.
	ahead := a.comment(alice, r, book, map[string]any{"page": 200, "anchor_text": "three", "body": "alice ahead"})
	a.comment(carol, r, book, map[string]any{"page": 200, "parent_id": ahead.ID, "body": "carol ahead in alice's"})
	a.comment(carol, r, book, map[string]any{"page": 201, "anchor_text": "four", "body": "carol ahead"})

	a.block(bob, alice)

	got := a.comments(bob, r, book)
	if len(got.Threads) != 1 || len(got.Threads[0].Comments) != 1 || got.Threads[0].Comments[0].ID != carolRoot.ID {
		t.Fatalf("bob sees %+v, want only carol's thread", got.Threads)
	}
	if n := len(got.Threads[0].Comments[0].Replies); n != 0 {
		t.Errorf("bob sees %d replies in carol's thread, want alice's left out", n)
	}
	if got.LockedCount != 1 {
		t.Errorf("bob's locked_count %d, want 1: carol's own thread only", got.LockedCount)
	}

	// Alice sees what she did before (R5.7): her three threads and carol's,
	// all but carol's page 201.
	if got := a.comments(alice, r, book); len(got.Threads) != 3 || got.LockedCount != 1 {
		t.Errorf("alice sees %d threads, %d locked; want 3 and 1", len(got.Threads), got.LockedCount)
	}

	// Unblocking brings them back on the next load (R6.3).
	a.must(http.StatusNoContent, "DELETE", "/users/me/blocks/"+alice.ID, &bob, nil, nil)
	if got := a.comments(bob, r, book); len(got.Threads) != 2 || got.LockedCount != 3 {
		t.Errorf("after unblocking: %d threads, %d locked; want 2 and 3", len(got.Threads), got.LockedCount)
	}
}

// report-block R5.4, R5.5, R5.6: no notifications from them, none of their
// open rooms, still in the member list but marked, and not a "friend" who
// read the book.
func TestBlockElsewhere(t *testing.T) {
	a := newAPI(t)
	bob, alice, carol := a.user("bob"), a.user("alice"), a.user("carol")
	book := a.book("Dune")
	r := a.newRoom(bob, "Bob's")
	a.joinRoom(carol, r)
	a.joinRoom(alice, r)
	a.putProgress(alice, book, 10, 300)
	a.putProgress(carol, book, 10, 300)
	a.block(bob, alice)
	a.exec(`DELETE FROM notifications`)

	// Notifications (R5.5): a comment, a reply to bob, a join.
	bobs := a.comment(bob, r, book, map[string]any{"page": 1, "body": "bob's"})
	a.comment(alice, r, book, map[string]any{"page": 1, "parent_id": bobs.ID, "body": "a reply"})
	a.comment(alice, r, book, map[string]any{"page": 2, "body": "a comment"})
	other := a.newRoom(bob, "Another")
	a.joinRoom(alice, other)
	for _, typ := range []string{"REPLY", "ROOM_COMMENTS", "ROOM_JOIN"} {
		if rows := a.inbox(bob, typ); len(rows) != 0 {
			t.Errorf("bob got %s from alice: %+v", typ, rows)
		}
	}
	if rows := a.inbox(carol, "ROOM_COMMENTS"); len(rows) != 1 {
		t.Errorf("carol got %d room notifications, want 1", len(rows))
	}

	// Open rooms (R5.4).
	open := a.newRoom(alice, "Alice's open")
	a.setOpen(alice, open, true)
	if ids := openRoomIDs(a.openRooms(bob, "")); len(ids) != 0 {
		t.Errorf("bob sees open rooms %v, want none of alice's", ids)
	}
	if ids := openRoomIDs(a.openRooms(carol, "")); len(ids) != 1 {
		t.Errorf("carol sees %v, want alice's room", ids)
	}

	// Members (R5.6): still there, marked for bob only.
	var detail struct {
		Members []struct {
			Username string `json:"username"`
			Blocked  bool   `json:"blocked"`
		} `json:"members"`
	}
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &bob, nil, &detail)
	marked := map[string]bool{}
	for _, m := range detail.Members {
		marked[m.Username] = m.Blocked
	}
	if len(detail.Members) != 3 || !marked["alice"] || marked["carol"] {
		t.Errorf("bob's members %+v, want all three with alice marked", detail.Members)
	}
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &carol, nil, &detail)
	for _, m := range detail.Members {
		if m.Blocked {
			t.Errorf("carol sees %s marked blocked", m.Username)
		}
	}

	// Book detail's room-mates who've read it.
	var bd struct {
		FriendsRead struct {
			Count int `json:"count"`
		} `json:"friends_read"`
	}
	a.must(http.StatusOK, "GET", "/books/"+book, &bob, nil, &bd)
	if bd.FriendsRead.Count != 1 {
		t.Errorf("bob's friends_read %d, want 1: carol, not alice", bd.FriendsRead.Count)
	}
}
