package server_test

import (
	"net/http"
	"testing"
)

type adminReportRow struct {
	ID           string `json:"id"`
	TargetType   string `json:"target_type"`
	Snapshot     string `json:"snapshot"`
	Status       string `json:"status"`
	Action       string `json:"action"`
	ResolvedBy   string `json:"resolved_by"`
	CommentState string `json:"comment_state"`
	Reporter     *struct {
		Username string `json:"username"`
	} `json:"reporter"`
	TargetUser *struct {
		Username  string `json:"username"`
		Suspended bool   `json:"suspended"`
	} `json:"target_user"`
	Room *struct {
		Name  string `json:"name"`
		State string `json:"state"`
	} `json:"room"`
}

type adminReportList struct {
	Reports   []adminReportRow `json:"reports"`
	OpenCount int              `json:"open_count"`
}

func (a *api) adminReports(admin user, status string) adminReportList {
	a.t.Helper()
	var list adminReportList
	a.must(http.StatusOK, "GET", "/admin/reports?status="+status, &admin, nil, &list)
	return list
}

func (a *api) resolve(want int, admin user, reportID, action string) {
	a.t.Helper()
	a.must(want, "POST", "/admin/reports/"+reportID+"/resolve", &admin, map[string]string{"action": action}, nil)
}

// report-block R9.1, R9.2, R9.5, R9.7: the queue, removing a comment, and
// every report on it closing with the action.
func TestModerateComment(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	admin := a.userWithRole("mod", "admin")
	carol := a.user("carol")
	a.joinRoom(carol, r)
	for _, u := range []user{alice, bob, carol} {
		a.putProgress(u, book, 100, 300)
	}
	mean := a.comment(alice, r, book, map[string]any{"page": 1, "body": "something mean"})
	a.comment(carol, r, book, map[string]any{"page": 1, "parent_id": mean.ID, "body": "a reply"})
	a.report(http.StatusCreated, bob, map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "harassment"})
	a.exec(`UPDATE content_reports SET created_at = created_at - interval '1 minute'`)
	a.report(http.StatusCreated, carol, map[string]any{"target_type": "comment", "target_id": mean.ID, "reason": "hate"})

	// Admins only (R9.7).
	a.must(http.StatusForbidden, "GET", "/admin/reports", &bob, nil, nil)

	open := a.adminReports(admin, "open")
	if open.OpenCount != 2 || len(open.Reports) != 2 {
		t.Fatalf("open %+v, want carol's and bob's", open)
	}
	first := open.Reports[0]
	if first.Reporter == nil || first.Reporter.Username != "carol" || first.TargetUser == nil || first.TargetUser.Username != "alice" ||
		first.Room == nil || first.Room.Name != "Club" || first.Room.State != "invite_only" ||
		first.Snapshot != "something mean" || first.CommentState != "live" {
		t.Errorf("newest report %+v", first)
	}

	a.resolve(http.StatusForbidden, bob, first.ID, "dismiss")
	a.resolve(http.StatusBadRequest, admin, first.ID, "close_room") // not a room
	a.resolve(http.StatusOK, admin, first.ID, "remove_comment")
	a.resolve(http.StatusConflict, admin, open.Reports[1].ID, "dismiss") // closed with it (R9.5)

	if got := a.adminReports(admin, "open"); got.OpenCount != 0 || len(got.Reports) != 0 {
		t.Errorf("still open: %+v", got)
	}
	for _, rp := range a.adminReports(admin, "all").Reports {
		if rp.Status != "actioned" || rp.Action != "remove_comment" || rp.ResolvedBy != "mod" || rp.CommentState != "removed" {
			t.Errorf("closed report %+v", rp)
		}
	}

	// Everyone sees "Removed by ReadPanda" with the reply kept (R9.2).
	got := a.comments(alice, r, book)
	if len(got.Threads) != 1 || got.Threads[0].Comments[0].Deleted != "moderator" || len(got.Threads[0].Comments[0].Replies) != 1 {
		t.Errorf("alice sees %+v, want the moderator's placeholder with carol's reply", got.Threads)
	}

	// The author is told why, and not by whom (R9.8).
	want := inboxRow{Type: "SYSTEM", Title: "Your comment was removed", Count: 1,
		Message: `Your comment in "Club" broke our community rules (Hate speech). Questions? readpanda.support@gmail.com`}
	if got := a.inbox(alice, "SYSTEM"); len(got) != 1 || got[0] != want {
		t.Errorf("alice's inbox %+v, want %+v", got, want)
	}
	if got := a.inbox(carol, "SYSTEM"); len(got) != 0 {
		t.Errorf("carol was told %+v", got)
	}

	// One the author had already deleted: nothing to tell them.
	gone := a.comment(alice, r, book, map[string]any{"page": 1, "body": "second thoughts"})
	a.report(http.StatusCreated, bob, map[string]any{"target_type": "comment", "target_id": gone.ID, "reason": "other", "details": "rude"})
	a.must(http.StatusNoContent, "DELETE", "/comments/"+gone.ID, &alice, nil, nil)
	a.resolve(http.StatusOK, admin, a.adminReports(admin, "open").Reports[0].ID, "remove_comment")
	if got := a.inbox(alice, "SYSTEM"); len(got) != 1 {
		t.Errorf("alice's inbox %+v, want only the first removal", got)
	}
}

