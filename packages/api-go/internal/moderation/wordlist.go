package moderation

// blockedWords is the whole list (report-block R8.5): add or remove a word
// here and nothing else changes. Entries match as whole words, ignoring
// case, accents and look-alike characters; an entry of several words matches
// those words in a row. Plurals are separate entries.
//
// Slurs and explicit sexual terms only. Words that are also ordinary words
// stay off it ("dick" is in Moby Dick, "cock" is a rooster, "pussy" a cat):
// this is the first line, and reports catch the rest.
var blockedWords = []string{
	// Racial and ethnic slurs
	"nigger", "niggers", "nigga", "niggas",
	"chink", "chinks",
	"gook", "gooks",
	"spic", "spics",
	"wetback", "wetbacks",
	"kike", "kikes",
	"raghead", "ragheads",
	"towelhead", "towelheads",
	"paki", "pakis",

	// Slurs about sexuality, gender and disability
	"faggot", "faggots", "fag", "fags",
	"tranny", "trannies",
	"retard", "retards",

	// Explicit sexual terms
	"cunt", "cunts",
	"blowjob", "blowjobs", "blow job", "blow jobs",
	"handjob", "handjobs", "hand job", "hand jobs",
	"cumshot", "cumshots",
	"creampie", "gangbang", "bukkake",
	"jizz",
	"dildo", "dildos",
	"milf",
}
