package handlers

import (
	"testing"
	"time"
)

func TestProgressPct(t *testing.T) {
	cases := []struct{ current, total, want int }{
		{0, 0, 0},     // length not known yet
		{0, -5, 0},    // nonsense length
		{0, 200, 1},   // the page in front of you counts as read
		{99, 200, 50}, // 100/200
		{199, 200, 100},
		{250, 200, 100}, // clamped
		{0, 3, 33},
		{1, 3, 67}, // rounds, doesn't truncate
	}
	for _, c := range cases {
		if got := progressPct(c.current, c.total); got != c.want {
			t.Errorf("progressPct(%d, %d) = %d, want %d", c.current, c.total, got, c.want)
		}
	}
}

func TestAnchorKey(t *testing.T) {
	a := anchorKey(12, "It was the best of times")
	if len(a) != 32 {
		t.Fatalf("key %q has length %d, want 32", a, len(a))
	}

	// Two selections of the same sentence rarely agree on case or spacing.
	if b := anchorKey(12, "  it was   the BEST\nof times "); b != a {
		t.Errorf("normalised selections should share a key: %q vs %q", a, b)
	}
	if b := anchorKey(13, "It was the best of times"); b == a {
		t.Error("the same text on a different page is a different thread")
	}
	if b := anchorKey(12, "It was the worst of times"); b == a {
		t.Error("different text must not collide")
	}

	if got := anchorKey(7, ""); got != "page:7" {
		t.Errorf("page-level key = %q, want page:7", got)
	}
	if got := anchorKey(7, "   "); got != "page:7" {
		t.Errorf("whitespace-only anchor = %q, want page:7", got)
	}
}

func TestEffectiveReadAt(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, raw string
		want      time.Time
	}{
		{"missing", "", now}, // older apps send none
		{"unparseable", "yesterday", now},
		{"past", "2026-10-26T09:30:00Z", time.Date(2026, 10, 26, 9, 30, 0, 0, time.UTC)},
		// What the app sends: JS toISOString, with milliseconds.
		{"past with millis", "2026-10-26T09:30:00.250Z", time.Date(2026, 10, 26, 9, 30, 0, 250e6, time.UTC)},
		{"offset", "2026-10-26T15:00:00+05:30", time.Date(2026, 10, 26, 9, 30, 0, 0, time.UTC)},
		{"future", "2026-10-28T12:00:00Z", now}, // a phone clock set ahead
	}
	for _, c := range cases {
		got := effectiveReadAt(c.raw, now)
		if !got.Equal(c.want) || got.Location() != time.UTC {
			t.Errorf("%s: effectiveReadAt(%q) = %v, want %v in UTC", c.name, c.raw, got, c.want)
		}
	}
}
