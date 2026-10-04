package metadata

import "testing"

func TestHumanize(t *testing.T) {
	cases := map[string]string{
		"DataScienceAndPredictiveAnalyt": "Data Science and Predictive Analyt",
		"PythonNotesForProfessionals":    "Python Notes for Professionals",
		"QuantumMechanics.pdf":           "Quantum Mechanics",
		"DataMining":                     "Data Mining",
		"PDFReaderGuide":                 "PDF Reader Guide",
		"Python3Basics":                  "Python 3 Basics",
		"the_art_of_war":                 "The Art of War",
		"Art Heist Baby":                 "Art Heist Baby",
		"The Three-Body Problem":         "The Three-Body Problem",
		"TheHobbit":                      "The Hobbit",
	}
	for in, want := range cases {
		if got := Humanize(in); got != want {
			t.Errorf("Humanize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAccepts(t *testing.T) {
	cases := []struct {
		query, candidate string
		want             bool
	}{
		{"Data Science and Predictive Analyt", "Data Science and Predictive Analytics", true},
		{"Data Mining", "Data Mining: Concepts and Techniques", true},
		{"Data Mining", "Mining Data", false},
		{"Dune", "Dune", false}, // too short to trust
		{"Quantum Mechanics", "Quantum Physics", false},
	}
	for _, c := range cases {
		if got := accepts(c.query, c.candidate); got != c.want {
			t.Errorf("accepts(%q, %q) = %v, want %v", c.query, c.candidate, got, c.want)
		}
	}
}

func TestPick(t *testing.T) {
	quantum := []Result{
		{Title: "Quantum Mechanics", Author: "Leonard Susskind", Pages: 384},
		{Title: "Quantum Mechanics", Author: "Eugene Hecht", Pages: 760},
		{Title: "Quantum Mechanics", Author: "Leonard Susskind", Pages: 400},
	}
	cases := []struct {
		name       string
		cands      []Result
		pdfPages   int
		wantAuthor string
		wantOK     bool
	}{
		{"nothing found", nil, 0, "", true},
		{"one author", []Result{{Author: "Ivo D. Dinov", Pages: 851}}, 0, "Ivo D. Dinov", true},
		{"same author twice", []Result{{Author: "Ivo D. Dinov"}, {Author: "ivo d dinov"}}, 0, "Ivo D. Dinov", true},
		{"several authors, length unknown", quantum, 0, "", false},
		{"several authors, the file's length picks one", quantum, 772, "Eugene Hecht", true},
		{"length matches nobody", quantum, 120, "", false},
		{"no author named", []Result{{Title: "X"}}, 0, "", true},
		{"one author, but a different length", quantum[:1], 772, "", false},
		{"one author, length unknown", []Result{{Author: "Eugene Hecht"}}, 772, "Eugene Hecht", true},
	}
	for _, c := range cases {
		r, ok := pick(c.cands, c.pdfPages)
		got := ""
		if r != nil {
			got = r.Author
		}
		if got != c.wantAuthor || ok != c.wantOK {
			t.Errorf("%s: pick = (%q, %v), want (%q, %v)", c.name, got, ok, c.wantAuthor, c.wantOK)
		}
	}
}
