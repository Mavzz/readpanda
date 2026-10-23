package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Mavzz/readpanda/api-go/internal/moderation"
)

// refusedWords answers 400 {"error", "code": "blocked_words", "field"} and
// returns true when text holds a word on the list (report-block R8.1). The
// field tells the app which input to show the message under (R8.4).
func refusedWords(w http.ResponseWriter, field, text string) bool {
	if moderation.Check(text) == nil {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error": moderation.Message,
		"code":  "blocked_words",
		"field": field,
	})
	return true
}
