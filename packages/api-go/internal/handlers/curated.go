package handlers

import (
	"database/sql"
	"math"
	"sort"

	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
)

// Shared by Discover (8a), Our Picks and the curated bucket screen (9b):
// every active curated bucket with its books, and what's derived from them.

// Reading time for "~{hrs} hrs".
const minutesPerPage = 1.5

// A genre is derived as a bucket's tag when at least this many of its books
// share it, or when it's at least tagSharePct of the books.
const (
	tagMinBooks = 2
	tagSharePct = 40
)

type curatedSet struct {
	bucket     models.CuratedBucket
	editorTags []string
	books      []models.BucketBook
}

// loadCurated reads the active curated buckets (or just one, by id) with their
// books in bucket order, and fills in tags, count and reading time.
func loadCurated(onlyID string) ([]*curatedSet, error) {
	rows, err := database.DB.Query(
		`SELECT id, title, sort_order, cover_image_url, is_active, description, genre_tags
		   FROM curated_buckets
		  WHERE is_active = true AND ($1 = '' OR id = $1)
		  ORDER BY sort_order, title`,
		onlyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sets []*curatedSet
	byID := map[string]*curatedSet{}
	for rows.Next() {
		c := &curatedSet{}
		var desc sql.NullString
		if err := rows.Scan(
			&c.bucket.ID, &c.bucket.Title, &c.bucket.SortOrder, &c.bucket.CoverImageURL,
			&c.bucket.IsActive, &desc, pq.Array(&c.editorTags),
		); err != nil {
			return nil, err
		}
		if desc.Valid && desc.String != "" {
			c.bucket.Description = &desc.String
		}
		sets = append(sets, c)
		byID[c.bucket.ID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	bookRows, err := database.DB.Query(
		`SELECT cbb.bucket_id, b.book_id, b.title, b.author_name, b.cover_image_url,
		        b.manuscript_url, COALESCE(b.subgenre, ''), b.page_count
		   FROM curated_bucket_books cbb
		   JOIN books b ON b.book_id = cbb.book_id
		   JOIN curated_buckets cb ON cb.id = cbb.bucket_id
		  WHERE cb.is_active = true AND ($1 = '' OR cbb.bucket_id = $1)
		  ORDER BY cbb.bucket_id, cbb.sort_order, b.title`,
		onlyID,
	)
	if err != nil {
		return nil, err
	}
	defer bookRows.Close()
	for bookRows.Next() {
		var bucketID string
		var b models.BucketBook
		if err := bookRows.Scan(
			&bucketID, &b.BookID, &b.Title, &b.AuthorName, &b.CoverImageURL,
			&b.ManuscriptURL, &b.Subgenre, &b.PageCount,
		); err != nil {
			return nil, err
		}
		if c := byID[bucketID]; c != nil {
			c.books = append(c.books, b)
		}
	}
	if err := bookRows.Err(); err != nil {
		return nil, err
	}

	for _, c := range sets {
		c.bucket.BookCount = len(c.books)
		c.bucket.GenreTags = c.tags()
		c.bucket.ReadingMinutes = c.readingMinutes()
	}
	return sets, nil
}

// tags are the editor's when there are any. Otherwise a subgenre becomes a
// tag when ≥2 books (or ≥40% of them) share it, so a mixed bucket can carry
// several and appear under several chips. Most-shared first.
func (c *curatedSet) tags() []string {
	if len(c.editorTags) > 0 {
		return c.editorTags
	}
	counts := map[string]int{}
	for _, b := range c.books {
		if b.Subgenre != "" {
			counts[b.Subgenre]++
		}
	}
	n := len(c.books)
	tags := []string{}
	for g, k := range counts {
		if k >= tagMinBooks || k*100 >= tagSharePct*n {
			tags = append(tags, g)
		}
	}
	sort.Slice(tags, func(i, j int) bool {
		if counts[tags[i]] != counts[tags[j]] {
			return counts[tags[i]] > counts[tags[j]]
		}
		return tags[i] < tags[j]
	})
	return tags
}

func (c *curatedSet) hasTag(genre string) bool {
	for _, t := range c.bucket.GenreTags {
		if t == genre {
			return true
		}
	}
	return false
}

// matching counts the books in any of the given subgenres.
func (c *curatedSet) matching(genres ...string) int {
	want := map[string]bool{}
	for _, g := range genres {
		want[g] = true
	}
	m := 0
	for _, b := range c.books {
		if want[b.Subgenre] {
			m++
		}
	}
	return m
}

// readingMinutes is null unless every book's length is known — a total over
// some of the books would understate it.
func (c *curatedSet) readingMinutes() *int {
	if len(c.books) == 0 {
		return nil
	}
	pages := 0
	for _, b := range c.books {
		if b.PageCount == nil || *b.PageCount <= 0 {
			return nil
		}
		pages += *b.PageCount
	}
	mins := int(math.Round(float64(pages) * minutesPerPage))
	return &mins
}

// preview is the tile's cover stack: books in the selected genre first, then
// the rest, each in bucket order. limit 0 = every book.
func (c *curatedSet) preview(genre string, limit int) []models.BookPreview {
	ordered := make([]models.BucketBook, 0, len(c.books))
	if genre != "" {
		for _, b := range c.books {
			if b.Subgenre == genre {
				ordered = append(ordered, b)
			}
		}
		for _, b := range c.books {
			if b.Subgenre != genre {
				ordered = append(ordered, b)
			}
		}
	} else {
		ordered = append(ordered, c.books...)
	}
	if limit > 0 && len(ordered) > limit {
		ordered = ordered[:limit]
	}
	out := make([]models.BookPreview, 0, len(ordered))
	for _, b := range ordered {
		p := models.BookPreview{
			BookID:        b.BookID,
			Title:         b.Title,
			CoverImageURL: b.CoverImageURL,
			ManuscriptURL: b.ManuscriptURL,
		}
		if b.AuthorName != nil {
			p.AuthorName = *b.AuthorName
		}
		out = append(out, p)
	}
	return out
}

// attachSaved marks the buckets the user has saved a copy of.
func attachSaved(userID string, sets []*curatedSet) error {
	rows, err := database.DB.Query(
		`SELECT source_curated_id, id FROM user_buckets
		  WHERE user_id = $1 AND source_curated_id IS NOT NULL`,
		userID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	saved := map[string]string{}
	for rows.Next() {
		var src, id string
		if err := rows.Scan(&src, &id); err != nil {
			return err
		}
		saved[src] = id
	}
	for _, c := range sets {
		if id, ok := saved[c.bucket.ID]; ok {
			id := id
			c.bucket.SavedBucketID = &id
		}
	}
	return rows.Err()
}

// progressFor returns the user's position in each of the given books that
// they've opened.
func progressFor(userID string, bookIDs []string) (map[string]*models.BookProgress, error) {
	out := map[string]*models.BookProgress{}
	if len(bookIDs) == 0 {
		return out, nil
	}
	rows, err := database.DB.Query(
		`SELECT book_id, current_page, total_pages, last_read_at
		   FROM reading_progress
		  WHERE user_id = $1 AND book_id = ANY($2)`,
		userID, pq.Array(bookIDs),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		p := &models.BookProgress{}
		if err := rows.Scan(&id, &p.CurrentPage, &p.TotalPages, &p.LastReadAt); err != nil {
			return nil, err
		}
		p.ProgressPct = progressPct(p.CurrentPage, p.TotalPages)
		out[id] = p
	}
	return out, rows.Err()
}

// finishedSQL matches progressPct(...) >= 100, so the server's "finished"
// agrees with the progress bars the app draws.
const finishedSQL = `p.total_pages > 0 AND ROUND((p.current_page + 1) * 100.0 / p.total_pages) >= 100`
