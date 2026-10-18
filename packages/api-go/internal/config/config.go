package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port       string
	APIVersion string

	// Database
	DBUser     string
	DBHost     string
	DBName     string
	DBPassword string
	DBPort     string
	// Hosted Postgres (Supabase, Neon) needs "require"; local Docker has no TLS.
	DBSSLMode string

	// JWT
	JWTSecret        string
	JWTRefreshSecret string

	// Google OAuth
	GoogleClientID    string
	GoogleIOSClientID string

	// Sign in with Apple. The bundle id is the identity token's audience. The
	// key (team, key id, .p8 contents — from APPLE_PRIVATE_KEY or the file at
	// APPLE_PRIVATE_KEY_PATH) signs the client secret used to
	// exchange and revoke Apple tokens; without it Apple sign-in still works,
	// but deleting an Apple account can't revoke its token.
	AppleBundleID   string
	AppleTeamID     string
	AppleKeyID      string
	ApplePrivateKey string

	// Firebase
	FirebaseType               string
	FirebaseProjectID          string
	FirebasePrivateKeyID       string
	FirebasePrivateKey         string
	FirebaseClientEmail        string
	FirebaseClientID           string
	FirebaseAuthURI            string
	FirebaseTokenURI           string
	FirebaseAuthProvider       string
	FirebaseCertURL            string
	FirebaseStorageBucket      string
	FirebaseServiceAccountPath string

	// Object storage (Cloudflare R2 / MinIO)
	R2Endpoint        string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	// Notifications. A reader's unread room-comments row is pushed at most
	// once per window; comments inside it only update the row.
	RoomCommentWindow time.Duration
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env.local file if it exists
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("No .env.local file found, using environment variables")
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	if jwtSecret == "" || jwtRefreshSecret == "" {
		// Silently falling back to an empty-string HMAC key would mean every
		// token is signed/verified with a well-known secret — fail loudly
		// instead of booting into a broken-but-running state.
		log.Fatal("JWT_SECRET and JWT_REFRESH_SECRET must both be set")
	}

	// The Sign in with Apple key: inline in APPLE_PRIVATE_KEY, or — as in
	// production, mounted from Secret Manager like the Firebase service
	// account — a .p8 file at APPLE_PRIVATE_KEY_PATH. Optional: without it
	// Apple sign-in works, but Apple tokens can't be revoked.
	applePrivateKey := getEnv("APPLE_PRIVATE_KEY", "")
	if path := getEnv("APPLE_PRIVATE_KEY_PATH", ""); applePrivateKey == "" && path != "" {
		if raw, err := os.ReadFile(path); err != nil {
			log.Printf("APPLE_PRIVATE_KEY_PATH %q unreadable (%v); Apple token revocation is off", path, err)
		} else {
			applePrivateKey = string(raw)
		}
	}

	return &Config{
		Port:       getEnv("PORT", "3000"),
		APIVersion: getEnv("API_VERSION", "/api/v1"),

		DBUser:     getEnv("PG_USER", ""),
		DBHost:     getEnv("PG_HOST", "localhost"),
		DBName:     getEnv("PG_DB", ""),
		DBPassword: getEnv("PG_PASSWORD", ""),
		DBPort:     getEnv("PG_PORT", "5432"),
		DBSSLMode:  getEnv("PG_SSLMODE", "disable"),

		JWTSecret:        jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,

		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleIOSClientID: getEnv("GOOGLE_IOS_CLIENT_ID", ""),

		AppleBundleID:   getEnv("APPLE_BUNDLE_ID", "com.readpanda.app"),
		AppleTeamID:     getEnv("APPLE_TEAM_ID", ""),
		AppleKeyID:      getEnv("APPLE_KEY_ID", ""),
		ApplePrivateKey: applePrivateKey,

		FirebaseType:               getEnv("FIREBASE_TYPE", "service_account"),
		FirebaseProjectID:          getEnv("FIREBASE_PROJECT_ID", ""),
		FirebasePrivateKeyID:       getEnv("FIREBASE_PRIVATE_KEY_ID", ""),
		FirebasePrivateKey:         getEnv("FIREBASE_PRIVATE_KEY", ""),
		FirebaseClientEmail:        getEnv("FIREBASE_CLIENT_EMAIL", ""),
		FirebaseClientID:           getEnv("FIREBASE_CLIENT_ID", ""),
		FirebaseAuthURI:            getEnv("FIREBASE_AUTH_URI", "https://accounts.google.com/o/oauth2/auth"),
		FirebaseTokenURI:           getEnv("FIREBASE_TOKEN_URI", "https://oauth2.googleapis.com/token"),
		FirebaseAuthProvider:       getEnv("FIREBASE_AUTH_PROVIDER", "https://www.googleapis.com/oauth2/v1/certs"),
		FirebaseCertURL:            getEnv("FIREBASE_CERT_URL", ""),
		FirebaseStorageBucket:      getEnv("FIREBASE_STORAGE_BUCKET", ""),
		FirebaseServiceAccountPath: getEnv("FIREBASE_SERVICE_ACCOUNT_PATH", ""),

		R2Endpoint:        getEnv("R2_ENDPOINT", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		RoomCommentWindow: getDurationEnv("NOTIFY_ROOM_COMMENT_WINDOW", 30*time.Minute),
	}
}

// getDurationEnv reads a Go duration ("30m", "1h"). A value that doesn't parse,
// or isn't positive, is logged and the default used: a typo in env.yaml
// shouldn't stop the server from starting.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("%s=%q isn't a positive duration like \"30m\"; using %s", key, raw, defaultValue)
		return defaultValue
	}
	return d
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
