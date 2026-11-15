package server_test

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

// suspensionRow is one user_suspensions row, as the tests read it.
type suspensionRow struct {
	SuspendedBy, ReportID, Note, ReinstatedBy, ReinstateNote sql.NullString
	Source                                                   string
	Ended, Dated, ReinstatedDated                            bool
}

func (a *api) suspensions(userID string) []suspensionRow {
	a.t.Helper()
	rows, err := database.DB.Query(
		`SELECT suspended_by::text, report_id, note, reinstated_by::text, reinstate_note, source,
		        ended, suspended_at IS NOT NULL, reinstated_at IS NOT NULL
		   FROM user_suspensions WHERE user_id = $1 ORDER BY suspended_at, id`, userID)
	if err != nil {
		a.t.Fatal(err)
	}
	defer rows.Close()
	var out []suspensionRow
	for rows.Next() {
		var s suspensionRow
		if err := rows.Scan(&s.SuspendedBy, &s.ReportID, &s.Note, &s.ReinstatedBy, &s.ReinstateNote, &s.Source,
			&s.Ended, &s.Dated, &s.ReinstatedDated); err != nil {
			a.t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// historyMatchesAccounts checks portal-moderation R1.8: suspended exactly
// when there's an open suspension, the Deleted reader aside.
func (a *api) historyMatchesAccounts() {
	a.t.Helper()
	if n := a.count(
		`SELECT count(*) FROM users u
		  WHERE u.username <> 'deleted.reader'
		    AND (u.isactive IS FALSE) <> EXISTS (SELECT 1 FROM user_suspensions s WHERE s.user_id = u.uuid AND NOT s.ended)`,
	); n != 0 {
		a.t.Errorf("%d accounts whose suspended state doesn't match their history", n)
	}
}

func (a *api) setActive(want int, admin, reader user, active bool, note string) {
	a.t.Helper()
	a.must(want, "PATCH", "/admin/users/"+reader.ID, &admin, map[string]any{"isactive": active, "note": note}, nil)
}

func (a *api) resolveWithNote(want int, admin user, reportID, action, note string) {
	a.t.Helper()
	a.must(want, "POST", "/admin/reports/"+reportID+"/resolve", &admin, map[string]string{"action": action, "note": note}, nil)
}

// portal-moderation R1.1, R1.3: suspending from a report records it once,
// with the report and note; suspending someone already suspended adds
// nothing but still closes the report.
func TestSuspensionHistoryFromReport(t *testing.T) {
	a, alice, bob, _, _ := commentsSetup(t)
	admin := a.userWithRole("mod", "admin")
	reportOf := func() string {
		t.Helper()
		var id string
		a.must(http.StatusCreated, "POST", "/reports", &bob,
			map[string]any{"target_type": "user", "target_id": alice.ID, "reason": "spam"}, &struct {
				ID *string `json:"id"`
			}{&id})
		return id
	}

	first := reportOf()
	a.resolveWithNote(http.StatusOK, admin, first, "suspend_user", "  third spam wave  ")
	got := a.suspensions(alice.ID)
	if len(got) != 1 {
		t.Fatalf("%d suspensions, want 1", len(got))
	}
	if s := got[0]; s.Source != "report" || s.ReportID.String != first || s.SuspendedBy.String != admin.ID ||
		s.Note.String != "third spam wave" || s.Ended || !s.Dated {
		t.Errorf("suspension %+v, want from report %s by %s with the trimmed note, open", s, first, admin.ID)
	}

	// Still suspended, reported again: the second Suspend closes the report
	// and adds nothing.
	second := reportOf()
	a.resolve(http.StatusOK, admin, second, "suspend_user")
	if n := len(a.suspensions(alice.ID)); n != 1 {
		t.Errorf("%d suspensions after suspending again, want 1", n)
	}
	if a.count(`SELECT count(*) FROM content_reports WHERE id = $1 AND status = 'actioned'`, second) != 1 {
		t.Error("the second report should still close")
	}

	// A note on another action is ignored, however long.
	a.setActive(http.StatusOK, admin, alice, true, "")
	a.resolveWithNote(http.StatusOK, admin, reportOf(), "dismiss", strings.Repeat("x", 600))
	a.historyMatchesAccounts()
}

// portal-moderation R1.1, R1.2, R1.6: from the Users page, each suspension
// is a record and reinstating closes it; reinstating twice is harmless.
func TestSuspensionHistoryFromUsersPage(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	other := a.userWithRole("mod2", "admin")
	reader := a.user("reader")

	a.setActive(http.StatusOK, admin, reader, false, "")
	a.setActive(http.StatusOK, other, reader, false, "again") // already suspended (R1.3)
	if got := a.suspensions(reader.ID); len(got) != 1 || got[0].Source != "users" || got[0].Note.Valid || got[0].SuspendedBy.String != admin.ID {
		t.Fatalf("suspensions %+v, want one from the Users page by the first admin, no note", got)
	}

	a.setActive(http.StatusOK, other, reader, true, "appealed by email")
	got := a.suspensions(reader.ID)
	if s := got[0]; !s.Ended || !s.ReinstatedDated || s.ReinstatedBy.String != other.ID || s.ReinstateNote.String != "appealed by email" {
		t.Errorf("after reinstating %+v, want closed by %s with the note", s, other.ID)
	}
	a.setActive(http.StatusOK, admin, reader, true, "") // nothing open: still fine

	a.setActive(http.StatusOK, admin, reader, false, "back at it")
	got = a.suspensions(reader.ID)
	if len(got) != 2 || got[0].Ended == got[1].Ended || got[1].Note.String != "back at it" {
		t.Errorf("suspensions %+v, want two with only the newest open", got)
	}
	if n := a.count(`SELECT count(*) FROM user_suspensions WHERE user_id = $1`, reader.ID); n != 2 {
		t.Errorf("times suspended %d, want 2 (R1.6)", n)
	}
	a.historyMatchesAccounts()
}

// portal-moderation R1.4: a note over 500 characters changes nothing.
func TestSuspensionNoteTooLong(t *testing.T) {
	a, alice, bob, _, _ := commentsSetup(t)
	admin := a.userWithRole("mod", "admin")
	long := strings.Repeat("é", 501)

	a.setActive(http.StatusBadRequest, admin, alice, false, long)
	var id string
	a.must(http.StatusCreated, "POST", "/reports", &bob,
		map[string]any{"target_type": "user", "target_id": alice.ID, "reason": "spam"}, &struct {
			ID *string `json:"id"`
		}{&id})
	a.resolveWithNote(http.StatusBadRequest, admin, id, "suspend_user", long)

	if a.count(`SELECT count(*) FROM users WHERE uuid = $1 AND isactive = false`, alice.ID) != 0 ||
		len(a.suspensions(alice.ID)) != 0 ||
		a.count(`SELECT count(*) FROM content_reports WHERE id = $1 AND status = 'open'`, id) != 1 {
		t.Error("a refused note should leave the account, its history and the report as they were")
	}

	// Exactly 500, with spaces trimmed off, is fine.
	a.setActive(http.StatusOK, admin, alice, false, "  "+strings.Repeat("é", 500)+"  ")
	a.historyMatchesAccounts()
}

// portal-moderation R1.7: the history goes with the reader, and stays
// without a name when the admin is deleted.
func TestSuspensionHistoryOnDelete(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	keeper := a.userWithRole("keeper", "admin")
	reader := a.user("reader")
	gone := a.user("gone")

	a.setActive(http.StatusOK, admin, reader, false, "")
	a.setActive(http.StatusOK, admin, gone, false, "")
	a.setActive(http.StatusOK, admin, reader, true, "")
	a.setActive(http.StatusOK, keeper, reader, false, "")

	a.exec(`DELETE FROM users WHERE uuid = $1`, admin.ID)
	got := a.suspensions(reader.ID)
	if len(got) != 2 || got[0].SuspendedBy.Valid || got[0].ReinstatedBy.Valid || got[1].SuspendedBy.String != keeper.ID {
		t.Errorf("after deleting the admin %+v, want both kept, the deleted admin's name blank", got)
	}

	a.exec(`DELETE FROM users WHERE uuid = $1`, gone.ID)
	if n := a.count(`SELECT count(*) FROM user_suspensions WHERE user_id = $1`, gone.ID); n != 0 {
		t.Errorf("%d suspensions left for a deleted reader, want 0", n)
	}
	a.historyMatchesAccounts()
}

type suspensionsList struct {
	Readers []struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		SuspensionCount int    `json:"suspension_count"`
		Last            *struct {
			SuspendedAt *string `json:"suspended_at"`
			SuspendedBy *struct {
				Username string `json:"username"`
			} `json:"suspended_by"`
			Note         string  `json:"note"`
			ReportID     string  `json:"report_id"`
			Ended        bool    `json:"ended"`
			ReinstatedAt *string `json:"reinstated_at"`
		} `json:"last"`
	} `json:"readers"`
	HasMore       bool `json:"has_more"`
	CurrentCount  int  `json:"current_count"`
	PreviousCount int  `json:"previous_count"`
}

func (a *api) suspendedList(admin user, query string) suspensionsList {
	a.t.Helper()
	var out suspensionsList
	a.must(http.StatusOK, "GET", "/admin/suspensions"+query, &admin, nil, &out)
	return out
}

func names(l suspensionsList) []string {
	out := []string{}
	for _, r := range l.Readers {
		out = append(out, r.Username)
	}
	return out
}

// portal-moderation R2.1, R2.2, R2.3, R2.6, R8.2: the Suspended tab's two
// lists, their order, counts and paging.
func TestSuspensionsList(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	reader := a.user("reader")
	a.must(http.StatusUnauthorized, "GET", "/admin/suspensions", nil, nil, nil)
	a.must(http.StatusForbidden, "GET", "/admin/suspensions", &reader, nil, nil)
	a.must(http.StatusBadRequest, "GET", "/admin/suspensions?state=later", &admin, nil, nil)

	// Suspended now: older, newer, and one from before history began.
	older, newer, undated := a.user("older"), a.user("newer"), a.user("undated")
	a.setActive(http.StatusOK, admin, older, false, "")
	a.setActive(http.StatusOK, admin, newer, false, "spam again")
	a.exec(`UPDATE user_suspensions SET suspended_at = now() - interval '2 days' WHERE user_id = $1`, older.ID)
	a.exec(`UPDATE users SET isactive = false WHERE uuid = $1`, undated.ID)
	a.exec(`INSERT INTO user_suspensions (id, user_id, source) VALUES ('su_old', $1, 'backfill')`, undated.ID)

	// Reinstated: twice-suspended (reinstated last), once, and one whose
	// reinstatement date isn't known.
	twice, once, unknown := a.user("twice"), a.user("once"), a.user("unknown")
	a.setActive(http.StatusOK, admin, twice, false, "")
	a.setActive(http.StatusOK, admin, twice, true, "")
	a.setActive(http.StatusOK, admin, once, false, "")
	a.setActive(http.StatusOK, admin, once, true, "")
	a.exec(`UPDATE user_suspensions SET reinstated_at = now() - interval '1 day' WHERE user_id = $1`, once.ID)
	a.setActive(http.StatusOK, admin, twice, false, "")
	a.setActive(http.StatusOK, admin, twice, true, "")
	a.exec(`INSERT INTO user_suspensions (id, user_id, ended, source) VALUES ('su_gone', $1, true, 'backfill')`, unknown.ID)

	cur := a.suspendedList(admin, "")
	if got := strings.Join(names(cur), ","); got != "newer,older,undated" {
		t.Errorf("current %s, want newest first and the undated one last", got)
	}
	if cur.CurrentCount != 3 || cur.PreviousCount != 3 {
		t.Errorf("counts %d/%d, want 3/3 (the Deleted reader isn't suspended)", cur.CurrentCount, cur.PreviousCount)
	}
	first := cur.Readers[0]
	if first.SuspensionCount != 1 || first.Last == nil || first.Last.SuspendedAt == nil || first.Last.Note != "spam again" ||
		first.Last.SuspendedBy == nil || first.Last.SuspendedBy.Username != "mod" || first.Last.Ended {
		t.Errorf("newer %+v, want one open suspension by mod with its note", first)
	}
	if u := cur.Readers[2]; u.Last == nil || u.Last.SuspendedAt != nil || u.Last.SuspendedBy != nil {
		t.Errorf("undated %+v, want no date and no admin", u)
	}

	prev := a.suspendedList(admin, "?state=previous")
	if got := strings.Join(names(prev), ","); got != "twice,once,unknown" {
		t.Errorf("previous %s, want most recently reinstated first and the unknown date last", got)
	}
	if prev.Readers[0].SuspensionCount != 2 || prev.Readers[0].Last.ReinstatedAt == nil || !prev.Readers[0].Last.Ended {
		t.Errorf("twice %+v, want 2 suspensions, last one reinstated", prev.Readers[0])
	}

	page := a.suspendedList(admin, "?state=previous&limit=2")
	rest := a.suspendedList(admin, "?state=previous&limit=2&offset=2")
	if len(page.Readers) != 2 || !page.HasMore || len(rest.Readers) != 1 || rest.HasMore {
		t.Errorf("pages of %d (more %v) and %d (more %v), want 2 then 1", len(page.Readers), page.HasMore, len(rest.Readers), rest.HasMore)
	}
}

// portal-moderation R3.1, R8.2: one reader's history, newest first, with
// the admins' names and notes.
func TestUserSuspensions(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	other := a.userWithRole("mod2", "admin")
	reader := a.user("reader")
	path := "/admin/users/" + reader.ID + "/suspensions"
	a.must(http.StatusUnauthorized, "GET", path, nil, nil, nil)
	a.must(http.StatusForbidden, "GET", path, &reader, nil, nil)
	a.must(http.StatusNotFound, "GET", "/admin/users/6f1c3a56-0000-4000-8000-000000000000/suspensions", &admin, nil, nil)
	a.must(http.StatusNotFound, "GET", "/admin/users/nope/suspensions", &admin, nil, nil)

	var none struct {
		Count       int   `json:"count"`
		Suspensions []any `json:"suspensions"`
	}
	a.must(http.StatusOK, "GET", path, &admin, nil, &none)
	if none.Count != 0 || none.Suspensions == nil || len(none.Suspensions) != 0 {
		t.Errorf("never suspended: %+v, want count 0 and an empty list", none)
	}

	a.exec(`INSERT INTO user_suspensions (id, user_id, ended, source) VALUES ('su_old', $1, true, 'backfill')`, reader.ID)
	a.setActive(http.StatusOK, admin, reader, false, "first")
	a.setActive(http.StatusOK, other, reader, true, "appealed")
	a.setActive(http.StatusOK, other, reader, false, "second")

	type person struct {
		Username string `json:"username"`
	}
	var got struct {
		Count       int `json:"count"`
		Suspensions []struct {
			SuspendedAt   *string `json:"suspended_at"`
			SuspendedBy   *person `json:"suspended_by"`
			Note          string  `json:"note"`
			Ended         bool    `json:"ended"`
			ReinstatedBy  *person `json:"reinstated_by"`
			ReinstateNote string  `json:"reinstate_note"`
			Source        string  `json:"source"`
		} `json:"suspensions"`
	}
	a.must(http.StatusOK, "GET", path, &admin, nil, &got)
	if got.Count != 3 || len(got.Suspensions) != 3 {
		t.Fatalf("count %d, want 3", got.Count)
	}
	s := got.Suspensions
	if s[0].Note != "second" || s[0].Ended || s[0].SuspendedBy == nil || s[0].SuspendedBy.Username != "mod2" {
		t.Errorf("newest %+v, want the open one by mod2", s[0])
	}
	if s[1].Note != "first" || !s[1].Ended || s[1].ReinstatedBy == nil || s[1].ReinstatedBy.Username != "mod2" || s[1].ReinstateNote != "appealed" {
		t.Errorf("middle %+v, want the first, reinstated by mod2 with the note", s[1])
	}
	if s[2].Source != "backfill" || s[2].SuspendedAt != nil || s[2].SuspendedBy != nil {
		t.Errorf("oldest %+v, want the undated backfill", s[2])
	}
}
