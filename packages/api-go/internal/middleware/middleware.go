package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// CORS middleware to handle Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Application-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// maxLoggedErrorBody caps how much of a failed response is copied into the log.
const maxLoggedErrorBody = 1024

// statusRecorder remembers the status a handler wrote, and for server errors
// the start of the body — handlers put the real cause there (`{"error": ...}`)
// without logging it themselves.
type statusRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	if s.status >= 500 && len(s.body) < maxLoggedErrorBody {
		s.body = append(s.body, b[:min(len(b), maxLoggedErrorBody-len(s.body))]...)
	}
	return s.ResponseWriter.Write(b)
}

// Logging logs each request once it completes, with its status and duration.
// Server errors are logged with the error the handler returned, so a 500 shows
// its cause in Cloud Run's logs, not only in the client's response.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		elapsed := time.Since(start).Round(time.Millisecond)
		if rec.status >= 500 {
			log.Printf("ERROR %s %s -> %d (%s): %s", r.Method, r.URL.Path, rec.status, elapsed, strings.TrimSpace(string(rec.body)))
			return
		}
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, elapsed)
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
