package gutenberg

import (
	"regexp"
	"strings"
	"unicode"
)

// Shelf is where a book goes in the library: a genre and subgenre from the
// preferences table.
type Shelf struct {
	Genre    string
	Subgenre string
}

// A rule matches a book's subjects and Gutenberg categories, lower-cased and
// joined with "|", so a pattern can match one subject ("time travel") or a
// category ("category: romance"). Every pattern in all has to match.
type rule struct {
	shelf Shelf
	all   []*regexp.Regexp
}

func re(s string) *regexp.Regexp { return regexp.MustCompile(s) }

var (
	juvenile = re(`juvenile fiction|children's stories|category: children`)
	horror   = re(`horror tales|ghost stories|vampires|werewolves|haunted|supernatural|monsters|occultism`)
	fantasy  = re(`fantasy fiction|fairy tales|magic|imaginary places|category: science-fiction & fantasy`)
	romance  = re(`love stories|courtship|man-woman relationships|category: romance`)
)

// fictionRules are tried in order and the first match wins, so the specific
// signals (time travel, vampires, detectives) come before the broad ones
// (crime, love stories). Literature that matches none is a Classic.
var fictionRules = []rule{
	{Shelf{"Science Fiction", "Time Travel"}, []*regexp.Regexp{re(`time travel`)}},
	{Shelf{"Science Fiction", "Dystopian"}, []*regexp.Regexp{re(`dystopias|utopias -- fiction`)}},
	{Shelf{"Science Fiction", "Space Opera"}, []*regexp.Regexp{re(`interplanetary voyages|outer space|life on other planets|mars \(planet\)|space flight`)}},
	{Shelf{"Horror", "Psychological Horror"}, []*regexp.Regexp{re(`multiple personality`)}},
	{Shelf{"Horror", "Psychological Horror"}, []*regexp.Regexp{horror, re(`psycholog|insanity|madness`)}},
	{Shelf{"Horror", "Supernatural Horror"}, []*regexp.Regexp{horror}},
	// Science fiction with no space in it (Verne's submarines, Wells's
	// invisible man) still belongs with the rest of the genre.
	{Shelf{"Science Fiction", "Space Opera"}, []*regexp.Regexp{re(`science fiction`)}},
	{Shelf{"Fiction", "Literary Fiction"}, []*regexp.Regexp{re(`psychological fiction|psychological aspects|bildungsromans|stream of consciousness`)}},
	{Shelf{"Mystery", "Police Procedural"}, []*regexp.Regexp{re(`police --|scotland yard`)}},
	{Shelf{"Mystery", "Detective"}, []*regexp.Regexp{re(`detective and mystery|private investigators|detectives|holmes, sherlock|brown, father`)}},
	{Shelf{"Mystery", "Thriller"}, []*regexp.Regexp{re(`spy stories|espionage|category: crime, thrillers|murder|kidnapping|revenge`)}},
	{Shelf{"Young Adult", "YA Fantasy"}, []*regexp.Regexp{juvenile, fantasy}},
	{Shelf{"Fantasy", "Mythic Fantasy"}, []*regexp.Regexp{re(`mythology|legends|arthurian|knights and knighthood|folklore|epic poetry`)}},
	{Shelf{"Fantasy", "Epic Fantasy"}, []*regexp.Regexp{fantasy}},
	{Shelf{"Young Adult", "YA Romance"}, []*regexp.Regexp{juvenile, romance}},
	{Shelf{"Romance", "Historical Romance"}, []*regexp.Regexp{romance}},
	{Shelf{"Fiction", "Humor"}, []*regexp.Regexp{re(`humorous|humor|satire|category: humour`)}},
}

// Classes outside literature, by Library of Congress class prefix.
var nonFiction = []struct {
	class string
	shelf Shelf
}{
	{"CT", Shelf{"Non-Fiction", "Biography"}},
	{"BJ", Shelf{"Non-Fiction", "Self-Help"}},
	{"Q", Shelf{"Non-Fiction", "Science"}},
	{"D", Shelf{"Non-Fiction", "History"}},
	{"E", Shelf{"Non-Fiction", "History"}},
	{"F", Shelf{"Non-Fiction", "History"}},
	{"B", Shelf{"Non-Fiction", "Self-Help"}},
}

var biography = re(`biography|autobiograph|diaries|memoirs|letters`)

