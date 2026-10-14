package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyAppleIdentityToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	// Stand in for Apple's key set, fresh so nothing is fetched.
	appleKeys.mu.Lock()
	appleKeys.keys = map[string]*rsa.PublicKey{"test-kid": &priv.PublicKey}
	appleKeys.fetchedAt = time.Now()
	appleKeys.mu.Unlock()

	const bundle = "com.readpanda.app"
	sign := func(claims jwt.MapClaims, kid string, method jwt.SigningMethod) string {
		tok := jwt.NewWithClaims(method, claims)
		tok.Header["kid"] = kid
		var key interface{} = priv
		if method == jwt.SigningMethodHS256 {
			key = []byte("not-apple")
		}
		s, err := tok.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	base := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss":              appleIssuer,
			"aud":              bundle,
			"sub":              "001234.abc",
			"email":            "x7@privaterelay.appleid.com",
			"email_verified":   "true",
			"is_private_email": true,
			"exp":              time.Now().Add(time.Hour).Unix(),
		}
	}

	id, err := VerifyAppleIdentityToken(sign(base(), "test-kid", jwt.SigningMethodRS256), bundle)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if id.Subject != "001234.abc" || !id.EmailVerified || !id.PrivateEmail || id.Email == "" {
		t.Errorf("identity = %+v", id)
	}

	bad := map[string]string{}
	wrongAud := base()
	wrongAud["aud"] = "com.someone.else"
	bad["wrong audience"] = sign(wrongAud, "test-kid", jwt.SigningMethodRS256)
	wrongIss := base()
	wrongIss["iss"] = "https://evil.example"
	bad["wrong issuer"] = sign(wrongIss, "test-kid", jwt.SigningMethodRS256)
	expired := base()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	bad["expired"] = sign(expired, "test-kid", jwt.SigningMethodRS256)
	bad["unknown key"] = sign(base(), "other-kid", jwt.SigningMethodRS256)
	bad["hmac"] = sign(base(), "test-kid", jwt.SigningMethodHS256)
	for name, token := range bad {
		if _, err := VerifyAppleIdentityToken(token, bundle); err == nil {
			t.Errorf("%s: accepted, want rejected", name)
		}
	}
}

func TestAppleClientSecret(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pemText := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	// As it arrives from an env var: newlines written as "\n".
	key := AppleKey{TeamID: "6F6H5A6JCC", KeyID: "KEY123", BundleID: "com.readpanda.app",
		PrivateKey: strings.ReplaceAll(pemText, "\n", `\n`)}
	if !key.Configured() {
		t.Fatal("key reports unconfigured")
	}
	secret, err := key.clientSecret()
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(secret, claims, func(*jwt.Token) (interface{}, error) { return &priv.PublicKey, nil },
		jwt.WithValidMethods([]string{"ES256"}))
	if err != nil {
		t.Fatalf("secret doesn't verify: %v", err)
	}
	if tok.Header["kid"] != "KEY123" || claims["iss"] != "6F6H5A6JCC" || claims["sub"] != "com.readpanda.app" || claims["aud"] != appleIssuer {
		t.Errorf("header %v claims %v", tok.Header, claims)
	}
}
