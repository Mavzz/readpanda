package server_test

import (
	"net/http"
	"testing"
)

// App spec coming-soon R5, R6.2.

type account struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	UserID       string  `json:"user_id"`
	Username     string  `json:"username"`
	DisplayName  *string `json:"display_name"`
	Email        string  `json:"email"`
	LoginType    string  `json:"login_type"`
}

// signUp creates an email account through /signup, so it has a password.
func (a *api) signUp(name string) (user, account) {
	a.t.Helper()
	var got account
	a.must(http.StatusCreated, "POST", "/signup", nil,
		map[string]string{"username": name, "email": name + "@example.com", "password": "hunter22"}, &got)
	return user{ID: got.UserID, Name: name, Token: got.AccessToken}, got
}

func TestSignInResponsesCarryTheAccount(t *testing.T) {
	a := newAPI(t)
	_, created := a.signUp("ada")
	if created.UserID == "" || created.LoginType != "email" || created.DisplayName != nil {
		t.Fatalf("signup = %+v", created)
	}

	var login account
	a.must(http.StatusOK, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "hunter22"}, &login)
	if login.UserID != created.UserID || login.LoginType != "email" {
		t.Fatalf("login = %+v, signup id %s", login, created.UserID)
	}

	var refreshed account
	a.must(http.StatusOK, "POST", "/token/refresh", nil, map[string]string{"refreshToken": login.RefreshToken}, &refreshed)
	if refreshed.UserID != created.UserID || refreshed.AccessToken == "" {
		t.Fatalf("refresh = %+v", refreshed)
	}
}

// An older session reads its id and sign-in kind here (R5.7, R6.2).
func TestGetMeIsTheAccount(t *testing.T) {
	a := newAPI(t)
	me, created := a.signUp("ada")
	var got account
	a.must(http.StatusOK, "GET", "/users/me", &me, nil, &got)
	if got.UserID != created.UserID || got.Username != "ada" || got.LoginType != "email" || got.Email != "ada@example.com" {
		t.Fatalf("GET /users/me = %+v", got)
	}
	a.must(http.StatusUnauthorized, "GET", "/users/me", nil, nil, nil)
}

func TestPatchMeDisplayName(t *testing.T) {
	a := newAPI(t)
	u, _ := a.signUp("ada")

	var got account
	a.must(http.StatusOK, "PATCH", "/users/me", &u, map[string]string{"display_name": "  Ada Lovelace "}, &got)
	if got.DisplayName == nil || *got.DisplayName != "Ada Lovelace" {
		t.Fatalf("display name = %v", got.DisplayName)
	}

	a.must(http.StatusOK, "PATCH", "/users/me", &u, map[string]string{"display_name": "  "}, &got)
	if got.DisplayName != nil {
		t.Fatalf("empty should clear, got %q", *got.DisplayName)
	}

	long := "abcdefghijabcdefghijabcdefghijabcdefghijX"
	a.must(http.StatusBadRequest, "PATCH", "/users/me", &u, map[string]string{"display_name": long}, nil)
}

func TestPatchMeUsername(t *testing.T) {
	a := newAPI(t)
	u, _ := a.signUp("ada")
	a.signUp("grace")

	var got account
	a.must(http.StatusOK, "PATCH", "/users/me", &u, map[string]string{"username": " Ada.Reads "}, &got)
	if got.Username != "ada.reads" {
		t.Fatalf("username = %q, want ada.reads", got.Username)
	}
	// The new name signs in; the id behind it is unchanged.
	var login account
	a.must(http.StatusOK, "POST", "/auth/login", nil, map[string]string{"username": "ada.reads", "password": "hunter22"}, &login)
	if login.UserID != u.ID {
		t.Fatalf("login id %s, want %s", login.UserID, u.ID)
	}

	// Its own name again, in another case, is no change.
	a.must(http.StatusOK, "PATCH", "/users/me", &u, map[string]string{"username": "ADA.READS"}, nil)

	var taken struct {
		Field      string `json:"field"`
		Suggestion string `json:"suggestion"`
	}
	a.must(http.StatusConflict, "PATCH", "/users/me", &u, map[string]string{"username": "Grace"}, &taken)
	if taken.Field != "username" || taken.Suggestion == "" || taken.Suggestion == "grace" {
		t.Fatalf("taken = %+v", taken)
	}

	a.must(http.StatusBadRequest, "PATCH", "/users/me", &u, map[string]string{"username": "a b"}, nil)
	a.must(http.StatusConflict, "PATCH", "/users/me", &u, map[string]string{"username": "deleted.reader"}, nil)
	a.must(http.StatusUnauthorized, "PATCH", "/users/me", nil, map[string]string{"username": "x_y_z"}, nil)
}

func TestChangeEmail(t *testing.T) {
	a := newAPI(t)
	u, _ := a.signUp("ada")
	a.signUp("grace")

	a.must(http.StatusForbidden, "PUT", "/users/me/email", &u,
		map[string]string{"email": "new@example.com", "current_password": "wrong"}, nil)
	a.must(http.StatusBadRequest, "PUT", "/users/me/email", &u,
		map[string]string{"email": "not-an-email", "current_password": "hunter22"}, nil)
	a.must(http.StatusConflict, "PUT", "/users/me/email", &u,
		map[string]string{"email": "Grace@Example.com", "current_password": "hunter22"}, nil)

	var got account
	a.must(http.StatusOK, "PUT", "/users/me/email", &u,
		map[string]string{"email": " New@Example.com ", "current_password": "hunter22"}, &got)
	if got.Email != "new@example.com" {
		t.Fatalf("email = %q", got.Email)
	}
}

func TestChangePassword(t *testing.T) {
	a := newAPI(t)
	u, _ := a.signUp("ada")

	a.must(http.StatusForbidden, "PUT", "/users/me/password", &u,
		map[string]string{"current_password": "wrong", "new_password": "longenough"}, nil)
	a.must(http.StatusBadRequest, "PUT", "/users/me/password", &u,
		map[string]string{"current_password": "hunter22", "new_password": "short"}, nil)
	a.must(http.StatusNoContent, "PUT", "/users/me/password", &u,
		map[string]string{"current_password": "hunter22", "new_password": "longenough"}, nil)

	a.must(http.StatusOK, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "longenough"}, nil)
	a.must(http.StatusUnauthorized, "POST", "/auth/login", nil, map[string]string{"username": "ada", "password": "hunter22"}, nil)
}

func TestSocialAccountsCantChangeEmailOrPassword(t *testing.T) {
	a := newAPI(t)
	u := a.user("gwen")
	a.exec(`UPDATE users SET login_type = 'social_google' WHERE uuid = $1`, u.ID)

	a.must(http.StatusForbidden, "PUT", "/users/me/email", &u,
		map[string]string{"email": "gwen2@example.com", "current_password": "anything"}, nil)
	a.must(http.StatusForbidden, "PUT", "/users/me/password", &u,
		map[string]string{"current_password": "anything", "new_password": "longenough"}, nil)
	// Display name and username are for everyone.
	a.must(http.StatusOK, "PATCH", "/users/me", &u, map[string]string{"display_name": "Gwen"}, nil)
}
