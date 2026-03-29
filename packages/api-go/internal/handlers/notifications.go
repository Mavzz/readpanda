package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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

// GetUserNotifications retrieves all notifications for a user
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
		return
	}

	token := parts[1]
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, `{"error": "Username required"}`, http.StatusBadRequest)
		return
	}

	// Get user ID
	var userID string
	err := database.DB.QueryRow("SELECT uuid FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, user_id, message, is_read, created_at FROM notifications WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.IsRead, &notification.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		notifications = append(notifications, notification)
	}

	if len(notifications) == 0 {
		http.Error(w, `{"message": "No notifications found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GetUnreadNotificationCount retrieves the count of unread notifications
func (h *NotificationHandler) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
		return
	}

	token := parts[1]
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, `{"error": "Username required"}`, http.StatusBadRequest)
		return
	}

	// Get user ID
	var userID string
	err := database.DB.QueryRow("SELECT uuid FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	var count int
	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&count)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"unread_count": count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
