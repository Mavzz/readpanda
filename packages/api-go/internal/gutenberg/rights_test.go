package gutenberg

import (
	"strings"
	"testing"
)

func book(rights string, people ...Contributor) Book {
	return Book{ID: 1, Title: "A Book", Type: "Text", Rights: rights, Languages: []string{"en"}, LoCC: []string{"PR"}, Contributors: people}
}

func author(name string, death int) Contributor {
	return Contributor{Name: name, Role: "author", Death: death}
}

func TestCheck(t *testing.T) {
	pd := "Public domain in the USA."
	p := Policy{Year: 2026, Term: 70, Languages: []string{"en"}, LoCC: []string{"P", "Q"}}

	cases := []struct {
		name   string
		book   Book
		ok     bool
		reason string
	}{
		{"Austen", book(pd, author("Austen, Jane", 1817)), true, ""},
		{"died in the last cleared year", book(pd, author("Shaw, Bernard", 1950), author("Mann, Thomas", 1955)), true, ""},
		{"Milne is a year short", book(pd, author("Milne, A. A.", 1956)), false, "until the end of 2026"},
		{"old translation", book(pd, author("Dostoyevsky, Fyodor", 1881), Contributor{Name: "Garnett, Constance", Role: "translator", Death: 1946}), true, ""},
		{"recent translator", book(pd, author("Kafka, Franz", 1924), Contributor{Name: "Muir, Willa", Role: "translator", Death: 1970}), false, "Muir, Willa (translator) died in 1970"},
		{"illustrator counts too", book(pd, author("Milne, A. A.", 1950), Contributor{Name: "Shepard, E. H.", Role: "illustrator", Death: 1976}), false, "(illustrator)"},
		{"no death year", book(pd, author("Austen, Jane", 1817), Contributor{Name: "Smith, John", Role: "editor"}), false, "no death year"},
		{"anonymous", book(pd, author("Anonymous", 0)), false, "no one person"},
		{"copyrighted", book("Copyrighted. Read the copyright notice inside this book for details.", author("Doctorow, Cory", 0)), false, "not public domain in the US"},
		{"no author", book(pd), false, "no credited author"},
		{"Peter Pan", func() Book {
			b := book(pd, author("Barrie, J. M. (James Matthew)", 1937))
			b.Title = "Peter Pan"
			return b
		}(), false, "Great Ormond Street"},
		{"French", func() Book { b := book(pd, author("Hugo, Victor", 1885)); b.Languages = []string{"fr"}; return b }(), false, "language fr"},
		{"science is allowed", func() Book { b := book(pd, author("Darwin, Charles", 1882)); b.LoCC = []string{"QH"}; return b }(), true, ""},
		{"law isn't", func() Book { b := book(pd, author("Blackstone, William", 1780)); b.LoCC = []string{"KD"}; return b }(), false, "class KD"},
		{"audio book", func() Book { b := book(pd, author("Austen, Jane", 1817)); b.Type = "Sound"; return b }(), false, "not a text"},
	}
	for _, c := range cases {
		ok, reason := p.Check(c.book)
		if ok != c.ok || !strings.Contains(reason, c.reason) {
			t.Errorf("%s: Check = %v, %q; want %v, %q", c.name, ok, reason, c.ok, c.reason)
		}
	}
}

func TestCheckRollsForward(t *testing.T) {
	milne := book("Public domain in the USA.", author("Milne, A. A.", 1956))
	if ok, _ := (Policy{Year: 2027, Term: 70}).Check(milne); !ok {
		t.Error("Milne should clear life+70 in 2027")
	}
	doyle := book("Public domain in the USA.", author("Doyle, Arthur Conan", 1930))
	if ok, _ := (Policy{Year: 2026, Term: 100}).Check(doyle); ok {
		t.Error("Doyle shouldn't clear life+100 in 2026")
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]string{
		"Austen, Jane":                     "Jane Austen",
		"Montgomery, L. M. (Lucy Maud)":    "L. M. Montgomery",
		"Tolstoy, Leo, graf":               "Leo Tolstoy",
		"Homer":                            "Homer",
		"Doyle, Arthur Conan":              "Arthur Conan Doyle",
		"Augustine, of Hippo, Saint":       "Augustine of Hippo",
		"Marcus Aurelius, Emperor of Rome": "Marcus Aurelius",
	}
	for in, want := range cases {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", in, got, want)
		}
	}
	b := book("", author("Brontë, Charlotte", 1855), author("Brontë, Emily", 1848), Contributor{Name: "Garnett, Constance", Role: "translator"})
	if got := AuthorLine(b); got != "Charlotte Brontë and Emily Brontë" {
		t.Errorf("AuthorLine = %q", got)
	}
}
