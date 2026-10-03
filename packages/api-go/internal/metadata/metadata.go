// Package metadata turns the catalogue's file-name titles into real book
// records. Seeded books are named after their files ("DataScienceAndPredictiveAnalyt",
// truncated by the storage path); the bucket screens (9a/9b) show title,
// author and length, so each book is looked up once in Open Library, then
// Google Books, and the result is cached on the books row.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Mavzz/readpanda/api-go/internal/database"
)

// Result is what a lookup found. Author and Pages are empty/0 when the source
// didn't say.
type Result struct {
	Title  string
	Author string
	Pages  int
}

// pageTolerance is how far a candidate's page count may be from the PDF's
// own and still be taken for the same book: editions differ by front matter
// and indexes, other books by far more.
const pageTolerance = 0.15

// Small words stay lower-case inside a title ("Notes for Professionals").
var smallWords = map[string]bool{
	"a": true, "an": true, "and": true, "as": true, "at": true, "but": true,
	"by": true, "for": true, "from": true, "in": true, "into": true, "of": true,
	"on": true, "or": true, "the": true, "to": true, "with": true,
}

// Humanize derives a readable title from a file name: strip the extension,
// split camelCase, digits and underscores into words, then title-case.
// "DataScienceAndPredictiveAnalyt" → "Data Science and Predictive Analyt".
// A name that already has spaces is treated as a real title and only has its
// extension and underscores removed — "The Three-Body Problem" stays as is.
func Humanize(name string) string {
	name = strings.TrimSpace(name)
	if dot := strings.LastIndex(name, "."); dot > 0 && len(name)-dot <= 5 {
		name = name[:dot]
	}
	if strings.Contains(name, " ") {
		return strings.Join(strings.Fields(strings.ReplaceAll(name, "_", " ")), " ")
	}

	var words []string
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == '_' }) {
		words = append(words, splitCamel(part)...)
	}
	for i, w := range words {
		lower := strings.ToLower(w)
		switch {
		case isAcronym(w):
			// Keep as written.
		case i > 0 && smallWords[lower]:
			words[i] = lower
		default:
			r := []rune(w)
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

// splitCamel splits at case and digit boundaries: "Python3Basics" →
// ["Python", "3", "Basics"].
func splitCamel(part string) []string {
	rs := []rune(part)
	if len(rs) == 0 {
		return nil
	}
	var words []string
	start := 0
	for i := 1; i < len(rs); i++ {
		prev, cur := rs[i-1], rs[i]
		var next rune
		if i+1 < len(rs) {
			next = rs[i+1]
		}
		boundary := (unicode.IsLower(prev) && unicode.IsUpper(cur)) ||
			(unicode.IsLetter(prev) && unicode.IsDigit(cur)) ||
			(unicode.IsDigit(prev) && unicode.IsLetter(cur)) ||
			// The end of an acronym: "PDFReader" → "PDF" + "Reader".
			(unicode.IsUpper(prev) && unicode.IsUpper(cur) && next != 0 && unicode.IsLower(next))
		if boundary {
			words = append(words, string(rs[start:i]))
			start = i
		}
	}
	return append(words, string(rs[start:]))
}

func isAcronym(w string) bool {
	if len([]rune(w)) < 2 {
		return false
	}
	for _, r := range w {
		if unicode.IsLower(r) {
			return false
		}
	}
	return true
}

// normalize keeps letters and digits only, lower-cased, so "Data Science and
// Predictive Analyt" and "Data science & predictive analytics" compare.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// accepts guards against a search returning some other book. The file name
// is the start of the real title (storage paths truncate it), so a match has
// to begin with everything we know. Very short names are too ambiguous.
func accepts(query, candidate string) bool {
	q, c := normalize(query), normalize(candidate)
	return len(q) >= 6 && strings.HasPrefix(c, q)
}

var client = &http.Client{Timeout: 8 * time.Second}

func getJSON(ctx context.Context, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	// Open Library asks API clients to identify themselves.
	req.Header.Set("User-Agent", "ReadPanda/1.0 (book metadata lookup)")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status %d", endpoint, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func openLibrary(ctx context.Context, query, search string) ([]Result, error) {
	var body struct {
		Docs []struct {
			Title  string   `json:"title"`
			Author []string `json:"author_name"`
			Pages  int      `json:"number_of_pages_median"`
		} `json:"docs"`
	}
	endpoint := "https://openlibrary.org/search.json?limit=5&fields=title,author_name,number_of_pages_median&title=" +
		url.QueryEscape(search)
	if err := getJSON(ctx, endpoint, &body); err != nil {
		return nil, err
	}
	var found []Result
	for _, d := range body.Docs {
		if accepts(query, d.Title) {
			r := Result{Title: d.Title, Pages: d.Pages}
			if len(d.Author) > 0 {
				r.Author = d.Author[0]
			}
			found = append(found, r)
		}
	}
	return found, nil
}

func googleBooks(ctx context.Context, query, search string) ([]Result, error) {
	var body struct {
		Items []struct {
			Info struct {
				Title   string   `json:"title"`
				Authors []string `json:"authors"`
				Pages   int      `json:"pageCount"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	endpoint := "https://www.googleapis.com/books/v1/volumes?maxResults=5&q=" +
		url.QueryEscape("intitle:"+search)
	if err := getJSON(ctx, endpoint, &body); err != nil {
		return nil, err
	}
	var found []Result
	for _, it := range body.Items {
		if accepts(query, it.Info.Title) {
			r := Result{Title: it.Info.Title, Pages: it.Info.Pages}
			if len(it.Info.Authors) > 0 {
				r.Author = it.Info.Authors[0]
			}
			found = append(found, r)
		}
	}
	return found, nil
}

// pick chooses the record that describes OUR file, or nothing. A generic
// title ("Quantum Mechanics", "Data Mining") matches books by several
// authors, and taking the first one captioned a Hecht cover "Leonard
// Susskind". So the author (and length) are only taken when they can be tied
// to the file the cover comes from:
//
//   - When the PDF's page count is known, only candidates within
//     pageTolerance of it are kept; if every candidate with a length is
//     further off than that, none of them is this book.
//   - What's left must name exactly one author. Two or more authors is
//     ambiguous, and an ambiguous match is no match: the book keeps its
//     humanized title and no author, which is better than a wrong one.
//
// ok=false means the candidates disagree (stop looking: another source
// won't know better); a nil result with ok=true means there was nothing.
func pick(cands []Result, pdfPages int) (r *Result, ok bool) {
	if len(cands) == 0 {
		return nil, true
	}
	if pdfPages > 0 {
		var near []Result
		measured := false
		for _, c := range cands {
			if c.Pages > 0 {
				measured = true
				if abs(c.Pages-pdfPages) <= int(pageTolerance*float64(pdfPages)) {
					near = append(near, c)
				}
			}
		}
		switch {
		case len(near) > 0:
			cands = near
		case measured:
			// Every candidate with a known length is some other book.
			return nil, false
		}
	}
	var chosen *Result
	for i := range cands {
		c := cands[i]
		if normalize(c.Author) == "" {
			continue
		}
		if chosen != nil && normalize(chosen.Author) != normalize(c.Author) {
			return nil, false
		}
		if chosen == nil {
			chosen = &c
		}
	}
	if chosen == nil {
		return nil, true
	}
	// Its length only when it isn't contradicted by the file's own.
	if pdfPages > 0 && chosen.Pages > 0 && abs(chosen.Pages-pdfPages) > int(pageTolerance*float64(pdfPages)) {
		chosen.Pages = 0
	}
	return chosen, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Resolve looks a humanized title up. Search engines match whole words, so a
// truncated last word ("Analyt") can sink the search: if the full title finds
// nothing, it tries again without the last word, still requiring the match
// to start with the full title. pdfPages is the file's own page count (0 if
// nobody has opened it yet); see pick for how a match is chosen.
func Resolve(ctx context.Context, title string, pdfPages int) (*Result, error) {
	searches := []string{title}
	if words := strings.Fields(title); len(words) > 2 {
		searches = append(searches, strings.Join(words[:len(words)-1], " "))
	}
	var lastErr error
	for _, source := range []func(context.Context, string, string) ([]Result, error){openLibrary, googleBooks} {
		for _, s := range searches {
			cands, err := source(ctx, title, s)
			if err != nil {
				lastErr = err
				continue
			}
			r, ok := pick(cands, pdfPages)
			if !ok {
				// Several books share this title and nothing ties one to the file.
				return nil, nil
			}
			if r != nil {
				return r, nil
			}
		}
	}
	return nil, lastErr
}

var running sync.Mutex

// EnrichPending looks up every book the job hasn't checked yet, at most
// `limit` of them (0 = all), and caches what it finds. A book is always given
// at least its humanized title. Only one run happens at a time; a call while
// another is running returns immediately with ran=false.
//
// Set METADATA_LOOKUP=off to skip the network (the humanized title is still
// written).
func EnrichPending(ctx context.Context, limit int) (checked, resolved int, ran bool, err error) {
	if !running.TryLock() {
		return 0, 0, false, nil
	}
	defer running.Unlock()

	// The file's own length: the count the reader recorded on the book, else
	// the longest any reader's PDF reported. A count the lookup itself wrote
	// is no evidence about the file.
	q := `SELECT b.book_id, COALESCE(b.source_title, b.title),
	             COALESCE(CASE WHEN NOT b.pages_from_lookup THEN b.page_count END,
	                      (SELECT MAX(p.total_pages) FROM reading_progress p WHERE p.book_id = b.book_id),
	                      0)
	        FROM books b
	       WHERE b.metadata_checked_at IS NULL ORDER BY b.created_at`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := database.DB.QueryContext(ctx, q)
	if err != nil {
		return 0, 0, true, err
	}
	type pending struct {
		id, source string
		pdfPages   int
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.source, &p.pdfPages); err != nil {
			rows.Close()
			return 0, 0, true, err
		}
		todo = append(todo, p)
	}
	rows.Close()

	lookup := !strings.EqualFold(os.Getenv("METADATA_LOOKUP"), "off")
	for i, p := range todo {
		if ctx.Err() != nil {
			return checked, resolved, true, ctx.Err()
		}
		title := Humanize(p.source)
		var found *Result
		if lookup {
			// Both APIs throttle anonymous callers; stay well under.
			if i > 0 {
				time.Sleep(time.Second)
			}
			found, err = Resolve(ctx, title, p.pdfPages)
			if err != nil {
				// A network failure or rate limit leaves the book unchecked so
				// the next run retries it, rather than caching "not found" —
				// but it still gets its readable title now.
				log.Printf("metadata: lookup failed for %s (%q): %v", p.id, title, err)
				if _, err := database.DB.ExecContext(ctx,
					`UPDATE books SET source_title = COALESCE(source_title, title), title = $2
					  WHERE book_id = $1`,
					p.id, title,
				); err != nil {
					return checked, resolved, true, err
				}
				continue
			}
		}

		var author *string
		var pages *int
		if found != nil {
			title = strings.Join(strings.Fields(found.Title), " ")
			found.Author = strings.Join(strings.Fields(found.Author), " ")
			if found.Author != "" {
				author = &found.Author
			}
			if found.Pages > 0 {
				pages = &found.Pages
			}
			resolved++
		}
		if len(title) > 255 {
			title = title[:255]
		}
		if _, err := database.DB.ExecContext(ctx,
			`UPDATE books
			    SET source_title = COALESCE(source_title, title),
			        title = $2,
			        author_from_lookup = author_from_lookup OR (author_name IS NULL AND $3::text IS NOT NULL),
			        author_name = COALESCE(author_name, $3),
			        pages_from_lookup = pages_from_lookup OR (page_count IS NULL AND $4::int IS NOT NULL),
			        page_count = COALESCE(page_count, $4),
			        metadata_checked_at = NOW()
			  WHERE book_id = $1`,
			p.id, title, author, pages,
		); err != nil {
			return checked, resolved, true, err
		}
		checked++
	}
	return checked, resolved, true, nil
}

// EnrichInBackground runs EnrichPending without blocking the caller, for use
// after books are added (seed, upload) and at startup.
func EnrichInBackground() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		checked, resolved, ran, err := EnrichPending(ctx, 0)
		if err != nil {
			log.Printf("metadata: enrichment stopped: %v", err)
		}
		if ran && checked > 0 {
			log.Printf("metadata: checked %d books, resolved %d", checked, resolved)
		}
	}()
}
