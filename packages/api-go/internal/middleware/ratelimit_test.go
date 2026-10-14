package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
	limited := RateLimit(3, ok)

	call := func(forwarded string) int {
		req := httptest.NewRequest(http.MethodGet, "/signup/available?username=mira", nil)
		req.Header.Set("X-Forwarded-For", forwarded)
		rec := httptest.NewRecorder()
		limited(rec, req)
		return rec.Code
	}

	for i := 0; i < 3; i++ {
		if code := call("203.0.113.7"); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i+1, code)
		}
	}
	if code := call("203.0.113.7"); code != http.StatusTooManyRequests {
		t.Errorf("4th request: got %d, want 429", code)
	}
	// Another client has its own allowance.
	if code := call("198.51.100.2"); code != http.StatusOK {
		t.Errorf("other client: got %d, want 200", code)
	}
	// A spoofed first hop doesn't escape: the last hop is what's counted.
	if code := call("10.0.0.1, 203.0.113.7"); code != http.StatusTooManyRequests {
		t.Errorf("spoofed first hop: got %d, want 429", code)
	}
}
