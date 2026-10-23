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
		"kya chutiya hai", // romanized Hindi
		"BHENCHOD",        // in capitals
		"तू चूतिया है",    // Devanagari, vowel signs kept in the word
		"भोसड़ीके",        // with the nukta precomposed…
		"भोसड\u093cीके",   // …or typed as two characters
		"లం\u200cజ",       // a zero-width non-joiner inside the word doesn't hide it
		"sule maga",       // an entry of two words, romanized
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
		"chhoot milegi on this edition", // "discount", romanized
		"Sulla crossed the Rubicon",     // people and places stay off
		"Niki Lauda", "ABO blood group",
		"chhakka on the last ball",        // a six, in cricket
		"रडार पर कुछ नहीं",                // shares letters with a listed word, not the word
		"चमारी गाँव की कहानी",             // a listed word inside a longer one
		"मैं यह किताब पढ़ रहा हूँ",        // an ordinary sentence
		"ఈ పుస్తకం చాలా బాగుంది",          // Telugu: "this book is very good"
		"இந்த புத்தகம் நன்றாக இருக்கிறது", // Tamil: the same
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
