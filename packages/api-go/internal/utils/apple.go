package utils

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Sign in with Apple: verifying the identity token the app gets from Apple.
// https://developer.apple.com/documentation/signinwithapplerestapi

const (
	appleIssuer    = "https://appleid.apple.com"
	appleKeysURL   = "https://appleid.apple.com/auth/keys"
	appleTokenURL  = "https://appleid.apple.com/auth/token"
	appleRevokeURL = "https://appleid.apple.com/auth/revoke"
	// Apple rotates its keys rarely; a day's cache is plenty, and an unknown
	// key id forces a refetch anyway.
	appleKeysTTL = 24 * time.Hour
	// Refetches triggered by unknown key ids are spaced out, so a stream of
	// forged tokens can't turn into a stream of requests to Apple.
	appleMinRefetch = time.Minute
)

// AppleIdentity is what the app may trust once the token checks out.
type AppleIdentity struct {
	// Subject is Apple's stable id for this user in this app.
	Subject string
	// Email as shared by the user — possibly a private relay address. Empty
	// if they didn't share one.
	Email         string
	EmailVerified bool
	PrivateEmail  bool
}

type appleKeySet struct {
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	client    *http.Client
}

var appleKeys = &appleKeySet{client: &http.Client{Timeout: 10 * time.Second}}

func (s *appleKeySet) key(kid string) (*rsa.PublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stale := time.Since(s.fetchedAt) > appleKeysTTL
	if k, ok := s.keys[kid]; ok && !stale {
		return k, nil
	}
	if stale || time.Since(s.fetchedAt) > appleMinRefetch {
		if err := s.fetch(); err != nil {
			return nil, err
		}
	}
	if k, ok := s.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown Apple key id %q", kid)
}

// fetch replaces the cached keys with Apple's current set. Caller holds mu.
func (s *appleKeySet) fetch() error {
	resp, err := s.client.Get(appleKeysURL)
	if err != nil {
		return fmt.Errorf("fetching Apple keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching Apple keys: status %d", resp.StatusCode)
	}
	var body struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding Apple keys: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range body.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("Apple returned no usable keys")
	}
	s.keys = keys
	s.fetchedAt = time.Now()
	return nil
}

func rsaKey(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(new(big.Int).SetBytes(eb).Int64())}, nil
}

// Apple sends these as booleans or as the strings "true"/"false".
func claimBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true"
	}
	return false
}

// VerifyAppleIdentityToken checks an identity token's signature (against
// Apple's published keys), issuer, audience (the app's bundle id) and expiry,
// and returns who it identifies.
func VerifyAppleIdentityToken(token, bundleID string) (*AppleIdentity, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token has no key id")
		}
		return appleKeys.key(kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(bundleID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("token has no subject")
	}
	email, _ := claims["email"].(string)
	return &AppleIdentity{
		Subject:       sub,
		Email:         email,
		EmailVerified: claimBool(claims["email_verified"]),
		PrivateEmail:  claimBool(claims["is_private_email"]),
	}, nil
}

// AppleKey is the Sign in with Apple key from the developer account: the
// team, the key's id and the .p8 file's contents. It signs the short-lived
// client secret Apple's token and revoke endpoints ask for.
type AppleKey struct {
	TeamID     string
	KeyID      string
	PrivateKey string // PEM; literal "\n" sequences (from an env var) are accepted
	BundleID   string
}

// Configured reports whether every part of the key is set.
func (k AppleKey) Configured() bool {
	return k.TeamID != "" && k.KeyID != "" && k.PrivateKey != "" && k.BundleID != ""
}

func (k AppleKey) clientSecret() (string, error) {
	pem := strings.ReplaceAll(k.PrivateKey, `\n`, "\n")
	key, err := jwt.ParseECPrivateKeyFromPEM([]byte(pem))
	if err != nil {
		return "", fmt.Errorf("Apple private key: %w", err)
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": k.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": appleIssuer,
		"sub": k.BundleID,
	})
	tok.Header["kid"] = k.KeyID
	return tok.SignedString(key)
}

func (k AppleKey) post(endpoint string, form url.Values) (*http.Response, error) {
	secret, err := k.clientSecret()
	if err != nil {
		return nil, err
	}
	form.Set("client_id", k.BundleID)
	form.Set("client_secret", secret)
	return appleKeys.client.PostForm(endpoint, form)
}

// ExchangeAppleCode trades the authorization code from a sign-in for Apple's
// refresh token — the token to revoke when the account is deleted.
func ExchangeAppleCode(k AppleKey, code string) (string, error) {
	resp, err := k.post(appleTokenURL, url.Values{"code": {code}, "grant_type": {"authorization_code"}})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("Apple token exchange: %w", err)
	}
	if resp.StatusCode != http.StatusOK || body.RefreshToken == "" {
		return "", fmt.Errorf("Apple token exchange: status %d %s", resp.StatusCode, body.Error)
	}
	return body.RefreshToken, nil
}

// RevokeAppleToken revokes a stored Apple refresh token, ending the app's
// access to the user's Apple ID (App Store 5.1.1(v): required when an account
// that signed in with Apple is deleted).
func RevokeAppleToken(k AppleKey, refreshToken string) error {
	resp, err := k.post(appleRevokeURL, url.Values{"token": {refreshToken}, "token_type_hint": {"refresh_token"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Apple revoke: status %d", resp.StatusCode)
	}
	return nil
}
