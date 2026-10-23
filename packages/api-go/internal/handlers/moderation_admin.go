package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/notify"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// The portal's moderation (report-block R9). Every endpoint here requires
// the admin role (R9.7).

type reportPerson struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Suspended bool   `json:"suspended"`
}

type reportRoom struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	// "open", "invite_only", or "gone" once deleted.
	State string `json:"state"`
}

// adminReport is one row of the Reports page (R9.1).
type adminReport struct {
	ID         string        `json:"id"`
	TargetType string        `json:"target_type"`
	TargetID   string        `json:"target_id"`
	Reason     string        `json:"reason"`
	Details    string        `json:"details,omitempty"`
	Snapshot   string        `json:"snapshot"`
	Status     string        `json:"status"`
	Action     string        `json:"action,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
	ResolvedBy string        `json:"resolved_by,omitempty"`
	Reporter   *reportPerson `json:"reporter"`
	TargetUser *reportPerson `json:"target_user"`
	Room       *reportRoom   `json:"room"`
	// For a comment: "live", "deleted" (by its author), "removed" (by an
	// admin) or "gone" (deleted with no replies, or with its room).
	CommentState string `json:"comment_state,omitempty"`
}

// ListReports — GET /admin/reports?status=open|actioned|dismissed|all&limit=&offset=
// Newest first; open by default. open_count is for the portal's nav badge.
func (h *AdminHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	switch status {
	case "":
		status = "open"
	case "open", "actioned", "dismissed", "all":
	default:
		adminError(w, http.StatusBadRequest, "Unknown status")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	rows, err := database.DB.Query(
		`SELECT cr.id, cr.target_type, cr.target_id, cr.reason, COALESCE(cr.details, ''), cr.snapshot,
		        cr.status, COALESCE(cr.action, ''), cr.created_at, cr.resolved_at,
		        COALESCE(res.username, ''),
		        rep.uuid, COALESCE(rep.username, ''), COALESCE(rep.isactive IS FALSE, false),
		        tu.uuid, COALESCE(tu.username, ''), COALESCE(tu.isactive IS FALSE, false),
		        cr.room_id, COALESCE(rm.name, ''), rm.is_private,
		        CASE WHEN cr.target_type <> 'comment' THEN ''
		             WHEN c.id IS NULL THEN 'gone'
		             WHEN c.deleted_by = 'moderator' THEN 'removed'
		             WHEN c.deleted_at IS NOT NULL THEN 'deleted'
		             ELSE 'live' END
		   FROM content_reports cr
		   LEFT JOIN users res ON res.uuid = cr.resolved_by
		   LEFT JOIN users rep ON rep.uuid = cr.reporter_id
		   LEFT JOIN users tu ON tu.uuid = cr.target_user_id
		   LEFT JOIN rooms rm ON rm.id = cr.room_id
		   LEFT JOIN book_comments c ON cr.target_type = 'comment' AND c.id = cr.target_id
		  WHERE ($1 = 'all' OR cr.status = $1)
		  ORDER BY cr.created_at DESC, cr.id
		  LIMIT $2 OFFSET $3`,
		status, limit+1, offset,
	)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()

	reports := []adminReport{}
	for rows.Next() {
		var rp adminReport
		var resolvedAt sql.NullTime
		var repID, tuID, roomID sql.NullString
		var repName, tuName, roomName string
		var repSuspended, tuSuspended bool
		var roomPrivate sql.NullBool
		if err := rows.Scan(
			&rp.ID, &rp.TargetType, &rp.TargetID, &rp.Reason, &rp.Details, &rp.Snapshot,
			&rp.Status, &rp.Action, &rp.CreatedAt, &resolvedAt, &rp.ResolvedBy,
			&repID, &repName, &repSuspended, &tuID, &tuName, &tuSuspended,
			&roomID, &roomName, &roomPrivate, &rp.CommentState,
		); err != nil {
			adminDBError(w, err)
			return
		}
		if resolvedAt.Valid {
			rp.ResolvedAt = &resolvedAt.Time
		}
		// A deleted account leaves these null (R3.3).
		if repID.Valid {
			rp.Reporter = &reportPerson{ID: repID.String, Username: repName, Suspended: repSuspended}
		}
		if tuID.Valid {
			rp.TargetUser = &reportPerson{ID: tuID.String, Username: tuName, Suspended: tuSuspended}
		}
		if roomID.Valid {
			rp.Room = &reportRoom{ID: roomID.String, Name: roomName, State: "gone"}
			if roomPrivate.Valid {
				rp.Room.State = "open"
				if roomPrivate.Bool {
					rp.Room.State = "invite_only"
				}
			}
		}
		reports = append(reports, rp)
	}
	if err := rows.Err(); err != nil {
		adminDBError(w, err)
		return
	}
	hasMore := len(reports) > limit
	if hasMore {
		reports = reports[:limit]
	}

	var openCount int
	if err := database.DB.QueryRow(`SELECT count(*) FROM content_reports WHERE status = 'open'`).Scan(&openCount); err != nil {
		adminDBError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reports":    reports,
		"has_more":   hasMore,
		"open_count": openCount,
	})
}

// ResolveReport — POST /admin/reports/{id}/resolve {"action"}
//
//	remove_comment  a comment report: "Removed by ReadPanda", replies kept (R9.2)
//	close_room      a room report: Invite only, out of open-room lists (R9.3)
//	delete_room     a room report: the room and everything in it (R9.3)
//	suspend_user    any report about someone: suspended and signed out (R9.4)
//	dismiss         nothing done (R9.4)
//
// The report, and every other open report on the same thing, closes with the
// action, who took it and when (R9.5). 409 if it's already closed. Once it's
// done, the people it affects are told (R9.8).
func (h *AdminHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	adminID, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	reportID := mux.Vars(r)["id"]
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		adminError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer tx.Rollback()

	var targetType, targetID, status, reason string
	var targetUser sql.NullString
	err = tx.QueryRow(
		`SELECT target_type, target_id, target_user_id, status, reason FROM content_reports WHERE id = $1 FOR UPDATE`,
		reportID,
	).Scan(&targetType, &targetID, &targetUser, &status, &reason)
	if err == sql.ErrNoRows {
		adminError(w, http.StatusNotFound, "Report not found")
		return
	}
	if err != nil {
		adminDBError(w, err)
		return
	}
	if status != "open" {
		adminError(w, http.StatusConflict, "This report is already closed")
		return
	}

	// Who to tell is read before the action: a deleted room takes its name and
	// members with it.
	var notices []moderationNotice
	switch {
	case req.Action == "dismiss":
	case req.Action == "remove_comment" && targetType == "comment":
		if notices, err = commentRemovedNotices(tx, targetID, reason); err == nil {
			err = removeReportedComment(tx, targetID)
		}
	case req.Action == "close_room" && targetType == "room":
		if notices, err = roomNotices(tx, targetID, reason, false); err == nil {
			_, err = tx.Exec(`UPDATE rooms SET is_private = true, updated_at = now() WHERE id = $1`, targetID)
		}
	case req.Action == "delete_room" && targetType == "room":
		// Comments and memberships go with it; reports keep its id and copy.
		if notices, err = roomNotices(tx, targetID, reason, true); err == nil {
			_, err = tx.Exec(`DELETE FROM rooms WHERE id = $1`, targetID)
		}
	case req.Action == "suspend_user":
		if !targetUser.Valid || targetUser.String == deletedReaderUUID {
			adminError(w, http.StatusBadRequest, "That account no longer exists")
			return
		}
		err = setSuspended(tx, targetUser.String, true)
	default:
		adminError(w, http.StatusBadRequest, "That action doesn't fit this report")
		return
	}
	if err != nil {
		log.Printf("moderation: %s on report %s: %v", req.Action, reportID, err)
		adminDBError(w, err)
		return
	}

	newStatus := "actioned"
	if req.Action == "dismiss" {
		newStatus = "dismissed"
	}
	res, err := tx.Exec(
		`UPDATE content_reports
		    SET status = $1, action = $2, resolved_by = $3, resolved_at = now()
		  WHERE status = 'open' AND target_type = $4 AND target_id = $5`,
		newStatus, req.Action, adminID, targetType, targetID,
	)
	if err != nil {
		adminDBError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		adminDBError(w, err)
		return
	}
	closed, _ := res.RowsAffected()
	for _, n := range notices {
		if err := notify.Send(n.userIDs, n.notification); err != nil {
			log.Printf("moderation: telling %d about %s on report %s: %v", len(n.userIDs), req.Action, reportID, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": newStatus, "action": req.Action, "closed": closed})
}

// moderationNotice is one message to the people an admin's action affects
// (R9.8). A SYSTEM notification with no room: a deleted room would take the
// row with it, and a tap only marks it read.
type moderationNotice struct {
	userIDs      []string
	notification notify.Notification
}

const supportLine = " Questions? readpanda.support@gmail.com"

// brokeRules is "broke our community rules", with the report's reason when
// it says more than "Something else".
func brokeRules(verb, reason string) string {
	text := verb + " our community rules"
	if label, ok := reportReasons[reason]; ok && reason != "other" {
		text += " (" + label + ")"
	}
	return text + "."
}

func systemNotice(userIDs []string, title, message string) moderationNotice {
	return moderationNotice{userIDs, notify.Notification{
		Type:    models.NotificationTypeSystem,
		Title:   title,
		Message: message,
	}}
}

// commentRemovedNotices tells the author (R9.8). Not one who'd already
// deleted it, and not the Deleted reader.
func commentRemovedNotices(tx *sql.Tx, commentID, reason string) ([]moderationNotice, error) {
	var authorID string
	var roomName sql.NullString
	var deletedAt sql.NullTime
	err := tx.QueryRow(
		`SELECT c.user_id, r.name, c.deleted_at
		   FROM book_comments c LEFT JOIN rooms r ON r.id = c.room_id
		  WHERE c.id = $1`, commentID,
	).Scan(&authorID, &roomName, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil || deletedAt.Valid || authorID == deletedReaderUUID {
		return nil, err
	}
	where := ""
	if roomName.String != "" {
		where = ` in "` + roomName.String + `"`
	}
	return []moderationNotice{systemNotice([]string{authorID}, "Your comment was removed",
		"Your comment"+where+" "+brokeRules("broke", reason)+supportLine)}, nil
}

// roomNotices tells a room's creator it was closed or deleted, and, when
// it's deleted, its other members too, so it doesn't just vanish (R9.8).
func roomNotices(tx *sql.Tx, roomID, reason string, deleting bool) ([]moderationNotice, error) {
	var name, creatorID string
	var private bool
	err := tx.QueryRow(`SELECT name, admin_id, is_private FROM rooms WHERE id = $1`, roomID).
		Scan(&name, &creatorID, &private)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !deleting {
		// Already invite-only: nothing changes for them.
		if private {
			return nil, nil
		}
		return []moderationNotice{systemNotice([]string{creatorID}, "Your room is invite-only now",
			`"`+name+`" was taken out of open rooms for `+brokeRules("breaking", reason)+supportLine)}, nil
	}

	notices := []moderationNotice{systemNotice([]string{creatorID}, "Your room was removed",
		`"`+name+`" was removed for `+brokeRules("breaking", reason)+supportLine)}
	rows, err := tx.Query(`SELECT user_id FROM room_members WHERE room_id = $1 AND user_id <> $2`, roomID, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		members = append(members, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(members) > 0 {
		notices = append(notices, systemNotice(members, "A room was removed",
			`"`+name+`" was removed for breaking our community rules.`))
	}
	return notices, nil
}

// removeReportedComment is the admin's remove (R9.2). A comment already gone
// needs nothing more; one its author deleted becomes the admin's.
func removeReportedComment(tx *sql.Tx, commentID string) error {
	var parentID sql.NullString
	var deletedAt sql.NullTime
	err := tx.QueryRow(
		`SELECT parent_id, deleted_at FROM book_comments WHERE id = $1 FOR UPDATE`, commentID,
	).Scan(&parentID, &deletedAt)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if deletedAt.Valid {
		_, err = tx.Exec(`UPDATE book_comments SET deleted_by = 'moderator' WHERE id = $1`, commentID)
		return err
	}
	return removeComment(tx, commentID, parentID.String, "moderator")
}

// setSuspended suspends (or reinstates) an account. Suspending also deletes
// its refresh tokens, so the app is signed out at its next refresh (R10.2).
func setSuspended(tx *sql.Tx, userID string, suspended bool) error {
	if _, err := tx.Exec(`UPDATE users SET isactive = $2, updated_at = now() WHERE uuid = $1`, userID, !suspended); err != nil {
		return err
	}
	if !suspended {
		return nil
	}
	_, err := tx.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

// SetUserActive — PATCH /admin/users/{uuid} {"isactive"}: suspend or
// reinstate from the Users page (R9.6, R10.4).
func (h *AdminHandler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	adminID, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	userID := mux.Vars(r)["uuid"]
	var req struct {
		IsActive *bool `json:"isactive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IsActive == nil {
		adminError(w, http.StatusBadRequest, "isactive is required")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		adminError(w, http.StatusNotFound, "User not found")
		return
	}
	if userID == deletedReaderUUID || (userID == adminID && !*req.IsActive) {
		adminError(w, http.StatusBadRequest, "That account can't be suspended")
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE uuid = $1)`, userID).Scan(&exists); err != nil {
		adminDBError(w, err)
		return
	}
	if !exists {
		adminError(w, http.StatusNotFound, "User not found")
		return
	}
	if err := setSuspended(tx, userID, !*req.IsActive); err != nil {
		adminDBError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		adminDBError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"uuid": userID, "isactive": *req.IsActive})
}
