package server_test

import (
	"net/http"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

type tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Username     string `json:"username"`
}

func TestSignupLoginRefresh(t *testing.T) {
	a := newAPI(t)
	a.exec(`INSERT INTO preferences (subgenre, genre) VALUES ('Space opera', 'Sci-Fi')`)

	signup := map[string]string{"username": "ada", "email": "ada@example.com", "password": "hunter22"}
	var created tokens
	a.must(http.StatusCreated, "POST", "/signup", nil, signup, &created)
	if created.AccessToken == "" || created.RefreshToken == "" {
		t.Fatalf("signup returned no tokens: %+v", created)
	}

	t.Run("duplicate signup is a conflict", func(t *testing.T) {
		a.must(http.StatusConflict, "POST", "/signup", nil, signup, nil)
	})

	t.Run("login", func(t *testing.T) {
		var got tokens
		a.must(http.StatusOK, "POST", "/auth/login", nil,
			map[string]string{"username": "ada", "password": "hunter22"}, &got)
		if got.Username != "ada" || got.AccessToken == "" {
			t.Errorf("login = %+v", got)
		}
		a.must(http.StatusUnauthorized, "POST", "/auth/login", nil,
			map[string]string{"username": "ada", "password": "wrong"}, nil)
		a.must(http.StatusUnauthorized, "POST", "/auth/login", nil,
			map[string]string{"username": "nobody", "password": "hunter22"}, nil)
	})

	t.Run("refresh rotates the token", func(t *testing.T) {
		var login tokens
		a.must(http.StatusOK, "POST", "/auth/login", nil,
			map[string]string{"username": "ada", "password": "hunter22"}, &login)

		var refreshed tokens
		a.must(http.StatusOK, "POST", "/token/refresh", nil,
			map[string]string{"refreshToken": login.RefreshToken}, &refreshed)
		if refreshed.RefreshToken == "" || refreshed.RefreshToken == login.RefreshToken {
			t.Fatal("refresh should hand back a new refresh token")
		}
		// The old one was replaced, so replaying it must fail.
		a.must(http.StatusUnauthorized, "POST", "/token/refresh", nil,
			map[string]string{"refreshToken": login.RefreshToken}, nil)
		// An access token is not a refresh token.
		a.must(http.StatusUnauthorized, "POST", "/token/refresh", nil,
			map[string]string{"refreshToken": refreshed.AccessToken}, nil)
	})
}

func TestProtectedRoutesNeedAToken(t *testing.T) {
	a := newAPI(t)
	for _, r := range []struct{ method, path string }{
		{"GET", "/progress"},
		{"GET", "/room/my-rooms"},
		{"GET", "/room/open"},
		{"DELETE", "/comments/cm_1"},
		{"GET", "/users/me/blocks"},
		{"POST", "/users/me/blocks"},
		{"POST", "/reports"},
		{"POST", "/room/create"},
		{"GET", "/books/bk_1/highlights"},
		{"GET", "/users/me/buckets"},
	} {
		if got := a.do(r.method, r.path, nil, nil, nil); got != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", r.method, r.path, got)
		}
	}
}

// specs/offline-first R12: a phone and an iPad on one account each keep their
// own sign-in. With one stored token per account, either one signing in or
// refreshing signed the other out at its next refresh.
func TestEachDeviceKeepsItsSignIn(t *testing.T) {
	a := newAPI(t)
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": "ada", "email": "ada@example.com", "password": "hunter22"}, nil)

	signIn := func() tokens {
		t.Helper()
		var got tokens
		a.must(http.StatusOK, "POST", "/auth/login", nil,
			map[string]string{"username": "ada", "password": "hunter22"}, &got)
		return got
	}
	refresh := func(refreshToken string) tokens {
		t.Helper()
		var got tokens
		a.must(http.StatusOK, "POST", "/token/refresh", nil,
			map[string]string{"refreshToken": refreshToken}, &got)
		return got
	}

	phone := signIn()
	ipad := signIn()

	// Each refreshes, in turn and again: neither knocks the other out.
	phone = refresh(phone.RefreshToken)
	ipad = refresh(ipad.RefreshToken)
	phone = refresh(phone.RefreshToken)
	ipad = refresh(ipad.RefreshToken)

	// A token is still single use: the phone's old one is refused.
	old := phone.RefreshToken
	phone = refresh(old)
	a.must(http.StatusUnauthorized, "POST", "/token/refresh", nil,
		map[string]string{"refreshToken": old}, nil)

	// Signing out on the phone ends the phone's sign-in only.
	a.must(http.StatusOK, "POST", "/auth/logout?username=ada", &user{Token: phone.RefreshToken}, nil, nil)
	a.must(http.StatusUnauthorized, "POST", "/token/refresh", nil,
		map[string]string{"refreshToken": phone.RefreshToken}, nil)
	refresh(ipad.RefreshToken)
}

// R12.5: signing in clears the reader's expired tokens.
func TestSignInClearsExpiredTokens(t *testing.T) {
	a := newAPI(t)
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": "ada", "email": "ada@example.com", "password": "hunter22"}, nil)
	a.exec(`UPDATE refresh_tokens SET expires_at = NOW() - INTERVAL '1 day'`)
	a.must(http.StatusOK, "POST", "/auth/login", nil,
		map[string]string{"username": "ada", "password": "hunter22"}, nil)

	var expired, live int
	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FILTER (WHERE expires_at <= NOW()), COUNT(*) FILTER (WHERE expires_at > NOW()) FROM refresh_tokens`,
	).Scan(&expired, &live); err != nil {
		t.Fatal(err)
	}
	if expired != 0 || live != 1 {
		t.Errorf("after sign-in: %d expired, %d live tokens; want 0 and 1", expired, live)
	}
}
