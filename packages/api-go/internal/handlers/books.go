package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/metadata"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/notify"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
	"github.com/google/uuid"
)

// BookHandler handles book-related operations
type BookHandler struct {
	Config *config.Config
}

// NewBookHandler creates a new book handler
func NewBookHandler(cfg *config.Config) *BookHandler {
	return &BookHandler{Config: cfg}
}

// Upload limits. The portal sends files straight to storage through presigned
// URLs (see CreateUploadURLs), so these are ours, not Cloud Run's 32 MiB cap.
const (
	maxManuscriptBytes = 500 << 20
	maxCoverBytes      = 20 << 20
	uploadURLTTL       = time.Hour
)

var (
	manuscriptTypes = map[string]string{".pdf": "application/pdf", ".epub": "application/epub+zip"}
	coverTypes      = map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".gif": "image/gif"}
	bookIDPattern   = regexp.MustCompile(`^bk_[0-9a-f]{8}$`)
)

// bookFields are the publish form's text fields.
type bookFields struct {
	Title         string
	Description   string
	Genre         string
	Subgenre      string
	AuthorName    string
	NotifyReaders bool
}

func newBookID() string {
	return "bk_" + uuid.New().String()[:8]
}

// storageKey places a book's file in storage. Keys carry the book id: uploads
// routinely share file names ("cover.jpeg" from an EPUB), and a shared key
// would overwrite the other book's file.
func storageKey(folder, bookID, filename string) string {
	name := path.Base(strings.ReplaceAll(filename, `\`, "/"))
	return fmt.Sprintf("books/%s/%s-%s", folder, bookID, name)
}

type uploadFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type uploadTarget struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

// CreateUploadURLs starts a publish: it picks the book id and returns one
// presigned PUT URL per file. The client uploads to those URLs, then calls
// PublishBook with the book id and keys.
func (h *BookHandler) CreateUploadURLs(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret); !ok {
		return
	}

	var req struct {
		Manuscript *uploadFile `json:"manuscript"`
		Cover      *uploadFile `json:"cover"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		adminError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Manuscript == nil {
		adminError(w, http.StatusBadRequest, "A manuscript is required")
		return
	}

	bookID := newBookID()
	resp := map[string]interface{}{"book_id": bookID}

	files := []struct {
		field  string
		folder string
		file   *uploadFile
		types  map[string]string
		max    int64
		label  string
	}{
		{"manuscript", "manuscripts", req.Manuscript, manuscriptTypes, maxManuscriptBytes, "The manuscript has to be a PDF or EPUB"},
		{"cover", "covers", req.Cover, coverTypes, maxCoverBytes, "The cover has to be a JPEG, PNG, WebP or GIF image"},
	}
	for _, f := range files {
		if f.file == nil {
			continue
		}
		contentType, ok := f.types[strings.ToLower(path.Ext(f.file.Name))]
		if !ok {
			adminError(w, http.StatusBadRequest, f.label)
			return
		}
		if f.file.Size > f.max {
			adminError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("The %s is over the %d MB limit", f.field, f.max>>20))
			return
		}
		key := storageKey(f.folder, bookID, f.file.Name)
		url, err := utils.PresignUpload(key, contentType, uploadURLTTL)
		if err != nil {
			adminError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp[f.field] = uploadTarget{Key: key, URL: url, ContentType: contentType}
	}

	adminJSON(w, http.StatusOK, resp)
}

// PublishBook adds a book. The portal sends JSON naming files it already put
// in storage through CreateUploadURLs; a multipart form with the files
// themselves still works for small files (Cloud Run rejects bodies over 32 MiB).
func (h *BookHandler) PublishBook(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		h.publishUploadedBook(w, r, userID)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // in memory up to 32 MB, the rest on disk
	if err != nil {
		http.Error(w, `{"error": "Failed to parse form"}`, http.StatusBadRequest)
		return
	}

	fields := bookFields{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Genre:       r.FormValue("genre"),
		Subgenre:    r.FormValue("subgenre"),
		AuthorName:  strings.TrimSpace(r.FormValue("author_name")),
		// The portal's bulk upload sends notify=false so a batch of test books
		// doesn't send every reader one notification per book.
		NotifyReaders: r.FormValue("notify") != "false",
	}

	bookID := newBookID()

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

		coverPath := storageKey("covers", bookID, coverHeader.Filename)
		url, err := utils.UploadFileToStorage(coverData, coverHeader.Header.Get("Content-Type"), coverPath)
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

		manuscriptPath := storageKey("manuscripts", bookID, manuscriptHeader.Filename)
		url, err := utils.UploadFileToStorage(manuscriptData, manuscriptHeader.Header.Get("Content-Type"), manuscriptPath)
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

	h.saveBook(w, userID, bookID, fields, coverLink, manuscriptLink)
}

// publishUploadedBook is PublishBook for files already in storage. It checks
// each key belongs to the book id CreateUploadURLs issued and that the
// upload landed within the size limit.
func (h *BookHandler) publishUploadedBook(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		BookID        string `json:"book_id"`
		Title         string `json:"title"`
		Description   string `json:"description"`
		Genre         string `json:"genre"`
		Subgenre      string `json:"subgenre"`
		AuthorName    string `json:"author_name"`
		Notify        *bool  `json:"notify"`
		ManuscriptKey string `json:"manuscript_key"`
		CoverKey      string `json:"cover_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		adminError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if !bookIDPattern.MatchString(req.BookID) {
		adminError(w, http.StatusBadRequest, "Invalid book_id")
		return
	}
	if req.ManuscriptKey == "" {
		adminError(w, http.StatusBadRequest, "manuscript_key is required")
		return
	}

	// Returns the file's public URL, or "" after writing the error.
	checkUpload := func(key, folder string, max int64, field string) string {
		if !strings.HasPrefix(key, "books/"+folder+"/"+req.BookID+"-") {
			adminError(w, http.StatusBadRequest, "Invalid "+field+"_key")
			return ""
		}
		size, err := utils.StatObject(key)
		if err != nil {
			adminError(w, http.StatusBadRequest, "The "+field+" upload didn't finish. Try again.")
			return ""
		}
		if size > max {
			if err := utils.DeleteObject(key); err != nil {
				log.Printf("books: failed to delete oversized %s: %v", key, err)
			}
			adminError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("The %s is over the %d MB limit", field, max>>20))
			return ""
		}
		return utils.PublicURL(key)
	}

	manuscriptURL := checkUpload(req.ManuscriptKey, "manuscripts", maxManuscriptBytes, "manuscript")
	if manuscriptURL == "" {
		return
	}
	var coverLink *string
	if req.CoverKey != "" {
		coverURL := checkUpload(req.CoverKey, "covers", maxCoverBytes, "cover")
		if coverURL == "" {
			return
		}
		coverLink = &coverURL
	}

	fields := bookFields{
		Title:         req.Title,
		Description:   req.Description,
		Genre:         req.Genre,
		Subgenre:      req.Subgenre,
		AuthorName:    strings.TrimSpace(req.AuthorName),
		NotifyReaders: req.Notify == nil || *req.Notify,
	}
	h.saveBook(w, userID, req.BookID, fields, coverLink, &manuscriptURL)
}

// saveBook inserts the book row, tells readers and answers the publish request.
func (h *BookHandler) saveBook(w http.ResponseWriter, userID, bookID string, fields bookFields, coverLink, manuscriptLink *string) {
	// Insert book into database
	_, err := database.DB.Exec(
		"INSERT INTO books (book_id, title, description, subgenre, genre, author_name, cover_image_url, manuscript_url, status, views, user_id) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, $11)",
		bookID, fields.Title, fields.Description, fields.Subgenre, fields.Genre, fields.AuthorName, coverLink, manuscriptLink, 1, 0, userID,
	)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Tell every other reader. Only once there is a manuscript: a NEW_BOOK
	// notification opens the reader, and a cover alone has nothing to open.
	// A failure here costs the announcement, not the upload.
	if manuscriptLink != nil && fields.NotifyReaders {
		displayTitle := strings.TrimSpace(fields.Title)
		if displayTitle == "" {
			displayTitle = "A new book"
		}
		if err := notify.ToAllUsersExcept(userID, notify.Notification{
			Type:    models.NotificationTypeNewBook,
			Title:   "New book added",
			Message: displayTitle + " is ready to read.",
			BookID:  bookID,
		}); err != nil {
			log.Printf("books: failed to announce %s: %v", bookID, err)
		}
	}

	metadata.EnrichInBackground()

	response := map[string]string{
		"message": "Book uploaded successfully",
		"book_id": bookID,
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

	rows, err := database.DB.Query("SELECT book_id, title, description, subgenre, genre, author_name, page_count, cover_image_url, manuscript_url, status, views, created_at FROM books WHERE user_id = $1", claims.UserID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var book models.Book
		err := rows.Scan(&book.ID, &book.Title, &book.Description, &book.Subgenre, &book.Genre, &book.AuthorName, &book.PageCount, &book.CoverImageURL, &book.ManuscriptURL, &book.Status, &book.Views, &book.CreatedAt)
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

	rows, err := database.DB.Query("SELECT book_id, title, description, subgenre, genre, author_name, page_count, cover_image_url, manuscript_url, status, views, created_at FROM books")
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var book models.Book
		err := rows.Scan(&book.ID, &book.Title, &book.Description, &book.Subgenre, &book.Genre, &book.AuthorName, &book.PageCount, &book.CoverImageURL, &book.ManuscriptURL, &book.Status, &book.Views, &book.CreatedAt)
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

// parseCoverPath extracts genre, subgenre, and title from a cover object key.
// Expected format: books/covers/{genre}/{subgenre}/{title}.ext
func parseCoverPath(key string) (genre, subgenre, title string) {
	// Remove the "books/covers/" prefix
	trimmed := strings.TrimPrefix(key, "books/covers/")
	parts := strings.Split(trimmed, "/")

	switch len(parts) {
	case 3:
		// genre/subgenre/title.ext
		genre = parts[0]
		subgenre = parts[1]
		title = strings.TrimSuffix(parts[2], filepath.Ext(parts[2]))
	case 2:
		// genre/title.ext (no subgenre)
		genre = parts[0]
		title = strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
	default:
		// title.ext only (flat structure)
		title = strings.TrimSuffix(filepath.Base(key), filepath.Ext(key))
	}
	return
}

// SeedBooksFromStorage populates the books table from files in object storage.
// Covers should be organised as: books/covers/{genre}/{subgenre}/{title}.ext
// Manuscripts should mirror the same structure under books/manuscripts/.
func (h *BookHandler) SeedBooksFromStorage(w http.ResponseWriter, r *http.Request) {
	// List covers and manuscripts from object storage
	covers, err := utils.ListFilesFromStorage("books/covers/")
	if err != nil {
		http.Error(w, `{"error": "Failed to list covers: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	manuscripts, err := utils.ListFilesFromStorage("books/manuscripts/")
	if err != nil {
		http.Error(w, `{"error": "Failed to list manuscripts: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Index manuscripts by relative path (without extension) for matching.
	// e.g. "Fiction/Fantasy/The Great Adventure" → URL
	manuscriptMap := make(map[string]string)
	for _, m := range manuscripts {
		trimmed := strings.TrimPrefix(m.Name, "books/manuscripts/")
		base := strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
		manuscriptMap[base] = m.URL
	}

	inserted := 0
	for _, cover := range covers {
		genre, subgenre, title := parseCoverPath(cover.Name)

		coverURL := cover.URL
		var manuscriptURL *string

		// Build the manuscript lookup key to match cover's relative path
		trimmed := strings.TrimPrefix(cover.Name, "books/covers/")
		lookupKey := strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
		if mURL, ok := manuscriptMap[lookupKey]; ok {
			manuscriptURL = &mURL
		}

		bookID := "bk_" + uuid.New().String()[:8]
		_, err := database.DB.Exec(
			"INSERT INTO books (book_id, title, description, subgenre, genre, cover_image_url, manuscript_url, status, views) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			bookID, title, "", subgenre, genre, &coverURL, manuscriptURL, 1, 0,
		)
		if err != nil {
			http.Error(w, `{"error": "Failed to insert book: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		inserted++
	}

	// New books arrive with file-name titles; look them up in the background.
	if inserted > 0 {
		metadata.EnrichInBackground()
	}

	response := map[string]interface{}{
		"message":  fmt.Sprintf("Seeded %d books from object storage", inserted),
		"inserted": inserted,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminEnrichBooks — POST /admin/books/enrich?limit=N
// Runs the metadata lookup now over books it hasn't checked yet and reports
// what it found. The server also runs it at startup and after a seed or
// upload; this is for doing it on demand. ?recheck=true clears every book's
// checked mark first, and with it whatever an earlier lookup filled in
// (never an uploader's author or the reader's page count), so failed or
// outdated lookups are redone from scratch.
func (h *BookHandler) AdminEnrichBooks(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	if r.URL.Query().Get("recheck") == "true" {
		if _, err := database.DB.Exec(
			`UPDATE books
			    SET metadata_checked_at = NULL,
			        author_name = CASE WHEN author_from_lookup THEN NULL ELSE author_name END,
			        author_from_lookup = false,
			        page_count = CASE WHEN pages_from_lookup THEN NULL ELSE page_count END,
			        pages_from_lookup = false`,
		); err != nil {
			http.Error(w, `{"error": "Failed to reset metadata"}`, http.StatusInternalServerError)
			return
		}
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	checked, resolved, ran, err := metadata.EnrichPending(r.Context(), limit)
	if err != nil {
		http.Error(w, `{"error": "Metadata lookup failed"}`, http.StatusInternalServerError)
		return
	}
	if !ran {
		http.Error(w, `{"error": "A metadata lookup is already running"}`, http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"checked": checked, "resolved": resolved})
}
