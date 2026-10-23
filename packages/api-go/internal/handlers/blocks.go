package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// BlockedReader is one row of Settings › Blocked readers (report-block R6).
type BlockedReader struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	BlockedAt time.Time `json:"blocked_at"`
}

// BlockUser — POST /users/me/blocks {"user_id"} (report-block R5). 204, and
// again if they're already blocked. The blocked reader isn't told (R5.7).
func (h *UserHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	if !requireActive(w, userID) {
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == userID || req.UserID == deletedReaderUUID {
		http.Error(w, `{"error": "You can't block that reader"}`, http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		http.Error(w, `{"error": "Reader not found"}`, http.StatusNotFound)
		return
	}
	res, err := database.DB.Exec(
		`INSERT INTO user_blocks (blocker_id, blocked_id)
		 SELECT $1, uuid FROM users WHERE uuid = $2
		 ON CONFLICT DO NOTHING`,
		userID, req.UserID,
	)
	if err != nil {
		http.Error(w, `{"error": "Couldn't block that reader"}`, http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Nothing written: already blocked, or no such reader.
		var exists bool
		if err := database.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE uuid = $1)`, req.UserID).Scan(&exists); err != nil || !exists {
			http.Error(w, `{"error": "Reader not found"}`, http.StatusNotFound)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnblockUser — DELETE /users/me/blocks/{userId}. 204 whether or not they
// were blocked.
func (h *UserHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	if !requireActive(w, userID) {
		return
	}
	blockedID := mux.Vars(r)["userId"]
	if _, err := uuid.Parse(blockedID); err == nil {
		if _, err := database.DB.Exec(
			`DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`, userID, blockedID,
		); err != nil {
			http.Error(w, `{"error": "Couldn't unblock that reader"}`, http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListBlocks — GET /users/me/blocks: who the reader has blocked, newest first.
func (h *UserHandler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	rows, err := database.DB.Query(
		`SELECT b.blocked_id, u.username, b.created_at
		   FROM user_blocks b JOIN users u ON u.uuid = b.blocked_id
		  WHERE b.blocker_id = $1
		  ORDER BY b.created_at DESC, u.username`,
		userID,
	)
	if err != nil {
		http.Error(w, `{"error": "Couldn't load blocked readers"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	list := []BlockedReader{}
	for rows.Next() {
		var b BlockedReader
		if err := rows.Scan(&b.UserID, &b.Username, &b.BlockedAt); err != nil {
			http.Error(w, `{"error": "Couldn't load blocked readers"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Couldn't load blocked readers"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
