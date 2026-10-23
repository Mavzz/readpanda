package moderation

import "testing"

func TestCheck(t *testing.T) {
	refused := []string{
		"you absolute cunt",
		"CUNT",               // case
		"c.u.n.t? no, cunt.", // punctuation around the word
		"n1gger",             // look-alikes
		"f@g",                // a symbol inside the word
		"spïc",               // accents
		"what a blow job",    // an entry of two words
		"blow   job",         // however they're spaced
		"some_kike_name",     // usernames split on _ and .
		"mira.faggot",
	}
	for _, text := range refused {
		if Check(text) == nil {
			t.Errorf("Check(%q) passed, want it refused", text)
		}
	}

	allowed := []string{
		"",
		"Scunthorpe United won again", // R8.2: inside a longer word
		"Moby Dick, chapter 42",
		"a spicy blowfish job",
		"blow the job up", // the two words, but not in a row
		"page 15 of 400",  // digits on their own aren't words on the list
		"mira.anand",
		"Retardant on the forest fire",
	}
	for _, text := range allowed {
		if err := Check(text); err != nil {
			t.Errorf("Check(%q) = %v, want it allowed", text, err)
		}
	}
}

// Every entry on the list is caught as written, so a typo there shows up.
func TestEveryEntryIsCaught(t *testing.T) {
	for _, entry := range blockedWords {
		if Check(entry) == nil {
			t.Errorf("the list's own entry %q isn't caught", entry)
		}
	}
}
