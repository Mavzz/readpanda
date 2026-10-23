// Package moderation keeps words we don't allow out of what readers write:
// comments, room names and descriptions, and usernames (report-block R8).
package moderation

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// ErrBlockedWords is returned when text holds a word on the list.
var ErrBlockedWords = errors.New("includes blocked words")

// Message is what readers are told, everywhere the check refuses something.
const Message = "That includes words we don't allow."

// Look-alikes people swap in to get past a filter ("h4te"). Folded before
// the text is split, so "$" and "@" become letters of the word around them.
var lookAlikes = strings.NewReplacer(
	"0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "@", "a", "$", "s",
)

// The list, split once: single words in a set, and entries of several words
// as runs to look for.
var (
	singles = map[string]bool{}
	runs    [][]string
)

func init() {
	for _, entry := range blockedWords {
		ws := words(entry)
		switch len(ws) {
		case 0:
		case 1:
			singles[ws[0]] = true
		default:
			runs = append(runs, ws)
		}
	}
}

// Check returns ErrBlockedWords if text contains a listed word. It matches
// whole words, ignoring case, accents and look-alike characters, so a
// listed word inside a longer ordinary one ("Scunthorpe") passes (R8.2).
func Check(text string) error {
	ws := words(text)
	for i, w := range ws {
		if singles[w] {
			return ErrBlockedWords
		}
		for _, run := range runs {
			if run[0] == w && i+len(run) <= len(ws) && equal(ws[i:i+len(run)], run) {
				return ErrBlockedWords
			}
		}
	}
	return nil
}

// Scripts where a combining mark is an accent to ignore. In Devanagari,
// Telugu, Tamil and the rest, marks are vowel signs and viramas: part of the
// word, and dropping them would turn different words into one.
var accented = []*unicode.RangeTable{unicode.Latin, unicode.Greek, unicode.Cyrillic}

// words lowercases text, drops accents ("é" → "e"), folds look-alikes, and
// splits it on anything that isn't a letter, a mark or a digit. Zero-width
// joiners go, since Indic text is typed both with and without them.
func words(text string) []string {
	var b strings.Builder
	accentable := false
	for _, r := range norm.NFKD.String(strings.ToLower(text)) {
		switch {
		case r == '\u200c' || r == '\u200d':
			continue
		case unicode.Is(unicode.Mn, r):
			if accentable {
				continue
			}
		case unicode.IsLetter(r):
			accentable = unicode.In(r, accented...)
		}
		b.WriteRune(r)
	}
	folded := lookAlikes.Replace(b.String())
	return strings.FieldsFunc(folded, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r)
	})
}

func equal(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
