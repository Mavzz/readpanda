package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

const (
	maxCommentBody  = 2000
	maxAnchorLength = 1000
)

// CommentHandler handles passage-anchored, room-scoped comments (handoff 6a/6b).
type CommentHandler struct {
	Config *config.Config
}

func NewCommentHandler(cfg *config.Config) *CommentHandler {
	return &CommentHandler{Config: cfg}
}

// anchorKey groups everything written about the same passage into one thread
// and one gutter dot. Selections of the same sentence rarely agree on their
// exact edges, so the text is lowercased and its whitespace collapsed before
// hashing; a page-level comment keys on the page alone.
//
// Always computed here. A client-supplied key would let one device drop its
// comment into somebody else's thread.
func anchorKey(page int, anchorText string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(anchorText), " "))
	if normalized == "" {
		return "page:" + strconv.Itoa(page)
	}
	sum := sha256.Sum256([]byte(strconv.Itoa(page) + ":" + normalized))
	return hex.EncodeToString(sum[:])[:32]
}

// furthestPageFor is the spoiler line: the furthest page this reader has ever
// reached in this book. A reader who has never opened it is at 0.
func furthestPageFor(userID, bookID string) (int, error) {
	var furthest int
	err := database.DB.QueryRow(
		`SELECT furthest_page FROM reading_progress WHERE user_id = $1 AND book_id = $2`,
		userID, bookID,
	).Scan(&furthest)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return furthest, err
}

