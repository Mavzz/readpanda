package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/notify"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// ReportHandler takes readers' reports of comments, readers and rooms
// (report-block R1–R4).
type ReportHandler struct {
	Config *config.Config
}

func NewReportHandler(cfg *config.Config) *ReportHandler {
	return &ReportHandler{Config: cfg}
}

// reportReasons are the reasons the app offers (R1.2), with how the admins'
// notification names them.
var reportReasons = map[string]string{
	"harassment": "Harassment or bullying",
	"hate":       "Hate speech",
	"sexual":     "Sexual content",
	"violence":   "Violence or threats",
	"self_harm":  "Self-harm",
	"spam":       "Spam or scam",
	"other":      "Something else",
}

const (
	maxReportDetails = 500
	// Reports one reader can make in a day (R3.4).
	maxReportsPerDay = 30
)

// reportTarget is what a report is about, as stored: whose it is, the room
// it's in, a copy of its text, and how the admins' notification names it.
type reportTarget struct {
	userID   sql.NullString
	roomID   sql.NullString
	snapshot string
	label    string
}

// CreateReport — POST /reports {target_type, target_id, room_id?, reason, details?}
//
// 201 for a new report, 200 when the reader already has one open on the same
// thing (R1.7). Every admin is told before it answers; that failing never
// loses the report (R4.3).
func (h *ReportHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	if !requireActive(w, userID) {
		return
	}
	var req struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		RoomID     string `json:"room_id"`
		Reason     string `json:"reason"`
		Details    string `json:"details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	reasonLabel, known := reportReasons[req.Reason]
	if !known {
		http.Error(w, `{"error": "Choose a reason"}`, http.StatusBadRequest)
		return
	}
	details := strings.TrimSpace(req.Details)
	if req.Reason == "other" && details == "" {
		http.Error(w, `{"error": "Tell us what's wrong"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(details)) > maxReportDetails {
		http.Error(w, `{"error": "Keep it to 500 characters"}`, http.StatusBadRequest)
		return
	}

	var target reportTarget
	var status int
	switch req.TargetType {
	case "comment":
		target, status = commentReportTarget(userID, req.TargetID)
	case "user":
		target, status = userReportTarget(userID, req.TargetID, req.RoomID)
	case "room":
		target, status = roomReportTarget(userID, req.TargetID)
	default:
		http.Error(w, `{"error": "Unknown target_type"}`, http.StatusBadRequest)
		return
	}
	switch status {
	case 0:
	case http.StatusNotFound:
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	case http.StatusBadRequest:
		// Yourself, your own content, or the Deleted reader (R2.4).
		http.Error(w, `{"error": "You can't report that"}`, http.StatusBadRequest)
		return
	default:
		http.Error(w, `{"error": "Couldn't send your report"}`, http.StatusInternalServerError)
		return
	}

	var today int
	if err := database.DB.QueryRow(
		`SELECT count(*) FROM content_reports WHERE reporter_id = $1 AND created_at > now() - interval '1 day'`,
		userID,
	).Scan(&today); err != nil {
		http.Error(w, `{"error": "Couldn't send your report"}`, http.StatusInternalServerError)
		return
	}
	if today >= maxReportsPerDay {
		http.Error(w, `{"error": "You've sent a lot of reports today. Email us if it's urgent."}`, http.StatusTooManyRequests)
		return
	}

	reportID := "rp_" + uuid.New().String()
	res, err := database.DB.Exec(
		`INSERT INTO content_reports
		     (id, reporter_id, target_type, target_id, target_user_id, room_id, reason, details, snapshot)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)
		 ON CONFLICT (reporter_id, target_type, target_id) WHERE status = 'open' DO NOTHING`,
		reportID, userID, req.TargetType, req.TargetID, target.userID, target.roomID,
		req.Reason, details, target.snapshot,
	)
	if err != nil {
		log.Printf("reports: saving %s %s: %v", req.TargetType, req.TargetID, err)
		http.Error(w, `{"error": "Couldn't send your report"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if n, _ := res.RowsAffected(); n == 0 {
		// Already reported and still open: one report, and no second alert.
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "already_reported"})
		return
	}

	notifyAdmins(reasonLabel, target.label)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": reportID})
}

// commentReportTarget: a live comment in a room the reporter is in, not
// their own and not the Deleted reader's.
func commentReportTarget(reporterID, commentID string) (reportTarget, int) {
	var t reportTarget
	var authorID, roomID, roomName, body string
	err := database.DB.QueryRow(
		`SELECT c.user_id, c.room_id, r.name, c.body
		   FROM book_comments c JOIN rooms r ON r.id = c.room_id
		  WHERE c.id = $1 AND c.deleted_at IS NULL`,
		commentID,
	).Scan(&authorID, &roomID, &roomName, &body)
	if err == sql.ErrNoRows {
		return t, http.StatusNotFound
	}
	if err != nil {
		return t, http.StatusInternalServerError
	}
	member, err := isRoomMember(roomID, reporterID)
	if err != nil {
		return t, http.StatusInternalServerError
	}
	if !member {
		return t, http.StatusNotFound
	}
	if authorID == reporterID || authorID == deletedReaderUUID {
		return t, http.StatusBadRequest
	}
	t.userID = sql.NullString{String: authorID, Valid: true}
	t.roomID = sql.NullString{String: roomID, Valid: true}
	t.snapshot = body
	t.label = "comment in " + roomName
	return t, 0
}

// userReportTarget: another reader the reporter shares a room with. roomID,
// if given, is where it happened and must be one of those rooms.
func userReportTarget(reporterID, targetID, roomID string) (reportTarget, int) {
	var t reportTarget
	if targetID == reporterID || targetID == deletedReaderUUID {
		return t, http.StatusBadRequest
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return t, http.StatusNotFound
	}
	var username string
	var shared sql.NullString
	err := database.DB.QueryRow(
		`WITH rooms_of AS (
		     SELECT id AS room_id, admin_id AS user_id FROM rooms
		     UNION
		     SELECT room_id, user_id FROM room_members
		 )
		 SELECT u.username,
		        (SELECT a.room_id FROM rooms_of a JOIN rooms_of b ON b.room_id = a.room_id
		          WHERE a.user_id = $1 AND b.user_id = u.uuid AND ($3 = '' OR a.room_id = $3)
		          ORDER BY a.room_id LIMIT 1)
		   FROM users u WHERE u.uuid = $2`,
		reporterID, targetID, roomID,
	).Scan(&username, &shared)
	if err == sql.ErrNoRows || (err == nil && !shared.Valid) {
		return t, http.StatusNotFound
	}
	if err != nil {
		return t, http.StatusInternalServerError
	}
	t.userID = sql.NullString{String: targetID, Valid: true}
	if roomID != "" {
		t.roomID = shared
	}
	t.snapshot = username
	t.label = "reader " + username
	return t, 0
}

// roomReportTarget: a room the reporter is in or can find (an open room),
// that they didn't create (R2.2).
func roomReportTarget(reporterID, roomID string) (reportTarget, int) {
	var t reportTarget
	var adminID, name, description string
	var isPrivate bool
	err := database.DB.QueryRow(
		`SELECT admin_id, name, COALESCE(description, ''), is_private FROM rooms WHERE id = $1`, roomID,
	).Scan(&adminID, &name, &description, &isPrivate)
	if err == sql.ErrNoRows {
		return t, http.StatusNotFound
	}
	if err != nil {
		return t, http.StatusInternalServerError
	}
	if adminID == reporterID {
		return t, http.StatusBadRequest
	}
	if isPrivate {
		member, err := isRoomMember(roomID, reporterID)
		if err != nil {
			return t, http.StatusInternalServerError
		}
		if !member {
			return t, http.StatusNotFound
		}
	}
	t.userID = sql.NullString{String: adminID, Valid: true}
	t.roomID = sql.NullString{String: roomID, Valid: true}
	t.snapshot = name
	if description != "" {
		t.snapshot += "\n\n" + description
	}
	t.label = "room " + name
	return t, 0
}

// notifyAdmins tells every admin a report came in (R4): the reason and what
// was reported, never its text (R4.2). Failures are only logged; the report
// is saved and shows in the portal either way (R4.3).
func notifyAdmins(reasonLabel, what string) {
	rows, err := database.DB.Query(`SELECT uuid FROM users WHERE role = 'admin'`)
	if err != nil {
		log.Printf("reports: reading admins: %v", err)
		return
	}
	var admins []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			admins = append(admins, id)
		}
	}
	rows.Close()
	if err := notify.Send(admins, notify.Notification{
		Type:    models.NotificationTypeSystem,
		Title:   "New report",
		Message: reasonLabel + " · " + what,
	}); err != nil {
		log.Printf("reports: notifying admins: %v", err)
	}
}
