package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// CORS middleware to handle Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware to log incoming requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Incoming %s request to %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates JWT tokens
type AuthMiddleware struct {
	JWTSecret string
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{JWTSecret: jwtSecret}
}

// Authenticate validates the JWT token from the Authorization header
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		if !utils.CheckToken(token, am.JWTSecret) {
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
