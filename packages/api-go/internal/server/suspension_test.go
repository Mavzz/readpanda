package server_test

import (
	"net/http"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

type suspendedError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// suspended checks one request is refused as suspended (report-block R10.1).
func (a *api) suspended(method, path string, as *user, body any) {
	a.t.Helper()
	var got suspendedError
	a.must(http.StatusForbidden, method, path, as, body, &got)
	if got.Code != "account_suspended" ||
		got.Error != "Your account has been suspended. Email readpanda.support@gmail.com if you think this is a mistake." {
		a.t.Errorf("%s %s: %+v, want account_suspended", method, path, got)
	}
}

// report-block R10.1, R10.3, R10.4: a suspended reader can't sign in,
// refresh, write or refresh the app; other reads hold until the token
// expires; their comments stay; reinstating lets them back in.
func TestSuspendedAccount(t *testing.T) {
	a := newAPI(t)
	var signedUp tokens
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": "ada", "email": "ada@example.com", "password": "hunter22"}, &signedUp)
	var ada user
	a.must(http.StatusOK, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "hunter22"}, &signedUp)
	ada.Token = signedUp.AccessToken
	if err := database.DB.QueryRow(`SELECT uuid FROM users WHERE username = 'ada'`).Scan(&ada.ID); err != nil {
		t.Fatal(err)
	}

	bob := a.user("bob")
	book := a.book("Dune")
	club := a.newRoom(bob, "Club")
	a.joinRoom(ada, club)
	open := a.newRoom(bob, "Open")
	a.setOpen(bob, open, true)
	mine := a.newRoom(ada, "Ada's")
	a.comment(ada, club, book, map[string]any{"page": 1, "body": "before"})
	a.putProgress(bob, book, 10, 300)

	a.exec(`UPDATE users SET isactive = false WHERE uuid = $1`, ada.ID)

	// Signing in and refreshing. A wrong password still reads as one.
	a.suspended("POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "hunter22"})
	a.must(http.StatusUnauthorized, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "nope"}, nil)
	a.suspended("POST", "/token/refresh", nil, map[string]string{"refreshToken": signedUp.RefreshToken})

	// Writes.
	a.suspended("POST", "/room/"+club.ID+"/book/"+book+"/comments", &ada, map[string]any{"page": 1, "body": "after"})
	a.suspended("POST", "/room/create", &ada, map[string]any{"name": "New"})
	a.suspended("POST", "/room/join", &ada, map[string]string{"invite_code": open.InviteCode})
	a.suspended("POST", "/room/"+open.ID+"/join", &ada, nil)
	a.suspended("PATCH", "/room/"+mine.ID, &ada, map[string]any{"is_private": false})
	a.suspended("PATCH", "/room/"+mine.ID+"/reading", &ada, map[string]any{"current_book_id": book})
	a.suspended("POST", "/reports", &ada, map[string]any{"target_type": "user", "target_id": bob.ID, "reason": "spam"})
	a.suspended("POST", "/users/me/blocks", &ada, map[string]string{"user_id": bob.ID})
	a.suspended("DELETE", "/users/me/blocks/"+bob.ID, &ada, nil)

	// The app's refresh reads.
	a.suspended("GET", "/notifications/unread/count", &ada, nil)
	a.suspended("GET", "/room/my-rooms", &ada, nil)

	// Other reads still answer until the access token runs out (R10.2), and
	// what they wrote stays (R10.3).
	a.must(http.StatusOK, "GET", "/room/"+club.ID, &ada, nil, nil)
	if got := a.comments(bob, club, book); len(got.Threads) != 1 {
		t.Errorf("bob sees %d threads, want ada's comment kept", len(got.Threads))
	}

	// Reinstated (R10.4).
	a.exec(`UPDATE users SET isactive = true WHERE uuid = $1`, ada.ID)
	a.must(http.StatusOK, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "hunter22"}, nil)
	a.must(http.StatusCreated, "POST", "/room/create", &ada, map[string]any{"name": "Back"}, nil)
	a.must(http.StatusOK, "GET", "/room/my-rooms", &ada, nil, nil)
}
