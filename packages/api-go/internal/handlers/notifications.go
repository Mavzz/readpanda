package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// NotificationHandler handles notification operations
type NotificationHandler struct {
	Config *config.Config
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(cfg *config.Config) *NotificationHandler {
	return &NotificationHandler{Config: cfg}
}

// Every route here is "mine": the user comes from the access token. These
// used to take ?username=, which let any signed-in user read anyone's inbox.
// Clients that still send it are unaffected — it is ignored.

// GetUserNotifications — GET /notifications
// The caller's notifications, newest first. An empty inbox is [], not a 404.
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	// created_at is a zoneless timestamp filled with the database's local
	// time. Casting reads it in the session zone, so the JSON carries the real
	// instant instead of local time mislabelled as UTC.
	rows, err := database.DB.Query(
		`SELECT id, user_id, type, title, COALESCE(message, ''), book_id, COALESCE(is_read, false), created_at::timestamptz
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to load notifications"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.BookID, &n.IsRead, &n.CreatedAt); err != nil {
			http.Error(w, `{"error": "Failed to load notifications"}`, http.StatusInternalServerError)
			return
		}
		notifications = append(notifications, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GetUnreadNotificationCount — GET /notifications/unread/count
func (h *NotificationHandler) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	var count int
	err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&count)
	if err != nil {
		http.Error(w, `{"error": "Failed to count notifications"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unread_count": count})
}

// MarkNotificationRead — PUT /notifications/{id}/read
// Idempotent. Someone else's notification is a 404, same as a missing one.
func (h *NotificationHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid notification id"}`, http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(
		"UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to update notification"}`, http.StatusInternalServerError)
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		http.Error(w, `{"error": "Notification not found"}`, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterDevice — POST /users/me/devices  {"token": "<fcm token>", "platform": "ios"}
// Upserts on the token, so a phone that signs into another account moves to it.
func (h *NotificationHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" || len(req.Token) > 4096 {
		http.Error(w, `{"error": "A device token is required"}`, http.StatusBadRequest)
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform != "ios" && platform != "android" {
		platform = "unknown"
	}

	_, err := database.DB.Exec(
		`INSERT INTO device_tokens (token, user_id, platform)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (token) DO UPDATE
		    SET user_id = EXCLUDED.user_id,
		        platform = EXCLUDED.platform,
		        updated_at = CURRENT_TIMESTAMP`,
		req.Token, userID, platform,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to register device"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnregisterDevice — DELETE /users/me/devices/{token}
// Called on sign-out so the next person on this phone doesn't get these
// pushes. Only removes the token if it's the caller's; idempotent either way.
func (h *NotificationHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, `{"error": "A device token is required"}`, http.StatusBadRequest)
		return
	}

	if _, err := database.DB.Exec(
		"DELETE FROM device_tokens WHERE token = $1 AND user_id = $2",
		token, userID,
	); err != nil {
		http.Error(w, `{"error": "Failed to unregister device"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
