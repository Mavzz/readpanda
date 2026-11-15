package server_test

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// rawReport inserts a report as it would be after its life so far: about
// targetUser (nil once their account is gone), with status, made ago
// (a Postgres interval) before now.
func (a *api) rawReport(id, targetType, targetID string, targetUser any, reason, status, ago string) {
	a.t.Helper()
	a.exec(
		`INSERT INTO content_reports (id, target_type, target_id, target_user_id, reason, snapshot, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, now() - $8::interval)`,
		id, targetType, targetID, targetUser, reason, "copy of "+id, status, ago,
	)
}

type reasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type targetList struct {
	Targets []struct {
		User *struct {
			ID              string `json:"id"`
			Username        string `json:"username"`
			Suspended       bool   `json:"suspended"`
			SuspensionCount int    `json:"suspension_count"`
		} `json:"user"`
		Room *struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"room"`
		Admin *struct {
			Username string `json:"username"`
		} `json:"admin"`
		Total  int `json:"total"`
		Open   int `json:"open"`
		ByType *struct {
			User    int `json:"user"`
			Comment int `json:"comment"`
			Room    int `json:"room"`
		} `json:"by_type"`
		TopReasons     []reasonCount `json:"top_reasons"`
		LastReportedAt string        `json:"last_reported_at"`
	} `json:"targets"`
	HasMore bool `json:"has_more"`
}

func (a *api) targets(admin user, path string) targetList {
	a.t.Helper()
	var out targetList
	a.must(http.StatusOK, "GET", path, &admin, nil, &out)
	return out
}

func targetNames(l targetList) string {
	out := []string{}
	for _, t := range l.Targets {
		if t.User != nil {
			out = append(out, t.User.Username)
		} else {
			out = append(out, t.Room.Name)
		}
	}
	return strings.Join(out, ",")
}

// portal-moderation R4.1, R4.2, R4.3, R4.6, R4.7, R8.2: reports grouped by
// the reader they're about.
func TestReportedReaders(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	alice, bob, carol := a.user("alice"), a.user("bob"), a.user("carol")
	path := "/admin/reports/targets?type=user"
	a.must(http.StatusUnauthorized, "GET", path, nil, nil, nil)
	a.must(http.StatusForbidden, "GET", path, &alice, nil, nil)
	for _, bad := range []string{"?type=book", "?type=user&status=closed", "?type=user&sort=best"} {
		a.must(http.StatusBadRequest, "GET", "/admin/reports/targets"+bad, &admin, nil, nil)
	}

	// alice: her profile, a comment, a room she runs; four reasons.
	a.rawReport("a1", "user", alice.ID, alice.ID, "spam", "open", "3 days")
	a.rawReport("a2", "user", alice.ID, alice.ID, "spam", "open", "3 days")
	a.rawReport("a3", "user", alice.ID, alice.ID, "hate", "actioned", "3 days")
	a.rawReport("a4", "comment", "c1", alice.ID, "harassment", "open", "2 days")
	a.rawReport("a5", "room", "r1", alice.ID, "spam", "dismissed", "3 days")
	a.rawReport("a6", "comment", "c2", alice.ID, "other", "dismissed", "3 days")
	// bob: nothing open; carol: one open, the latest.
	a.rawReport("b1", "user", bob.ID, bob.ID, "spam", "dismissed", "1 day")
	a.rawReport("c1", "user", carol.ID, carol.ID, "spam", "open", "1 minute")
	// About an account that's since been deleted.
	a.rawReport("g1", "user", "6f1c3a56-0000-4000-8000-000000000000", nil, "spam", "open", "1 minute")
	a.setActive(http.StatusOK, admin, alice, false, "")

	open := a.targets(admin, path)
	if got := targetNames(open); got != "alice,carol" {
		t.Errorf("open targets %s, want alice,carol (bob has none open, the deleted account isn't listed)", got)
	}
	al := open.Targets[0]
	if al.Total != 6 || al.Open != 3 || al.ByType == nil || *al.ByType != (struct {
		User    int `json:"user"`
		Comment int `json:"comment"`
		Room    int `json:"room"`
	}{3, 2, 1}) {
		t.Errorf("alice %+v by type %+v, want 6 total, 3 open, 3/2/1", al, al.ByType)
	}
	want := []reasonCount{{"spam", 3}, {"harassment", 1}, {"hate", 1}}
	if len(al.TopReasons) != 3 || al.TopReasons[0] != want[0] || al.TopReasons[1] != want[1] || al.TopReasons[2] != want[2] {
		t.Errorf("alice's reasons %+v, want %+v", al.TopReasons, want)
	}
	if !al.User.Suspended || al.User.SuspensionCount != 1 || al.LastReportedAt == "" {
		t.Errorf("alice %+v, want suspended once, with a last-reported time", al.User)
	}

	for sort, want := range map[string]string{
		"open":   "alice,carol,bob",
		"total":  "alice,carol,bob", // carol and bob tie on one; carol's is later
		"latest": "carol,bob,alice",
	} {
		if got := targetNames(a.targets(admin, path+"&status=all&sort="+sort)); got != want {
			t.Errorf("sort %s: %s, want %s", sort, got, want)
		}
	}

	page := a.targets(admin, path+"&status=all&limit=2")
	rest := a.targets(admin, path+"&status=all&limit=2&offset=2")
	if targetNames(page) != "alice,carol" || !page.HasMore || targetNames(rest) != "bob" || rest.HasMore {
		t.Errorf("pages %s (more %v) and %s (more %v)", targetNames(page), page.HasMore, targetNames(rest), rest.HasMore)
	}
}

// portal-moderation R5.1, R5.2: reports grouped by room, a deleted one
// named from its stored copy.
func TestReportedRooms(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	alice, bob := a.user("alice"), a.user("bob")
	club := a.newRoom(alice, "Club")

	a.rawReport("r1", "room", club.ID, alice.ID, "spam", "open", "2 days")
	a.rawReport("r2", "room", club.ID, alice.ID, "spam", "open", "1 day")
	a.rawReport("r3", "room", club.ID, alice.ID, "hate", "actioned", "3 days")
	a.exec(
		`INSERT INTO content_reports (id, target_type, target_id, target_user_id, reason, snapshot, status, created_at)
		 VALUES ('r4', 'room', 'rm_gone', $1, 'sexual', E'Old name\n\nWhat it was about', 'actioned', now() - interval '4 days'),
		        ('r5', 'room', 'rm_gone', $1, 'sexual', E'Older name\n\nBefore a rename', 'actioned', now() - interval '5 days')`,
		bob.ID,
	)
	// Not a room report: never a room row.
	a.rawReport("u1", "user", alice.ID, alice.ID, "spam", "open", "1 day")

	path := "/admin/reports/targets?type=room"
	open := a.targets(admin, path)
	if targetNames(open) != "Club" {
		t.Fatalf("open rooms %s, want Club", targetNames(open))
	}
	c := open.Targets[0]
	if c.Room.ID != club.ID || c.Room.State != "invite_only" || c.Total != 3 || c.Open != 2 ||
		c.Admin == nil || c.Admin.Username != "alice" || c.ByType != nil || c.User != nil {
		t.Errorf("Club %+v room %+v admin %+v", c, c.Room, c.Admin)
	}
	if len(c.TopReasons) != 2 || c.TopReasons[0] != (reasonCount{"spam", 2}) || c.TopReasons[1] != (reasonCount{"hate", 1}) {
		t.Errorf("Club's reasons %+v", c.TopReasons)
	}

	all := a.targets(admin, path+"&status=all&sort=total")
	if targetNames(all) != "Club,Old name" {
		t.Fatalf("all rooms %s, want Club then the deleted one, named from its latest report", targetNames(all))
	}
	if g := all.Targets[1]; g.Room.State != "gone" || g.Room.ID != "rm_gone" || g.Admin == nil || g.Admin.Username != "bob" {
		t.Errorf("deleted room %+v admin %+v, want gone, run by bob", g.Room, g.Admin)
	}

	a.setOpen(alice, club, true)
	if s := a.targets(admin, path).Targets[0].Room.State; s != "open" {
		t.Errorf("state %s, want open", s)
	}
}

func reportIDs(l adminReportList) string {
	out := []string{}
	for _, r := range l.Reports {
		out = append(out, r.ID)
	}
	return strings.Join(out, ",")
}

// portal-moderation R6.1: the Queue narrowed to one reader or room, under
// any status, with the nav badge's count left alone.
func TestQueueAboutFilter(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	alice, bob := a.user("alice"), a.user("bob")
	a.rawReport("a1", "user", alice.ID, alice.ID, "spam", "open", "3 days")
	a.rawReport("a2", "comment", "c1", alice.ID, "spam", "dismissed", "2 days")
	a.rawReport("a3", "room", "rm_a", alice.ID, "spam", "open", "1 day")
	a.rawReport("b1", "user", bob.ID, bob.ID, "spam", "open", "1 hour")
	a.rawReport("b2", "room", "rm_b", bob.ID, "hate", "actioned", "2 hours")

	get := func(query string) adminReportList {
		t.Helper()
		var list adminReportList
		a.must(http.StatusOK, "GET", "/admin/reports"+query, &admin, nil, &list)
		return list
	}
	for query, want := range map[string]string{
		"?status=all&about_user=" + alice.ID:  "a3,a2,a1",
		"?status=open&about_user=" + alice.ID: "a3,a1",
		"?status=all&about_room=rm_b":         "b2",
		"?status=open&about_room=rm_b":        "",
		"?status=all&about_room=" + alice.ID:  "", // a reader's id isn't a room
		"?status=all&about_user=nope":         "",
	} {
		got := get(query)
		if reportIDs(got) != want {
			t.Errorf("%s: %s, want %s", query, reportIDs(got), want)
		}
		if got.OpenCount != 3 {
			t.Errorf("%s: open_count %d, want 3 for the badge", query, got.OpenCount)
		}
	}
	a.must(http.StatusBadRequest, "GET", "/admin/reports?about_user="+alice.ID+"&about_room=rm_a", &admin, nil, nil)
}

// statsReport inserts a report made at hours after this week's Monday
// (negative for earlier), resolved resolvedAfter hours later if not nil.
func (a *api) statsReport(id, reason, status string, action any, hours float64, resolvedAfter any) {
	a.t.Helper()
	a.exec(
		`INSERT INTO content_reports (id, target_type, target_id, reason, snapshot, status, action, created_at, resolved_at)
		 SELECT $1, 'user', $1, $2, 'x', $3, $4, m + $5 * interval '1 hour',
		        m + ($5 + $6::float8) * interval '1 hour'
		   FROM (SELECT date_trunc('week', localtimestamp) AS m) w`,
		id, reason, status, action, hours, resolvedAfter,
	)
}

type statsResponse struct {
	RangeStart string `json:"range_start"`
	Weeks      []struct {
		WeekStart  string `json:"week_start"`
		Received   int    `json:"received"`
		Resolved   int    `json:"resolved"`
		Suspended  int    `json:"suspended"`
		Reinstated int    `json:"reinstated"`
	} `json:"weeks"`
	Totals struct {
		Received, Resolved, Suspended, Reinstated int
	} `json:"totals"`
	OpenNow      int            `json:"open_now"`
	OldestOpenAt *string        `json:"oldest_open_at"`
	Actions      map[string]int `json:"actions"`
	Reasons      map[string]int `json:"reasons"`
	ResolveHours struct {
		Median *float64 `json:"median"`
		P90    *float64 `json:"p90"`
	} `json:"resolve_hours"`
	UndatedSuspensions int `json:"undated_suspensions"`
}

func (a *api) stats(admin user, query string) statsResponse {
	a.t.Helper()
	var out statsResponse
	a.must(http.StatusOK, "GET", "/admin/moderation/stats"+query, &admin, nil, &out)
	return out
}

// portal-moderation R7.1, R7.6, R7.7, R7.8, R8.2: an empty range is every
// week at zero, Monday-start, with no resolve times.
func TestModerationStatsEmpty(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	reader := a.user("reader")
	a.must(http.StatusUnauthorized, "GET", "/admin/moderation/stats", nil, nil, nil)
	a.must(http.StatusForbidden, "GET", "/admin/moderation/stats", &reader, nil, nil)
	a.must(http.StatusBadRequest, "GET", "/admin/moderation/stats?weeks=7", &admin, nil, nil)

	if n := len(a.stats(admin, "").Weeks); n != 12 {
		t.Errorf("default %d weeks, want 12", n)
	}
	for _, weeks := range []int{4, 52} {
		s := a.stats(admin, "?weeks="+strconv.Itoa(weeks))
		if len(s.Weeks) != weeks || s.RangeStart != s.Weeks[0].WeekStart {
			t.Fatalf("%d weeks: got %d starting %s / %s", weeks, len(s.Weeks), s.RangeStart, s.Weeks[0].WeekStart)
		}
		for i, wk := range s.Weeks {
			d, err := time.Parse("2006-01-02", wk.WeekStart)
			if err != nil || d.Weekday() != time.Monday || wk.Received+wk.Resolved+wk.Suspended+wk.Reinstated != 0 {
				t.Errorf("week %d %+v, want an empty Monday", i, wk)
			}
			if i > 0 {
				prev, _ := time.Parse("2006-01-02", s.Weeks[i-1].WeekStart)
				if d.Sub(prev) != 7*24*time.Hour {
					t.Errorf("week %d starts %s, not a week after %s", i, wk.WeekStart, s.Weeks[i-1].WeekStart)
				}
			}
		}
		if s.ResolveHours.Median != nil || s.ResolveHours.P90 != nil || s.OldestOpenAt != nil || s.OpenNow != 0 {
			t.Errorf("%d weeks: %+v, want no resolve times and nothing open", weeks, s)
		}
		if len(s.Actions) != 5 || len(s.Reasons) != 7 {
			t.Errorf("actions %v reasons %v, want every one listed at zero", s.Actions, s.Reasons)
		}
	}
}

// portal-moderation R7.2, R7.3, R7.4, R7.5, R7.7, R7.8: counts land in the
// right week and only the range counts.
func TestModerationStats(t *testing.T) {
	a := newAPI(t)
	admin := a.userWithRole("mod", "admin")
	reader := a.user("reader")
	const week = 168.0

	a.statsReport("old", "sexual", "actioned", "close_room", -5*week, 1.0) // before the range
	a.statsReport("x", "spam", "dismissed", "dismiss", -3*week+24, 2.0)    // week 0
	a.statsReport("y", "spam", "actioned", "suspend_user", -week+24, 10.0) // week 2
	a.statsReport("z", "hate", "open", nil, -week+48, nil)                 // week 2, oldest open
	a.statsReport("b2", "spam", "open", nil, -0.000001, nil)               // week 2, just before Monday
	a.statsReport("b1", "spam", "open", nil, 0, nil)                       // this week, on the dot
	a.statsReport("w", "spam", "actioned", "remove_comment", 0.0167, 4.0)  // this week

	a.exec(`INSERT INTO user_suspensions (id, user_id, suspended_at, ended, reinstated_at, source)
	        SELECT 's1', $1, m - interval '6 days', true, m + interval '1 minute', 'users'
	          FROM (SELECT date_trunc('week', localtimestamp) AS m) w`, reader.ID)
	a.exec(`INSERT INTO user_suspensions (id, user_id, suspended_at, ended, source)
	        SELECT 's2', $1, m - interval '10 weeks', true, 'users'
	          FROM (SELECT date_trunc('week', localtimestamp) AS m) w`, reader.ID)
	a.exec(`INSERT INTO user_suspensions (id, user_id, ended, source) VALUES ('s3', $1, true, 'backfill')`, reader.ID)

	s := a.stats(admin, "?weeks=4")
	type counts struct{ rec, res, sus, rei int }
	want := []counts{{1, 1, 0, 0}, {0, 0, 0, 0}, {3, 1, 1, 0}, {2, 1, 0, 1}}
	for i, wk := range s.Weeks {
		if got := (counts{wk.Received, wk.Resolved, wk.Suspended, wk.Reinstated}); got != want[i] {
			t.Errorf("week %d (%s): %+v, want %+v", i, wk.WeekStart, got, want[i])
		}
	}
	if s.Totals.Received != 6 || s.Totals.Resolved != 3 || s.Totals.Suspended != 1 || s.Totals.Reinstated != 1 {
		t.Errorf("totals %+v", s.Totals)
	}
	if s.OpenNow != 3 || s.OldestOpenAt == nil {
		t.Errorf("open now %d, oldest %v, want 3", s.OpenNow, s.OldestOpenAt)
	}
	wantActions := map[string]int{"dismiss": 1, "suspend_user": 1, "remove_comment": 1, "close_room": 0, "delete_room": 0}
	for k, v := range wantActions {
		if s.Actions[k] != v {
			t.Errorf("actions %v, want %v", s.Actions, wantActions)
			break
		}
	}
	if s.Reasons["spam"] != 5 || s.Reasons["hate"] != 1 || s.Reasons["sexual"] != 0 {
		t.Errorf("reasons %v, want spam 5, hate 1, the old sexual one outside the range", s.Reasons)
	}
	// Over 2, 4 and 10 hours.
	m, p := s.ResolveHours.Median, s.ResolveHours.P90
	if m == nil || p == nil || math.Abs(*m-4) > 0.01 || math.Abs(*p-8.8) > 0.01 {
		t.Errorf("resolve hours %v / %v, want median 4, p90 8.8", m, p)
	}
	if s.UndatedSuspensions != 1 {
		t.Errorf("undated %d, want 1", s.UndatedSuspensions)
	}
}
