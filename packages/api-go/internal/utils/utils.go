package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CryptPassword hashes a password using bcrypt
func CryptPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// DecryptPassword compares a password with a hash
func DecryptPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateUserUID generates a new UUID for a user
func GenerateUserUID() string {
	return uuid.New().String()
}

// DecryptAES decrypts an AES encrypted string (for password decryption from frontend)
func DecryptAES(encryptedText, key string) (string, error) {
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	// Create cipher
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// Get GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Get nonce size
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptAES encrypts a string using AES (for compatibility)
func EncryptAES(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// JWTClaims represents custom JWT claims
type JWTClaims struct {
	UserID string `json:"userId"`
	Role   string `json:"role,omitempty"`
	Type   string `json:"type,omitempty"`
	jwt.RegisteredClaims
}

// GenerateTokens generates access and refresh tokens. role is embedded in the
// access token only (refresh tokens don't need it — access is re-derived from
// the DB on every /token/refresh call so a role change takes effect promptly).
func GenerateTokens(userID, role, jwtSecret, jwtRefreshSecret string) (accessToken, refreshToken string, err error) {
	// Generate access token (1 hour)
	accessClaims := JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	// Generate refresh token (7 days)
	refreshClaims := JWTClaims{
		UserID: userID,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// VerifyToken verifies and decodes a JWT token
func VerifyToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// CheckToken validates a JWT token
func CheckToken(tokenString, secret string) bool {
	_, err := VerifyToken(tokenString, secret)
	return err == nil
}

// DecodeToken decodes and verifies a JWT token
// Note: This is an alias for VerifyToken kept for API compatibility
func DecodeToken(tokenString, secret string) (*JWTClaims, error) {
	return VerifyToken(tokenString, secret)
}

// extractUserID validates the bearer token and returns the user ID from claims.
func ExtractUserID(w http.ResponseWriter, r *http.Request, secret string) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return "", false
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
		return "", false
	}
	claims, err := DecodeToken(parts[1], secret)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return "", false
	}
	return claims.UserID, true
}

// RequireAdmin validates the bearer token like ExtractUserID, and additionally
// requires the "admin" role claim. Use this instead of ExtractUserID for any
// endpoint that should be restricted to admins (e.g. curated-bucket writes).
func RequireAdmin(w http.ResponseWriter, r *http.Request, secret string) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
		return "", false
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
		return "", false
	}
	claims, err := DecodeToken(parts[1], secret)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return "", false
	}
	if claims.Role != "admin" {
		http.Error(w, `{"error": "Forbidden"}`, http.StatusForbidden)
		return "", false
	}
	return claims.UserID, true
}

const inviteCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateInviteCode generates a random 6-character unique-looking invite code.
func GenerateInviteCode() (string, error) {
	code := make([]byte, 6)
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCharset))))
		if err != nil {
			return "", err
		}
		code[i] = inviteCharset[num.Int64()]
	}
	return string(code), nil
}
