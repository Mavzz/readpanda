package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// BookHandler handles book-related operations
type BookHandler struct {
	Config *config.Config
}

// NewBookHandler creates a new book handler
func NewBookHandler(cfg *config.Config) *BookHandler {
	return &BookHandler{Config: cfg}
}

// PublishBook handles book publishing with cover and manuscript upload
func (h *BookHandler) PublishBook(w http.ResponseWriter, r *http.Request) {
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
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, `{"error": "Failed to parse form"}`, http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	genre := r.FormValue("genre")
	subgenre := r.FormValue("subgenre")

	claims, err := utils.DecodeToken(token, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
		return
	}

	var coverLink, manuscriptLink *string

	// Upload cover if present
	coverFile, coverHeader, err := r.FormFile("cover")
	if err == nil {
		defer coverFile.Close()
		coverData, err := io.ReadAll(coverFile)
		if err != nil {
			http.Error(w, `{"error": "Failed to read cover file"}`, http.StatusInternalServerError)
			return
		}

		coverPath := fmt.Sprintf("books/covers/%s", coverHeader.Filename)
		url, err := utils.UploadFileToFirebase(coverData, coverHeader.Header.Get("Content-Type"), coverPath)
		if err != nil {
			http.Error(w, `{"error": "Failed to upload cover: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		coverLink = &url
	}

	// Upload manuscript if present
	manuscriptFile, manuscriptHeader, err := r.FormFile("manuscript")
	if err == nil {
		defer manuscriptFile.Close()
		manuscriptData, err := io.ReadAll(manuscriptFile)
		if err != nil {
			http.Error(w, `{"error": "Failed to read manuscript file"}`, http.StatusInternalServerError)
			return
		}

		manuscriptPath := fmt.Sprintf("books/manuscripts/%s", manuscriptHeader.Filename)
		url, err := utils.UploadFileToFirebase(manuscriptData, manuscriptHeader.Header.Get("Content-Type"), manuscriptPath)
		if err != nil {
			http.Error(w, `{"error": "Failed to upload manuscript: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		manuscriptLink = &url
	}

	if coverLink == nil && manuscriptLink == nil {
		http.Error(w, `{"message": "No files uploaded."}`, http.StatusBadRequest)
		return
	}

	// Insert book into database
	_, err = database.DB.Exec(
		"INSERT INTO books (title, description, subgenre, genre, cover_image_url, manuscript_url, status, views, user_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		title, description, subgenre, genre, coverLink, manuscriptLink, 1, 0, claims.UserID,
	)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Book uploaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBooksForUser retrieves all books for the authenticated user
func (h *BookHandler) GetBooksForUser(w http.ResponseWriter, r *http.Request) {
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
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	claims, err := utils.DecodeToken(token, h.Config.JWTSecret)
	if err != nil {
		http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query("SELECT id, title, description, subgenre, genre, cover_image_url, manuscript_url, status, views, created_at FROM books WHERE user_id = $1", claims.UserID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var book models.Book
		err := rows.Scan(&book.ID, &book.Title, &book.Description, &book.Subgenre, &book.Genre, &book.CoverImageURL, &book.ManuscriptURL, &book.Status, &book.Views, &book.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		books = append(books, book)
	}

	response := map[string]interface{}{
		"books": books,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllBooks retrieves all books
func (h *BookHandler) GetAllBooks(w http.ResponseWriter, r *http.Request) {
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
	if !utils.CheckToken(token, h.Config.JWTSecret) {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query("SELECT id, title, description, subgenre, genre, cover_image_url, manuscript_url, status, views, created_at FROM books")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var book models.Book
		err := rows.Scan(&book.ID, &book.Title, &book.Description, &book.Subgenre, &book.Genre, &book.CoverImageURL, &book.ManuscriptURL, &book.Status, &book.Views, &book.CreatedAt)
		if err != nil {
			http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		books = append(books, book)
	}

	response := map[string]interface{}{
		"books": books,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