// report-block R9.3, R9.4: closing or deleting a room, suspending, dismissing.
func TestModerateRoomAndReader(t *testing.T) {
	a, alice, bob, r, _ := commentsSetup(t)
	admin := a.userWithRole("mod", "admin")
	a.setOpen(alice, r, true)
	gone := a.newRoom(alice, "Gone")
	a.joinRoom(bob, gone)
	a.exec(`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, 't', NOW() + interval '1 day')`, alice.ID)

	reportOf := func(targetType, targetID string) string {
		t.Helper()
		var id string
		a.must(http.StatusCreated, "POST", "/reports", &bob,
			map[string]any{"target_type": targetType, "target_id": targetID, "reason": "spam"}, &struct {
				ID *string `json:"id"`
			}{&id})
		return id
	}

	a.resolve(http.StatusOK, admin, reportOf("room", r.ID), "close_room")
	if a.count(`SELECT count(*) FROM rooms WHERE id = $1 AND is_private`, r.ID) != 1 {
		t.Error("closing the room left it open")
	}

	a.resolve(http.StatusOK, admin, reportOf("room", gone.ID), "delete_room")
	if a.count(`SELECT count(*) FROM rooms WHERE id = $1`, gone.ID) != 0 {
		t.Error("the room wasn't deleted")
	}

	// The creator is told both, with the reason; a deleted room's members are
	// told it's gone (R9.8).
	support := " Questions? readpanda.support@gmail.com"
	wantAlice := []inboxRow{
		{Type: "SYSTEM", Title: "Your room is invite-only now", Count: 1,
			Message: `"Club" was taken out of open rooms for breaking our community rules (Spam or scam).` + support},
		{Type: "SYSTEM", Title: "Your room was removed", Count: 1,
			Message: `"Gone" was removed for breaking our community rules (Spam or scam).` + support},
	}
	if got := a.inbox(alice, "SYSTEM"); len(got) != 2 || got[0] != wantAlice[0] || got[1] != wantAlice[1] {
		t.Errorf("alice's inbox %+v, want %+v", got, wantAlice)
	}
	wantBob := inboxRow{Type: "SYSTEM", Title: "A room was removed", Count: 1,
		Message: `"Gone" was removed for breaking our community rules.`}
	if got := a.inbox(bob, "SYSTEM"); len(got) != 1 || got[0] != wantBob {
		t.Errorf("bob's inbox %+v, want %+v", got, wantBob)
	}
	for _, rp := range a.adminReports(admin, "all").Reports {
		if rp.Action == "delete_room" && (rp.Room == nil || rp.Room.State != "gone" || rp.Snapshot != "Gone") {
			t.Errorf("deleted room's report %+v, want its copy kept and the room gone", rp)
		}
	}

	a.resolve(http.StatusOK, admin, reportOf("user", alice.ID), "suspend_user")
	if a.count(`SELECT count(*) FROM users WHERE uuid = $1 AND isactive = false`, alice.ID) != 1 ||
		a.count(`SELECT count(*) FROM refresh_tokens WHERE user_id = $1`, alice.ID) != 0 {
		t.Error("suspending should mark the account and drop its refresh tokens")
	}

	a.exec(`UPDATE users SET isactive = true WHERE uuid = $1`, alice.ID)
	a.exec(`DELETE FROM content_reports`)
	id := reportOf("user", alice.ID)
	a.resolve(http.StatusOK, admin, id, "dismiss")
	if rows := a.adminReports(admin, "dismissed").Reports; len(rows) != 1 || rows[0].ID != id {
		t.Errorf("dismissed %+v", rows)
	}
}

// report-block R9.6, R10.4: suspending and reinstating from the Users page.
func TestSuspendFromUsersPage(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	reader := a.user("reader")
	path := "/admin/users/" + reader.ID

	a.must(http.StatusForbidden, "PATCH", path, &reader, map[string]bool{"isactive": false}, nil)
	a.must(http.StatusBadRequest, "PATCH", path, &admin, map[string]string{}, nil)
	a.must(http.StatusBadRequest, "PATCH", "/admin/users/"+admin.ID, &admin, map[string]bool{"isactive": false}, nil)
	a.must(http.StatusBadRequest, "PATCH", "/admin/users/00000000-0000-0000-0000-000000000000", &admin, map[string]bool{"isactive": true}, nil)
	a.must(http.StatusNotFound, "PATCH", "/admin/users/6f1c3a56-0000-4000-8000-000000000000", &admin, map[string]bool{"isactive": false}, nil)

	a.must(http.StatusOK, "PATCH", path, &admin, map[string]bool{"isactive": false}, nil)
	a.suspended("POST", "/room/create", &reader, map[string]any{"name": "Nope"})
	a.must(http.StatusOK, "PATCH", path, &admin, map[string]bool{"isactive": true}, nil)
	a.must(http.StatusCreated, "POST", "/room/create", &reader, map[string]any{"name": "Yes"}, nil)
}
