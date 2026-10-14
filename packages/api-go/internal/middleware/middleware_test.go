package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
})

func TestCORS(t *testing.T) {
	t.Run("preflight is answered without reaching the handler", func(t *testing.T) {
		w := httptest.NewRecorder()
		CORS(okHandler).ServeHTTP(w, httptest.NewRequest("OPTIONS", "/x", nil))
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
			t.Error("missing Access-Control-Allow-Headers")
		}
	})

	t.Run("other methods pass through with headers set", func(t *testing.T) {
		w := httptest.NewRecorder()
		CORS(okHandler).ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
		if w.Code != http.StatusTeapot {
			t.Errorf("status = %d, want the handler's 418", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Allow-Origin = %q", got)
		}
	})
}

func TestAuthenticate(t *testing.T) {
	const secret = "s3cret"
	good, refresh, _ := utils.GenerateTokens("u", "user", secret, "other")

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong scheme", "Token " + good, http.StatusUnauthorized},
		{"invalid", "Bearer abc", http.StatusUnauthorized},
		{"refresh token", "Bearer " + refresh, http.StatusUnauthorized},
		{"valid", "Bearer " + good, http.StatusTeapot},
	}
	am := NewAuthMiddleware(secret)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if c.header != "" {
				r.Header.Set("Authorization", c.header)
			}
			w := httptest.NewRecorder()
			am.Authenticate(okHandler).ServeHTTP(w, r)
			if w.Code != c.want {
				t.Errorf("status = %d, want %d", w.Code, c.want)
			}
		})
	}
}

func TestStatusRecorderCapsLoggedBody(t *testing.T) {
	big := make([]byte, maxLoggedErrorBody*2)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(big)
		w.Write(big)
	})
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w}
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.status != 500 {
		t.Errorf("status = %d", rec.status)
	}
	if len(rec.body) != maxLoggedErrorBody {
		t.Errorf("captured %d bytes, want %d", len(rec.body), maxLoggedErrorBody)
	}
	if w.Body.Len() != len(big)*2 {
		t.Error("the client must still receive the full body")
	}
}

func TestStatusRecorderIgnoresSuccessBodies(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	rec.Write([]byte("fine"))
	if rec.status != http.StatusOK || len(rec.body) != 0 {
		t.Errorf("status=%d body=%q, want 200 and nothing captured", rec.status, rec.body)
	}
}
