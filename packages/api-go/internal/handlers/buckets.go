package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// BucketHandler handles bucket-related operations
type BucketHandler struct {
	Config *config.Config
}

// NewBucketHandler creates a new bucket handler
func NewBucketHandler(cfg *config.Config) *BucketHandler {
	return &BucketHandler{Config: cfg}
}

// extractUserID validates the bearer token and returns the user ID from claims.
func (h *BucketHandler) extractUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
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
	claims, err := utils.DecodeToken(parts[1], h.Config.JWTSecret)
	if err != nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return "", false
	}
	return claims.UserID, true
}

// ── User Buckets ────────────────────────────────────────────

// ListUserBuckets — GET /users/me/buckets
func (h *BucketHandler) ListUserBuckets(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, name, created_at FROM user_buckets WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to list buckets"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	buckets := []models.UserBucket{}
	for rows.Next() {
		var b models.UserBucket
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt); err != nil {
			http.Error(w, `{"error": "Failed to scan bucket"}`, http.StatusInternalServerError)
			return
		}
		b.UserID = userID
		buckets = append(buckets, b)
	}

	// Enrich each bucket with book_count and books_preview (first 2)
	for i := range buckets {
		var count int
		_ = database.DB.QueryRow(
			`SELECT COUNT(*) FROM user_bucket_books WHERE bucket_id = $1`, buckets[i].ID,
		).Scan(&count)
		buckets[i].BookCount = count

		previewRows, err := database.DB.Query(
			`SELECT b.book_id, b.title, b.cover_image_url
			 FROM user_bucket_books ubb
			 JOIN books b ON b.book_id = ubb.book_id
			 WHERE ubb.bucket_id = $1
			 ORDER BY ubb.added_at
			 LIMIT 2`,
			buckets[i].ID,
		)
		if err == nil {
			defer previewRows.Close()
			for previewRows.Next() {
				var bp models.BookPreview
				if err := previewRows.Scan(&bp.BookID, &bp.Title, &bp.CoverImageURL); err == nil {
					buckets[i].BooksPreview = append(buckets[i].BooksPreview, bp)
				}
			}
		}
		if buckets[i].BooksPreview == nil {
			buckets[i].BooksPreview = []models.BookPreview{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"buckets": buckets})
}

// CreateUserBucket — POST /users/me/buckets
func (h *BucketHandler) CreateUserBucket(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name    string   `json:"name"`
		BookIDs []string `json:"book_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 40 {
		http.Error(w, `{"error": "name is required and must be 1-40 characters"}`, http.StatusBadRequest)
		return
	}

	// Check max 20 buckets per user
	var count int
	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM user_buckets WHERE user_id = $1`, userID,
	).Scan(&count); err != nil {
		http.Error(w, `{"error": "Failed to check bucket count"}`, http.StatusInternalServerError)
		return
	}
	if count >= 20 {
		http.Error(w, `{"error": "Maximum of 20 buckets per user reached"}`, http.StatusConflict)
		return
	}

	bucketID := "ub_" + uuid.New().String()[:8]
	now := time.Now().UTC()
	bookCount := len(req.BookIDs)

	_, err := database.DB.Exec(
		`INSERT INTO user_buckets (id, user_id, name, created_at, updated_at, book_count) VALUES ($1, $2, $3, $4, $5, $6)`,
		bucketID, userID, name, now, now, bookCount,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to create bucket"}`, http.StatusInternalServerError)
		return
	}

	// Add books if provided
	for _, bookID := range req.BookIDs {
		_, _ = database.DB.Exec(
			`INSERT INTO user_bucket_books (bucket_id, book_id, added_at)
			 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			bucketID, bookID, now,
		)
	}

	bucket := models.UserBucket{
		ID:        bucketID,
		Name:      name,
		BookCount: bookCount,
	}
	rows, err := database.DB.Query(
		`SELECT ubb.book_id, b.title, b.cover_image_url, b.manuscript_url
		FROM user_bucket_books ubb
		JOIN books b ON b.book_id = ubb.book_id
		WHERE ubb.bucket_id = $1`, bucketID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var book models.BookPreview
			if err := rows.Scan(&book.BookID, &book.Title, &book.CoverImageURL, &book.ManuscriptURL); err == nil {
				bucket.BooksPreview = append(bucket.BooksPreview, book)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bucket": bucket,
	})
}

// UpdateUserBucket — PUT /users/me/buckets/{id}
func (h *BucketHandler) UpdateUserBucket(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["id"]

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 40 {
		http.Error(w, `{"error": "name is required and must be 1-40 characters"}`, http.StatusBadRequest)
		return
	}

	res, err := database.DB.Exec(
		`UPDATE user_buckets SET name = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		name, bucketID, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to update bucket"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   bucketID,
		"name": name,
	})
}

