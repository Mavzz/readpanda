package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
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

// GetUsers retrieves all users
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT id, username, email, isactive, login_type, uuid, created_at FROM users")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.IsActive, &user.LoginType, &user.UUID, &user.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// DecryptAESCryptoJS decrypts AES encrypted data from CryptoJS (frontend)
func DecryptAESCryptoJS(encryptedText, key string) (string, error) {
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	// CryptoJS uses "Salted__" prefix
	if len(ciphertext) < 16 || string(ciphertext[:8]) != "Salted__" {
		return "", fmt.Errorf("invalid ciphertext format")
	}

	// Extract salt
	salt := ciphertext[8:16]
	ciphertext = ciphertext[16:]

	// Derive key and IV using EVP_BytesToKey (OpenSSL compatible)
	keyIV := evpBytesToKey([]byte(key), salt, 32+16, 1) // 32 bytes key + 16 bytes IV
	derivedKey := keyIV[:32]
	iv := keyIV[32:48]

	// Decrypt
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Helper functions for CryptoJS compatibility
func evpBytesToKey(password, salt []byte, keyLen, iterations int) []byte {
	var (
		concat   []byte
		lastHash []byte
	)
	for len(concat) < keyLen {
		// Hash the password + salt (or previous hash + password + salt)
		hash := md5Hash(append(append(lastHash, password...), salt...))
		lastHash = hash
		concat = append(concat, hash...)
	}
	return concat[:keyLen]
}

func md5Hash(data []byte) []byte {
	// For compatibility with CryptoJS, we need MD5
	// WARNING: MD5 is NOT cryptographically secure and is only used here for
	// compatibility with the existing frontend CryptoJS implementation.
	// TODO: Migrate frontend to use PBKDF2, Argon2, or another secure KDF
	hash := md5.Sum(data)
	return hash[:]
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:length-padding], nil
}

// CreateUser creates a new user (signup)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Check if user already exists
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1 OR email = $2", req.Username, req.Email).Scan(&count)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, `{"error": "User already exists"}`, http.StatusConflict)
		return
	}

	// Decrypt the password from frontend
	decryptedPassword, err := DecryptAESCryptoJS(req.Password, h.Config.CryptoSecret)
	if err != nil {
		http.Error(w, `{"error": "Failed to decrypt password"}`, http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := utils.CryptPassword(decryptedPassword)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Generate user UUID
	newUserUID := utils.GenerateUserUID()

	// Insert new user
	var user models.User
	err = tx.QueryRow(
		"INSERT INTO users (username, password, email, isactive, login_type, uuid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username, email",
		req.Username, hashedPassword, req.Email, true, models.LoginTypeEmail, newUserUID,
	).Scan(&user.ID, &user.Username, &user.Email)
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
	accessToken, refreshToken, err := utils.GenerateTokens(newUserUID, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
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

	// Decrypt the password from frontend
	decryptedPassword, err := DecryptAESCryptoJS(req.Password, h.Config.CryptoSecret)
	if err != nil {
		http.Error(w, `{"error": "Failed to decrypt password"}`, http.StatusBadRequest)
		return
	}

	// Check if user exists
	var user models.User
	err = database.DB.QueryRow(
		"SELECT id, username, email, password, uuid FROM users WHERE username = $1",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.UUID)
	if err != nil {
		http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	// Verify password
	if !utils.DecryptPassword(decryptedPassword, user.Password) {
		http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.UUID, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
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

	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	sub := payload.Subject
	picture := payload.Claims["picture"].(string)

	// Check if user exists
	var user models.User
	err = database.DB.QueryRow("SELECT email, username, uuid FROM users WHERE email = $1", email).
		Scan(&user.Email, &user.Username, &user.UUID)

	if err == nil {
		// User exists, generate tokens
		accessToken, refreshToken, err := utils.GenerateTokens(user.UUID, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
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
	err = tx.QueryRow(
		"INSERT INTO users (username, email, isactive, login_type, uuid, google_sub) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username",
		name, email, true, models.LoginTypeSocialGoogle, newUserUID, sub,
	).Scan(&user.ID, &user.Username)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Initialize user preferences
	_, err = initializeUserPreferences(tx, newUserUID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(newUserUID, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
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
		Picture:      picture,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

	// Generate new access token
	accessToken, _, err := utils.GenerateTokens(claims.UserID, h.Config.JWTSecret, h.Config.JWTRefreshSecret)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"accessToken": accessToken,
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
