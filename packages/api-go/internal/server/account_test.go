package server_test

import (
	"net/http"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

// The reserved account deleted readers' comments move to (account-deletion
// R5.2); its id is fixed by scripts/migrate_account_deletion.sql.
const deletedReaderID = "00000000-0000-0000-0000-000000000000"

// count runs a COUNT query, failing the test on error.
func (a *api) count(query string, args ...any) int {
	a.t.Helper()
	var n int
	if err := database.DB.QueryRow(query, args...).Scan(&n); err != nil {
		a.t.Fatalf("%s: %v", query, err)
	}
	return n
}

// commentRow inserts a comment straight into the database.
func (a *api) commentRow(id, roomID, bookID string, by user, parentID any) {
	a.t.Helper()
	a.exec(`INSERT INTO book_comments (id, room_id, book_id, user_id, anchor_key, body, client_id, parent_id)
	        VALUES ($1, $2, $3, $4, 'k', 'a comment', $5, $6)`,
		id, roomID, bookID, by.ID, "cmt_"+id, parentID)
}

func TestDeleteMeNeedsAToken(t *testing.T) {
	a := newAPI(t)
	if got := a.do("DELETE", "/users/me", nil, nil, nil); got != http.StatusUnauthorized {
		t.Errorf("without a token = %d, want 401", got)
	}
}

// R5.1: the earliest-joined member takes over, ties broken by id; a room
// nobody else is in goes.
func TestDeleteMeHandsRoomsOver(t *testing.T) {
	a := newAPI(t)
	owner, early, late := a.user("owner"), a.user("early"), a.user("late")
	club := a.newRoom(owner, "Club")
	a.joinRoom(late, club)
	a.joinRoom(early, club)
	a.exec(`UPDATE room_members SET joined_at = '2026-01-01' WHERE user_id = $1`, early.ID)
	a.exec(`UPDATE room_members SET joined_at = '2026-02-01' WHERE user_id = $1`, late.ID)

	// Two members who joined at the same moment: the smaller id wins.
	tieA, tieB := a.user("tie_a"), a.user("tie_b")
	tied := a.newRoom(owner, "Tied")
	a.joinRoom(tieA, tied)
	a.joinRoom(tieB, tied)
	a.exec(`UPDATE room_members SET joined_at = '2026-03-01' WHERE room_id = $1`, tied.ID)
	tieWinner := tieA
	if tieB.ID < tieA.ID {
		tieWinner = tieB
	}

	alone := a.newRoom(owner, "Alone")

	a.must(http.StatusNoContent, "DELETE", "/users/me", &owner, nil, nil)

	for _, c := range []struct {
		room  room
		admin user
	}{{club, early}, {tied, tieWinner}} {
		if a.count(`SELECT count(*) FROM rooms WHERE id = $1 AND admin_id = $2`, c.room.ID, c.admin.ID) != 1 {
			t.Errorf("%s: not handed to %s", c.room.Name, c.admin.Name)
		}
		if a.count(`SELECT count(*) FROM room_members WHERE room_id = $1 AND user_id = $2 AND role = 'admin'`, c.room.ID, c.admin.ID) != 1 {
			t.Errorf("%s: %s isn't an admin member", c.room.Name, c.admin.Name)
		}
		// The new admin can open it, as its creator now.
		a.must(http.StatusOK, "GET", "/room/"+c.room.ID, &c.admin, nil, nil)
	}
	if a.count(`SELECT count(*) FROM rooms WHERE id = $1`, alone.ID) != 0 {
		t.Error("a room nobody else was in is still there")
	}
	if a.count(`SELECT count(*) FROM room_members WHERE room_id = $1 AND role = 'admin'`, club.ID) != 1 {
		t.Error("the handed-over room should have exactly one admin")
	}
}

// R5.2, R5.3: comments stay as Deleted reader with others' replies intact,
// in rooms that pass on; uploaded books stay with no uploader.
func TestDeleteMeKeepsWhatOthersShare(t *testing.T) {
	a := newAPI(t)
	owner, reader := a.user("owner"), a.user("reader")
	other, gone := a.user("other"), a.user("gone")
	club := a.newRoom(other, "Club")
	a.joinRoom(owner, club)
	a.joinRoom(gone, club)
	bk := a.book("Theirs")
	a.exec(`UPDATE books SET user_id = $1 WHERE book_id = $2`, owner.ID, bk)

	a.commentRow("c1", club.ID, bk, owner, nil)
	a.commentRow("c2", club.ID, bk, reader, "c1")
	// Same client id as c1: two deleted readers' comments must not collide
	// under the one Deleted reader account.
	a.exec(`INSERT INTO book_comments (id, room_id, book_id, user_id, anchor_key, body, client_id)
	        VALUES ('c3', $1, $2, $3, 'k', 'mine too', 'cmt_c1')`, club.ID, bk, gone.ID)

	a.must(http.StatusNoContent, "DELETE", "/users/me", &owner, nil, nil)
	a.must(http.StatusNoContent, "DELETE", "/users/me", &gone, nil, nil)

	if n := a.count(`SELECT count(*) FROM book_comments WHERE id IN ('c1', 'c3') AND user_id = $1 AND client_id IS NULL`, deletedReaderID); n != 2 {
		t.Errorf("%d of 2 comments moved to Deleted reader", n)
	}
	if a.count(`SELECT count(*) FROM book_comments WHERE id = 'c2' AND parent_id = 'c1' AND user_id = $1`, reader.ID) != 1 {
		t.Error("the reply to a deleted reader's comment is gone")
	}
	if a.count(`SELECT count(*) FROM users WHERE uuid = $1 AND username = 'deleted.reader' AND password IS NULL AND NOT isactive`, deletedReaderID) != 1 {
		t.Error("Deleted reader row missing or not locked down")
	}
	if a.count(`SELECT count(*) FROM books WHERE book_id = $1 AND user_id IS NULL`, bk) != 1 {
		t.Error("uploaded book not kept unlinked")
	}
	if a.count(`SELECT count(*) FROM rooms WHERE id = $1 AND admin_id = $2`, club.ID, other.ID) != 1 {
		t.Error("a room the deleted reader only joined changed hands")
	}
}

// R4.1, R6.1: nothing of theirs is left in any table that points at users,
// nor in reading_progress, which doesn't; a second call is a no-op 204.
func TestDeleteMeLeavesNothingBehind(t *testing.T) {
	a := newAPI(t)
	owner, other := a.user("owner"), a.user("other")
	club := a.newRoom(other, "Club")
	a.joinRoom(owner, club)
	bk := a.book("Book")
	a.commentRow("theirs", club.ID, bk, other, nil)

	a.exec(`INSERT INTO reading_progress (user_id, book_id) VALUES ($1, $2)`, owner.ID, bk)
	a.exec(`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, 't', NOW() + interval '1 day')`, owner.ID)
	a.exec(`INSERT INTO device_tokens (token, user_id) VALUES ('dev', $1)`, owner.ID)
	a.exec(`INSERT INTO notifications (user_id) VALUES ($1)`, owner.ID)
	a.exec(`INSERT INTO user_preferences (user_id) VALUES ($1)`, owner.ID)
	a.exec(`INSERT INTO user_buckets (id, user_id, name) VALUES ('b1', $1, 'Mine')`, owner.ID)
	a.exec(`INSERT INTO book_highlights (id, user_id, book_id, anchor_text) VALUES ('h1', $1, $2, 'x')`, owner.ID, bk)
	a.exec(`INSERT INTO book_comment_likes (comment_id, user_id) VALUES ('theirs', $1)`, owner.ID)
	a.exec(`INSERT INTO book_comment_reads (user_id, comment_id) VALUES ($1, 'theirs')`, owner.ID)
	a.exec(`UPDATE users SET apple_refresh_token = 'apple-token' WHERE uuid = $1`, owner.ID)

	// No Apple key in the test config: the revoke is skipped and logged, and
	// the deletion still succeeds (R4.3).
	a.must(http.StatusNoContent, "DELETE", "/users/me", &owner, nil, nil)
	a.must(http.StatusNoContent, "DELETE", "/users/me", &owner, nil, nil)

	// Every column with a foreign key to users, so a new table can't be
	// missed.
	rows, err := database.DB.Query(`
		SELECT cl.relname, att.attname
		FROM pg_constraint con
		JOIN pg_class cl ON cl.oid = con.conrelid
		JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ANY (con.conkey)
		WHERE con.contype = 'f' AND con.confrelid = 'public.users'::regclass`)
	if err != nil {
		t.Fatal(err)
	}
	type column struct{ table, name string }
	columns := []column{{"reading_progress", "user_id"}}
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.table, &c.name); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, c)
	}
	rows.Close()
	for _, c := range columns {
		if n := a.count(`SELECT count(*) FROM public.`+c.table+` WHERE `+c.name+` = $1`, owner.ID); n != 0 {
			t.Errorf("%s.%s: %d rows left", c.table, c.name, n)
		}
	}
	if a.count(`SELECT count(*) FROM users WHERE uuid = $1`, owner.ID) != 0 {
		t.Error("the account is still there")
	}
	// The other reader's room, comment and membership are untouched.
	if a.count(`SELECT count(*) FROM book_comments WHERE id = 'theirs' AND user_id = $1`, other.ID) != 1 {
		t.Error("someone else's comment was touched")
	}

	// Signing up again with the same email is a new account (R6.1).
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": "owner2", "email": "owner@example.com", "password": "longenough"}, nil)
}

// R5.4: the reserved name can't be signed up with, and reads as taken.
func TestDeletedReaderNameIsReserved(t *testing.T) {
	a := newAPI(t)
	if got := a.do("POST", "/signup", nil,
		map[string]string{"username": "Deleted.Reader", "email": "dr@example.com", "password": "longenough"}, nil); got != http.StatusBadRequest {
		t.Errorf("sign-up as deleted.reader = %d, want 400", got)
	}
	var avail struct {
		Available bool `json:"available"`
	}
	a.must(http.StatusOK, "GET", "/signup/available?username=deleted.reader", nil, nil, &avail)
	if avail.Available {
		t.Error("deleted.reader reported as available")
	}
}
