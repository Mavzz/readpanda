package server_test

import (
	"net/http"
	"testing"
)

// report-block R7.3, R7.4, R7.6: only the author deletes; a comment with
// replies stays as an empty placeholder, one without goes, and the
// placeholder goes with its last reply.
func TestDeleteComment(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(bob, book, 100, 300)
	a.putProgress(alice, book, 100, 300)

	lone := a.comment(alice, r, book, map[string]any{"page": 3, "body": "on its own"})
	root := a.comment(alice, r, book, map[string]any{"page": 5, "anchor_text": "a passage", "body": "start of a thread"})
	reply := a.comment(bob, r, book, map[string]any{"page": 5, "parent_id": root.ID, "body": "an answer"})
	a.must(http.StatusNoContent, "POST", "/comments/"+root.ID+"/like", &bob, nil, nil)

	// Only the author (R7.6).
	a.must(http.StatusForbidden, "DELETE", "/comments/"+root.ID, &bob, nil, nil)
	a.must(http.StatusNotFound, "DELETE", "/comments/cm_missing", &alice, nil, nil)

	// No replies: gone.
	a.must(http.StatusNoContent, "DELETE", "/comments/"+lone.ID, &alice, nil, nil)
	if a.count(`SELECT count(*) FROM book_comments WHERE id = $1`, lone.ID) != 0 {
		t.Error("a comment without replies is still stored")
	}

	// Replies: a placeholder with nothing of the author's, the reply kept,
	// the quote kept (R7.3, R7.4).
	a.must(http.StatusNoContent, "DELETE", "/comments/"+root.ID, &alice, nil, nil)
	a.must(http.StatusNotFound, "DELETE", "/comments/"+root.ID, &alice, nil, nil)
	got := a.comments(bob, r, book)
	if len(got.Threads) != 1 || len(got.Threads[0].Comments) != 1 {
		t.Fatalf("bob sees %+v, want one thread with the placeholder", got.Threads)
	}
	ph := got.Threads[0].Comments[0]
	if ph.Deleted != "author" || ph.Body != "" || ph.UserID != "" || ph.Username != "" || ph.Likes != 0 {
		t.Errorf("placeholder %+v, want deleted by author with no body, author or likes", ph)
	}
	if len(ph.Replies) != 1 || ph.Replies[0].ID != reply.ID || ph.Replies[0].Body != "an answer" {
		t.Errorf("replies %+v, want bob's answer kept", ph.Replies)
	}
	if q := a.count(`SELECT count(*) FROM book_comments WHERE id = $1 AND anchor_text = 'a passage'`, root.ID); q != 1 {
		t.Error("the quote went with the comment")
	}

	// Nothing more hangs off it: no replies, no likes.
	a.must(http.StatusNotFound, "POST", "/room/"+r.ID+"/book/"+book+"/comments", &bob,
		map[string]any{"page": 5, "parent_id": root.ID, "body": "too late"}, nil)
	a.must(http.StatusNotFound, "POST", "/comments/"+root.ID+"/like", &bob, nil, nil)

	// The last reply takes the placeholder with it.
	a.must(http.StatusNoContent, "DELETE", "/comments/"+reply.ID, &bob, nil, nil)
	if a.count(`SELECT count(*) FROM book_comments WHERE id IN ($1, $2)`, root.ID, reply.ID) != 0 {
		t.Error("the placeholder outlived its last reply")
	}
	if got := a.comments(bob, r, book); len(got.Threads) != 0 {
		t.Errorf("bob still sees %d threads", len(got.Threads))
	}
}

// A placeholder isn't something new waiting for you, locked or unread.
func TestDeletedCommentIsNotCounted(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(alice, book, 100, 300)
	root := a.comment(alice, r, book, map[string]any{"page": 50, "body": "ahead of bob"})
	a.comment(alice, r, book, map[string]any{"page": 50, "parent_id": root.ID, "body": "and more"})
	a.must(http.StatusNoContent, "DELETE", "/comments/"+root.ID, &alice, nil, nil)

	if got := a.comments(bob, r, book).LockedCount; got != 1 {
		t.Errorf("locked_count %d, want 1: the reply, not the placeholder", got)
	}
}
