package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/api/idtoken"
)

// UserHandler handles user-related operations
type UserHandler struct {
	Config *config.Config
}

// NewUserHandler creates a new user handler
func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{Config: cfg}
}

// GetUsers retrieves all users. Admin-only: it lists every account's email.
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}

	rows, err := database.DB.Query("SELECT username, email, isactive, login_type, uuid, created_at FROM users")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.Username, &user.Email, &user.IsActive, &user.LoginType, &user.UUID, &user.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Sign-up rules (app spec signup-redesign R3.1, R8.4). New accounts only:
// usernames made before these rules keep working.
var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9._]{3,20}$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const minPasswordLength = 8

// fieldError writes {"error", "field"} so the app can show the message under
// the field it's about (spec R4.1).
func fieldError(w http.ResponseWriter, status int, field, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message, "field": field})
}

// validateSignup checks a sign-up's fields, returning the first field that
// fails and why, or "" when all pass. Username and email are normalised in
// place: trimmed, and lowercase, which is how they're stored and compared.
func validateSignup(req *models.SignupRequest) (string, string) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	switch {
	case !usernamePattern.MatchString(req.Username):
		return "username", "Use 3–20 letters, numbers, dots or underscores."
	case !emailPattern.MatchString(req.Email):
		return "email", "Enter a valid email address."
	case len([]rune(req.Password)) < minPasswordLength:
		return "password", "Use at least 8 characters."
	}
	return "", ""
}

// usernameTaken reports whether a name is in use, ignoring case.
func usernameTaken(name string) (bool, error) {
	var taken bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = $1)", name).Scan(&taken)
	return taken, err
}

// suggestUsername finds a free name near a taken one: name.reads, name_reads,
// then name plus two digits. The base is trimmed so every candidate still
// fits the 20-character rule. "" when nothing nearby is free.
func suggestUsername(name string) string {
	base := name
	if len(base) > 14 {
		base = base[:14]
	}
	candidates := []string{base + ".reads", base + "_reads"}
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(90))
		if err != nil {
			break
		}
		candidates = append(candidates, fmt.Sprintf("%s%d", base, n.Int64()+10))
	}
	for _, c := range candidates {
		if !usernamePattern.MatchString(c) {
			continue
		}
		if taken, err := usernameTaken(c); err == nil && !taken {
			return c
		}
	}
	return ""
}

