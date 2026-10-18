package gutenberg

import "testing"

func lit(subjects ...string) Book {
	return Book{LoCC: []string{"PR"}, Subjects: subjects}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		book Book
		want Shelf
	}{
		{"Crime and Punishment", parseFile(t, "pg2554.rdf"), Shelf{"Fiction", "Literary Fiction"}},
		{"Frankenstein", parseFile(t, "pg84.rdf"), Shelf{"Horror", "Supernatural Horror"}},
		{"The Time Machine", lit("Time travel -- Fiction", "Dystopias -- Fiction", "Category: Science-Fiction & Fantasy"), Shelf{"Science Fiction", "Time Travel"}},
		{"The War of the Worlds", lit("Mars (Planet) -- Fiction", "Science fiction"), Shelf{"Science Fiction", "Space Opera"}},
		{"Jekyll and Hyde", lit("Multiple personality -- Fiction", "Horror tales", "Psychological fiction"), Shelf{"Horror", "Psychological Horror"}},
		{"Dracula", lit("Vampires -- Fiction", "Horror tales"), Shelf{"Horror", "Supernatural Horror"}},
		{"Sherlock Holmes", lit("Holmes, Sherlock (Fictitious character) -- Fiction", "Private investigators -- England -- Fiction"), Shelf{"Mystery", "Detective"}},
		{"The Thirty-Nine Steps", lit("Spy stories"), Shelf{"Mystery", "Thriller"}},
		{"Alice", lit("Fantasy fiction", "Children's stories"), Shelf{"Young Adult", "YA Fantasy"}},
		{"Anne of Green Gables", lit("Orphans -- Juvenile fiction", "Love stories"), Shelf{"Young Adult", "YA Romance"}},
		{"Le Morte d'Arthur", lit("Arthurian romances"), Shelf{"Fantasy", "Mythic Fantasy"}},
		{"Pride and Prejudice", lit("Love stories", "Courtship -- Fiction", "Category: Romance"), Shelf{"Romance", "Historical Romance"}},
		{"Three Men in a Boat", lit("Humorous stories"), Shelf{"Fiction", "Humor"}},
		{"Moby Dick", lit("Whaling -- Fiction", "Sea stories"), Shelf{"Fiction", "Classics"}},
		{"Franklin's Autobiography", Book{LoCC: []string{"E"}, Subjects: []string{"Statesmen -- Biography"}}, Shelf{"Non-Fiction", "Biography"}},
		{"The Prince", Book{LoCC: []string{"JC"}}, Shelf{}},
		{"Meditations", Book{LoCC: []string{"B"}}, Shelf{"Non-Fiction", "Self-Help"}},
		{"On the Origin of Species", Book{LoCC: []string{"QH"}}, Shelf{"Non-Fiction", "Science"}},
		{"Gibbon", Book{LoCC: []string{"DG"}}, Shelf{"Non-Fiction", "History"}},
		{"The Republic", Book{LoCC: []string{"JC", "PA"}, Subjects: []string{"Utopias", "Political science"}}, Shelf{"Fiction", "Classics"}},
		{"The Catholic Church and History", Book{LoCC: []string{"BX"}}, Shelf{}},
	}
	for _, c := range cases {
		got, ok := Classify(c.book)
		if ok != (c.want != Shelf{}) || got != c.want {
			t.Errorf("%s: Classify = %v, %v; want %v", c.name, got, ok, c.want)
		}
	}
}

func TestDisplayTitle(t *testing.T) {
	cases := map[string]string{
		"Frankenstein; or, the modern prometheus":     "Frankenstein",
		"The Adventures of Tom Sawyer, Complete":      "The Adventures of Tom Sawyer",
		"Moby Dick; Or, The Whale":                    "Moby Dick",
		"Alice's Adventures in Wonderland":            "Alice's Adventures in Wonderland",
		"The Strange Case of Dr. Jekyll and Mr. Hyde": "The Strange Case of Dr. Jekyll and Mr. Hyde",
	}
	for in, want := range cases {
		if got := DisplayTitle(in); got != want {
			t.Errorf("DisplayTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDisplayTitleCleansUp(t *testing.T) {
	cases := map[string]string{
		"The war of the worlds":                       "The War of the Worlds",
		"Three sevens : $b A detective story":         "Three Sevens",
		"Jane Eyre: An Autobiography":                 "Jane Eyre",
		"Le Morte d'Arthur: Volume 1 (of 2)":          "Le Morte d'Arthur",
		"The strange case of Dr. Jekyll and Mr. Hyde": "The Strange Case of Dr. Jekyll and Mr. Hyde",
	}
	for in, want := range cases {
		if got := DisplayTitle(in); got != want {
			t.Errorf("DisplayTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadable(t *testing.T) {
	cases := map[string]bool{
		"Le Morte d'Arthur: Volume 1 (of 2)":        true,
		"The Works of Edgar Allan Poe — Volume 2":   false,
		"The Mysteries of London, v. 4/4":           false,
		"Modern English biography, volume 2 (of 6)": false,
		"The Complete Works of William Shakespeare": false,
		"Pride and Prejudice":                       true,
		"Dracula":                                   true,
	}
	for title, want := range cases {
		if ok, _ := Readable(Book{Title: title}); ok != want {
			t.Errorf("Readable(%q) = %v, want %v", title, ok, want)
		}
	}
	if ok, _ := Readable(Book{Title: "Body-build and its inheritance", Subjects: []string{"Eugenics"}}); ok {
		t.Error("eugenics should be left out")
	}
}
