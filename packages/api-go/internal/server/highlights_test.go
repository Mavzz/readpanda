package server_test

import (
	"net/http"
	"testing"
)

type highlight struct {
	ID         string `json:"id"`
	Page       int    `json:"page"`
	AnchorText string `json:"anchor_text"`
}

func TestHighlightsArePrivate(t *testing.T) {
	a := newAPI(t)
	alice, bob := a.user("alice"), a.user("bob")
	book := a.book("Dune")
	path := "/books/" + book + "/highlights"

	var hl highlight
	a.must(http.StatusCreated, "POST", path, &alice,
		map[string]any{"page": 4, "anchor_text": "I must not fear.", "client_id": "h-1"}, &hl)

	// Retried POSTs land on the same row.
	var again highlight
	a.must(http.StatusCreated, "POST", path, &alice,
		map[string]any{"page": 4, "anchor_text": "I must not fear.", "client_id": "h-1"}, &again)
	if again.ID != hl.ID {
		t.Errorf("retry created a second highlight")
	}

	var mine, theirs []highlight
	a.must(http.StatusOK, "GET", path, &alice, nil, &mine)
	a.must(http.StatusOK, "GET", path, &bob, nil, &theirs)
	if len(mine) != 1 || len(theirs) != 0 {
		t.Errorf("alice sees %d, bob sees %d; want 1 and 0", len(mine), len(theirs))
	}

	// Someone else's id reads as not found, and the highlight survives.
	a.must(http.StatusNotFound, "DELETE", "/highlights/"+hl.ID, &bob, nil, nil)
	a.must(http.StatusNoContent, "DELETE", "/highlights/"+hl.ID, &alice, nil, nil)
	a.must(http.StatusNotFound, "DELETE", "/highlights/"+hl.ID, &alice, nil, nil)
}

func TestHighlightUnknownBook(t *testing.T) {
	a := newAPI(t)
	u := a.user("u")
	a.must(http.StatusNotFound, "POST", "/books/bk_missing/highlights", &u,
		map[string]any{"page": 1, "anchor_text": "x"}, nil)
}
