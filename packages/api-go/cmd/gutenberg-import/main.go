// Command gutenberg-import adds public-domain books from Project Gutenberg to
// the catalogue.
//
// It picks books every credited person on which (author, translator,
// illustrator, editor) died long enough ago for the book to be free in every
// country the app ships to, strips Gutenberg's name from them as their
// licence asks of paid apps, prints them to PDF with a cover of our own, and
// records why each one was judged free (book_provenance) — the answer to a
// takedown request.
//
// Three steps, each safe to re-run:
//
//	# 1. Decide. Reads the catalogue, writes gutenberg-report.csv, touches nothing.
//	curl -LO https://www.gutenberg.org/cache/epub/feeds/rdf-files.tar.bz2
//	go run ./cmd/gutenberg-import -catalog rdf-files.tar.bz2
//
//	# 2. Look. Downloads and prints the chosen books into ./gutenberg-out.
//	go run ./cmd/gutenberg-import -catalog rdf-files.tar.bz2 -out gutenberg-out
//
//	# 3. Publish. Uploads to R2 and adds the rows (needs .env.local).
//	go run ./cmd/gutenberg-import -catalog rdf-files.tar.bz2 -apply
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/gutenberg"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

type options struct {
	catalog  string
	ids      []int
	limit    int
	policy   gutenberg.Policy
	report   string
	out      string
	apply    bool
	genre    string
	subgenre string
	fetcher  *gutenberg.Fetcher
	renderer gutenberg.Renderer
}

// row is one line of the report.
type row struct {
	book    gutenberg.Book
	missing bool
	ok      bool
	reason  string
	result  string
}

func main() {
	log.SetFlags(0)
	o := parseFlags()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	rows, err := choose(o)
	if err != nil {
		log.Fatal(err)
	}
	chosen := 0
	for _, r := range rows {
		if r.ok {
			chosen++
		}
	}
	log.Printf("%d %s chosen (life+%d: everyone credited died in or before %d)", chosen, plural(chosen, "book", "books"), o.policy.Term, o.policy.LatestDeath())

	if o.apply || o.out != "" {
		if err := process(ctx, o, rows); err != nil {
			writeReport(o.report, rows)
			log.Fatal(err)
		}
	}
	if err := writeReport(o.report, rows); err != nil {
		log.Fatal(err)
	}
	log.Printf("Report: %s", o.report)
	if !o.apply && o.out == "" {
		log.Printf("Nothing was downloaded or uploaded. Check the report, then run again with -out <dir> to see the PDFs, or -apply to publish.")
	}
}

func parseFlags() options {
	var o options
	var ids, langs string
	var term int
	var delay time.Duration
	var mirror, chrome string

	flag.StringVar(&o.catalog, "catalog", "", "Gutenberg's rdf-files.tar.bz2, or the directory it was unpacked into (download: "+gutenberg.CatalogURL+")")
	flag.StringVar(&ids, "ids", "", "comma-separated ebook numbers to consider, e.g. 1342,84 (default: the most downloaded books that pass)")
	flag.IntVar(&o.limit, "limit", 50, "how many books to choose when -ids isn't given")
	flag.StringVar(&langs, "lang", "en", "comma-separated languages to allow; empty allows any")
	flag.StringVar(&o.policy.LoCC, "class", "P", "Library of Congress class prefix to allow (P is literature); empty allows any")
	flag.IntVar(&term, "term", 70, "life-plus term to clear: 70 covers almost every country, 100 covers all of them")
	flag.StringVar(&o.report, "report", "gutenberg-report.csv", "where to write the report")
	flag.StringVar(&o.out, "out", "", "download and print the chosen books into this directory, to check them")
	flag.BoolVar(&o.apply, "apply", false, "upload the chosen books to storage and add them to the database")
	flag.StringVar(&o.genre, "genre", "Fiction", "genre for the books rows")
	flag.StringVar(&o.subgenre, "subgenre", "Classics", "subgenre for the books rows")
	flag.StringVar(&mirror, "mirror", "https://gutenberg.pglaf.org", "Gutenberg mirror to download from first; empty uses the catalogue's links")
	flag.DurationVar(&delay, "delay", 2*time.Second, "wait between downloads, as Gutenberg asks")
	flag.StringVar(&chrome, "chrome", gutenberg.FindChrome(), "Chrome or Chromium to print with")
	flag.Parse()

	if o.catalog == "" {
		fmt.Fprintf(os.Stderr, "-catalog is required. Download it once (about 100 MB):\n\n  curl -LO %s\n\n", gutenberg.CatalogURL)
		flag.Usage()
		os.Exit(2)
	}
	for _, s := range strings.Split(ids, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("-ids: %q isn't a number", s)
		}
		o.ids = append(o.ids, n)
	}
	for _, l := range strings.Split(langs, ",") {
		if l = strings.TrimSpace(l); l != "" {
			o.policy.Languages = append(o.policy.Languages, l)
		}
	}
	if term < 50 {
		log.Fatal("-term under 50 years wouldn't clear most countries")
	}
	o.policy.Term = term
	o.policy.Year = time.Now().Year()
	o.fetcher = &gutenberg.Fetcher{Mirror: mirror, Delay: delay}
	o.renderer = gutenberg.Renderer{Chrome: chrome}
	return o
}

