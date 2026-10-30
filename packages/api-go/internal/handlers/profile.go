package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// Edit profile (app spec coming-soon R5): display name and username for every
// account; email and password only for accounts that sign in with them, since
// Apple and Google own theirs. There's no email to confirm a change with yet,
// so the current password does.

const maxDisplayName = 40

// profile is what the app keeps about the signed-in account beside its
// tokens: returned by the profile routes, and folded into every sign-in.
type profile struct {
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	Email       string  `json:"email"`
	LoginType   string  `json:"login_type"`
}

func loadProfile(userID string) (profile, error) {
	var p profile
	err := database.DB.QueryRow(
		"SELECT uuid, username, display_name, email, COALESCE(login_type::text, '') FROM users WHERE uuid = $1",
		userID,
	).Scan(&p.UserID, &p.Username, &p.DisplayName, &p.Email, &p.LoginType)
	return p, err
}

// withAccount adds the account's id, display name and sign-in kind to a
// sign-in's response (R5.8). A failed read leaves them out rather than
// failing a sign-in that has already succeeded.
func withAccount(resp models.TokenResponse, userID string) models.TokenResponse {
	p, err := loadProfile(userID)
	if err != nil {
		log.Printf("profile for sign-in %s: %v", userID, err)
		return resp
	}
	resp.UserID = p.UserID
	resp.DisplayName = p.DisplayName
	resp.LoginType = p.LoginType
	return resp
}

func writeProfile(w http.ResponseWriter, userID string) {
	p, err := loadProfile(userID)
	if err != nil {
		http.Error(w, `{"error": "Couldn't read the account"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// GetMe is the account as Edit profile shows it. A session from before
// sign-ins carried the account's id and sign-in kind gets them here, rather
// than waiting for its next token refresh.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	writeProfile(w, userID)
}

// PatchMe changes the display name and/or username:
// { display_name?: string, username?: string }. An empty display name clears
// it. A username follows the sign-up rule; taken is 409 with a nearby free
// one, including when it's taken between the app's check and this save.
func (h *UserHandler) PatchMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	var req struct {
		DisplayName *string `json:"display_name"`
		Username    *string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	// The word filter covers a renamed account as it does a new one
	// (report-block R8.1). Both fields are checked before either is saved.
	if req.DisplayName != nil && refusedWords(w, "display_name", *req.DisplayName) {
		return
	}
	if req.Username != nil && refusedWords(w, "username", *req.Username) {
		return
	}

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if len([]rune(name)) > maxDisplayName {
			fieldError(w, http.StatusBadRequest, "display_name", "Use at most 40 characters.")
			return
		}
		var value interface{}
		if name != "" {
			value = name
		}
		if _, err := database.DB.Exec("UPDATE users SET display_name = $2, updated_at = NOW() WHERE uuid = $1", userID, value); err != nil {
			http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
			return
		}
	}

	if req.Username != nil {
		name := strings.ToLower(strings.TrimSpace(*req.Username))
		if !usernamePattern.MatchString(name) {
			fieldError(w, http.StatusBadRequest, "username", "Use 3–20 letters, numbers, dots or underscores.")
			return
		}
		current, err := loadProfile(userID)
		if err != nil {
			http.Error(w, `{"error": "Couldn't read the account"}`, http.StatusInternalServerError)
			return
		}
		if name != current.Username {
			taken, err := usernameTaken(name)
			if err != nil {
				http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
				return
			}
			if !taken {
				_, err = database.DB.Exec("UPDATE users SET username = $2, updated_at = NOW() WHERE uuid = $1", userID, name)
				taken = isUniqueViolation(err)
				if err != nil && !taken {
					http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
					return
				}
			}
			if taken {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "That username is taken.", "field": "username", "suggestion": suggestUsername(name),
				})
				return
			}
		}
	}

	writeProfile(w, userID)
}

// checkPassword is the gate for email and password changes: only accounts
// that sign in with a password have one to give (403 otherwise), and it has to
// be right (403).
func checkPassword(w http.ResponseWriter, userID, password string) bool {
	var hash sql.NullString
	var loginType string
	err := database.DB.QueryRow(
		"SELECT password, COALESCE(login_type::text, '') FROM users WHERE uuid = $1", userID,
	).Scan(&hash, &loginType)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Account not found"}`, http.StatusNotFound)
		return false
	}
	if err != nil {
		http.Error(w, `{"error": "Couldn't read the account"}`, http.StatusInternalServerError)
		return false
	}
	if loginType != models.LoginTypeEmail || !hash.Valid {
		fieldError(w, http.StatusForbidden, "current_password", "This account signs in with Apple or Google.")
		return false
	}
	if !utils.DecryptPassword(password, hash.String) {
		fieldError(w, http.StatusForbidden, "current_password", "That password isn't right.")
		return false
	}
	return true
}

// ChangeEmail: { email, current_password }. 409 when another account has the
// address.
func (h *UserHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	var req struct {
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailPattern.MatchString(email) {
		fieldError(w, http.StatusBadRequest, "email", "Enter a valid email address.")
		return
	}
	if !checkPassword(w, userID, req.CurrentPassword) {
		return
	}
	_, err := database.DB.Exec("UPDATE users SET email = $2, updated_at = NOW() WHERE uuid = $1", userID, email)
	if isUniqueViolation(err) {
		fieldError(w, http.StatusConflict, "email", "That email is already in use.")
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
		return
	}
	writeProfile(w, userID)
}

// ChangePassword: { current_password, new_password }. Other signed-in devices
// stay signed in: there's no email yet to tell the reader they were signed out.
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(req.NewPassword)) < minPasswordLength {
		fieldError(w, http.StatusBadRequest, "new_password", "Use at least 8 characters.")
		return
	}
	if !checkPassword(w, userID, req.CurrentPassword) {
		return
	}
	hash, err := utils.CryptPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
		return
	}
	if _, err := database.DB.Exec("UPDATE users SET password = $2, updated_at = NOW() WHERE uuid = $1", userID, hash); err != nil {
		http.Error(w, `{"error": "Couldn't save"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
