package config

import (
	"log"
	"os"

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

	// JWT
	JWTSecret        string
	JWTRefreshSecret string

	// Crypto
	CryptoSecret string

	// Google OAuth
	GoogleClientID    string
	GoogleIOSClientID string

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
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env.local file if it exists
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("No .env.local file found, using environment variables")
	}

	return &Config{
		Port:       getEnv("PORT", "3000"),
		APIVersion: getEnv("API_VERSION", "/api/v1"),

		DBUser:     getEnv("PG_USER", ""),
		DBHost:     getEnv("PG_HOST", "localhost"),
		DBName:     getEnv("PG_DB", ""),
		DBPassword: getEnv("PG_PASSWORD", ""),
		DBPort:     getEnv("PG_PORT", "5432"),

		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		CryptoSecret:     getEnv("CRYPTO_SECRET", ""),

		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleIOSClientID: getEnv("GOOGLE_IOS_CLIENT_ID", ""),

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
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
