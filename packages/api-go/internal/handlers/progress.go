package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// isForeignKeyViolation reports whether err is a Postgres FK violation — here,
// progress published for a book (or by a user) that no longer exists.
func isForeignKeyViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23503"
}

// ProgressHandler handles reading-progress operations.
type ProgressHandler struct {
	Config *config.Config
}

func NewProgressHandler(cfg *config.Config) *ProgressHandler {
	return &ProgressHandler{Config: cfg}
}

// progressPct mirrors the mobile client's arithmetic exactly (see
// readingProgressStore's toActiveBook): the reader is *on* currentPage, so the
// page in front of them counts as read. Both ends have to agree, or a reader's
// own marker would jump the moment the server's number replaced the local one.
func progressPct(currentPage, totalPages int) int {
	if totalPages <= 0 {
		return 0
	}
	pct := int(math.Round(float64(currentPage+1) / float64(totalPages) * 100))
	if pct > 100 {
		return 100
	}
	return pct
}

// PutMyProgress — PUT /progress/{bookId}  {"current_page": 40, "total_pages": 210}
//
// Publishes where the caller is in a book. The mobile app keeps its own
// position locally and reads from there, so this is a broadcast for other
// people's pace tracks rather than a save the reader waits on.
func (h *ProgressHandler) PutMyProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	bookID := mux.Vars(r)["bookId"]
	if bookID == "" {
		http.Error(w, `{"error": "Book id is required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		CurrentPage int `json:"current_page"`
		TotalPages  int `json:"total_pages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.CurrentPage < 0 || req.TotalPages < 0 {
		http.Error(w, `{"error": "Page numbers cannot be negative"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	var stored models.ReadingProgress
	// Backing out of the reader before the PDF reports its length sends a
	// total_pages of 0. That must not erase a length we already knew, or the
	// reader would drop to 0% on everyone else's track — the same guard the
	// client applies to its local copy.
	//
	// furthest_page only ever climbs. It is the line the comment spoiler rule
	// is drawn at, so flipping back to re-read an earlier chapter must not
	// re-lock comments the reader has already been shown.
	err := database.DB.QueryRow(
		`INSERT INTO reading_progress (user_id, book_id, current_page, total_pages, furthest_page, last_read_at, updated_at)
		 VALUES ($1, $2, $3, $4, $3, $5, $5)
		 ON CONFLICT (user_id, book_id) DO UPDATE
		    SET current_page = EXCLUDED.current_page,
		        total_pages  = CASE WHEN EXCLUDED.total_pages > 0
		                            THEN EXCLUDED.total_pages
		                            ELSE reading_progress.total_pages END,
		        furthest_page = GREATEST(reading_progress.furthest_page, EXCLUDED.current_page),
		        last_read_at = EXCLUDED.last_read_at,
		        updated_at   = EXCLUDED.updated_at
		 RETURNING user_id, book_id, current_page, total_pages, furthest_page, last_read_at`,
		userID, bookID, req.CurrentPage, req.TotalPages, now,
	).Scan(&stored.UserID, &stored.BookID, &stored.CurrentPage, &stored.TotalPages, &stored.FurthestPage, &stored.LastReadAt)
	if isForeignKeyViolation(err) {
		http.Error(w, `{"error": "Book not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to save reading progress"}`, http.StatusInternalServerError)
		return
	}
	stored.ProgressPct = progressPct(stored.CurrentPage, stored.TotalPages)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stored)
}

// GetRoomProgress — GET /room/{id}/progress
//
// The pace track for what the room is currently reading: every member of the
// room, whether or not they've opened the book. Members only — a room's pace
// is not public.
func (h *ProgressHandler) GetRoomProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]

	var currentBookID sql.NullString
	err := database.DB.QueryRow(`SELECT current_book_id FROM rooms WHERE id = $1`, roomID).Scan(&currentBookID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to get room progress"}`, http.StatusInternalServerError)
		return
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get room progress"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	progress := models.RoomProgress{RoomID: roomID, Members: []models.MemberProgress{}}
	// Nothing chosen yet: an empty track, not an error. The room page renders
	// this state (it's the room's own first run).
	if !currentBookID.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(progress)
		return
	}
	progress.BookID = currentBookID.String

	// LEFT JOIN, so a member who hasn't opened the book is still on the track
	// at page 0 rather than missing from it. Ordered like GetRoomDetail's
	// member list so the two screens agree on who comes first.
	rows, err := database.DB.Query(
		`SELECT m.user_id, u.username,
		        COALESCE(p.current_page, 0), COALESCE(p.total_pages, 0), p.last_read_at
		   FROM room_members m
		   JOIN users u ON u.uuid = m.user_id
		   LEFT JOIN reading_progress p ON p.user_id = m.user_id AND p.book_id = $2
		  WHERE m.room_id = $1
		  ORDER BY (m.role = 'admin') DESC, m.joined_at`,
		roomID, currentBookID.String,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to get room progress"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var m models.MemberProgress
		var lastReadAt sql.NullTime
		if err := rows.Scan(&m.UserID, &m.Username, &m.CurrentPage, &m.TotalPages, &lastReadAt); err != nil {
			http.Error(w, `{"error": "Failed to scan member progress"}`, http.StatusInternalServerError)
			return
		}
		if lastReadAt.Valid {
			t := lastReadAt.Time
			m.LastReadAt = &t
		}
		m.ProgressPct = progressPct(m.CurrentPage, m.TotalPages)
		progress.Members = append(progress.Members, m)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Failed to get room progress"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(progress)
}
