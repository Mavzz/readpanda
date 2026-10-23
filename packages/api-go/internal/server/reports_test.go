package server_test

import (
	"net/http"
	"strings"
	"testing"
)

func (a *api) report(want int, as user, body map[string]any) {
	a.t.Helper()
	a.must(want, "POST", "/reports", &as, body, nil)
}

// report-block R1.3, R1.7, R3.1, R3.2, R4.1, R4.2: a comment report is
// checked, stored with a copy of the text, made once, and every admin hears
// about it without the text.
func TestReportComment(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	admin := a.userWithRole("mod", "admin")
	dave := a.user("dave")
	a.putProgress(bob, book, 100, 300)
	mean := a.comment(alice, r, book, map[string]any{"page": 1, "body": "something mean"})
	report := map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "harassment"}

	// Checks.
	a.report(http.StatusBadRequest, bob, map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "boring"})
	a.report(http.StatusBadRequest, bob, map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "other", "details": "  "})
	a.report(http.StatusBadRequest, bob, map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "other", "details": strings.Repeat("x", 501)})
	a.report(http.StatusBadRequest, alice, report) // your own (R2.4)
	a.report(http.StatusNotFound, dave, report)    // not in the room
	a.report(http.StatusNotFound, bob, map[string]any{"target_type": "comment", "target_id": "cm_missing", "reason": "spam"})

	a.report(http.StatusCreated, bob, report)
	a.report(http.StatusOK, bob, report) // once (R1.7)

	if a.count(`SELECT count(*) FROM content_reports
	             WHERE reporter_id = $1 AND target_user_id = $2 AND room_id = $3
	               AND snapshot = 'something mean' AND status = 'open'`, bob.ID, alice.ID, r.ID) != 1 {
		t.Error("want one open report with the comment's text, its author and room")
	}
	// The copy outlives the comment (R3.2).
	a.must(http.StatusNoContent, "DELETE", "/comments/"+mean.ID, &alice, nil, nil)
	if a.count(`SELECT count(*) FROM content_reports WHERE snapshot = 'something mean'`) != 1 {
		t.Error("deleting the comment took the evidence with it")
	}

	got := a.inbox(admin, "SYSTEM")
	if len(got) != 1 || got[0].Title != "New report" || got[0].Message != "Harassment or bullying · comment in Club" {
		t.Errorf("admin's inbox %+v, want one New report naming the reason and room", got)
	}
	if rows := a.inbox(bob, "SYSTEM"); len(rows) != 0 {
		t.Errorf("a reader who isn't an admin was told: %+v", rows)
	}
}

// report-block R1.4: what you report is gone for you, and only for you.
func TestReportedCommentIsHiddenFromTheReporter(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	carol := a.user("carol")
	a.joinRoom(carol, r)
	for _, u := range []user{alice, bob, carol} {
		a.putProgress(u, book, 100, 300)
	}
	lone := a.comment(alice, r, book, map[string]any{"page": 1, "anchor_text": "one", "body": "lone"})
	root := a.comment(alice, r, book, map[string]any{"page": 2, "anchor_text": "two", "body": "a thread"})
	a.comment(carol, r, book, map[string]any{"page": 2, "parent_id": root.ID, "body": "carol's reply"})
	other := a.comment(carol, r, book, map[string]any{"page": 3, "anchor_text": "three", "body": "carol's"})
	reply := a.comment(alice, r, book, map[string]any{"page": 3, "parent_id": other.ID, "body": "alice's reply"})

	for _, id := range []string{lone.ID, root.ID, reply.ID} {
		a.report(http.StatusCreated, bob, map[string]any{"target_type": "comment", "target_id": id, "reason": "spam"})
	}

	got := a.comments(bob, r, book)
	if len(got.Threads) != 2 {
		t.Fatalf("bob sees %d threads, want 2: the lone comment gone", len(got.Threads))
	}
	hiddenRoot := got.Threads[0].Comments[0]
	if hiddenRoot.Hidden != "reported" || hiddenRoot.Body != "" || hiddenRoot.Username != "" || len(hiddenRoot.Replies) != 1 {
		t.Errorf("reported thread start %+v, want a placeholder keeping carol's reply", hiddenRoot)
	}
	if n := len(got.Threads[1].Comments[0].Replies); n != 0 {
		t.Errorf("bob still sees %d replies under carol's comment, want alice's gone", n)
	}

	// Everyone else sees them all.
	if got := a.comments(carol, r, book); len(got.Threads) != 3 {
		t.Errorf("carol sees %d threads, want 3", len(got.Threads))
	}
}

// report-block R2, R3.2: readers and rooms are reported with their name as
// the copy; only by someone who can see them, and never your own.
func TestReportReaderAndRoom(t *testing.T) {
	a, alice, bob, r, _ := commentsSetup(t)
	stranger := a.user("stranger")

	a.report(http.StatusCreated, bob, map[string]any{"target_type": "user", "target_id": alice.ID, "room_id": r.ID, "reason": "harassment"})
	if a.count(`SELECT count(*) FROM content_reports WHERE target_type = 'user' AND snapshot = 'alice' AND room_id = $1`, r.ID) != 1 {
		t.Error("a reader report should keep their username and the room")
	}
	a.report(http.StatusNotFound, stranger, map[string]any{"target_type": "user", "target_id": alice.ID, "reason": "spam"})
	a.report(http.StatusBadRequest, bob, map[string]any{"target_type": "user", "target_id": bob.ID, "reason": "spam"})
	a.report(http.StatusBadRequest, bob, map[string]any{"target_type": "user", "target_id": "00000000-0000-0000-0000-000000000000", "reason": "spam"})

	a.exec(`UPDATE rooms SET description = 'rude words here' WHERE id = $1`, r.ID)
	a.report(http.StatusCreated, bob, map[string]any{"target_type": "room", "target_id": r.ID, "reason": "hate"})
	if a.count(`SELECT count(*) FROM content_reports WHERE target_type = 'room' AND target_user_id = $1
	             AND snapshot = E'Club\n\nrude words here'`, alice.ID) != 1 {
		t.Error("a room report should keep its name and description, and its creator")
	}
	a.report(http.StatusBadRequest, alice, map[string]any{"target_type": "room", "target_id": r.ID, "reason": "hate"}) // R2.2
	a.report(http.StatusNotFound, stranger, map[string]any{"target_type": "room", "target_id": r.ID, "reason": "hate"})
	// An open room is reportable from its preview (R2.2).
	a.setOpen(alice, r, true)
	a.report(http.StatusCreated, stranger, map[string]any{"target_type": "room", "target_id": r.ID, "reason": "hate"})
}

// report-block R3.4: 30 a day.
func TestReportsAreLimited(t *testing.T) {
	a, alice, bob, r, _ := commentsSetup(t)
	a.exec(`INSERT INTO content_reports (id, reporter_id, target_type, target_id, reason, snapshot)
	        SELECT 'rp_' || g, $1, 'room', 'rm_' || g, 'spam', 'x' FROM generate_series(1, 30) g`, bob.ID)
	var got struct {
		Error string `json:"error"`
	}
	a.must(http.StatusTooManyRequests, "POST", "/reports", &bob,
		map[string]any{"target_type": "user", "target_id": alice.ID, "reason": "spam"}, &got)
	if got.Error != "You've sent a lot of reports today. Email us if it's urgent." {
		t.Errorf("error %q", got.Error)
	}
	// Yesterday's don't count.
	a.exec(`UPDATE content_reports SET created_at = created_at - interval '25 hours'`)
	a.report(http.StatusCreated, bob, map[string]any{"target_type": "room", "target_id": r.ID, "reason": "spam"})
}
