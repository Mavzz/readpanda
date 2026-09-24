package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// isUniqueViolation reports whether err is a Postgres unique-constraint violation.
func isUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505"
}

// RoomHandler handles room-related operations
type RoomHandler struct {
	Config *config.Config
}

func NewRoomHandler(cfg *config.Config) *RoomHandler {
	return &RoomHandler{Config: cfg}
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPrivate   bool   `json:"is_private"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 50 {
		http.Error(w, "name is required and must be 1-50 characters", http.StatusBadRequest)
		return
	}

	// Full UUID — a truncated 8-char prefix has no uniqueness check/retry, and
	// the rooms PK constraint assumes real UUID-level collision resistance.
	roomID := "rm_" + uuid.New().String()
	now := time.Now().UTC()

	// Every room gets an invite code, not just private ones — Room Detail
	// shows the code for any room ("Anyone with the code can join this room").
	maxRetries := 5
	var inviteCode string
	for i := 0; i < maxRetries; i++ {
		code, err := utils.GenerateInviteCode()
		if err != nil {
			http.Error(w, `{"error": "Failed to generate invite code"}`, http.StatusInternalServerError)
			return
		}

		_, err = database.DB.Exec(
			`INSERT INTO rooms (id,name,description,is_private,invite_code,created_at,updated_at,admin_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			roomID, name, req.Description, req.IsPrivate, code, now, now, userID,
		)
		if err == nil {
			inviteCode = code
			break
		}
		// Only retry on an invite_code collision (check-then-insert race);
		// any other error is a real failure.
		if !isUniqueViolation(err) {
			http.Error(w, `{"error": "Failed to create room"}`, http.StatusInternalServerError)
			return
		}
		if i == maxRetries-1 {
			http.Error(w, `{"error": "Could not generate a unique invite code"}`, http.StatusInternalServerError)
			return
		}
	}

	// The creator is a member of their own room (member #1 on Room Detail).
	if _, err := database.DB.Exec(
		`INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'admin', $3)
		 ON CONFLICT (room_id, user_id) DO NOTHING`,
		roomID, userID, now,
	); err != nil {
		http.Error(w, `{"error": "Failed to add creator to room"}`, http.StatusInternalServerError)
		return
	}

	respondRoomCreated(w, roomID, name, req.Description, req.IsPrivate, &inviteCode, userID, now)
}

func respondRoomCreated(w http.ResponseWriter, roomID, name, description string, isPrivate bool, inviteCode *string, adminID string, now time.Time) {
	room := models.Room{
		ID:          roomID,
		Name:        name,
		Description: description,
		IsPrivate:   isPrivate,
		InviteCode:  inviteCode,
		CreatedAt:   now,
		UpdatedAt:   now,
		AdminID:     adminID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

func (h *RoomHandler) GetMyRooms(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	var rows *sql.Rows
	var err error
	rows, err = database.DB.Query(
		`
			SELECT id, name, description, invite_code, is_private, admin_id, created_at 
			FROM rooms 
			WHERE admin_id = $1 OR id IN (SELECT room_id FROM room_members WHERE user_id = $1)`,
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to get rooms"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rooms := []models.Room{}

	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.InviteCode, &room.IsPrivate, &room.AdminID, &room.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "Failed to get room"}`, http.StatusInternalServerError)
			return
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Failed to get rooms"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rooms)
}

// isRoomMember reports whether the user belongs to the room (as creator or member).
func isRoomMember(roomID, userID string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM rooms WHERE id = $1 AND admin_id = $2
			UNION ALL
			SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2
		)`,
		roomID, userID,
	).Scan(&exists)
	return exists, err
}

// loadRoomBucket reads the room's reading list. Buckets live in two tables, so
// bucketType says which one to read from.
func loadRoomBucket(bucketID, bucketType string) (*models.RoomBucket, error) {
	bucket := models.RoomBucket{ID: bucketID, Type: bucketType, Books: []models.BookPreview{}}

	var nameQuery, booksQuery string
	if bucketType == "curated" {
		nameQuery = `SELECT title FROM curated_buckets WHERE id = $1`
		booksQuery = `SELECT b.book_id, b.title, b.cover_image_url, b.manuscript_url
			FROM curated_bucket_books cbb
			JOIN books b ON b.book_id = cbb.book_id
			WHERE cbb.bucket_id = $1
			ORDER BY cbb.sort_order`
	} else {
		nameQuery = `SELECT name FROM user_buckets WHERE id = $1`
		booksQuery = `SELECT b.book_id, b.title, b.cover_image_url, b.manuscript_url
			FROM user_bucket_books ubb
			JOIN books b ON b.book_id = ubb.book_id
			WHERE ubb.bucket_id = $1
			ORDER BY ubb.added_at`
	}

	if err := database.DB.QueryRow(nameQuery, bucketID).Scan(&bucket.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // bucket was deleted out from under the room
		}
		return nil, err
	}

	rows, err := database.DB.Query(booksQuery, bucketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var b models.BookPreview
		if err := rows.Scan(&b.BookID, &b.Title, &b.CoverImageURL, &b.ManuscriptURL); err != nil {
			return nil, err
		}
		bucket.Books = append(bucket.Books, b)
	}
	return &bucket, rows.Err()
}

