package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// PreferencesHandler handles user preferences operations
type PreferencesHandler struct {
	Config *config.Config
}

// NewPreferencesHandler creates a new preferences handler
func NewPreferencesHandler(cfg *config.Config) *PreferencesHandler {
	return &PreferencesHandler{Config: cfg}
}

// GetUserPreferences retrieves user preferences
func (h *PreferencesHandler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
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

	var userID string
	err := database.DB.QueryRow("SELECT uuid FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	var preferencesJSON string
	err = database.DB.QueryRow("SELECT preferences FROM user_preferences WHERE user_id = $1", userID).Scan(&preferencesJSON)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "No preferences found for the user. Please set your preferences.",
		})
		return
	}

	// Return the preferences JSON directly
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(preferencesJSON))
}

// UpdateUserPreferences updates user preferences
func (h *PreferencesHandler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
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

	token := strings.Trim(parts[1], `"`)
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, `{"error": "Username required"}`, http.StatusBadRequest)
		return
	}

	var reqBody struct {
		Preferences interface{} `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Get user ID
	var userID string
	err := database.DB.QueryRow("SELECT uuid FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	// Convert preferences to JSON
	preferencesJSON, err := json.Marshal(reqBody.Preferences)
	if err != nil {
		http.Error(w, `{"error": "Failed to encode preferences"}`, http.StatusInternalServerError)
		return
	}

	// Update preferences
	_, err = database.DB.Exec(
		"UPDATE user_preferences SET preferences = $1, updated_at = NOW() WHERE user_id = $2",
		string(preferencesJSON), userID,
	)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "User preferences updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetGenres retrieves all distinct genres
func (h *PreferencesHandler) GetGenres(w http.ResponseWriter, r *http.Request) {
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

	rows, err := database.DB.Query("SELECT DISTINCT genre FROM preferences ORDER BY genre")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	genres := []map[string]string{}
	for rows.Next() {
		var genre string
		if err := rows.Scan(&genre); err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		genres = append(genres, map[string]string{
			"value": genre,
			"label": genre,
		})
	}

	response := map[string]interface{}{
		"genre": genres,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSubgenres retrieves all subgenres
func (h *PreferencesHandler) GetSubgenres(w http.ResponseWriter, r *http.Request) {
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

	rows, err := database.DB.Query("SELECT DISTINCT subgenre, genre FROM preferences ORDER BY subgenre")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	subgenres := []map[string]string{}
	for rows.Next() {
		var subgenre, genre string
		if err := rows.Scan(&subgenre, &genre); err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		subgenres = append(subgenres, map[string]string{
			"value": subgenre,
			"label": subgenre,
			"genre": genre,
		})
	}

	response := map[string]interface{}{
		"subgenre": subgenres,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
