package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/handlers"
	"github.com/Mavzz/readpanda/api-go/internal/middleware"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize Firebase
	if err := utils.InitFirebase(cfg); err != nil {
		log.Printf("Warning: Failed to initialize Firebase: %v", err)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(cfg)
	bookHandler := handlers.NewBookHandler(cfg)
	preferencesHandler := handlers.NewPreferencesHandler(cfg)
	notificationHandler := handlers.NewNotificationHandler(cfg)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)

	// API version prefix
	apiPrefix := cfg.APIVersion

	// User routes
	router.HandleFunc(apiPrefix+"/users", userHandler.GetUsers).Methods("GET")
	router.HandleFunc(apiPrefix+"/signup", userHandler.CreateUser).Methods("POST")
	router.HandleFunc(apiPrefix+"/token/refresh", userHandler.RefreshAccessToken).Methods("GET")

	// Authentication routes
	router.HandleFunc(apiPrefix+"/auth/login", userHandler.LoginUser).Methods("POST")
	router.HandleFunc(apiPrefix+"/auth/google", userHandler.GoogleAuth).Methods("POST")
	router.HandleFunc(apiPrefix+"/auth/logout", userHandler.LogoutUser).Methods("POST")

	// User preferences routes
	router.HandleFunc(apiPrefix+"/user/preferences", preferencesHandler.GetUserPreferences).Methods("GET")
	router.HandleFunc(apiPrefix+"/user/preferences", preferencesHandler.UpdateUserPreferences).Methods("POST")

	// Books routes
	router.HandleFunc(apiPrefix+"/books/upload", bookHandler.PublishBook).Methods("POST")
	router.HandleFunc(apiPrefix+"/books", bookHandler.GetBooksForUser).Methods("GET")
	router.HandleFunc(apiPrefix+"/books/all", bookHandler.GetAllBooks).Methods("GET")
	router.HandleFunc(apiPrefix+"/books/seed", bookHandler.SeedBooksFromFirebase).Methods("POST")

	// Genres / Subgenres routes
	router.HandleFunc(apiPrefix+"/genres", preferencesHandler.GetGenres).Methods("GET")
	router.HandleFunc(apiPrefix+"/subgenres", preferencesHandler.GetSubgenres).Methods("GET")

	// Notifications routes
	router.HandleFunc(apiPrefix+"/notifications", notificationHandler.GetUserNotifications).Methods("GET")
	router.HandleFunc(apiPrefix+"/notifications/unread/count", notificationHandler.GetUnreadNotificationCount).Methods("GET")

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
