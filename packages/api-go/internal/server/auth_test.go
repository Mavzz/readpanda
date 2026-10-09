package server_test

import (
	"net/http"
	"testing"
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
		{"POST", "/room/create"},
		{"GET", "/books/bk_1/highlights"},
		{"GET", "/users/me/buckets"},
	} {
		if got := a.do(r.method, r.path, nil, nil, nil); got != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", r.method, r.path, got)
		}
	}
}
