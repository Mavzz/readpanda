package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"isactive"`
	LoginType string    `json:"login_type"`
	UUID      string    `json:"uuid"`
	GoogleSub *string   `json:"google_sub,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// UserPreferences represents user reading preferences
type UserPreferences struct {
	ID          int                    `json:"id"`
	UserID      string                 `json:"user_id"`
	Preferences map[string]interface{} `json:"preferences"`
}

// Book represents a book in the system
type Book struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Subgenre        string    `json:"subgenre"`
	Genre           string    `json:"genre"`
	CoverImageURL   *string   `json:"cover_image_url,omitempty"`
	ManuscriptURL   *string   `json:"manuscript_url,omitempty"`
	Status          int       `json:"status"`
	Views           int       `json:"views"`
	UserID          *string   `json:"user_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Notification represents a user notification
type Notification struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// Preference represents a genre/subgenre preference
type Preference struct {
	ID       int    `json:"id"`
	Genre    string `json:"genre"`
	Subgenre string `json:"subgenre"`
}

// LoginType constants
const (
	LoginTypeEmail        = "email"
	LoginTypeSocialGoogle = "social_google"
	LoginTypeLDAP         = "ldap"
)

// SignupRequest represents the signup request body
type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse represents the token response
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	Picture      string `json:"picture,omitempty"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// GoogleAuthPayload represents Google OAuth token payload
type GoogleAuthPayload struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Sub     string `json:"sub"`
	Picture string `json:"picture"`
}
