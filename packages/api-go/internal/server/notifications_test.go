package server_test

import (
	"net/http"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/notify"
)

const settingsPath = "/users/me/notification-settings"

func TestNotificationSettingsDefaultToAllOn(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")

	var got notify.Settings
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("a reader who never changed anything: got %+v, want all on", got)
	}
}

func TestNotificationSettingsUpdateASubset(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")
	bob := a.user("bob")

	var got notify.Settings
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"room_comments": false}, &got)
	want := notify.DefaultSettings()
	want.RoomComments = false
	if got != want {
		t.Fatalf("after turning room comments off: got %+v, want %+v", got, want)
	}

	// A second change leaves the first alone.
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"new_books": false}, &got)
	want.NewBooks = false
	if got != want {
		t.Fatalf("after turning new books off too: got %+v, want %+v", got, want)
	}

	// And it's what GET reads back.
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != want {
		t.Fatalf("GET after updates: got %+v, want %+v", got, want)
	}

	// Turning one back on.
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"room_comments": true}, &got)
	want.RoomComments = true
	if got != want {
		t.Fatalf("after turning room comments back on: got %+v, want %+v", got, want)
	}

	// Someone else's settings are untouched.
	a.must(http.StatusOK, "GET", settingsPath, &bob, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("bob's settings changed: %+v", got)
	}
}

func TestNotificationSettingsRejectsBadBodies(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")

	for name, body := range map[string]any{
		"unknown key":   map[string]bool{"room_comment": false},
		"not a boolean": map[string]string{"replies": "no"},
		"not an object": []bool{false},
	} {
		if code := a.do("PUT", settingsPath, &ann, body, nil); code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, code)
		}
	}

	// Nothing was saved by the rejected requests.
	var got notify.Settings
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("a rejected request changed settings: %+v", got)
	}
}

func TestNotificationSettingsNeedSignIn(t *testing.T) {
	a := newAPI(t)
	if code := a.do("GET", settingsPath, nil, nil, nil); code != http.StatusUnauthorized {
		t.Errorf("GET without a token: got %d, want 401", code)
	}
	if code := a.do("PUT", settingsPath, nil, map[string]bool{"replies": false}, nil); code != http.StatusUnauthorized {
		t.Errorf("PUT without a token: got %d, want 401", code)
	}
}

func TestNotificationSettingsForADeletedAccount(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")
	a.exec(`DELETE FROM users WHERE uuid = $1`, ann.ID)

	// The token is still valid, but there's no account to attach settings to.
	if code := a.do("PUT", settingsPath, &ann, map[string]bool{"replies": false}, nil); code != http.StatusNotFound {
		t.Errorf("PUT for a deleted account: got %d, want 404", code)
	}
}