// GetRoomDetail — GET /room/{id}
// The whole Room Detail screen in one call: the room, what it's reading
// (standalone book, or a current book from a bucket), and its members.
func (h *RoomHandler) GetRoomDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]

	var detail models.RoomDetail
	var currentBookID, currentBucketID, currentBucketType sql.NullString
	err := database.DB.QueryRow(
		`SELECT id, name, description, invite_code, is_private, admin_id, created_at, updated_at,
		        current_book_id, current_bucket_id, current_bucket_type
		 FROM rooms WHERE id = $1`,
		roomID,
	).Scan(
		&detail.ID, &detail.Name, &detail.Description, &detail.InviteCode, &detail.IsPrivate,
		&detail.AdminID, &detail.CreatedAt, &detail.UpdatedAt,
		&currentBookID, &currentBucketID, &currentBucketType,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to get room"}`, http.StatusInternalServerError)
		return
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get room"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	if currentBookID.Valid {
		var book models.BookPreview
		err := database.DB.QueryRow(
			`SELECT book_id, title, cover_image_url, manuscript_url FROM books WHERE book_id = $1`,
			currentBookID.String,
		).Scan(&book.BookID, &book.Title, &book.CoverImageURL, &book.ManuscriptURL)
		if err == nil {
			detail.CurrentBook = &book
		} else if err != sql.ErrNoRows {
			http.Error(w, `{"error": "Failed to get current book"}`, http.StatusInternalServerError)
			return
		}
	}

	if currentBucketID.Valid && currentBucketType.Valid {
		bucket, err := loadRoomBucket(currentBucketID.String, currentBucketType.String)
		if err != nil {
			http.Error(w, `{"error": "Failed to get room bucket"}`, http.StatusInternalServerError)
			return
		}
		detail.Bucket = bucket
	}

	rows, err := database.DB.Query(
		`SELECT m.user_id, u.username, m.role, m.joined_at
		   FROM room_members m
		   JOIN users u ON u.uuid = m.user_id
		  WHERE m.room_id = $1
		  ORDER BY (m.role = 'admin') DESC, m.joined_at`,
		roomID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to get room members"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	detail.Members = []models.RoomMemberDetail{}
	for rows.Next() {
		var m models.RoomMemberDetail
		if err := rows.Scan(&m.UserID, &m.Username, &m.Role, &m.JoinedAt); err != nil {
			http.Error(w, `{"error": "Failed to scan room member"}`, http.StatusInternalServerError)
			return
		}
		detail.Members = append(detail.Members, m)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Failed to get room members"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(detail)
}

// JoinRoom — POST /room/join  {"invite_code": "AB12CD"}
func (h *RoomHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.InviteCode))
	if code == "" {
		http.Error(w, `{"error": "invite_code is required"}`, http.StatusBadRequest)
		return
	}

	var room models.Room
	err := database.DB.QueryRow(
		`SELECT id, name, description, invite_code, is_private, admin_id, created_at, updated_at
		   FROM rooms WHERE invite_code = $1`,
		code,
	).Scan(&room.ID, &room.Name, &room.Description, &room.InviteCode, &room.IsPrivate,
		&room.AdminID, &room.CreatedAt, &room.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "No room found for that code"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to join room"}`, http.StatusInternalServerError)
		return
	}

	already, err := isRoomMember(room.ID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to join room"}`, http.StatusInternalServerError)
		return
	}
	if already {
		http.Error(w, `{"error": "You're already in this room"}`, http.StatusConflict)
		return
	}

	if _, err := database.DB.Exec(
		`INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'reader', $3)
		 ON CONFLICT (room_id, user_id) DO NOTHING`,
		room.ID, userID, time.Now().UTC(),
	); err != nil {
		http.Error(w, `{"error": "Failed to join room"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(room)
}

// SetRoomReading — PATCH /room/{id}/reading
// Body: {"current_book_id": "...", "bucket_id": "...", "bucket_type": "user"|"curated"}
// Sending a bucket without a book, or neither, clears that side. The room's
// creator decides what the room reads.
func (h *RoomHandler) SetRoomReading(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]

	var req struct {
		CurrentBookID *string `json:"current_book_id"`
		BucketID      *string `json:"bucket_id"`
		BucketType    *string `json:"bucket_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	var adminID string
	err := database.DB.QueryRow(`SELECT admin_id FROM rooms WHERE id = $1`, roomID).Scan(&adminID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to update room"}`, http.StatusInternalServerError)
		return
	}
	if adminID != userID {
		http.Error(w, `{"error": "Only the room creator can change what the room reads"}`, http.StatusForbidden)
		return
	}

	if (req.BucketID == nil) != (req.BucketType == nil) {
		http.Error(w, `{"error": "bucket_id and bucket_type must be sent together"}`, http.StatusBadRequest)
		return
	}
	if req.BucketType != nil && *req.BucketType != "user" && *req.BucketType != "curated" {
		http.Error(w, `{"error": "bucket_type must be 'user' or 'curated'"}`, http.StatusBadRequest)
		return
	}

	// Invariant from the design: with a bucket set, the current book comes
	// from that bucket.
	if req.BucketID != nil && req.CurrentBookID != nil {
		bucket, err := loadRoomBucket(*req.BucketID, *req.BucketType)
		if err != nil {
			http.Error(w, `{"error": "Failed to read bucket"}`, http.StatusInternalServerError)
			return
		}
		if bucket == nil {
			http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
			return
		}
		inBucket := false
		for _, b := range bucket.Books {
			if b.BookID == *req.CurrentBookID {
				inBucket = true
				break
			}
		}
		if !inBucket {
			http.Error(w, `{"error": "current_book_id must be a book in the bucket"}`, http.StatusBadRequest)
			return
		}
	}

	if _, err := database.DB.Exec(
		`UPDATE rooms
		    SET current_book_id = $1, current_bucket_id = $2, current_bucket_type = $3, updated_at = $4
		  WHERE id = $5`,
		req.CurrentBookID, req.BucketID, req.BucketType, time.Now().UTC(), roomID,
	); err != nil {
		http.Error(w, `{"error": "Failed to update what the room is reading"}`, http.StatusInternalServerError)
		return
	}

	h.GetRoomDetail(w, r)
}

// DeleteRoom — DELETE /room/{id}
// Only the creator can delete a room. room_members rows cascade with it.
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]

	var adminID string
	err := database.DB.QueryRow(`SELECT admin_id FROM rooms WHERE id = $1`, roomID).Scan(&adminID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to delete room"}`, http.StatusInternalServerError)
		return
	}
	if adminID != userID {
		http.Error(w, `{"error": "Only the room creator can delete this room"}`, http.StatusForbidden)
		return
	}

	if _, err := database.DB.Exec(`DELETE FROM rooms WHERE id = $1`, roomID); err != nil {
		http.Error(w, `{"error": "Failed to delete room"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LeaveRoom — DELETE /room/{id}/members/me
// Anyone but the creator can walk away; the creator deletes the room instead.
func (h *RoomHandler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]

	var adminID string
	err := database.DB.QueryRow(`SELECT admin_id FROM rooms WHERE id = $1`, roomID).Scan(&adminID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to leave room"}`, http.StatusInternalServerError)
		return
	}
	if adminID == userID {
		http.Error(w, `{"error": "The room creator can't leave — delete the room instead"}`, http.StatusForbidden)
		return
	}

	res, err := database.DB.Exec(
		`DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`, roomID, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to leave room"}`, http.StatusInternalServerError)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		http.Error(w, `{"error": "You're not in this room"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
