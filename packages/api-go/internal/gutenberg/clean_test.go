package gutenberg

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func cleanFile(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Clean(src, "The Title by The Author")
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	return string(out)
}

func TestCleanNewStyle(t *testing.T) {
	out := cleanFile(t, "new-style.html")
	for _, want := range []string{
		"It is a truth universally acknowledged",
		"Mr. Bennet was among the earliest",
		`<img src="images/i_001.jpg"`,
		"<title>The Title by The Author</title>",
		"page-break-before: always", // the book's own styles stay
		"@page",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, gone := range []string{"Gutenberg", "gutenberg", "Produced by", "Updated editions", "<script", "alert(1)", "DC.Publisher"} {
		if strings.Contains(out, gone) {
			t.Errorf("still has %q", gone)
		}
	}
}

func TestCleanOldStyleLatin1(t *testing.T) {
	out := cleanFile(t, "old-style.html")
	if !strings.Contains(out, "There was no possibility of taking a walk that day. Café Brontë") {
		t.Errorf("book text missing or mis-decoded:\n%s", out)
	}
	if !strings.Contains(out, "JANE EYRE") {
		t.Error("heading missing")
	}
	for _, gone := range []string{"utenberg", "Transcribed from", "This eBook is for the use", "should be named", "iso-8859-1"} {
		if strings.Contains(out, gone) {
			t.Errorf("still has %q", gone)
		}
	}
}

func TestCleanWholeBookInOnePre(t *testing.T) {
	body := strings.Repeat("Call me Ishmael. Some years ago, never mind how long precisely.\n", 100)
	src := "<html><head><title>x</title></head><body><pre>The Project Gutenberg EBook of Moby Dick\n\n" +
		"*** START OF THIS PROJECT GUTENBERG EBOOK MOBY DICK ***\n\n" + body +
		"\n*** END OF THIS PROJECT GUTENBERG EBOOK MOBY DICK ***\n\nLicence text.</pre></body></html>"
	out, err := Clean([]byte(src), "Moby Dick")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(out), "Call me Ishmael") != 100 {
		t.Error("book text lost")
	}
	if strings.Contains(string(out), "Licence text") || strings.Contains(string(out), "utenberg") {
		t.Errorf("boilerplate left:\n%s", out)
	}
}

func TestCleanWrapperDiv(t *testing.T) {
	// A marker straight inside a wrapper that holds the whole book: only the
	// marker may go, not the wrapper.
	chapter := strings.Repeat("<p>It was the best of times, it was the worst of times.</p>", 80)
	src := `<html><head></head><body><div class="book">*** START OF THE PROJECT GUTENBERG EBOOK X ***` +
		chapter + `*** END OF THE PROJECT GUTENBERG EBOOK X ***</div></body></html>`
	out, err := Clean([]byte(src), "X")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(out), "best of times") != 80 {
		t.Errorf("book text lost:\n%s", out)
	}
}

func TestCleanNeedsReview(t *testing.T) {
	cases := map[string]string{
		"no markers": `<html><body><p>Just a book.</p></body></html>`,
		"long mention left": `<html><body><p>*** START OF THE PROJECT GUTENBERG EBOOK X ***</p><p>` +
			strings.Repeat("A long paragraph of the book itself. ", 30) + `Thanks to Project Gutenberg.</p>` +
			`<p>*** END OF THE PROJECT GUTENBERG EBOOK X ***</p></body></html>`,
	}
	for name, src := range cases {
		if _, err := Clean([]byte(src), "X"); !errors.Is(err, ErrNeedsReview) {
			t.Errorf("%s: err = %v, want ErrNeedsReview", name, err)
		}
	}
}
