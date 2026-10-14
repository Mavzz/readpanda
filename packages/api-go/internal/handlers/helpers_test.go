package handlers

import "testing"

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
