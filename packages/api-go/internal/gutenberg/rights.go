package gutenberg

import (
	"fmt"
	"regexp"
	"strings"
)

// Policy is the copyright rule a book has to pass to be hosted everywhere.
//
// Gutenberg only clears books for the US, which is the catalogue's rights
// field. Most other countries protect a work until a number of years after
// its author's death (Term): 70 in the EU, UK and Australia, 60 in India,
// 100 in Mexico. A translation, illustrations or an introduction are
// separate works with their own authors, so every credited person has to be
// past the term, not only the author.
type Policy struct {
	// Year is the current year; terms run to the end of a calendar year.
	Year int
	// Term is the life-plus term to clear: 70 covers almost every country,
	// 100 covers all of them.
	Term int
	// Languages the books have to be in, e.g. ["en"]. Empty allows any.
	Languages []string
	// LoCC is a Library of Congress class prefix the book must have, e.g. "P"
	// (language and literature). Empty allows any.
	LoCC string
}

// LatestDeath is the latest death year still past the term this year: with
// life+70 in 2026, someone who died in 1955 (public domain since 1 January
// 2026) but not 1956.
func (p Policy) LatestDeath() int {
	return p.Year - p.Term - 1
}

// Rule describes the policy in a sentence, for the provenance record.
func (p Policy) Rule() string {
	return fmt.Sprintf("Public domain in the USA per the Gutenberg catalogue; every credited contributor died in or before %d (life+%d, checked %d)",
		p.LatestDeath(), p.Term, p.Year)
}

// Names Gutenberg uses when there is no one person to date.
var undatable = map[string]bool{
	"anonymous": true,
	"unknown":   true,
	"various":   true,
}

// Books that pass the dates but carry a right of their own somewhere.
var cautions = []struct {
	title  *regexp.Regexp
	reason string
}{
	{regexp.MustCompile(`(?i)\bpeter pan\b`), "the UK gives Great Ormond Street Hospital a perpetual right to royalties on Peter Pan"},
}

// Check says whether b can be hosted, and if not, why. The reasons are
// written for a person reading the report.
func (p Policy) Check(b Book) (ok bool, reason string) {
	if ok, reason := p.InScope(b); !ok {
		return false, reason
	}
	return p.Clear(b)
}

// InScope is the part of Check that isn't about rights: the kind of book,
// its language and class.
func (p Policy) InScope(b Book) (ok bool, reason string) {
	if b.Type != "Text" {
		return false, fmt.Sprintf("not a text (%s)", orNone(b.Type))
	}
	if len(p.Languages) > 0 && !overlaps(b.Languages, p.Languages) {
		return false, fmt.Sprintf("language %s", orNone(strings.Join(b.Languages, ", ")))
	}
	if p.LoCC != "" && !hasPrefix(b.LoCC, p.LoCC) {
		return false, fmt.Sprintf("class %s, not %s", orNone(strings.Join(b.LoCC, ", ")), p.LoCC)
	}
	return true, ""
}

// Clear is the copyright part of Check: public domain in the US, and every
// credited person dead long enough for the term everywhere else.
func (p Policy) Clear(b Book) (ok bool, reason string) {
	if !strings.HasPrefix(b.Rights, "Public domain in the USA") {
		return false, fmt.Sprintf("not public domain in the US: %q", b.Rights)
	}
	if len(b.Authors()) == 0 {
		return false, "no credited author"
	}
	latest := p.LatestDeath()
	for _, c := range b.Contributors {
		who := fmt.Sprintf("%s (%s)", orNone(c.Name), c.Role)
		switch {
		case undatable[strings.ToLower(c.Name)]:
			return false, who + ": no one person to date the work by"
		case c.Death == 0:
			return false, who + ": no death year in the catalogue"
		case c.Death > latest:
			return false, fmt.Sprintf("%s died in %d: protected in life+%d countries until the end of %d",
				who, c.Death, p.Term, c.Death+p.Term)
		}
	}
	for _, c := range cautions {
		if c.title.MatchString(b.Title) {
			return false, c.reason
		}
	}
	return true, ""
}

var parenthetical = regexp.MustCompile(`\s*\([^)]*\)`)

// DisplayName turns the catalogue's "Surname, Given" into "Given Surname":
// "Austen, Jane" → "Jane Austen", "Montgomery, L. M. (Lucy Maud)" →
// "L. M. Montgomery", "Tolstoy, Leo, graf" → "Leo Tolstoy".
func DisplayName(name string) string {
	name = parenthetical.ReplaceAllString(name, "")
	parts := strings.Split(name, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < 2 || parts[1] == "" {
		return strings.TrimSpace(name)
	}
	return parts[1] + " " + parts[0]
}

// AuthorLine is the authors' names for the books row: "Jane Austen",
// "Charlotte Brontë and Emily Brontë".
func AuthorLine(b Book) string {
	var names []string
	for _, a := range b.Authors() {
		names = append(names, DisplayName(a.Name))
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}

func overlaps(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}

func hasPrefix(values []string, prefix string) bool {
	for _, v := range values {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
