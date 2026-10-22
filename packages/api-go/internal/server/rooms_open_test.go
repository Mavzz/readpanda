package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type openRoomList struct {
	Rooms   []map[string]json.RawMessage `json:"rooms"`
	HasMore bool                         `json:"has_more"`
}

// setOpen switches who can join through the creator's PATCH.
func (a *api) setOpen(owner user, r room, open bool) {
	a.t.Helper()
	a.must(http.StatusOK, "PATCH", "/room/"+r.ID, &owner, map[string]any{"is_private": !open}, nil)
}

func (a *api) openRooms(as user, query string) openRoomList {
	a.t.Helper()
	var list openRoomList
	a.must(http.StatusOK, "GET", "/room/open"+query, &as, nil, &list)
	return list
}

func openRoomIDs(list openRoomList) []string {
	ids := make([]string, len(list.Rooms))
	for i, r := range list.Rooms {
		var id string
		json.Unmarshal(r["id"], &id)
		ids[i] = id
	}
	return ids
}

func sameIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestCreateRoomDefaultsToInviteOnly(t *testing.T) {
	a := newAPI(t)
	owner := a.user("owner")

	var made struct {
		IsPrivate bool `json:"is_private"`
	}
	a.must(http.StatusCreated, "POST", "/room/create", &owner, map[string]any{"name": "Quiet"}, &made)
	if !made.IsPrivate {
		t.Error("a room created without is_private is open, want invite-only")
	}
	a.must(http.StatusCreated, "POST", "/room/create", &owner, map[string]any{"name": "Loud", "is_private": false}, &made)
	if made.IsPrivate {
		t.Error("is_private false was ignored")
	}
}

func TestListOpenRooms(t *testing.T) {
	a := newAPI(t)
	owner, member, reader := a.user("owner"), a.user("member"), a.user("reader")
	book := a.book("Middlemarch")

	open := a.newRoom(owner, "Open")
	a.setOpen(owner, open, true)
	a.newRoom(owner, "Closed")
	joined := a.newRoom(owner, "Joined")
	a.setOpen(owner, joined, true)
	a.joinRoom(reader, joined)

	// Only open rooms, and not ones the reader is in or made.
	if got := openRoomIDs(a.openRooms(reader, "")); !sameIDs(got, []string{open.ID}) {
		t.Errorf("reader sees %v, want only the open room they aren't in", got)
	}
	if got := a.openRooms(owner, "").Rooms; len(got) != 0 {
		t.Errorf("the creator sees %d of their own rooms, want none", len(got))
	}

	// Nothing a member alone should see.
	row := a.openRooms(reader, "").Rooms[0]
	for _, key := range []string{"invite_code", "members", "admin_id"} {
		if _, ok := row[key]; ok {
			t.Errorf("open room carries %q", key)
		}
	}
	var creator string
	var count int
	json.Unmarshal(row["creator_username"], &creator)
	json.Unmarshal(row["member_count"], &count)
	if creator != "owner" || count != 1 {
		t.Errorf("creator %q, member_count %d; want owner, 1", creator, count)
	}

	// Reading a book puts a room first, even ahead of a bigger room; and the
	// book filter keeps only rooms reading it.
	a.joinRoom(member, joined)
	a.must(http.StatusOK, "PATCH", "/room/"+open.ID+"/reading", &owner, map[string]any{"current_book_id": book}, nil)
	bigger := a.newRoom(owner, "Bigger")
	a.setOpen(owner, bigger, true)
	a.joinRoom(member, bigger)
	if got := openRoomIDs(a.openRooms(reader, "")); !sameIDs(got, []string{open.ID, bigger.ID}) {
		t.Errorf("order %v, want the room reading a book first", got)
	}
	if got := openRoomIDs(a.openRooms(reader, "?book_id="+book)); !sameIDs(got, []string{open.ID}) {
		t.Errorf("book filter gives %v, want only the room reading it", got)
	}

	// Then bigger rooms, then newer ones.
	newer := a.newRoom(owner, "Newer")
	a.setOpen(owner, newer, true)
	a.exec(`UPDATE rooms SET created_at = created_at - interval '1 day' WHERE id = $1`, bigger.ID)
	older := a.newRoom(owner, "Older")
	a.setOpen(owner, older, true)
	a.exec(`UPDATE rooms SET created_at = created_at - interval '2 days' WHERE id = $1`, older.ID)
	if got := openRoomIDs(a.openRooms(reader, "")); !sameIDs(got, []string{open.ID, bigger.ID, newer.ID, older.ID}) {
		t.Errorf("order %v, want book, then size, then newest", got)
	}

	// Pages say whether there are more.
	first := a.openRooms(reader, "?limit=3")
	last := a.openRooms(reader, "?limit=3&offset=3")
	if len(first.Rooms) != 3 || !first.HasMore || len(last.Rooms) != 1 || last.HasMore {
		t.Errorf("pages %d (more %v) and %d (more %v), want 3 (true) and 1 (false)",
			len(first.Rooms), first.HasMore, len(last.Rooms), last.HasMore)
	}
}

func TestJoinOpenRoom(t *testing.T) {
	a := newAPI(t)
	owner, reader, other := a.user("owner"), a.user("reader"), a.user("other")
	r := a.newRoom(owner, "Club")

	// No code, no way in while it's invite-only.
	a.must(http.StatusForbidden, "POST", "/room/"+r.ID+"/join", &reader, nil, nil)

	a.setOpen(owner, r, true)
	a.must(http.StatusOK, "POST", "/room/"+r.ID+"/join", &reader, nil, nil)
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &reader, nil, nil)
	a.must(http.StatusConflict, "POST", "/room/"+r.ID+"/join", &reader, nil, nil)
	a.must(http.StatusNotFound, "POST", "/room/rm_missing/join", &reader, nil, nil)

	// The creator hears about it as for a join by code.
	var inbox []struct {
		Type string `json:"type"`
	}
	a.must(http.StatusOK, "GET", "/notifications", &owner, nil, &inbox)
	if len(inbox) != 1 || inbox[0].Type != "ROOM_JOIN" {
		t.Errorf("creator's inbox %v, want one ROOM_JOIN", inbox)
	}

	// Closing it keeps the reader in and the code working; a member tapping a
	// stale preview is told they're in, not that it's invite-only.
	a.setOpen(owner, r, false)
	a.must(http.StatusOK, "GET", "/room/"+r.ID, &reader, nil, nil)
	a.must(http.StatusConflict, "POST", "/room/"+r.ID+"/join", &reader, nil, nil)
	a.must(http.StatusForbidden, "POST", "/room/"+r.ID+"/join", &other, nil, nil)
	a.joinRoom(other, r)
}

func TestUpdateRoomIsCreatorOnly(t *testing.T) {
	a := newAPI(t)
	owner, reader := a.user("owner"), a.user("reader")
	r := a.newRoom(owner, "Club")
	a.joinRoom(reader, r)

	a.must(http.StatusForbidden, "PATCH", "/room/"+r.ID, &reader, map[string]any{"is_private": false}, nil)
	a.must(http.StatusNotFound, "PATCH", "/room/rm_missing", &owner, map[string]any{"is_private": false}, nil)

	var detail struct {
		IsPrivate  bool   `json:"is_private"`
		InviteCode string `json:"invite_code"`
	}
	a.must(http.StatusOK, "PATCH", "/room/"+r.ID, &owner, map[string]any{"is_private": false}, &detail)
	if detail.IsPrivate || detail.InviteCode != r.InviteCode {
		t.Errorf("after opening: is_private %v, code %q; want false and the same code", detail.IsPrivate, detail.InviteCode)
	}
}
