package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

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

	roomID := "rm_" + uuid.New().String()[:8]
	now := time.Now().UTC()

	var inviteCode *string
	var err error
	maxRetries := 5

	if req.IsPrivate {
		for i := 0; i < maxRetries; i++ {
			code, err := utils.GenerateInviteCode()
			if err != nil {
				http.Error(w, `{"error": "Failed to generate invite code"}`, http.StatusInternalServerError)
				return
			}
			// Check if invite code already exists
			var exists bool
			err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE invite_code = $1)", code).Scan(&exists)
			if err != nil {
				http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
				return
			}
			if !exists {
				inviteCode = &code
				break
			}

			if i == maxRetries-1 {
				http.Error(w, `{"error": "Could not generate a unique invite code"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	_, err = database.DB.Exec(
		`INSERT INTO rooms (id,name,description,is_private,invite_code,created_at,updated_at,admin_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		roomID, name, req.Description, req.IsPrivate, inviteCode, now, now, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to create room"}`, http.StatusInternalServerError)
		return
	}

	room := models.Room{
		ID:          roomID,
		Name:        name,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
		InviteCode:  inviteCode,
		CreatedAt:   now,
		UpdatedAt:   now,
		AdminID:     userID,
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

	var rooms []models.RoomeResponse

	for rows.Next() {
		var room models.RoomeResponse
		err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.InviteCode, &room.IsPrivate, &room.AdminID, &room.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "Failed to get room"}`, http.StatusInternalServerError)
			return
		}
		rooms = append(rooms, room)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rooms)
}