// UsernameAvailable answers the sign-up form's check when you leave the
// Username field (app spec R3.2): GET /signup/available?username=mira →
// { "available": false, "suggestion": "mira.reads" }. No auth; rate-limited
// where it's routed, since it tells anyone whether a name exists.
func (h *UserHandler) UsernameAvailable(w http.ResponseWriter, r *http.Request) {
	name := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("username")))
	if !usernamePattern.MatchString(name) {
		fieldError(w, http.StatusBadRequest, "username", "Use 3–20 letters, numbers, dots or underscores.")
		return
	}
	taken, err := usernameTaken(name)
	if err != nil {
		http.Error(w, `{"error": "Couldn't check that name"}`, http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{"available": !taken}
	if taken {
		resp["suggestion"] = suggestUsername(name)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// usernameFrom makes a sign-up-valid username from a name or an email's local
// part ("Mira Anand" → "mira.anand", "Zoë" → "zoe"), padded to the 3-character minimum and
// cut to 20. Not checked for uniqueness.
func usernameFrom(source string) string {
	var b strings.Builder
	// Decompose accented letters ("é" → "e" + accent) so the letter survives
	// and only the accent is dropped below.
	for _, r := range norm.NFD.String(strings.ToLower(source)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteRune('.')
		}
	}
	name := strings.Trim(b.String(), "._")
	if len(name) < 3 {
		name += "reader"
	}
	if len(name) > 20 {
		name = strings.TrimRight(name[:20], "._")
	}
	return name
}

// appleKey is the Sign in with Apple key from config; it may be unset.
func (h *UserHandler) appleKey() utils.AppleKey {
	return utils.AppleKey{
		TeamID:     h.Config.AppleTeamID,
		KeyID:      h.Config.AppleKeyID,
		PrivateKey: h.Config.ApplePrivateKey,
		BundleID:   h.Config.AppleBundleID,
	}
}

// keepAppleRefreshToken exchanges a sign-in's authorization code for Apple's
// refresh token and stores it, so deleting the account can revoke it (App
// Store 5.1.1(v)). Best effort: sign-in doesn't fail over it, and without a
// configured key it's skipped (logged once per sign-in).
func (h *UserHandler) keepAppleRefreshToken(userID, code string) {
	key := h.appleKey()
	if code == "" || !key.Configured() {
		if code != "" {
			log.Printf("apple: key not configured, not storing a refresh token for %s", userID)
		}
		return
	}
	token, err := utils.ExchangeAppleCode(key, code)
	if err != nil {
		log.Printf("apple: code exchange failed for %s: %v", userID, err)
		return
	}
	if _, err := database.DB.Exec("UPDATE users SET apple_refresh_token = $1 WHERE uuid = $2", token, userID); err != nil {
		log.Printf("apple: storing refresh token failed for %s: %v", userID, err)
	}
}

// AppleAuth signs in with Apple (app spec signup-redesign R8.1): the app sends
// Apple's identity token; a returning user is found by Apple's id, an existing
// account by its verified email (and linked), and anyone else gets a new
// account. Same response as /auth/google: 200 for a sign-in, 201 for a
// sign-up.
func (h *UserHandler) AppleAuth(w http.ResponseWriter, r *http.Request) {
	var req models.AppleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IdentityToken == "" {
		http.Error(w, `{"error": "identity_token required"}`, http.StatusBadRequest)
		return
	}
	identity, err := utils.VerifyAppleIdentityToken(req.IdentityToken, h.Config.AppleBundleID)
	if err != nil {
		log.Printf("apple: rejected identity token: %v", err)
		http.Error(w, `{"error": "Invalid Apple token"}`, http.StatusUnauthorized)
		return
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))

	var user models.User
	err = database.DB.QueryRow("SELECT email, username, uuid, role FROM users WHERE apple_sub = $1", identity.Subject).
		Scan(&user.Email, &user.Username, &user.UUID, &user.Role)
	if err == sql.ErrNoRows && email != "" && identity.EmailVerified {
		// An account made another way with this email: link it, so Apple
		// signs into it from now on.
		err = database.DB.QueryRow(
			"UPDATE users SET apple_sub = $1 WHERE LOWER(email) = $2 AND apple_sub IS NULL RETURNING email, username, uuid, role",
			identity.Subject, email,
		).Scan(&user.Email, &user.Username, &user.UUID, &user.Role)
	}
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if err == nil {
		accessToken, refreshToken, err := utils.GenerateTokens(user.UUID, user.Role, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if err := storeRefreshToken(database.DB, user.UUID, refreshToken); err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		h.keepAppleRefreshToken(user.UUID, req.AuthorizationCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.TokenResponse{
			Username:     user.Username,
			Email:        user.Email,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		})
		return
	}

	// A new account. Apple shares an email on first sign-in unless the user
	// declined; users.email is required, so there's nothing to create without
	// one.
	if email == "" {
		http.Error(w, `{"error": "Apple didn't share an email. Try again and choose Share My Email or Hide My Email."}`, http.StatusBadRequest)
		return
	}
	// The name if Apple sent one; else the email's local part — unless it's a
	// private relay address, whose local part is random, so just "reader".
	source := strings.TrimSpace(req.FullName)
	if source == "" && !identity.PrivateEmail {
		source = strings.SplitN(email, "@", 2)[0]
	}
	if source == "" {
		source = "reader"
	}
	username := usernameFrom(source)
	if taken, err := usernameTaken(username); err == nil && taken {
		if suggestion := suggestUsername(username); suggestion != "" {
			username = suggestion
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	newUserUID := utils.GenerateUserUID()
	for attempt := 0; ; attempt++ {
		err = tx.QueryRow(
			"INSERT INTO users (username, email, isactive, login_type, uuid, apple_sub) VALUES ($1, $2, $3, $4, $5, $6) RETURNING username",
			username, email, true, models.LoginTypeSocialApple, newUserUID, identity.Subject,
		).Scan(&user.Username)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) || attempt >= 5 {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		suffix, genErr := utils.GenerateInviteCode()
		if genErr != nil {
			http.Error(w, `{"error": "Failed to generate a unique username"}`, http.StatusInternalServerError)
			return
		}
		base := username
		if len(base) > 13 {
			base = base[:13]
		}
		username = base + "_" + strings.ToLower(suffix)
	}

	preferencesData, err := initializeUserPreferences(tx, newUserUID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	accessToken, refreshToken, err := utils.GenerateTokens(newUserUID, models.RoleUser, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if err = storeRefreshToken(tx, newUserUID, refreshToken); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	h.keepAppleRefreshToken(newUserUID, req.AuthorizationCode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.TokenResponse{
		Username:     user.Username,
		Email:        email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Preferences:  preferencesData,
	})
}

// CreateUser creates a new user (signup)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if field, message := validateSignup(&req); field != "" {
		fieldError(w, http.StatusBadRequest, field, message)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Check if user already exists, and say which field clashes so the app can
	// show it under that field. Case-insensitive: older accounts may have
	// mixed-case names.
	var usernameTaken, emailTaken bool
	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = $1), EXISTS(SELECT 1 FROM users WHERE LOWER(email) = $2)",
		req.Username, req.Email,
	).Scan(&usernameTaken, &emailTaken)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if usernameTaken {
		fieldError(w, http.StatusConflict, "username", "That username is taken.")
		return
	}
	if emailTaken {
		fieldError(w, http.StatusConflict, "email", "An account with this email already exists. Log in instead.")
		return
	}

	// Hash the password. Transport is TLS-protected already, so the password
	// arrives as plaintext — no client-side encryption step is needed (it
	// used to be AES-encrypted with a secret baked into the app bundle,
	// which added no real security over TLS while making the login/signup
	// flow depend on both sides agreeing on a matching CryptoJS format).
	hashedPassword, err := utils.CryptPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Generate user UUID
	newUserUID := utils.GenerateUserUID()

	// Insert new user
	var user models.User
	err = tx.QueryRow(
		"INSERT INTO users (username, password, email, isactive, login_type, uuid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING uuid, username, email",
		req.Username, hashedPassword, req.Email, true, models.LoginTypeEmail, newUserUID,
	).Scan(&user.UUID, &user.Username, &user.Email)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Initialize user preferences
	preferencesData, err := initializeUserPreferences(tx, newUserUID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(newUserUID, models.RoleUser, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Store refresh token
	err = storeRefreshToken(tx, newUserUID, refreshToken)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"username":     user.Username,
		"email":        user.Email,
		"preferences":  preferencesData,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// LoginUser handles user login
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Check if user exists
	var user models.User
	err := database.DB.QueryRow(
		// New usernames are stored lowercase (validateSignup), so match case-
		// insensitively, preferring an exact match: older accounts may differ
		// only by case and each must still reach its own.
		"SELECT uuid, username, email, password, role FROM users WHERE username = $1 OR LOWER(username) = LOWER($1) ORDER BY (username = $1) DESC LIMIT 1",
		req.Username,
	).Scan(&user.UUID, &user.Username, &user.Email, &user.Password, &user.Role)
	if err != nil {
		http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	// Verify password. Transport is TLS-protected already, so req.Password
	// arrives as plaintext — no client-side decryption step needed.
	if !utils.DecryptPassword(req.Password, user.Password) {
		http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.UUID, user.Role, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Store refresh token
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	err = storeRefreshToken(tx, user.UUID, refreshToken)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	tx.Commit()

	response := models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     user.Username,
		Email:        user.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GoogleAuth handles Google OAuth authentication
func (h *UserHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
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

	// Verify Google ID token against known client IDs (web and iOS)
	ctx := context.Background()
	payload, err := idtoken.Validate(ctx, token, h.Config.GoogleClientID)
	if err != nil && h.Config.GoogleIOSClientID != "" {
		payload, err = idtoken.Validate(ctx, token, h.Config.GoogleIOSClientID)
	}
	if err != nil {
		http.Error(w, `{"error": "Invalid Google token"}`, http.StatusUnauthorized)
		return
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		http.Error(w, `{"error": "Google token missing email claim"}`, http.StatusUnauthorized)
		return
	}
	// Only trust the email for account lookup/linking if Google has verified
	// it — otherwise an unverified address could be used to log into (or
	// create) an account it doesn't actually own.
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		http.Error(w, `{"error": "Google email not verified"}`, http.StatusUnauthorized)
		return
	}
	name, _ := payload.Claims["name"].(string)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	sub := payload.Subject
	picture, _ := payload.Claims["picture"].(string)

	// Check if user exists
	var user models.User
	err = database.DB.QueryRow("SELECT email, username, uuid, role FROM users WHERE email = $1", email).
		Scan(&user.Email, &user.Username, &user.UUID, &user.Role)

	if err == nil {
		// User exists, generate tokens
		accessToken, refreshToken, err := utils.GenerateTokens(user.UUID, user.Role, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		tx, err := database.DB.Begin()
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		storeRefreshToken(tx, user.UUID, refreshToken)
		tx.Commit()

		response := models.TokenResponse{
			Username:     user.Username,
			Picture:      picture,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create new user
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	newUserUID := utils.GenerateUserUID()
	// The Google display name isn't guaranteed unique and username has a
	// UNIQUE constraint — try the raw name first, then fall back to
	// name+random suffix on collision (same retry-on-conflict shape used for
	// room invite codes).
	username := name
	for attempt := 0; ; attempt++ {
		err = tx.QueryRow(
			"INSERT INTO users (username, email, isactive, login_type, uuid, google_sub) VALUES ($1, $2, $3, $4, $5, $6) RETURNING username",
			username, email, true, models.LoginTypeSocialGoogle, newUserUID, sub,
		).Scan(&user.Username)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) || attempt >= 5 {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		suffix, genErr := utils.GenerateInviteCode()
		if genErr != nil {
			http.Error(w, `{"error": "Failed to generate a unique username"}`, http.StatusInternalServerError)
			return
		}
		username = name + "_" + strings.ToLower(suffix)
	}

	// Initialize user preferences
	preferencesData, err := initializeUserPreferences(tx, newUserUID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(newUserUID, models.RoleUser, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	err = storeRefreshToken(tx, newUserUID, refreshToken)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := models.TokenResponse{
		Username:     user.Username,
		Email:        email,
		Picture:      picture,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Preferences:  preferencesData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// RefreshAccessToken generates a new access token from a refresh token
func (h *UserHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, `{"error": "Refresh token required"}`, http.StatusUnauthorized)
		return
	}

	// Verify refresh token
	claims, err := utils.VerifyToken(req.RefreshToken, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "Invalid or expired refresh token"}`, http.StatusUnauthorized)
		return
	}

	if claims.Type != "refresh" {
		http.Error(w, `{"error": "Invalid token type"}`, http.StatusUnauthorized)
		return
	}

	// Check if refresh token exists in database
	var count int
	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND token = $2 AND expires_at > NOW()",
		claims.UserID, req.RefreshToken,
	).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, `{"error": "Invalid or expired refresh token"}`, http.StatusUnauthorized)
		return
	}

	// Re-fetch the user's current role rather than trusting a role embedded
	// in the refresh token, so a role change takes effect on the very next
	// refresh instead of waiting out the old access token's TTL.
	var role string
	if err := database.DB.QueryRow("SELECT role FROM users WHERE uuid = $1", claims.UserID).Scan(&role); err != nil {
		http.Error(w, `{"error": "User not found"}`, http.StatusUnauthorized)
		return
	}

	// Generate a new access token AND rotate the refresh token — reusing the
	// same refresh token for its full 7-day life means a leaked token stays
	// valid with no way to detect or limit reuse.
	accessToken, newRefreshToken, err := utils.GenerateTokens(claims.UserID, role, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if err := storeRefreshToken(tx, claims.UserID, newRefreshToken); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LogoutUser logs out a user by invalidating the refresh token
func (h *UserHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
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

	// Delete refresh token
	if utils.CheckToken(token, h.Config.JWTRefreshSecret) {
		_, err = database.DB.Exec("DELETE FROM refresh_tokens WHERE token = $1 AND user_id = $2", token, userID)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	response := map[string]string{
		"message": "Logged out successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

type dbExecutor interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func initializeUserPreferences(tx dbExecutor, userID string) (map[string]interface{}, error) {
	preferencesData := make(map[string]interface{})

	rows, err := tx.Query("SELECT id, genre, subgenre FROM preferences ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pref models.Preference
		if err := rows.Scan(&pref.ID, &pref.Genre, &pref.Subgenre); err != nil {
			return nil, err
		}

		if preferencesData[pref.Genre] == nil {
			preferencesData[pref.Genre] = []map[string]interface{}{}
		}

		genrePrefs := preferencesData[pref.Genre].([]map[string]interface{})
		genrePrefs = append(genrePrefs, map[string]interface{}{
			"preference_id":       pref.ID,
			"preference_subgenre": pref.Subgenre,
			"preference_value":    false,
		})
		preferencesData[pref.Genre] = genrePrefs
	}

	// Store preferences in database
	prefsJSON, err := json.Marshal(preferencesData)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("INSERT INTO user_preferences (user_id, preferences) VALUES ($1, $2)", userID, string(prefsJSON))
	if err != nil {
		return nil, err
	}

	return preferencesData, nil
}

func storeRefreshToken(tx dbExecutor, userID, refreshToken string) error {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	var tokenID int
	err := tx.QueryRow(
		"SELECT id FROM refresh_tokens WHERE user_id = $1 AND expires_at > NOW()",
		userID,
	).Scan(&tokenID)

	if err != nil {
		// No existing token, insert new one
		_, err = tx.Exec(
			"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
			userID, refreshToken, expiresAt,
		)
		return err
	}

	// Update existing token
	_, err = tx.Exec(
		"UPDATE refresh_tokens SET token = $1, expires_at = $2 WHERE id = $3",
		refreshToken, expiresAt, tokenID,
	)
	return err
}
