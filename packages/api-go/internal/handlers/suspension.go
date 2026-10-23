package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

// suspendedMessage is what the app shows when it signs a suspended reader
// out (report-block R10.2).
const suspendedMessage = "Your account has been suspended. Email readpanda.support@gmail.com if you think this is a mistake."

// requireActive answers 403 {"error", "code": "account_suspended"} and
// returns false when the account is suspended (isactive = false, report-block
// R10.1). An account that's gone is left to the caller's own checks.
func requireActive(w http.ResponseWriter, userID string) bool {
	var suspended bool
	err := database.DB.QueryRow(`SELECT isactive IS FALSE FROM users WHERE uuid = $1`, userID).Scan(&suspended)
	if err == sql.ErrNoRows {
		return true
	}
	if err != nil {
		http.Error(w, `{"error": "Something went wrong"}`, http.StatusInternalServerError)
		return false
	}
	if suspended {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": suspendedMessage, "code": "account_suspended"})
		return false
	}
	return true
}
