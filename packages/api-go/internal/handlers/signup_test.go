package handlers

import (
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/models"
)

func TestValidateSignup(t *testing.T) {
	cases := []struct {
		name               string
		req                models.SignupRequest
		wantField          string
		wantUser, wantMail string
	}{
		{"valid", models.SignupRequest{Username: "mira.reads", Email: "mira@hey.com", Password: "12345678"}, "", "mira.reads", "mira@hey.com"},
		{"normalised", models.SignupRequest{Username: "  Mira_Reads ", Email: " Mira@Hey.com ", Password: "longenough"}, "", "mira_reads", "mira@hey.com"},
		{"username too short", models.SignupRequest{Username: "mi", Email: "a@b.co", Password: "12345678"}, "username", "", ""},
		{"username too long", models.SignupRequest{Username: "abcdefghijklmnopqrstu", Email: "a@b.co", Password: "12345678"}, "username", "", ""},
		{"username bad character", models.SignupRequest{Username: "mira-reads", Email: "a@b.co", Password: "12345678"}, "username", "", ""},
		{"email without domain", models.SignupRequest{Username: "mira", Email: "mira@", Password: "12345678"}, "email", "", ""},
		{"email without dot", models.SignupRequest{Username: "mira", Email: "mira@hey", Password: "12345678"}, "email", "", ""},
		{"password short", models.SignupRequest{Username: "mira", Email: "mira@hey.com", Password: "1234567"}, "password", "", ""},
		{"reserved name", models.SignupRequest{Username: "deleted.reader", Email: "a@b.co", Password: "12345678"}, "username", "", ""},
		{"reserved name, any case", models.SignupRequest{Username: " Deleted.Reader ", Email: "a@b.co", Password: "12345678"}, "username", "", ""},
		{"password multibyte counts runes", models.SignupRequest{Username: "mira", Email: "mira@hey.com", Password: "ééééééé"}, "password", "", ""},
	}
	for _, c := range cases {
		req := c.req
		field, _ := validateSignup(&req)
		if field != c.wantField {
			t.Errorf("%s: field = %q, want %q", c.name, field, c.wantField)
		}
		if c.wantField == "" && (req.Username != c.wantUser || req.Email != c.wantMail) {
			t.Errorf("%s: normalised to %q / %q, want %q / %q", c.name, req.Username, req.Email, c.wantUser, c.wantMail)
		}
	}
}

func TestUsernameFrom(t *testing.T) {
	cases := map[string]string{
		"Mira Anand":                      "mira.anand",
		"mira-anand":                      "mira.anand",
		"Zoë":                             "zoe",
		"..mira__":                        "mira",
		"Al":                              "alreader",
		"averyveryverylongname.surname":   "averyveryverylongnam",
		"reader":                          "reader",
		"José María de la Cruz Fernández": "jose.maria.de.la.cru",
		"Christopher Montgomery-Smith":    "christopher.montgome",
	}
	for in, want := range cases {
		got := usernameFrom(in)
		if got != want {
			t.Errorf("usernameFrom(%q) = %q, want %q", in, got, want)
		}
		if !usernamePattern.MatchString(got) {
			t.Errorf("usernameFrom(%q) = %q breaks the sign-up rules", in, got)
		}
	}
}

// A clash after the first choice keeps within the rule: a Google name used
// to get its suffix added in full, past 20 (coming-soon R5.3).
func TestWithSuffix(t *testing.T) {
	cases := map[string]string{
		"mira":                 "mira_x7k2qa",
		"christopher.montgome": "christopher.m_x7k2qa",
		"mira.anand.reads":     "mira.anand.re_x7k2qa",
		"abcdefghijk._more":    "abcdefghijk_x7k2qa",
	}
	for in, want := range cases {
		got := withSuffix(in, "X7K2QA")
		if got != want {
			t.Errorf("withSuffix(%q) = %q, want %q", in, got, want)
		}
		if !usernamePattern.MatchString(got) {
			t.Errorf("withSuffix(%q) = %q breaks the sign-up rules", in, got)
		}
	}
}

// The reserved name reads as taken without a database lookup, so it holds
// even where the Deleted reader row doesn't exist (account-deletion R5.4).
func TestUsernameTakenReserved(t *testing.T) {
	for _, name := range []string{"deleted.reader", "Deleted.Reader"} {
		taken, err := usernameTaken(name)
		if err != nil || !taken {
			t.Errorf("usernameTaken(%q) = %v, %v; want true, nil", name, taken, err)
		}
	}
}
