package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/handlers"
	"github.com/Mavzz/readpanda/api-go/internal/middleware"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize object storage (R2 / MinIO)
	if err := utils.InitObjectStorage(cfg); err != nil {
		log.Printf("Warning: Failed to initialize object storage: %v", err)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(cfg)
	bookHandler := handlers.NewBookHandler(cfg)
	preferencesHandler := handlers.NewPreferencesHandler(cfg)
	notificationHandler := handlers.NewNotificationHandler(cfg)
	bucketHandler := handlers.NewBucketHandler(cfg)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)

	// Handle CORS preflight for all routes
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("OPTIONS")

	// API version prefix
	apiPrefix := cfg.APIVersion

	// User routes
	router.HandleFunc(apiPrefix+"/users", userHandler.GetUsers).Methods("GET")
	router.HandleFunc(apiPrefix+"/signup", userHandler.CreateUser).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/token/refresh", userHandler.RefreshAccessToken).Methods("POST", "OPTIONS")

	// Authentication routes
	router.HandleFunc(apiPrefix+"/auth/login", userHandler.LoginUser).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/auth/google", userHandler.GoogleAuth).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/auth/logout", userHandler.LogoutUser).Methods("POST", "OPTIONS")

	// User preferences routes
	router.HandleFunc(apiPrefix+"/user/preferences", preferencesHandler.GetUserPreferences).Methods("GET")
	router.HandleFunc(apiPrefix+"/user/preferences", preferencesHandler.UpdateUserPreferences).Methods("POST", "OPTIONS")

	// Books routes
	router.HandleFunc(apiPrefix+"/books/upload", bookHandler.PublishBook).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/books", bookHandler.GetBooksForUser).Methods("GET")
	router.HandleFunc(apiPrefix+"/books/all", bookHandler.GetAllBooks).Methods("GET")
	router.HandleFunc(apiPrefix+"/books/seed", bookHandler.SeedBooksFromStorage).Methods("POST", "OPTIONS")

	// Genres / Subgenres routes
	router.HandleFunc(apiPrefix+"/genres", preferencesHandler.GetGenres).Methods("GET")
	router.HandleFunc(apiPrefix+"/subgenres", preferencesHandler.GetSubgenres).Methods("GET")

	// Notifications routes
	router.HandleFunc(apiPrefix+"/notifications", notificationHandler.GetUserNotifications).Methods("GET")
	router.HandleFunc(apiPrefix+"/notifications/unread/count", notificationHandler.GetUnreadNotificationCount).Methods("GET")

	// User Buckets routes
	router.HandleFunc(apiPrefix+"/users/me/buckets", bucketHandler.ListUserBuckets).Methods("GET")
	router.HandleFunc(apiPrefix+"/users/me/buckets", bucketHandler.CreateUserBucket).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/users/me/buckets/{id}", bucketHandler.UpdateUserBucket).Methods("PUT", "OPTIONS")
	router.HandleFunc(apiPrefix+"/users/me/buckets/{id}", bucketHandler.DeleteUserBucket).Methods("DELETE", "OPTIONS")
	router.HandleFunc(apiPrefix+"/users/me/buckets/{id}/books", bucketHandler.AddBooksToBucket).Methods("POST", "OPTIONS")
	router.HandleFunc(apiPrefix+"/users/me/buckets/{id}/books/{bookId}", bucketHandler.RemoveBookFromBucket).Methods("DELETE", "OPTIONS")

	// Curated "Our Picks" routes
	// GET  — returns active buckets for mobile, all buckets for portal (X-Application-Type: portal)
	// POST/PUT/DELETE — admin management (portal only)
	router.HandleFunc(apiPrefix+"/home/our-picks", bucketHandler.GetOurPicks).Methods("GET")
	router.HandleFunc(apiPrefix+"/home/our-picks", bucketHandler.AdminCreateCuratedBucket).Methods("POST")
	router.HandleFunc(apiPrefix+"/home/our-picks/{bucketId}", bucketHandler.AdminUpdateCuratedBucket).Methods("PUT")
	router.HandleFunc(apiPrefix+"/home/our-picks/{bucketId}", bucketHandler.AdminDeleteCuratedBucket).Methods("DELETE")
	router.HandleFunc(apiPrefix+"/home/our-picks/{bucketId}/books", bucketHandler.GetOurPicksBucketBooks).Methods("GET")
	router.HandleFunc(apiPrefix+"/home/our-picks/{bucketId}/books", bucketHandler.AdminAddBooksToCuratedBucket).Methods("POST")
	router.HandleFunc(apiPrefix+"/home/our-picks/{bucketId}/books/{bookId}", bucketHandler.AdminRemoveBookFromCuratedBucket).Methods("DELETE")

	// Start server
	port := cfg.Port
	addr := fmt.Sprintf("0.0.0.0:%s", port)

	// Check if TLS certificates exist
	certFile := "cert.pem"
	keyFile := "key.pem"

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			log.Printf("App running on %s HTTPS and port %s...", addr, port)
			if err := http.ListenAndServeTLS(addr, certFile, keyFile, router); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	// Fall back to HTTP if certificates don't exist
	log.Printf("App running on %s HTTP and port %s...", addr, port)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