var religion = []string{"BL", "BM", "BP", "BQ", "BR", "BS", "BT", "BV", "BX"}

// Classify puts b on a shelf. ok is false for a non-literature book none of
// the non-fiction shelves suit; it's left out rather than misfiled.
func Classify(b Book) (s Shelf, ok bool) {
	tags := strings.ToLower(strings.Join(append(append([]string{}, b.Subjects...), b.Bookshelves...), "|"))

	if !hasAnyPrefix(b.LoCC, []string{"P"}) {
		if biography.MatchString(tags) {
			return Shelf{"Non-Fiction", "Biography"}, true
		}
		// Religion (BL–BX) has no shelf in the library, and filed under
		// Self-Help or History it would mislead.
		if hasAnyPrefix(b.LoCC, religion) {
			return Shelf{}, false
		}
		for _, n := range nonFiction {
			if hasAnyPrefix(b.LoCC, []string{n.class}) {
				return n.shelf, true
			}
		}
		return Shelf{}, false
	}

	for _, r := range fictionRules {
		if matchesAll(tags, r.all) {
			return r.shelf, true
		}
	}
	return Shelf{"Fiction", "Classics"}, true
}

func matchesAll(tags string, all []*regexp.Regexp) bool {
	for _, p := range all {
		if !p.MatchString(tags) {
			return false
		}
	}
	return true
}

var (
	// MARC subfield codes that leak into some titles: "Three sevens : $b A
	// detective story".
	marcCode = regexp.MustCompile(`\s*\$[a-z]\s*`)
	// "Volume 2", "Vol. II", "v. 4/4", "(of 6)".
	volume  = regexp.MustCompile(`(?i)\b(?:volume|vol\.?|v\.)\s*(\d+|[ivxlc]+)\b`)
	omnibus = regexp.MustCompile(`(?i)\b(complete|collected|selected) (works|writings|poems)\b|\bworks of\b|^(the )?works\b|\banthology\b`)
	// Left out of a library meant to look inviting, whatever its rights.
	unwanted = regexp.MustCompile(`(?i)eugenics|racial|race (?:problems|question)`)
)

// Readable says whether b is a book a reader would pick up on its own: not a
// later volume of a set, not a works-of omnibus (thousands of pages, and the
// pieces are usually there as books of their own), not a subject we'd rather
// not put on the shelves. Books a rule can't catch go in -exclude.
func Readable(b Book) (ok bool, reason string) {
	if m := volume.FindStringSubmatch(b.Title); m != nil && !strings.EqualFold(m[1], "1") && !strings.EqualFold(m[1], "i") {
		return false, "a later volume of a set"
	}
	if omnibus.MatchString(b.Title) {
		return false, "an omnibus of works"
	}
	if unwanted.MatchString(strings.Join(b.Subjects, "|")) {
		return false, "subject left out of the library"
	}
	return true, ""
}

// Small words stay lower-case inside a title.
var smallWords = map[string]bool{
	"a": true, "an": true, "and": true, "as": true, "at": true, "but": true, "by": true,
	"for": true, "from": true, "in": true, "into": true, "nor": true, "of": true, "on": true,
	"or": true, "the": true, "to": true, "with": true,
}

// DisplayTitle is the title readers see: the main title without the
// alternative after a semicolon, the subtitle after a colon or edition notes
// Gutenberg appends, in title case (some records are in sentence case).
// "Frankenstein; or, the modern prometheus" → "Frankenstein",
// "The war of the worlds" → "The War of the Worlds".
func DisplayTitle(title string) string {
	title = marcCode.ReplaceAllString(title, " ")
	if i := strings.IndexAny(title, ";:"); i > 0 {
		title = title[:i]
	}
	title = strings.TrimSpace(title)
	for _, suffix := range []string{", Complete", " — Complete", " - Complete"} {
		title = strings.TrimSuffix(title, suffix)
	}
	words := strings.Fields(title)
	for i, w := range words {
		r := []rune(w)
		switch {
		case i > 0 && smallWords[strings.ToLower(w)]:
		case len(r) > 2 && r[1] == '\'':
			// d'Arthur, l'Homme: the particle stays as written.
		case unicode.IsLower(r[0]):
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

// Description is the catalogue's summary without its machine-made note.
func Description(b Book) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(b.Summary), "(This is an automatically generated summary.)"))
}
