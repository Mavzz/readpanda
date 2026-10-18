// Command draw-covers gives every book without a cover one of our own: its
// title and author on a quiet colour, as the Gutenberg import draws them. A
// book with no cover shows as a blank swatch in the app.
//
//	go run ./cmd/draw-covers              # lists the books without a cover
//	go run ./cmd/draw-covers -out covers  # draws them into ./covers to look at
//	go run ./cmd/draw-covers -apply       # uploads them and sets cover_image_url
//
// It uses the database and bucket in .env.local; PG_HOST and friends set in
// the environment win, to run it against another database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/gutenberg"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

type book struct {
	id, title, author string
}

func main() {
	log.SetFlags(0)
	out := flag.String("out", "", "draw the covers into this directory, to look at them")
	apply := flag.Bool("apply", false, "upload the covers and set the books' cover_image_url")
	chrome := flag.String("chrome", gutenberg.FindChrome(), "Chrome or Chromium to draw with")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.Load()
	if err := database.Connect(cfg); err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	rows, err := database.DB.Query(`SELECT book_id, COALESCE(title, ''), COALESCE(author_name, '')
	                                  FROM books WHERE COALESCE(cover_image_url, '') = '' ORDER BY title`)
	if err != nil {
		log.Fatal(err)
	}
	var books []book
	for rows.Next() {
		var b book
		if err := rows.Scan(&b.id, &b.title, &b.author); err != nil {
			log.Fatal(err)
		}
		books = append(books, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books without a cover", len(books))
	for _, b := range books {
		log.Printf("  %s  %s — %s", b.id, b.title, b.author)
	}
	if len(books) == 0 || (*out == "" && !*apply) {
		if len(books) > 0 {
			log.Printf("Nothing was drawn. Run again with -out <dir> to look at them, or -apply to set them.")
		}
		return
	}
	if *chrome == "" {
		log.Fatal("no Chrome found: install Google Chrome, or pass -chrome / set CHROME_PATH")
	}
	if *apply {
		if err := utils.InitObjectStorage(cfg); err != nil {
			log.Fatal(err)
		}
	}
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	r := gutenberg.Renderer{Chrome: *chrome}
	work, err := os.MkdirTemp("", "covers-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(work)

	failed := 0
	for _, b := range books {
		if err := ctx.Err(); err != nil {
			log.Fatal(err)
		}
		if err := draw(ctx, r, b, work, *out, *apply); err != nil {
			failed++
			log.Printf("  %s: %v", b.id, err)
			continue
		}
		log.Printf("  %s: done", b.id)
	}
	if failed > 0 {
		log.Fatalf("%d of %d failed", failed, len(books))
	}
}

func draw(ctx context.Context, r gutenberg.Renderer, b book, work, out string, apply bool) error {
	png := filepath.Join(work, b.id+".png")
	// The full title: "The Constitution of India: Appendix" and its siblings
	// would otherwise get the same cover.
	if err := r.Cover(ctx, strings.TrimSpace(b.title), b.author, b.id, png); err != nil {
		return err
	}
	data, err := os.ReadFile(png)
	if err != nil {
		return err
	}
	if out != "" {
		if err := os.WriteFile(filepath.Join(out, b.id+".png"), data, 0o644); err != nil {
			return err
		}
	}
	if !apply {
		return nil
	}
	// Same key the import uses, so a book only ever has one cover file.
	url, err := utils.UploadFileToStorage(data, "image/png", fmt.Sprintf("books/covers/%s-cover.png", b.id))
	if err != nil {
		return err
	}
	// Only if it still has none: an upload since the listing wins.
	_, err = database.DB.Exec(`UPDATE books SET cover_image_url = $2, updated_at = NOW()
	                            WHERE book_id = $1 AND COALESCE(cover_image_url, '') = ''`, b.id, url)
	return err
}
