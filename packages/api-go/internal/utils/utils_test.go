package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret        = "test-access-secret"
	testRefreshSecret = "test-refresh-secret"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := CryptPassword("correct horse")
	if err != nil {
		t.Fatalf("CryptPassword: %v", err)
	}
	if hash == "correct horse" {
		t.Fatal("hash must not be the plaintext password")
	}
	if !DecryptPassword("correct horse", hash) {
		t.Error("the right password should verify")
	}
	if DecryptPassword("wrong horse", hash) {
		t.Error("a wrong password must not verify")
	}
}

func TestAESRoundTrip(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef" // AES-256
	enc, err := EncryptAES("hello", key)
	if err != nil {
		t.Fatalf("EncryptAES: %v", err)
	}
	got, err := DecryptAES(enc, key)
	if err != nil {
		t.Fatalf("DecryptAES: %v", err)
	}
	if got != "hello" {
		t.Errorf("DecryptAES = %q, want %q", got, "hello")
	}
	if _, err := DecryptAES(enc, "fedcba9876543210fedcba9876543210"); err == nil {
		t.Error("decrypting with the wrong key should fail")
	}
	if _, err := DecryptAES("AAAA", key); err == nil {
		t.Error("a ciphertext shorter than the nonce should fail")
	}
}

func TestGenerateTokens(t *testing.T) {
	access, refresh, err := GenerateTokens("user-1", "admin", testSecret, testRefreshSecret)
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}

	claims, err := VerifyToken(access, testSecret)
	if err != nil {
		t.Fatalf("access token should verify with the access secret: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != "admin" || claims.Type != "" {
		t.Errorf("access claims = %+v", claims)
	}
	if ttl := time.Until(claims.ExpiresAt.Time); ttl < 59*time.Minute || ttl > time.Hour {
		t.Errorf("access token TTL = %s, want ~1h", ttl)
	}

	rc, err := VerifyToken(refresh, testRefreshSecret)
	if err != nil {
		t.Fatalf("refresh token should verify with the refresh secret: %v", err)
	}
	if rc.Type != "refresh" || rc.Role != "" {
		t.Errorf("refresh claims = %+v, want type=refresh and no role", rc)
	}

	// The two secrets must not be interchangeable: a refresh token is never
	// a valid access token, and vice versa.
	if CheckToken(refresh, testSecret) {
		t.Error("refresh token must not pass as an access token")
	}
	if CheckToken(access, testRefreshSecret) {
		t.Error("access token must not pass as a refresh token")
	}
}

func TestVerifyTokenRejects(t *testing.T) {
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		UserID:           "u",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	})
	expiredStr, _ := expired.SignedString([]byte(testSecret))

	// alg=none must never be accepted, whatever the claims say.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, JWTClaims{UserID: "u"})
	noneStr, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)

	cases := map[string]string{
		"expired":  expiredStr,
		"alg none": noneStr,
		"garbage":  "not.a.jwt",
		"empty":    "",
		"tampered": expiredStr[:len(expiredStr)-2] + "xx",
	}
	for name, tok := range cases {
		if _, err := VerifyToken(tok, testSecret); err == nil {
			t.Errorf("%s: VerifyToken accepted it", name)
		}
	}
}

func TestExtractUserIDAndRequireAdmin(t *testing.T) {
	userTok, _, _ := GenerateTokens("u-1", "user", testSecret, testRefreshSecret)
	adminTok, _, _ := GenerateTokens("a-1", "admin", testSecret, testRefreshSecret)

	cases := []struct {
		name        string
		header      string
		wantUser    int // status from ExtractUserID, 0 = ok
		wantAdmin   int // status from RequireAdmin, 0 = ok
		wantSubject string
	}{
		{"no header", "", 401, 401, ""},
		{"not bearer", "Basic " + userTok, 401, 401, ""},
		{"extra parts", "Bearer " + userTok + " x", 401, 401, ""},
		{"bad token", "Bearer nope", 401, 401, ""},
		{"user", "Bearer " + userTok, 0, 403, "u-1"},
		{"admin", "Bearer " + adminTok, 0, 0, "a-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, fn := range []struct {
				label string
				call  func(http.ResponseWriter, *http.Request, string) (string, bool)
				want  int
			}{
				{"ExtractUserID", ExtractUserID, c.wantUser},
				{"RequireAdmin", RequireAdmin, c.wantAdmin},
			} {
				r := httptest.NewRequest("GET", "/", nil)
				if c.header != "" {
					r.Header.Set("Authorization", c.header)
				}
				w := httptest.NewRecorder()
				id, ok := fn.call(w, r, testSecret)
				if fn.want == 0 {
					if !ok || id != c.wantSubject {
						t.Errorf("%s = (%q, %v), want (%q, true)", fn.label, id, ok, c.wantSubject)
					}
					continue
				}
				if ok {
					t.Errorf("%s accepted the request", fn.label)
				}
				if w.Code != fn.want {
					t.Errorf("%s status = %d, want %d", fn.label, w.Code, fn.want)
				}
			}
		})
	}
}

func TestGenerateInviteCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			t.Fatalf("GenerateInviteCode: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("code %q has length %d, want 6", code, len(code))
		}
		// No look-alike characters: 0/O and 1/I are left out of the charset.
		if strings.ContainsAny(code, "01OI") {
			t.Errorf("code %q contains an ambiguous character", code)
		}
		seen[code] = true
	}
	if len(seen) < 190 {
		t.Errorf("only %d distinct codes in 200 draws", len(seen))
	}
}

func TestTokensAreUniqueWithinTheSameSecond(t *testing.T) {
	a1, r1, _ := GenerateTokens("u", "user", testSecret, testRefreshSecret)
	a2, r2, _ := GenerateTokens("u", "user", testSecret, testRefreshSecret)
	if a1 == a2 || r1 == r2 {
		t.Error("back-to-back tokens for the same user must differ, or rotation is a no-op")
	}
}