// choose reads the catalogue and decides on every book in scope. With -ids
// that's the books asked for, each reported; otherwise the most downloaded
// books in scope, until -limit pass.
func choose(o options) ([]*row, error) {
	wanted := map[int]bool{}
	for _, id := range o.ids {
		wanted[id] = true
	}

	var books []gutenberg.Book
	log.Printf("Reading %s…", o.catalog)
	err := gutenberg.ReadCatalog(o.catalog, func(b gutenberg.Book) error {
		if len(wanted) == 0 || wanted[b.ID] {
			books = append(books, b)
		}
		return nil
	}, func(name string, err error) {
		log.Printf("skipping %s: %v", name, err)
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(books, func(i, j int) bool { return books[i].Downloads > books[j].Downloads })

	var rows []*row
	seen := map[string]int{} // title and authors → the edition already chosen
	chosen := 0
	for _, b := range books {
		if len(wanted) == 0 && chosen >= o.limit {
			break
		}
		r := &row{book: b}
		if ok, reason := o.policy.InScope(b); !ok {
			if len(wanted) == 0 {
				continue // the report is about books we could have taken
			}
			r.reason = reason
		} else if ok, reason := o.policy.Clear(b); !ok {
			r.reason = reason
		} else if b.HTMLZip == "" {
			r.reason = "no HTML edition to print"
		} else if first, dup := seen[editionKey(b)]; dup {
			r.reason = fmt.Sprintf("another edition (#%d) is already chosen", first)
		} else {
			r.ok = true
			seen[editionKey(b)] = b.ID
			chosen++
		}
		rows = append(rows, r)
	}

	found := map[int]bool{}
	for _, b := range books {
		found[b.ID] = true
	}
	for _, id := range o.ids {
		if !found[id] {
			rows = append(rows, &row{book: gutenberg.Book{ID: id}, missing: true, reason: "not in the catalogue"})
		}
	}
	return rows, nil
}

func editionKey(b gutenberg.Book) string {
	title := strings.ToLower(b.Title)
	if i := strings.IndexAny(title, ":;"); i > 0 {
		title = title[:i] // "Frankenstein; Or, The Modern Prometheus" is Frankenstein
	}
	return strings.TrimSpace(title) + "|" + strings.ToLower(gutenberg.AuthorLine(b))
}

// process downloads, cleans and prints each chosen book, and with -apply
// publishes it. A book that fails is noted in the report and the rest carry
// on; only setup problems stop the run.
func process(ctx context.Context, o options, rows []*row) error {
	if o.renderer.Chrome == "" {
		return errors.New("no Chrome found: install Google Chrome, or pass -chrome / set CHROME_PATH")
	}
	if o.out != "" {
		if err := os.MkdirAll(o.out, 0o755); err != nil {
			return err
		}
	}
	if o.apply {
		if err := connect(o); err != nil {
			return err
		}
		defer database.Close()
	}

	done := 0
	for _, r := range rows {
		if !r.ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		done++
		log.Printf("[%d] #%d %s", done, r.book.ID, r.book.Title)

		if o.apply {
			imported, err := alreadyImported(r.book.ID)
			if err != nil {
				return err
			}
			if imported != "" {
				r.result = "already imported as " + imported
				log.Printf("    %s", r.result)
				continue
			}
		}
		r.result = importBook(ctx, o, r.book)
		log.Printf("    %s", r.result)
	}
	return nil
}

// importBook does one book and says how it went, for the report.
func importBook(ctx context.Context, o options, b gutenberg.Book) string {
	work, err := os.MkdirTemp("", fmt.Sprintf("pg%d-", b.ID))
	if err != nil {
		return "failed: " + err.Error()
	}
	defer os.RemoveAll(work)

	zipData, source, err := o.fetcher.HTMLZip(ctx, b)
	if err != nil {
		return "failed to download: " + err.Error()
	}
	// The archive goes in a directory of its own; the PDF and cover are
	// written beside it, never inside.
	unpacked := filepath.Join(work, "src")
	if err := os.Mkdir(unpacked, 0o700); err != nil {
		return "failed: " + err.Error()
	}
	mainHTML, err := gutenberg.Unpack(zipData, unpacked)
	if err != nil {
		return "failed to unpack: " + err.Error()
	}
	raw, err := os.ReadFile(mainHTML)
	if err != nil {
		return "failed: " + err.Error()
	}
	title := gutenberg.AuthorLine(b)
	if title != "" {
		title = b.Title + " by " + title
	}
	clean, err := gutenberg.Clean(raw, title)
	if err != nil {
		if errors.Is(err, gutenberg.ErrNeedsReview) {
			return err.Error()
		}
		return "failed to clean: " + err.Error()
	}
	// Beside the original, so its relative image paths still resolve.
	cleanHTML := filepath.Join(filepath.Dir(mainHTML), "readpanda-book.html")
	if err := os.WriteFile(cleanHTML, clean, 0o600); err != nil {
		return "failed: " + err.Error()
	}

	pdfPath, coverPath := filepath.Join(work, "book.pdf"), filepath.Join(work, "cover.png")
	if err := o.renderer.PDF(ctx, cleanHTML, pdfPath); err != nil {
		return "failed to print: " + err.Error()
	}
	if err := o.renderer.Cover(ctx, b, coverPath); err != nil {
		return "failed to draw the cover: " + err.Error()
	}
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return "failed: " + err.Error()
	}
	cover, err := os.ReadFile(coverPath)
	if err != nil {
		return "failed: " + err.Error()
	}
	pages := gutenberg.PageCount(pdf)

	name := fmt.Sprintf("%d-%s", b.ID, slug(b.Title))
	result := fmt.Sprintf("printed, %d pages", pages)
	if o.out != "" {
		if err := os.WriteFile(filepath.Join(o.out, name+".pdf"), pdf, 0o644); err != nil {
			return "failed: " + err.Error()
		}
		if err := os.WriteFile(filepath.Join(o.out, name+".png"), cover, 0o644); err != nil {
			return "failed: " + err.Error()
		}
	}
	if !o.apply {
		return result
	}

	id := bookID(b.ID)
	manuscriptURL, err := utils.UploadFileToStorage(pdf, "application/pdf", fmt.Sprintf("books/manuscripts/%s-%s.pdf", id, slug(b.Title)))
	if err != nil {
		return "failed to upload: " + err.Error()
	}
	coverURL, err := utils.UploadFileToStorage(cover, "image/png", fmt.Sprintf("books/covers/%s-cover.png", id))
	if err != nil {
		return "failed to upload: " + err.Error()
	}
	if err := insert(o, b, id, manuscriptURL, coverURL, pages, source); err != nil {
		return "failed to save: " + err.Error()
	}
	return fmt.Sprintf("imported as %s, %d pages", id, pages)
}

// bookID is fixed per Gutenberg number, so a re-run writes to the same
// storage keys instead of leaving copies behind. It keeps the bk_ + 8 hex
// shape the API checks for.
func bookID(gutenbergID int) string {
	sum := sha256.Sum256([]byte("gutenberg:" + strconv.Itoa(gutenbergID)))
	return "bk_" + hex.EncodeToString(sum[:4])
}

func connect(o options) error {
	cfg := config.Load()
	if err := database.Connect(cfg); err != nil {
		return err
	}
	if err := utils.InitObjectStorage(cfg); err != nil {
		return err
	}
	var exists bool
	if err := database.DB.QueryRow(`SELECT to_regclass('public.book_provenance') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("no book_provenance table: run scripts/migrate_book_provenance.sql first")
	}
	err := database.DB.QueryRow(`SELECT true FROM preferences WHERE genre = $1 AND subgenre = $2 LIMIT 1`, o.genre, o.subgenre).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("no %s / %s in preferences; add it, or pick another with -genre / -subgenre", o.genre, o.subgenre)
	}
	return err
}

// alreadyImported is the book id a Gutenberg book was imported as, or "".
func alreadyImported(gutenbergID int) (string, error) {
	var id string
	err := database.DB.QueryRow(`SELECT book_id FROM book_provenance WHERE source = 'gutenberg' AND source_id = $1`,
		strconv.Itoa(gutenbergID)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func insert(o options, b gutenberg.Book, id, manuscriptURL, coverURL string, pages int, source string) error {
	contributors, err := json.Marshal(b.Contributors)
	if err != nil {
		return err
	}
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// No user_id: like the seeded books, nobody uploaded these. The title and
	// author come from the catalogue, so the metadata lookup is marked done
	// rather than left to rename them.
	if _, err := tx.Exec(`
		INSERT INTO books (book_id, user_id, title, source_title, description, genre, subgenre, author_name,
		                   cover_image_url, manuscript_url, page_count, status, views, metadata_checked_at)
		VALUES ($1, NULL, $2, $2, '', $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, 0), 1, 0, NOW())`,
		id, truncate(b.Title, 255), o.genre, o.subgenre, truncate(gutenberg.AuthorLine(b), 255),
		coverURL, manuscriptURL, pages); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO book_provenance (book_id, source, source_id, source_url, downloaded_from, rights, contributors, rule)
		VALUES ($1, 'gutenberg', $2, $3, $4, $5, $6, $7)`,
		id, strconv.Itoa(b.ID), fmt.Sprintf("https://www.gutenberg.org/ebooks/%d", b.ID), source,
		b.Rights, string(contributors), o.policy.Rule()); err != nil {
		return err
	}
	return tx.Commit()
}

