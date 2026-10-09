package server_test

import (
	"net/http"
	"strings"
	"testing"
)

type room struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code"`
	AdminID    string `json:"admin_id"`
}

// newRoom creates a room as owner and returns it.
func (a *api) newRoom(owner user, name string) room {
	a.t.Helper()
	var r room
	a.must(http.StatusCreated, "POST", "/room/create", &owner, map[string]any{"name": name}, &r)
	return r
}

// joinRoom adds reader to the room through the invite code.
func (a *api) joinRoom(reader user, r room) {
	a.t.Helper()
	a.must(http.StatusOK, "POST", "/room/join", &reader, map[string]string{"invite_code": r.InviteCode}, nil)
}

func TestCreateRoom(t *testing.T) {
	a := newAPI(t)
	owner := a.user("owner")

	r := a.newRoom(owner, "  Book Club  ")
	if r.Name != "Book Club" {
		t.Errorf("name = %q, want it trimmed", r.Name)
	}
	if len(r.InviteCode) != 6 {
		t.Errorf("invite code %q, want 6 characters", r.InviteCode)
	}
	if r.AdminID != owner.ID {
		t.Errorf("admin = %q, want the creator", r.AdminID)
	}

	// The creator can open their own room straight away.
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &owner, nil, nil)

	for _, name := range []string{"", "   ", strings.Repeat("x", 51)} {
		if got := a.do("POST", "/room/create", &owner, map[string]any{"name": name}, nil); got != http.StatusBadRequest {
			t.Errorf("name %q = %d, want 400", name, got)
		}
	}
}

func TestJoinRoom(t *testing.T) {
	a := newAPI(t)
	owner, reader, outsider := a.user("owner"), a.user("reader"), a.user("outsider")
	r := a.newRoom(owner, "Club")

	// A stranger can't see into the room before joining.
	a.must(http.StatusForbidden, "GET", "/room/"+r.ID, &outsider, nil, nil)

	// Codes are typed by people; case and stray spaces shouldn't matter.
	a.must(http.StatusOK, "POST", "/room/join", &reader,
		map[string]string{"invite_code": " " + strings.ToLower(r.InviteCode) + " "}, nil)
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &reader, nil, nil)

	a.must(http.StatusConflict, "POST", "/room/join", &reader, map[string]string{"invite_code": r.InviteCode}, nil)
	a.must(http.StatusConflict, "POST", "/room/join", &owner, map[string]string{"invite_code": r.InviteCode}, nil)
	a.must(http.StatusNotFound, "POST", "/room/join", &outsider, map[string]string{"invite_code": "ZZZZZZ"}, nil)
	a.must(http.StatusBadRequest, "POST", "/room/join", &outsider, map[string]string{"invite_code": ""}, nil)
}

func TestGetRoomUnknown(t *testing.T) {
	a := newAPI(t)
	u := a.user("u")
	a.must(http.StatusNotFound, "GET", "/room/rm_missing", &u, nil, nil)
}
