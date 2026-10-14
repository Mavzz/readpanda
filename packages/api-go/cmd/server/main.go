package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/metadata"
	"github.com/Mavzz/readpanda/api-go/internal/server"
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

	// Initialize object storage (R2 / MinIO)
	if err := utils.InitObjectStorage(cfg); err != nil {
		log.Printf("Warning: Failed to initialize object storage: %v", err)
	}

	// Initialize push (Firebase Cloud Messaging). Optional: without it,
	// notifications still land in the inbox, they just aren't pushed.
	if err := utils.InitPush(cfg); err != nil {
		log.Printf("Warning: Push notifications disabled: %v", err)
	}

	// Give books still named after their files a real title, author and
	// length (9a/9b). Runs in the background; already-checked books are skipped.
	metadata.EnrichInBackground()

	router := server.NewRouter(cfg)

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