func writeReport(path string, rows []*row) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"ebook", "title", "authors", "everyone credited", "downloads", "verdict", "reason", "result"})
	for _, r := range rows {
		verdict := "rejected"
		if r.ok {
			verdict = "chosen"
		}
		var people []string
		for _, c := range r.book.Contributors {
			people = append(people, fmt.Sprintf("%s (%s, %s–%s)", c.Name, c.Role, year(c.Birth), year(c.Death)))
		}
		w.Write([]string{
			strconv.Itoa(r.book.ID), r.book.Title, gutenberg.AuthorLine(r.book), strings.Join(people, "; "),
			strconv.Itoa(r.book.Downloads), verdict, r.reason, r.result,
		})
	}
	w.Flush()
	return w.Error()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func year(y int) string {
	if y == 0 {
		return "?"
	}
	return strconv.Itoa(y)
}

// slug is a title as a file name: "Brontë: Jane Eyre!" → "Bronte_Jane_Eyre".
func slug(title string) string {
	var b strings.Builder
	sep := false
	for _, r := range norm.NFD.String(title) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// The accent of a decomposed letter.
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			if sep && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			sep = false
		default:
			sep = true
		}
		if b.Len() >= 60 {
			break
		}
	}
	if b.Len() == 0 {
		return "book"
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
