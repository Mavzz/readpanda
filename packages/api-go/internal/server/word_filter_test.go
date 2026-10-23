package server_test

import (
	"net/http"
	"testing"
)

// report-block R8.1, R8.2: comments, room names and descriptions, and new
// usernames with a listed word are refused, naming the field; a listed word
// inside a longer ordinary one isn't.
func TestWordFilter(t *testing.T) {
	a := newAPI(t)
	reader := a.user("reader")
	club := a.newRoom(reader, "Club")
	bk := a.book("Book")

	type refusal struct {
		Error string `json:"error"`
		Code  string `json:"code"`
		Field string `json:"field"`
	}
	refused := func(field, method, path string, as *user, body any) {
		t.Helper()
		var got refusal
		a.must(http.StatusBadRequest, method, path, as, body, &got)
		if got.Code != "blocked_words" || got.Field != field || got.Error != "That includes words we don't allow." {
			t.Errorf("%s %s: %+v, want blocked_words on %q", method, path, got, field)
		}
	}

	comments := "/room/" + club.ID + "/book/" + bk + "/comments"
	refused("body", "POST", comments, &reader, map[string]any{"body": "what a c u n t, no: CUNT", "page": 1})
	a.must(http.StatusCreated, "POST", comments, &reader, map[string]any{"body": "Scunthorpe, again", "page": 1}, nil)

	refused("name", "POST", "/room/create", &reader, map[string]any{"name": "F@g club"})
	refused("description", "POST", "/room/create", &reader, map[string]any{"name": "Fine", "description": "no retards"})

	refused("username", "POST", "/signup", nil,
		map[string]string{"username": "big.cunt", "email": "x@example.com", "password": "longenough"})
	refused("username", "GET", "/signup/available?username=kike_reader", nil, nil)
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": "scunthorpe", "email": "s@example.com", "password": "longenough"}, nil)
}
