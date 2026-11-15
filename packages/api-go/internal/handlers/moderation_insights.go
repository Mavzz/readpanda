package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// The portal's moderation overview (portal-moderation): suspension history,
// reports grouped by who or what they're about, and stats. Read-only, and
// admins only (R8.2).

// adminPerson is an admin who acted, or null once their account is gone
// (R1.7).
type adminPerson struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func personOrNil(id, username sql.NullString) *adminPerson {
	if !id.Valid {
		return nil
	}
	return &adminPerson{ID: id.String, Username: username.String}
}

// pageParams reads limit (≤ 100, default 50) and offset.
func pageParams(r *http.Request) (limit, offset int) {
	q := r.URL.Query()
	limit, _ = strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ = strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// A reader's most recent suspension: the open one if they're suspended,
// otherwise the newest. An undated one (from before history began) sorts
// as the oldest unless it's the one still open.
const latestSuspensionOrder = `(NOT s.ended) DESC, s.suspended_at DESC NULLS LAST, s.id DESC`

type lastSuspension struct {
	SuspendedAt  *time.Time   `json:"suspended_at"`
	SuspendedBy  *adminPerson `json:"suspended_by"`
	Note         string       `json:"note,omitempty"`
	ReportID     string       `json:"report_id,omitempty"`
	Ended        bool         `json:"ended"`
	ReinstatedAt *time.Time   `json:"reinstated_at"`
}

type suspendedReader struct {
	ID              string          `json:"id"`
	Username        string          `json:"username"`
	SuspensionCount int             `json:"suspension_count"`
	Last            *lastSuspension `json:"last"`
}

// ListSuspensions — GET /admin/suspensions?state=current|previous&limit=&offset=
// The Suspended tab (R2.1, R2.2): readers suspended now, newest first, or
// readers reinstated since, most recently reinstated first. Both counts
// come every time, for the chips (R2.3).
func (h *AdminHandler) ListSuspensions(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	var where, order string
	switch r.URL.Query().Get("state") {
	case "", "current":
		where = `u.isactive IS FALSE`
		order = `l.suspended_at DESC NULLS LAST`
	case "previous":
		where = `u.isactive IS NOT FALSE AND l.id IS NOT NULL`
		order = `l.reinstated_at DESC NULLS LAST`
	default:
		adminError(w, http.StatusBadRequest, "Unknown state")
		return
	}
	limit, offset := pageParams(r)

	// The Deleted reader is inactive by design, not suspended (R1.8).
	rows, err := database.DB.Query(
		`WITH latest AS (
		     SELECT DISTINCT ON (s.user_id) s.*
		       FROM user_suspensions s
		      ORDER BY s.user_id, `+latestSuspensionOrder+`
		 ), counts AS (
		     SELECT user_id, count(*) AS n FROM user_suspensions GROUP BY user_id
		 )
		 SELECT u.uuid, u.username, COALESCE(c.n, 0),
		        l.id, l.suspended_at, sb.uuid, sb.username, COALESCE(l.note, ''), COALESCE(l.report_id, ''),
		        COALESCE(l.ended, false), l.reinstated_at
		   FROM users u
		   LEFT JOIN latest l ON l.user_id = u.uuid
		   LEFT JOIN counts c ON c.user_id = u.uuid
		   LEFT JOIN users sb ON sb.uuid = l.suspended_by
		  WHERE u.uuid <> $1 AND `+where+`
		  ORDER BY `+order+`, u.username, u.uuid
		  LIMIT $2 OFFSET $3`,
		deletedReaderUUID, limit+1, offset,
	)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()

	readers := []suspendedReader{}
	for rows.Next() {
		var sr suspendedReader
		var lastID, byID, byName sql.NullString
		var suspendedAt, reinstatedAt sql.NullTime
		var last lastSuspension
		if err := rows.Scan(&sr.ID, &sr.Username, &sr.SuspensionCount,
			&lastID, &suspendedAt, &byID, &byName, &last.Note, &last.ReportID, &last.Ended, &reinstatedAt); err != nil {
			adminDBError(w, err)
			return
		}
		// Suspended with nothing on record only happens to data edited by
		// hand; the row still shows, with no details.
		if lastID.Valid {
			last.SuspendedAt = timeOrNil(suspendedAt)
			last.ReinstatedAt = timeOrNil(reinstatedAt)
			last.SuspendedBy = personOrNil(byID, byName)
			sr.Last = &last
		}
		readers = append(readers, sr)
	}
	if err := rows.Err(); err != nil {
		adminDBError(w, err)
		return
	}
	hasMore := len(readers) > limit
	if hasMore {
		readers = readers[:limit]
	}

	var current, previous int
	if err := database.DB.QueryRow(
		`SELECT count(*) FILTER (WHERE u.isactive IS FALSE),
		        count(*) FILTER (WHERE u.isactive IS NOT FALSE
		                           AND EXISTS (SELECT 1 FROM user_suspensions s WHERE s.user_id = u.uuid))
		   FROM users u WHERE u.uuid <> $1`, deletedReaderUUID,
	).Scan(&current, &previous); err != nil {
		adminDBError(w, err)
		return
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{
		"readers":        readers,
		"has_more":       hasMore,
		"current_count":  current,
		"previous_count": previous,
	})
}

type suspensionEntry struct {
	ID            string       `json:"id"`
	SuspendedAt   *time.Time   `json:"suspended_at"`
	SuspendedBy   *adminPerson `json:"suspended_by"`
	ReportID      string       `json:"report_id,omitempty"`
	Note          string       `json:"note,omitempty"`
	Ended         bool         `json:"ended"`
	ReinstatedAt  *time.Time   `json:"reinstated_at"`
	ReinstatedBy  *adminPerson `json:"reinstated_by"`
	ReinstateNote string       `json:"reinstate_note,omitempty"`
	Source        string       `json:"source"`
}

// GetUserSuspensions — GET /admin/users/{uuid}/suspensions
// One reader's history for their user page (R3.1), newest first.
func (h *AdminHandler) GetUserSuspensions(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	userID := mux.Vars(r)["uuid"]
	if _, err := uuid.Parse(userID); err != nil {
		adminError(w, http.StatusNotFound, "User not found")
		return
	}
	var exists bool
	if err := database.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE uuid = $1)`, userID).Scan(&exists); err != nil {
		adminDBError(w, err)
		return
	}
	if !exists {
		adminError(w, http.StatusNotFound, "User not found")
		return
	}

	rows, err := database.DB.Query(
		`SELECT s.id, s.suspended_at, sb.uuid, sb.username, COALESCE(s.report_id, ''), COALESCE(s.note, ''),
		        s.ended, s.reinstated_at, rb.uuid, rb.username, COALESCE(s.reinstate_note, ''), s.source
		   FROM user_suspensions s
		   LEFT JOIN users sb ON sb.uuid = s.suspended_by
		   LEFT JOIN users rb ON rb.uuid = s.reinstated_by
		  WHERE s.user_id = $1
		  ORDER BY `+latestSuspensionOrder,
		userID,
	)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()
	list := []suspensionEntry{}
	for rows.Next() {
		var e suspensionEntry
		var suspendedAt, reinstatedAt sql.NullTime
		var sbID, sbName, rbID, rbName sql.NullString
		if err := rows.Scan(&e.ID, &suspendedAt, &sbID, &sbName, &e.ReportID, &e.Note,
			&e.Ended, &reinstatedAt, &rbID, &rbName, &e.ReinstateNote, &e.Source); err != nil {
			adminDBError(w, err)
			return
		}
		e.SuspendedAt = timeOrNil(suspendedAt)
		e.ReinstatedAt = timeOrNil(reinstatedAt)
		e.SuspendedBy = personOrNil(sbID, sbName)
		e.ReinstatedBy = personOrNil(rbID, rbName)
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		adminDBError(w, err)
		return
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "suspensions": list})
}

func timeOrNil(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// targetSorts orders the Readers and Rooms tabs (R4.3). Ties fall to the
// latest report, then the id, so paging is stable.
var targetSorts = map[string]string{
	"open":   `g.open DESC, g.last DESC, g.tid`,
	"total":  `g.total DESC, g.last DESC, g.tid`,
	"latest": `g.last DESC, g.tid`,
}

// topReasons is the three most-reported reasons for the target in g.tid,
// ties by name, as JSON. where picks that target's reports.
func topReasons(where string) string {
	return `(SELECT COALESCE(json_agg(json_build_object('reason', x.reason, 'count', x.n) ORDER BY x.n DESC, x.reason), '[]'::json)
	           FROM (SELECT c.reason, count(*) AS n FROM content_reports c WHERE ` + where + `
	                  GROUP BY c.reason ORDER BY n DESC, c.reason LIMIT 3) x)`
}

type reportedUser struct {
	ID              string `json:"id"`
	Username        string `json:"username"`
	Suspended       bool   `json:"suspended"`
	SuspensionCount int    `json:"suspension_count"`
}

type reportsByType struct {
	User    int `json:"user"`
	Comment int `json:"comment"`
	Room    int `json:"room"`
}

type userTarget struct {
	User           reportedUser    `json:"user"`
	Total          int             `json:"total"`
	Open           int             `json:"open"`
	ByType         reportsByType   `json:"by_type"`
	TopReasons     json.RawMessage `json:"top_reasons"`
	LastReportedAt time.Time       `json:"last_reported_at"`
}

type roomTarget struct {
	Room           reportRoom      `json:"room"`
	Admin          *reportPerson   `json:"admin"`
	Total          int             `json:"total"`
	Open           int             `json:"open"`
	TopReasons     json.RawMessage `json:"top_reasons"`
	LastReportedAt time.Time       `json:"last_reported_at"`
}

// ListReportTargets — GET /admin/reports/targets?type=user|room&status=open|all&sort=open|total|latest&limit=&offset=
// The Readers and Rooms tabs: one row per reader (R4.1) or room (R5.1)
// with reports about it. A reader's are reports on their profile, their
// comments and the rooms they run, i.e. everything with them as
// target_user_id. status=open keeps only those with open reports (R4.2).
func (h *AdminHandler) ListReportTargets(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	switch status {
	case "":
		status = "open"
	case "open", "all":
	default:
		adminError(w, http.StatusBadRequest, "Unknown status")
		return
	}
	sort := q.Get("sort")
	if sort == "" {
		sort = "open"
	}
	order, ok := targetSorts[sort]
	if !ok {
		adminError(w, http.StatusBadRequest, "Unknown sort")
		return
	}
	limit, offset := pageParams(r)

	var targets []interface{}
	var err error
	switch q.Get("type") {
	case "user":
		targets, err = userTargets(status, order, limit+1, offset)
	case "room":
		targets, err = roomTargets(status, order, limit+1, offset)
	default:
		adminError(w, http.StatusBadRequest, "Unknown type")
		return
	}
	if err != nil {
		adminDBError(w, err)
		return
	}
	hasMore := len(targets) > limit
	if hasMore {
		targets = targets[:limit]
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{"targets": targets, "has_more": hasMore})
}

func userTargets(status, order string, limit, offset int) ([]interface{}, error) {
	// A deleted account's reports lose their target_user_id, so they drop
	// out here and stay in the Queue (R4.6).
	rows, err := database.DB.Query(
		`WITH g AS (
		     SELECT cr.target_user_id AS tid,
		            count(*) AS total,
		            count(*) FILTER (WHERE cr.status = 'open') AS open,
		            count(*) FILTER (WHERE cr.target_type = 'user') AS by_user,
		            count(*) FILTER (WHERE cr.target_type = 'comment') AS by_comment,
		            count(*) FILTER (WHERE cr.target_type = 'room') AS by_room,
		            max(cr.created_at) AS last
		       FROM content_reports cr
		      WHERE cr.target_user_id IS NOT NULL AND cr.target_user_id <> $2
		      GROUP BY cr.target_user_id
		 )
		 SELECT g.tid, u.username, u.isactive IS FALSE,
		        (SELECT count(*) FROM user_suspensions s WHERE s.user_id = g.tid),
		        g.total, g.open, g.by_user, g.by_comment, g.by_room, g.last,
		        `+topReasons(`c.target_user_id = g.tid`)+`
		   FROM g JOIN users u ON u.uuid = g.tid
		  WHERE ($1 = 'all' OR g.open > 0)
		  ORDER BY `+order+`
		  LIMIT $3 OFFSET $4`,
		status, deletedReaderUUID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []interface{}{}
	for rows.Next() {
		var t userTarget
		var reasons []byte
		if err := rows.Scan(&t.User.ID, &t.User.Username, &t.User.Suspended, &t.User.SuspensionCount,
			&t.Total, &t.Open, &t.ByType.User, &t.ByType.Comment, &t.ByType.Room, &t.LastReportedAt, &reasons); err != nil {
			return nil, err
		}
		t.TopReasons = reasons
		out = append(out, t)
	}
	return out, rows.Err()
}

func roomTargets(status, order string, limit, offset int) ([]interface{}, error) {
	// A deleted room keeps its reports, so it keeps its row: named from the
	// latest report's copy, whose first line is the room's name (R5.1). Its
	// admin is whoever the latest report was about.
	rows, err := database.DB.Query(
		`WITH g AS (
		     SELECT cr.target_id AS tid,
		            count(*) AS total,
		            count(*) FILTER (WHERE cr.status = 'open') AS open,
		            max(cr.created_at) AS last
		       FROM content_reports cr
		      WHERE cr.target_type = 'room'
		      GROUP BY cr.target_id
		 )
		 SELECT g.tid, rm.name, rm.is_private, latest.snapshot,
		        tu.uuid, tu.username, COALESCE(tu.isactive IS FALSE, false),
		        g.total, g.open, g.last,
		        `+topReasons(`c.target_type = 'room' AND c.target_id = g.tid`)+`
		   FROM g
		   LEFT JOIN rooms rm ON rm.id = g.tid
		   LEFT JOIN LATERAL (
		       SELECT c.snapshot, c.target_user_id FROM content_reports c
		        WHERE c.target_type = 'room' AND c.target_id = g.tid
		        ORDER BY c.created_at DESC, c.id DESC LIMIT 1
		   ) latest ON true
		   LEFT JOIN users tu ON tu.uuid = latest.target_user_id
		  WHERE ($1 = 'all' OR g.open > 0)
		  ORDER BY `+order+`
		  LIMIT $2 OFFSET $3`,
		status, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []interface{}{}
	for rows.Next() {
		var t roomTarget
		var name, adminID, adminName sql.NullString
		var private sql.NullBool
		var snapshot string
		var adminSuspended bool
		var reasons []byte
		if err := rows.Scan(&t.Room.ID, &name, &private, &snapshot, &adminID, &adminName, &adminSuspended,
			&t.Total, &t.Open, &t.LastReportedAt, &reasons); err != nil {
			return nil, err
		}
		t.TopReasons = reasons
		switch {
		case !private.Valid:
			t.Room.State = "gone"
			t.Room.Name = strings.SplitN(snapshot, "\n", 2)[0]
		case private.Bool:
			t.Room.State, t.Room.Name = "invite_only", name.String
		default:
			t.Room.State, t.Room.Name = "open", name.String
		}
		if adminID.Valid {
			t.Admin = &reportPerson{ID: adminID.String, Username: adminName.String, Suspended: adminSuspended}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type statsWeek struct {
	WeekStart  string `json:"week_start"`
	Received   int    `json:"received"`
	Resolved   int    `json:"resolved"`
	Suspended  int    `json:"suspended"`
	Reinstated int    `json:"reinstated"`
}

type statsTotals struct {
	Received   int `json:"received"`
	Resolved   int `json:"resolved"`
	Suspended  int `json:"suspended"`
	Reinstated int `json:"reinstated"`
}

type resolveHours struct {
	Median *float64 `json:"median"`
	P90    *float64 `json:"p90"`
}

var reportActions = []string{"remove_comment", "close_room", "delete_room", "suspend_user", "dismiss"}

// ModerationStats — GET /admin/moderation/stats?weeks=4|12|52
// The Stats tab (R7). Weeks start on Monday (R7.7) and every week in the
// range is listed, empty or not (R7.6); the last is this unfinished one.
//
// Timestamps are written with now(), in the database session's zone, so
// weeks are cut in that zone too: UTC in production.
func (h *AdminHandler) ModerationStats(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	weeks := 12
	switch r.URL.Query().Get("weeks") {
	case "", "12":
	case "4":
		weeks = 4
	case "52":
		weeks = 52
	default:
		adminError(w, http.StatusBadRequest, "Choose 4, 12 or 52 weeks")
		return
	}

	var start time.Time
	if err := database.DB.QueryRow(
		`SELECT date_trunc('week', localtimestamp) - ($1 - 1) * interval '1 week'`, weeks,
	).Scan(&start); err != nil {
		adminDBError(w, err)
		return
	}

	rows, err := database.DB.Query(
		`SELECT ws,
		        (SELECT count(*) FROM content_reports WHERE created_at >= ws AND created_at < ws + interval '1 week'),
		        (SELECT count(*) FROM content_reports WHERE resolved_at >= ws AND resolved_at < ws + interval '1 week'),
		        (SELECT count(*) FROM user_suspensions WHERE suspended_at >= ws AND suspended_at < ws + interval '1 week'),
		        (SELECT count(*) FROM user_suspensions WHERE reinstated_at >= ws AND reinstated_at < ws + interval '1 week')
		   FROM generate_series($1::timestamp, $1::timestamp + ($2 - 1) * interval '1 week', interval '1 week') AS ws
		  ORDER BY ws`,
		start, weeks,
	)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()
	series := []statsWeek{}
	var totals statsTotals
	for rows.Next() {
		var wk statsWeek
		var ws time.Time
		if err := rows.Scan(&ws, &wk.Received, &wk.Resolved, &wk.Suspended, &wk.Reinstated); err != nil {
			adminDBError(w, err)
			return
		}
		wk.WeekStart = ws.Format("2006-01-02")
		totals.Received += wk.Received
		totals.Resolved += wk.Resolved
		totals.Suspended += wk.Suspended
		totals.Reinstated += wk.Reinstated
		series = append(series, wk)
	}
	if err := rows.Err(); err != nil {
		adminDBError(w, err)
		return
	}

	// What closed reports ended in, and what came in, over the range (R7.5).
	// Every action and reason is listed, at zero if need be.
	actions := map[string]int{}
	for _, a := range reportActions {
		actions[a] = 0
	}
	reasons := map[string]int{}
	for reason := range reportReasons {
		reasons[reason] = 0
	}
	if err := countInto(actions,
		`SELECT action, count(*) FROM content_reports WHERE resolved_at >= $1 AND action IS NOT NULL GROUP BY action`, start); err != nil {
		adminDBError(w, err)
		return
	}
	if err := countInto(reasons,
		`SELECT reason, count(*) FROM content_reports WHERE created_at >= $1 GROUP BY reason`, start); err != nil {
		adminDBError(w, err)
		return
	}

	// Time to resolve, over reports closed in the range (R7.8).
	var median, p90 sql.NullFloat64
	var openNow, undated int
	var oldestOpen sql.NullTime
	if err := database.DB.QueryRow(
		`SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch FROM resolved_at - created_at) / 3600),
		        percentile_cont(0.9) WITHIN GROUP (ORDER BY extract(epoch FROM resolved_at - created_at) / 3600)
		   FROM content_reports WHERE resolved_at >= $1`, start,
	).Scan(&median, &p90); err != nil {
		adminDBError(w, err)
		return
	}
	if err := database.DB.QueryRow(
		`SELECT count(*), min(created_at) FROM content_reports WHERE status = 'open'`,
	).Scan(&openNow, &oldestOpen); err != nil {
		adminDBError(w, err)
		return
	}
	// Suspensions from before history began can't go in any week (R7.4).
	if err := database.DB.QueryRow(
		`SELECT count(*) FROM user_suspensions WHERE suspended_at IS NULL`,
	).Scan(&undated); err != nil {
		adminDBError(w, err)
		return
	}

	hours := resolveHours{}
	if median.Valid {
		hours.Median = &median.Float64
	}
	if p90.Valid {
		hours.P90 = &p90.Float64
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{
		"range_start":         start.Format("2006-01-02"),
		"weeks":               series,
		"totals":              totals,
		"open_now":            openNow,
		"oldest_open_at":      timeOrNil(oldestOpen),
		"actions":             actions,
		"reasons":             reasons,
		"resolve_hours":       hours,
		"undated_suspensions": undated,
	})
}

// countInto adds the (key, count) rows a query returns to m.
func countInto(m map[string]int, query string, args ...interface{}) error {
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return err
		}
		m[k] = n
	}
	return rows.Err()
}
