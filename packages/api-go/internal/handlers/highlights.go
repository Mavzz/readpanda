package handlers

import (
	"database/sql"
	"encoding/json"
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

// HighlightHandler handles personal highlights: passages a reader marked for
// themselves. They key on the reader and the book — no room, no audience —
// so every query here is scoped to the caller's own user id.
type HighlightHandler struct {
	Config *config.Config
}

func NewHighlightHandler(cfg *config.Config) *HighlightHandler {
	return &HighlightHandler{Config: cfg}
}

// GetBookHighlights — GET /books/{bookId}/highlights
//
// The caller's highlights on one book, in page order. Always 200 with an
// array, empty when there are none.
func (h *HighlightHandler) GetBookHighlights(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	bookID := mux.Vars(r)["bookId"]
	if bookID == "" {
		http.Error(w, `{"error": "Book id is required"}`, http.StatusBadRequest)
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, page, anchor_text, anchor_bounds, COALESCE(file_hash, ''),
		        COALESCE(client_id, ''), created_at
		   FROM book_highlights
		  WHERE user_id = $1 AND book_id = $2
		  ORDER BY page, created_at`,
		userID, bookID,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to get highlights"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	highlights := []models.BookHighlight{}
	for rows.Next() {
		var hl models.BookHighlight
		var bounds []byte
		if err := rows.Scan(&hl.ID, &hl.Page, &hl.AnchorText, &bounds, &hl.FileHash,
			&hl.ClientID, &hl.CreatedAt); err != nil {
			http.Error(w, `{"error": "Failed to scan highlights"}`, http.StatusInternalServerError)
			return
		}
		hl.BookID = bookID
		if len(bounds) > 0 {
			hl.AnchorBounds = json.RawMessage(bounds)
		}
		highlights = append(highlights, hl)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Failed to get highlights"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(highlights)
}

// CreateHighlight — POST /books/{bookId}/highlights
//
// Body: {page, anchor_text, anchor_bounds, file_hash, client_id}
//
// Idempotent on client_id, like comments: the mobile API client retries a
// failed POST, and the second arrival gets the row the first one wrote.
func (h *HighlightHandler) CreateHighlight(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	bookID := mux.Vars(r)["bookId"]
	if bookID == "" {
		http.Error(w, `{"error": "Book id is required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Page         int             `json:"page"`
		AnchorText   string          `json:"anchor_text"`
		AnchorBounds json.RawMessage `json:"anchor_bounds"`
		FileHash     string          `json:"file_hash"`
		ClientID     string          `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Page < 0 {
		http.Error(w, `{"error": "Page number cannot be negative"}`, http.StatusBadRequest)
		return
	}
	// A highlight is a passage — there is no page-level version of one.
	anchorText := strings.TrimSpace(req.AnchorText)
	if anchorText == "" {
		http.Error(w, `{"error": "A highlight needs selected text"}`, http.StatusBadRequest)
		return
	}
	// Same cap as comment anchors: keep the head of a long drag rather than
	// rejecting it.
	if len(anchorText) > maxAnchorLength {
		anchorText = anchorText[:maxAnchorLength]
	}

	var boundsArg interface{}
	if len(req.AnchorBounds) > 0 && string(req.AnchorBounds) != "null" {
		boundsArg = []byte(req.AnchorBounds)
	}

	var stored models.BookHighlight
	var storedBounds []byte
	err := database.DB.QueryRow(
		`INSERT INTO book_highlights
		    (id, user_id, book_id, file_hash, page, anchor_text, anchor_bounds, client_id, created_at)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, NULLIF($8, ''), $9)
		 ON CONFLICT (user_id, client_id) WHERE client_id IS NOT NULL
		 DO UPDATE SET created_at = book_highlights.created_at
		 RETURNING id, page, anchor_text, anchor_bounds, COALESCE(file_hash, ''),
		           COALESCE(client_id, ''), created_at`,
		"hl_"+uuid.New().String(), userID, bookID, req.FileHash, req.Page, anchorText,
		boundsArg, req.ClientID, time.Now().UTC(),
	).Scan(&stored.ID, &stored.Page, &stored.AnchorText, &storedBounds, &stored.FileHash,
		&stored.ClientID, &stored.CreatedAt)
	if isForeignKeyViolation(err) {
		http.Error(w, `{"error": "Book not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to save highlight"}`, http.StatusInternalServerError)
		return
	}
	stored.BookID = bookID
	if len(storedBounds) > 0 {
		stored.AnchorBounds = json.RawMessage(storedBounds)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stored)
}

// DeleteHighlight — DELETE /highlights/{highlightId}
//
// Only the author's own highlight can be removed; anyone else's id reads as
// not found rather than forbidden, so ids can't be probed. Deleting one that
// is already gone is a 404 the client treats as done.
func (h *HighlightHandler) DeleteHighlight(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	highlightID := mux.Vars(r)["highlightId"]
	if highlightID == "" {
		http.Error(w, `{"error": "Highlight id is required"}`, http.StatusBadRequest)
		return
	}

	var deleted string
	err := database.DB.QueryRow(
		`DELETE FROM book_highlights WHERE id = $1 AND user_id = $2 RETURNING id`,
		highlightID, userID,
	).Scan(&deleted)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Highlight not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to delete highlight"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