// DeleteUserBucket — DELETE /users/me/buckets/{id}
func (h *BucketHandler) DeleteUserBucket(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["id"]

	res, err := database.DB.Exec(
		`DELETE FROM user_buckets WHERE id = $1 AND user_id = $2`,
		bucketID, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to delete bucket"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddBooksToBucket — POST /users/me/buckets/{id}/books
func (h *BucketHandler) AddBooksToBucket(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["id"]

	// Verify ownership
	var exists bool
	err := database.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_buckets WHERE id = $1 AND user_id = $2)`,
		bucketID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		BookIDs []string `json:"book_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	for _, bookID := range req.BookIDs {
		_, _ = database.DB.Exec(
			`INSERT INTO user_bucket_books (bucket_id, book_id, added_at)
			 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			bucketID, bookID, now,
		)
	}

	var bookCount int
	_ = database.DB.QueryRow(
		`SELECT COUNT(*) FROM user_bucket_books WHERE bucket_id = $1`, bucketID,
	).Scan(&bookCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         bucketID,
		"book_count": bookCount,
	})
}

// RemoveBookFromBucket — DELETE /users/me/buckets/{id}/books/{bookId}
func (h *BucketHandler) RemoveBookFromBucket(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	bucketID := vars["id"]
	bookID := vars["bookId"]

	// Verify ownership
	var exists bool
	err := database.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_buckets WHERE id = $1 AND user_id = $2)`,
		bucketID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	_, err = database.DB.Exec(
		`DELETE FROM user_bucket_books WHERE bucket_id = $1 AND book_id = $2`,
		bucketID, bookID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to remove book"}`, http.StatusInternalServerError)
		return
	}

	// Update book_count
	var bookCount int
	_ = database.DB.QueryRow(
		`UPDATE user_buckets SET book_count = book_count - 1 WHERE id = $1`,
		bucketID,
	).Scan(&bookCount)

	w.WriteHeader(http.StatusNoContent)
}

// ── Curated "Our Picks" Buckets ─────────────────────────────

// GetOurPicks — GET /home/our-picks
// Returns all curated buckets for admin portal (X-Application-Type: portal),
// or only active buckets for mobile/users.
func (h *BucketHandler) GetOurPicks(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	isAdmin := r.Header.Get("X-Application-Type") == "portal"

	var rows *sql.Rows
	var err error
	if isAdmin {
		rows, err = database.DB.Query(
			`SELECT id, title, sort_order, cover_image_url, is_active, book_count
			 FROM curated_buckets
			 ORDER BY sort_order`,
		)
	} else {
		rows, err = database.DB.Query(
			`SELECT id, title, sort_order, cover_image_url, is_active, book_count
			 FROM curated_buckets
			 WHERE is_active = true
			 ORDER BY sort_order`,
		)
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch curated buckets"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	buckets := []models.CuratedBucket{}
	for rows.Next() {
		var b models.CuratedBucket
		if err := rows.Scan(&b.ID, &b.Title, &b.SortOrder, &b.CoverImageURL, &b.IsActive, &b.BookCount); err != nil {
			http.Error(w, `{"error": "Failed to scan bucket"}`, http.StatusInternalServerError)
			return
		}
		buckets = append(buckets, b)
	}

	for i := range buckets {

		// Preview (first 2 books)
		previewRows, err := database.DB.Query(
			`SELECT b.book_id, b.title, b.cover_image_url, b.manuscript_url
			 FROM curated_bucket_books cbb
			 JOIN books b ON b.book_id = cbb.book_id
			 WHERE cbb.bucket_id = $1
			 ORDER BY cbb.sort_order`,
			buckets[i].ID,
		)
		if err == nil {
			defer previewRows.Close()
			for previewRows.Next() {
				var bp models.BookPreview
				if err := previewRows.Scan(&bp.BookID, &bp.Title, &bp.CoverImageURL, &bp.ManuscriptURL); err == nil {
					buckets[i].BooksPreview = append(buckets[i].BooksPreview, bp)
				}
			}
		}
		if buckets[i].BooksPreview == nil {
			buckets[i].BooksPreview = []models.BookPreview{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"buckets": buckets})
}

// GetOurPicksBucketBooks — GET /home/our-picks/{bucketId}/books
func (h *BucketHandler) GetOurPicksBucketBooks(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["bucketId"]

	var title string
	err := database.DB.QueryRow(
		`SELECT title FROM curated_buckets WHERE id = $1 AND is_active = true`, bucketID,
	).Scan(&title)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error": "Failed to fetch bucket"}`, http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query(
		`SELECT b.book_id, b.title, b.cover_image_url, b.genre
		 FROM curated_bucket_books cbb
		 JOIN books b ON b.book_id = cbb.book_id
		 WHERE cbb.bucket_id = $1
		 ORDER BY cbb.sort_order`,
		bucketID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch books: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := []models.CuratedBookEntry{}
	for rows.Next() {
		var entry models.CuratedBookEntry
		if err := rows.Scan(&entry.BookID, &entry.Title, &entry.CoverImageURL, &entry.Genre); err != nil {
			http.Error(w, `{"error": "Failed to scan book"}`, http.StatusInternalServerError)
			return
		}
		books = append(books, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CuratedBucketDetail{
		ID:    bucketID,
		Title: title,
		Books: books,
	})
}

// ── Admin: Curated Bucket Management ────────────────────────

// AdminListCuratedBuckets — GET /admin/curated-buckets
// Returns all curated buckets (including inactive) for admin management.
func (h *BucketHandler) AdminListCuratedBuckets(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, title, sort_order, cover_image_url, is_active
		 FROM curated_buckets
		 ORDER BY sort_order`,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to list curated buckets"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AdminCuratedBucket struct {
		ID            string               `json:"id"`
		Title         string               `json:"title"`
		SortOrder     int                  `json:"sort_order"`
		CoverImageURL *string              `json:"cover_image_url"`
		IsActive      bool                 `json:"is_active"`
		BookCount     int                  `json:"book_count"`
		BooksPreview  []models.BookPreview `json:"books_preview"`
	}

	buckets := []AdminCuratedBucket{}
	for rows.Next() {
		var b AdminCuratedBucket
		if err := rows.Scan(&b.ID, &b.Title, &b.SortOrder, &b.CoverImageURL, &b.IsActive); err != nil {
			http.Error(w, `{"error": "Failed to scan bucket"}`, http.StatusInternalServerError)
			return
		}
		buckets = append(buckets, b)
	}

	for i := range buckets {
		_ = database.DB.QueryRow(
			`SELECT COUNT(*) FROM curated_bucket_books WHERE bucket_id = $1`, buckets[i].ID,
		).Scan(&buckets[i].BookCount)

		previewRows, err := database.DB.Query(
			`SELECT b.book_id, b.title, b.cover_image_url
			 FROM curated_bucket_books cbb
			 JOIN books b ON b.book_id = cbb.book_id
			 WHERE cbb.bucket_id = $1
			 ORDER BY cbb.sort_order
			 LIMIT 2`,
			buckets[i].ID,
		)
		if err == nil {
			defer previewRows.Close()
			for previewRows.Next() {
				var bp models.BookPreview
				if err := previewRows.Scan(&bp.BookID, &bp.Title, &bp.CoverImageURL); err == nil {
					buckets[i].BooksPreview = append(buckets[i].BooksPreview, bp)
				}
			}
		}
		if buckets[i].BooksPreview == nil {
			buckets[i].BooksPreview = []models.BookPreview{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"buckets": buckets})
}

// AdminCreateCuratedBucket — POST /admin/curated-buckets
func (h *BucketHandler) AdminCreateCuratedBucket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		Title     string   `json:"title"`
		SortOrder int      `json:"sort_order"`
		BookIDs   []string `json:"book_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" || len(title) > 100 {
		http.Error(w, `{"error": "title is required and must be 1-100 characters"}`, http.StatusBadRequest)
		return
	}

	bucketID := "cb_" + uuid.New().String()[:8]
	now := time.Now().UTC()

	_, err := database.DB.Exec(
		`INSERT INTO curated_buckets (id, title, sort_order, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, true, $4, $5)`,
		bucketID, title, req.SortOrder, now, now,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to create curated bucket"}`, http.StatusInternalServerError)
		return
	}

	for i, bookID := range req.BookIDs {
		_, _ = database.DB.Exec(
			`INSERT INTO curated_bucket_books (bucket_id, book_id, sort_order) VALUES ($1, $2, $3)
			 ON CONFLICT (bucket_id, book_id) DO NOTHING`,
			bucketID, bookID, i,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         bucketID,
		"title":      title,
		"sort_order": req.SortOrder,
		"is_active":  true,
		"created_at": now.Format(time.RFC3339),
		"book_count": len(req.BookIDs),
	})
}

// AdminUpdateCuratedBucket — PUT /home/our-picks/{bucketId}
func (h *BucketHandler) AdminUpdateCuratedBucket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["bucketId"]

	var req struct {
		Title     string `json:"title"`
		SortOrder int    `json:"sort_order"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" || len(title) > 100 {
		http.Error(w, `{"error": "title is required and must be 1-100 characters"}`, http.StatusBadRequest)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	res, err := database.DB.Exec(
		`UPDATE curated_buckets SET title = $1, sort_order = $2, is_active = $3, updated_at = NOW()
		 WHERE id = $4`,
		title, req.SortOrder, isActive, bucketID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to update curated bucket"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         bucketID,
		"title":      title,
		"sort_order": req.SortOrder,
		"is_active":  isActive,
	})
}

// AdminDeleteCuratedBucket — DELETE /home/our-picks/{bucketId}
func (h *BucketHandler) AdminDeleteCuratedBucket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["bucketId"]

	res, err := database.DB.Exec(`DELETE FROM curated_buckets WHERE id = $1`, bucketID)
	if err != nil {
		http.Error(w, `{"error": "Failed to delete curated bucket"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AdminAddBooksToCuratedBucket — POST /home/our-picks/{bucketId}/books
func (h *BucketHandler) AdminAddBooksToCuratedBucket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	bucketID := mux.Vars(r)["bucketId"]

	var exists bool
	if err := database.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM curated_buckets WHERE id = $1)`, bucketID,
	).Scan(&exists); err != nil || !exists {
		http.Error(w, `{"error": "Bucket not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		BookIDs []string `json:"book_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	for i, bookID := range req.BookIDs {
		_, _ = database.DB.Exec(
			`INSERT INTO curated_bucket_books (bucket_id, book_id, sort_order, added_at)
			 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
			bucketID, bookID, i, now,
		)
	}

	var bookCount int
	_ = database.DB.QueryRow(
		`SELECT COUNT(*) FROM curated_bucket_books WHERE bucket_id = $1`, bucketID,
	).Scan(&bookCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         bucketID,
		"book_count": bookCount,
	})
}

// AdminRemoveBookFromCuratedBucket — DELETE /home/our-picks/{bucketId}/books/{bookId}
func (h *BucketHandler) AdminRemoveBookFromCuratedBucket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.extractUserID(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	bucketID := vars["bucketId"]
	bookID := vars["bookId"]

	_, err := database.DB.Exec(
		`DELETE FROM curated_bucket_books WHERE bucket_id = $1 AND book_id = $2`,
		bucketID, bookID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to remove book"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