// GetBookComments — GET /room/{id}/book/{bookId}/comments
//
// The comment layer for one book in one room: unlocked threads in full, and
// everything still ahead of the reader as a single number.
//
// The spoiler rule is applied here, in SQL, not in the client. A locked
// comment's body, quote, page and id are never read into the response, so
// there is nothing in the payload for a curious reader to pull apart.
func (h *CommentHandler) GetBookComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]
	bookID := mux.Vars(r)["bookId"]
	if roomID == "" || bookID == "" {
		http.Error(w, `{"error": "Room id and book id are required"}`, http.StatusBadRequest)
		return
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get comments"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	furthest, err := furthestPageFor(userID, bookID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get comments"}`, http.StatusInternalServerError)
		return
	}

	resp := models.BookCommentsResponse{
		RoomID:       roomID,
		BookID:       bookID,
		FurthestPage: furthest,
		Threads:      []models.BookCommentThread{},
	}

	// The thread is the unit of visibility: a reply is shown exactly when the
	// comment it answers is. Deciding per-comment instead would hand a reader
	// their own reply to a passage they can no longer see the root of — an
	// orphan with nothing to attach to.
	//
	// Within that, a reader always sees their own threads however far ahead
	// they were written. Progress is only published when they leave the reader
	// screen, so without the author bypass you could comment on page 200 and
	// have the server hand it straight back to you locked.
	visibleRoots := `SELECT id FROM book_comments
	                  WHERE room_id = $1 AND book_id = $2 AND parent_id IS NULL
	                    AND (page <= $4 OR user_id = $3)`

	rows, err := database.DB.Query(
		`SELECT c.id, c.user_id, u.username, c.page, COALESCE(c.anchor_text, ''),
		        c.anchor_bounds, c.anchor_key, COALESCE(c.file_hash, ''), c.body,
		        COALESCE(c.parent_id, ''), c.created_at,
		        (SELECT COUNT(*) FROM book_comment_likes l WHERE l.comment_id = c.id),
		        EXISTS (SELECT 1 FROM book_comment_likes l WHERE l.comment_id = c.id AND l.user_id = $3),
		        EXISTS (SELECT 1 FROM book_comment_reads rd WHERE rd.comment_id = c.id AND rd.user_id = $3)
		   FROM book_comments c
		   JOIN users u ON u.uuid = c.user_id
		  WHERE c.room_id = $1 AND c.book_id = $2
		    AND (c.id IN (`+visibleRoots+`) OR c.parent_id IN (`+visibleRoots+`))
		  ORDER BY c.page, c.created_at`,
		roomID, bookID, userID, furthest,
	)
	if err != nil {
		http.Error(w, `{"error": "Failed to get comments"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Roots in encounter order (already sorted by page then time), replies
	// bucketed by the root they belong to.
	var roots []models.BookComment
	repliesByParent := map[string][]models.BookComment{}
	// The edition each root's anchor was taken from, carried on the thread
	// rather than the comment — the reader compares it against the file it
	// actually opened before drawing anything on the page.
	fileHashByRoot := map[string]string{}

	for rows.Next() {
		var c models.BookComment
		var bounds []byte
		var anchorText, anchorKeyValue, fileHash string
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Username, &c.Page, &anchorText, &bounds, &anchorKeyValue,
			&fileHash, &c.Body, &c.ParentID, &c.CreatedAt, &c.Likes, &c.LikedByMe, &c.Read,
		); err != nil {
			http.Error(w, `{"error": "Failed to scan comments"}`, http.StatusInternalServerError)
			return
		}
		c.RoomID = roomID
		c.BookID = bookID
		c.Anchor = models.CommentAnchor{Page: c.Page, Text: anchorText, Key: anchorKeyValue}
		if len(bounds) > 0 {
			c.Anchor.Bounds = json.RawMessage(bounds)
		}
		c.Replies = []models.BookComment{}

		if c.ParentID == "" {
			roots = append(roots, c)
			fileHashByRoot[c.ID] = fileHash
		} else {
			repliesByParent[c.ParentID] = append(repliesByParent[c.ParentID], c)
		}
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Failed to get comments"}`, http.StatusInternalServerError)
		return
	}

	// Everything sharing an anchor is one thread — two people highlighting the
	// same sentence get one dot, not two.
	threadIndex := map[string]int{}
	for _, root := range roots {
		root.Replies = repliesByParent[root.ID]
		if root.Replies == nil {
			root.Replies = []models.BookComment{}
		}

		idx, seen := threadIndex[root.Anchor.Key]
		if !seen {
			resp.Threads = append(resp.Threads, models.BookCommentThread{
				AnchorKey:    root.Anchor.Key,
				Page:         root.Page,
				AnchorText:   root.Anchor.Text,
				AnchorBounds: root.Anchor.Bounds,
				FileHash:     fileHashByRoot[root.ID],
				Comments:     []models.BookComment{},
			})
			idx = len(resp.Threads) - 1
			threadIndex[root.Anchor.Key] = idx
		}
		resp.Threads[idx].Comments = append(resp.Threads[idx].Comments, root)

		if !root.Read {
			resp.Threads[idx].UnreadCount++
			resp.UnlockedUnreadCount++
		}
		for _, reply := range root.Replies {
			if !reply.Read {
				resp.Threads[idx].UnreadCount++
				resp.UnlockedUnreadCount++
			}
		}
	}

	// Locked comments are counted and nothing more. This query deliberately
	// selects no column that could describe them — not a page, not an id.
	//
	// Counted by thread, the same way they're hidden: a reply inside a thread
	// the reader hasn't reached is one of the things still waiting for them,
	// even though its own row carries no page of its own worth reading.
	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM book_comments c
		  WHERE c.room_id = $1 AND c.book_id = $2 AND c.user_id <> $3
		    AND COALESCE(c.parent_id, c.id) NOT IN (`+visibleRoots+`)`,
		roomID, bookID, userID, furthest,
	).Scan(&resp.LockedCount); err != nil {
		http.Error(w, `{"error": "Failed to get comments"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// CreateComment — POST /room/{id}/book/{bookId}/comments
//
// Body: {page, anchor_text, anchor_bounds, parent_id, body, client_id, file_hash}
//
// A reply inherits its root's page, anchor and room rather than trusting the
// body: posting into a thread is a statement about that passage, and letting a
// client re-anchor it would put a reply somewhere its thread isn't.
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]
	bookID := mux.Vars(r)["bookId"]
	if roomID == "" || bookID == "" {
		http.Error(w, `{"error": "Room id and book id are required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Page         int             `json:"page"`
		AnchorText   string          `json:"anchor_text"`
		AnchorBounds json.RawMessage `json:"anchor_bounds"`
		ParentID     string          `json:"parent_id"`
		Body         string          `json:"body"`
		ClientID     string          `json:"client_id"`
		FileHash     string          `json:"file_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" || len(body) > maxCommentBody {
		http.Error(w, `{"error": "Comment must be 1-2000 characters"}`, http.StatusBadRequest)
		return
	}
	if req.Page < 0 {
		http.Error(w, `{"error": "Page number cannot be negative"}`, http.StatusBadRequest)
		return
	}
	// A long drag shouldn't cost the reader their comment — keep the head of
	// the selection rather than rejecting the write.
	anchorText := strings.TrimSpace(req.AnchorText)
	if len(anchorText) > maxAnchorLength {
		anchorText = anchorText[:maxAnchorLength]
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to save comment"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	page := req.Page
	anchorBounds := req.AnchorBounds
	fileHash := req.FileHash
	var parentID interface{}

	if req.ParentID != "" {
		var parentOfParent sql.NullString
		var rootRoom string
		var rootBounds []byte
		var rootAnchor sql.NullString
		var rootHash sql.NullString
		err := database.DB.QueryRow(
			`SELECT parent_id, room_id, page, anchor_text, anchor_bounds, file_hash
			   FROM book_comments WHERE id = $1 AND book_id = $2`,
			req.ParentID, bookID,
		).Scan(&parentOfParent, &rootRoom, &page, &rootAnchor, &rootBounds, &rootHash)
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Comment not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, `{"error": "Failed to save comment"}`, http.StatusInternalServerError)
			return
		}
		// One level of nesting: a reply to a reply belongs on the root.
		if parentOfParent.Valid {
			http.Error(w, `{"error": "Replies cannot be nested"}`, http.StatusBadRequest)
			return
		}
		if rootRoom != roomID {
			http.Error(w, `{"error": "Comment not found"}`, http.StatusNotFound)
			return
		}
		anchorText = rootAnchor.String
		anchorBounds = json.RawMessage(rootBounds)
		fileHash = rootHash.String
		parentID = req.ParentID
	}

	commentID := "cm_" + uuid.New().String()
	key := anchorKey(page, anchorText)
	now := time.Now().UTC()

	var boundsArg interface{}
	if len(anchorBounds) > 0 {
		boundsArg = []byte(anchorBounds)
	}
	var anchorArg interface{}
	if anchorText != "" {
		anchorArg = anchorText
	}

	var stored models.BookComment
	var storedBounds []byte
	var storedParent sql.NullString
	// The mobile API client retries a failed POST, so the same comment can
	// arrive more than once. The client id makes the second arrival return the
	// row the first one wrote instead of adding another; the no-op DO UPDATE
	// is what lets RETURNING hand it back.
	err = database.DB.QueryRow(
		`INSERT INTO book_comments
		    (id, room_id, book_id, file_hash, user_id, page, anchor_text, anchor_bounds,
		     anchor_key, body, parent_id, client_id, created_at, updated_at)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11, NULLIF($12, ''), $13, $13)
		 ON CONFLICT (user_id, client_id) WHERE client_id IS NOT NULL
		 DO UPDATE SET updated_at = book_comments.updated_at
		 RETURNING id, page, COALESCE(anchor_text, ''), anchor_bounds, anchor_key,
		           body, parent_id, created_at`,
		commentID, roomID, bookID, fileHash, userID, page, anchorArg, boundsArg,
		key, body, parentID, req.ClientID, now,
	).Scan(&stored.ID, &stored.Page, &stored.Anchor.Text, &storedBounds, &stored.Anchor.Key,
		&stored.Body, &storedParent, &stored.CreatedAt)
	if isForeignKeyViolation(err) {
		http.Error(w, `{"error": "Room or book not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to save comment"}`, http.StatusInternalServerError)
		return
	}
	stored.RoomID = roomID
	stored.BookID = bookID
	stored.UserID = userID
	stored.ParentID = storedParent.String
	stored.Anchor.Page = stored.Page
	if len(storedBounds) > 0 {
		stored.Anchor.Bounds = json.RawMessage(storedBounds)
	}
	stored.Replies = []models.BookComment{}
	stored.Read = true

	// Writing about a page proves the author reached it. Without this their own
	// comment comes back locked until they next leave the reader and progress
	// is published.
	if _, err := database.DB.Exec(
		`INSERT INTO reading_progress (user_id, book_id, current_page, total_pages, furthest_page, last_read_at, updated_at)
		 VALUES ($1, $2, $3, 0, $3, $4, $4)
		 ON CONFLICT (user_id, book_id) DO UPDATE
		    SET furthest_page = GREATEST(reading_progress.furthest_page, EXCLUDED.furthest_page),
		        updated_at    = EXCLUDED.updated_at`,
		userID, bookID, page, now,
	); err != nil {
		// The comment is saved; a stale pace marker is not worth failing on.
		log.Printf("comments: failed to advance furthest_page for %s/%s: %v", userID, bookID, err)
	}

	// Your own comment is never an unread notification to yourself.
	if _, err := database.DB.Exec(
		`INSERT INTO book_comment_reads (user_id, comment_id, read_at) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		userID, stored.ID, now,
	); err != nil {
		log.Printf("comments: failed to mark own comment read %s: %v", stored.ID, err)
	}

	if err := database.DB.QueryRow(`SELECT username FROM users WHERE uuid = $1`, userID).Scan(&stored.Username); err != nil {
		// A missing display name is not worth losing a saved comment over —
		// the client already knows who it is.
		log.Printf("comments: failed to read username for %s: %v", userID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stored)
}

// MarkRead — POST /room/{id}/book/{bookId}/comments/read  {"comment_ids": [...]}
//
// Opening a thread marks the comments in it read, which is what clears the
// gutter dot, the scrubber tick and both counters.
func (h *CommentHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	roomID := mux.Vars(r)["id"]
	bookID := mux.Vars(r)["bookId"]

	var req struct {
		CommentIDs []string `json:"comment_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if len(req.CommentIDs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to mark comments read"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	// The subquery is the authorization: ids that aren't comments on this book
	// in this room are silently dropped rather than trusted.
	if _, err := database.DB.Exec(
		`INSERT INTO book_comment_reads (user_id, comment_id, read_at)
		 SELECT $1, c.id, $2 FROM book_comments c
		  WHERE c.id = ANY($3) AND c.room_id = $4 AND c.book_id = $5
		 ON CONFLICT DO NOTHING`,
		userID, time.Now().UTC(), pq.Array(req.CommentIDs), roomID, bookID,
	); err != nil {
		http.Error(w, `{"error": "Failed to mark comments read"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LikeComment — POST /comments/{commentId}/like
func (h *CommentHandler) LikeComment(w http.ResponseWriter, r *http.Request) {
	h.setLike(w, r, true)
}

// UnlikeComment — DELETE /comments/{commentId}/like
func (h *CommentHandler) UnlikeComment(w http.ResponseWriter, r *http.Request) {
	h.setLike(w, r, false)
}

// setLike is both halves of the heart. Membership is checked through the
// comment's own room, so a like can't be dropped on a conversation the caller
// can't see.
func (h *CommentHandler) setLike(w http.ResponseWriter, r *http.Request, liked bool) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	commentID := mux.Vars(r)["commentId"]
	if commentID == "" {
		http.Error(w, `{"error": "Comment id is required"}`, http.StatusBadRequest)
		return
	}

	var roomID string
	err := database.DB.QueryRow(`SELECT room_id FROM book_comments WHERE id = $1`, commentID).Scan(&roomID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Comment not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to update like"}`, http.StatusInternalServerError)
		return
	}

	member, err := isRoomMember(roomID, userID)
	if err != nil {
		http.Error(w, `{"error": "Failed to update like"}`, http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, `{"error": "Not a member of this room"}`, http.StatusForbidden)
		return
	}

	if liked {
		_, err = database.DB.Exec(
			`INSERT INTO book_comment_likes (comment_id, user_id, created_at) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`,
			commentID, userID, time.Now().UTC(),
		)
	} else {
		_, err = database.DB.Exec(
			`DELETE FROM book_comment_likes WHERE comment_id = $1 AND user_id = $2`,
			commentID, userID,
		)
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to update like"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
