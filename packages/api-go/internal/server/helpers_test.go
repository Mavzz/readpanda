package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/server"
	"github.com/Mavzz/readpanda/api-go/internal/testdb"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var testConfig = &config.Config{
	APIVersion:       "/api/v1",
	JWTSecret:        "test-access-secret",
	JWTRefreshSecret: "test-refresh-secret",
}

// api is one test's view of the server: a fresh, empty database behind the
// production router.
type api struct {
	t      *testing.T
	router http.Handler
}

func newAPI(t *testing.T) *api {
	t.Helper()
	testdb.Require(t)
	return &api{t: t, router: server.NewRouter(testConfig)}
}

type user struct {
	ID    string
	Name  string
	Token string
}

// user inserts a user straight into the database and returns an access token
// for them, skipping signup for tests that aren't about signup.
func (a *api) user(name string) user {
	a.t.Helper()
	return a.userWithRole(name, "user")
}

func (a *api) userWithRole(name, role string) user {
	a.t.Helper()
	id := uuid.New().String()
	a.exec(`INSERT INTO users (uuid, username, email, login_type, role) VALUES ($1, $2, $3, 'email', $4)`,
		id, name, name+"@example.com", role)
	tok, _, err := utils.GenerateTokens(id, role, testConfig.JWTSecret, testConfig.JWTRefreshSecret)
	if err != nil {
		a.t.Fatal(err)
	}
	return user{ID: id, Name: name, Token: tok}
}

// book inserts a published book and returns its id.
func (a *api) book(title string) string {
	a.t.Helper()
	id := "bk_" + uuid.New().String()[:8]
	a.exec(`INSERT INTO books (book_id, title, status) VALUES ($1, $2, 1)`, id, title)
	return id
}

func (a *api) exec(query string, args ...any) {
	a.t.Helper()
	if _, err := database.DB.Exec(query, args...); err != nil {
		a.t.Fatalf("exec %q: %v", query, err)
	}
}

// do sends a request through the router. body is JSON-encoded unless nil;
// as is the caller (nil for an anonymous request). The response body is
// decoded into out when out is non-nil and the call succeeded.
func (a *api) do(method, path string, as *user, body any, out any) int {
	a.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			a.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, testConfig.APIVersion+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if as != nil {
		req.Header.Set("Authorization", "Bearer "+as.Token)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)

	if out != nil && w.Code < 300 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			a.t.Fatalf("%s %s: decoding %q: %v", method, path, w.Body.String(), err)
		}
	}
	return w.Code
}

// must is do, failing the test unless the status is the one expected.
func (a *api) must(want int, method, path string, as *user, body any, out any) {
	a.t.Helper()
	if got := a.do(method, path, as, body, out); got != want {
		a.t.Fatalf("%s %s = %d, want %d", method, path, got, want)
	}
}
