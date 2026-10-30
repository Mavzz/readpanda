package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

type searchResult struct {
	Books []struct {
		ID     string  `json:"id"`
		Title  string  `json:"title"`
		Author *string `json:"author_name"`
	} `json:"books"`
}

func (a *api) search(as user, q string) []string {
	a.t.Helper()
	var out searchResult
	a.must(http.StatusOK, "GET", "/books/search?q="+url.QueryEscape(q), &as, nil, &out)
	titles := make([]string, len(out.Books))
	for i, b := range out.Books {
		titles[i] = b.Title
	}
	return titles
}

func (a *api) authoredBook(title, author string) {
	a.t.Helper()
	id := a.book(title)
	a.exec(`UPDATE books SET author_name = $2 WHERE book_id = $1`, id, author)
}

// coming-soon R1.6
func TestSearchBooksRanksTitlePrefixThenTitleThenAuthor(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	a.authoredBook("A Whale's Tale", "Nobody")
	a.authoredBook("Whale Songs", "Someone")
	a.authoredBook("Ocean Life", "Whaley Smith")
	a.authoredBook("Dune", "Frank Herbert")

	got := a.search(u, "whale")
	want := []string{"Whale Songs", "A Whale's Tale", "Ocean Life"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("search(whale) = %v, want %v", got, want)
	}
}

func TestSearchBooksIsCaseInsensitiveAndTrimmed(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	a.authoredBook("Moby Dick", "Herman Melville")

	if got := a.search(u, "  MELVILLE "); len(got) != 1 || got[0] != "Moby Dick" {
		t.Fatalf("search by author = %v, want [Moby Dick]", got)
	}
}

func TestSearchBooksMatchesWildcardsLiterally(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	a.book("100% Reading")
	a.book("1000 Readings")
	a.book("snake_case")
	a.book("snakeXcase")

	if got := a.search(u, "0%"); len(got) != 1 || got[0] != "100% Reading" {
		t.Fatalf("search(0%%) = %v, want [100%% Reading]", got)
	}
	if got := a.search(u, "e_c"); len(got) != 1 || got[0] != "snake_case" {
		t.Fatalf("search(e_c) = %v, want [snake_case]", got)
	}
}

func TestSearchBooksReturnsAtMost50(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	for i := 0; i < 55; i++ {
		a.book(fmt.Sprintf("Sea Story %02d", i))
	}
	if got := a.search(u, "sea story"); len(got) != 50 {
		t.Fatalf("got %d results, want 50", len(got))
	}
}

func TestSearchBooksRejects(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	a.must(http.StatusBadRequest, "GET", "/books/search?q=a", &u, nil, nil)
	a.must(http.StatusBadRequest, "GET", "/books/search?q=%20%20", &u, nil, nil)
	a.must(http.StatusUnauthorized, "GET", "/books/search?q=whale", nil, nil, nil)
}
