package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// DiscoverHandler serves the Discover tab (8a) and Book detail (8c).
type DiscoverHandler struct {
	Config *config.Config
}

// NewDiscoverHandler creates a new discover handler
func NewDiscoverHandler(cfg *config.Config) *DiscoverHandler {
	return &DiscoverHandler{Config: cfg}
}

// How many books the Popular row carries, and how many friend avatars Book
// detail's social line draws.
const (
	popularLimit   = 20
	friendsShown   = 3
	previewCovers  = 3
	popularWindow  = "7 days"
	friendWeight   = 2
	likedGenreBump = 3
)

// friendsCTE is everyone who shares a room with $1, the reader. There is no
// friends graph, and a room is the only relationship the app has between two
// readers, so "friends" means room-mates. Admins are included through
// rooms.admin_id because a room's creator isn't guaranteed a room_members row
// (see isRoomMember).
const friendsCTE = `
	my_rooms AS (
		SELECT id AS room_id FROM rooms WHERE admin_id = $1
		UNION
		SELECT room_id FROM room_members WHERE user_id = $1
	),
	friends AS (
		SELECT m.user_id FROM room_members m JOIN my_rooms r ON r.room_id = m.room_id
		UNION
		SELECT rm.admin_id FROM rooms rm JOIN my_rooms r ON r.room_id = rm.id
	)`

// likedSubgenres reads the subgenres the reader picked in Settings › Genres I
// like. Preferences are stored as the mobile picker writes them:
// { category: [{ preference_id, preference_subgenre, preference_value }] }.
// Categories are walked in name order so the chip row doesn't reshuffle
// between requests (Go map order is random).
func likedSubgenres(userID string) []string {
	var raw sql.NullString
	err := database.DB.QueryRow(
		`SELECT preferences FROM user_preferences WHERE user_id = $1`, userID,
	).Scan(&raw)
	if err != nil || !raw.Valid {
		return []string{}
	}

	var prefs map[string][]struct {
		Subgenre string `json:"preference_subgenre"`
		Value    bool   `json:"preference_value"`
	}
	if err := json.Unmarshal([]byte(raw.String), &prefs); err != nil {
		return []string{}
	}

	categories := make([]string, 0, len(prefs))
	for c := range prefs {
		categories = append(categories, c)
	}
	sort.Strings(categories)

	liked := []string{}
	seen := map[string]bool{}
	for _, c := range categories {
		for _, p := range prefs[c] {
			if p.Value && p.Subgenre != "" && !seen[p.Subgenre] {
				seen[p.Subgenre] = true
				liked = append(liked, p.Subgenre)
			}
		}
	}
	return liked
}

// GetDiscover — GET /discover?genre={subgenre}
//
// With no genre this is the "For you" feed: curated buckets ranked by how
// many of their books sit in the reader's liked genres, and a Popular row
// ranked by a mix of this week's readers, friends who've read the book, and
// the reader's genres. With a genre, curated buckets are those tagged with it
// (see discoverCurated) and Popular is a plain per-book genre filter.
func (h *DiscoverHandler) GetDiscover(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	liked := likedSubgenres(userID)

	genres, err := discoverGenres(liked)
	if err != nil {
		http.Error(w, `{"error": "Failed to load genres"}`, http.StatusInternalServerError)
		return
	}
	curated, err := discoverCurated(userID, genre, liked)
	if err != nil {
		http.Error(w, `{"error": "Failed to load curated buckets"}`, http.StatusInternalServerError)
		return
	}
	popular, err := discoverPopular(userID, genre, liked)
	if err != nil {
		http.Error(w, `{"error": "Failed to load popular books"}`, http.StatusInternalServerError)
		return
	}

	resp := models.DiscoverResponse{
		Genres:  genres,
		Curated: curated,
		Popular: popular,
	}
	if genre != "" {
		resp.Genre = &genre
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// discoverGenres is the chip row after "For you": the reader's liked
// subgenres first, in their own order, then every other subgenre the
// catalogue has a book in. A liked genre with no books is left out, since its
// chip would only ever open an empty page.
func discoverGenres(liked []string) ([]models.DiscoverGenre, error) {
	rows, err := database.DB.Query(
		`SELECT DISTINCT subgenre FROM books
		  WHERE subgenre IS NOT NULL AND subgenre <> ''
		  ORDER BY subgenre`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	available := map[string]bool{}
	ordered := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		available[s] = true
		ordered = append(ordered, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	genres := []models.DiscoverGenre{}
	isLiked := map[string]bool{}
	for _, s := range liked {
		if available[s] {
			isLiked[s] = true
			genres = append(genres, models.DiscoverGenre{Value: s, Label: s, Liked: true})
		}
	}
	for _, s := range ordered {
		if !isLiked[s] {
			genres = append(genres, models.DiscoverGenre{Value: s, Label: s})
		}
	}
	return genres, nil
}

// discoverCurated ranks the curated buckets.
//
// Under a genre: every bucket tagged with it (tags can be several, so a mixed
// bucket shows under several chips), ranked by the share of its books in
// that genre, so a pure Sci-fi bucket outranks one that's 3/9 Sci-fi. The
// cover stack leads with matching books and matching_count drives the tile's
// "{m} of {n} are {Genre}".
//
// For you: every bucket, ranked by how many books sit in the reader's liked
// genres, then editorial order.
func discoverCurated(userID, genre string, liked []string) ([]models.CuratedBucket, error) {
	sets, err := loadCurated("")
	if err != nil {
		return nil, err
	}
	if err := attachSaved(userID, sets); err != nil {
		return nil, err
	}

	type ranked struct {
		set   *curatedSet
		score float64
	}
	list := []ranked{}
	for _, c := range sets {
		if genre == "" {
			list = append(list, ranked{c, float64(c.matching(liked...))})
			continue
		}
		if !c.hasTag(genre) || len(c.books) == 0 {
			continue
		}
		m := c.matching(genre)
		c.bucket.MatchingCount = &m
		list = append(list, ranked{c, float64(m) / float64(len(c.books))})
	}
	// Stable, so equal scores keep editorial (sort_order) order.
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })

	buckets := make([]models.CuratedBucket, 0, len(list))
	for _, r := range list {
		r.set.bucket.BooksPreview = r.set.preview(genre, previewCovers)
		buckets = append(buckets, r.set.bucket)
	}
	return buckets, nil
}

// discoverPopular is the Popular row. Under a genre it is simply that genre's
// books by readers this week. On For you, a friend having read a book and the
// book sitting in a liked genre both lift it. Views and recency break ties so
// a catalogue nobody has read yet still has an order.
func discoverPopular(userID, genre string, liked []string) ([]models.PopularBook, error) {
	rows, err := database.DB.Query(
		`WITH `+friendsCTE+`,
		weekly AS (
			SELECT book_id, COUNT(DISTINCT user_id) AS n
			  FROM reading_progress
			 WHERE last_read_at >= NOW() - $5::interval
			 GROUP BY book_id
		),
		friend_reads AS (
			SELECT p.book_id, COUNT(DISTINCT p.user_id) AS n
			  FROM reading_progress p
			  JOIN friends f ON f.user_id = p.user_id
			 WHERE p.user_id <> $1
			 GROUP BY p.book_id
		)
		SELECT b.book_id, b.title, b.author_name, b.page_count, b.genre, b.subgenre,
		       b.cover_image_url, b.manuscript_url, COALESCE(b.created_at, NOW()),
		       COALESCE(w.n, 0), COALESCE(fr.n, 0)
		  FROM books b
		  LEFT JOIN weekly w ON w.book_id = b.book_id
		  LEFT JOIN friend_reads fr ON fr.book_id = b.book_id
		 WHERE $2 = '' OR b.subgenre = $2
		 ORDER BY CASE WHEN $2 = ''
		               THEN COALESCE(w.n, 0)
		                    + $6 * COALESCE(fr.n, 0)
		                    + CASE WHEN b.subgenre = ANY($3) THEN $7 ELSE 0 END
		               ELSE COALESCE(w.n, 0) END DESC,
		          b.views DESC, b.created_at DESC
		 LIMIT $4`,
		userID, genre, pq.Array(liked), popularLimit, popularWindow, friendWeight, likedGenreBump,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []models.PopularBook{}
	for rows.Next() {
		var b models.PopularBook
		var g, sg sql.NullString
		if err := rows.Scan(
			&b.BookID, &b.Title, &b.AuthorName, &b.PageCount, &g, &sg,
			&b.CoverImageURL, &b.ManuscriptURL, &b.AddedAt, &b.ReadersThisWeek, &b.FriendsRead,
		); err != nil {
			return nil, err
		}
		b.Genre, b.Subgenre = g.String, sg.String
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetBookDetail — GET /books/{bookId}
//
// Everything Book detail (8c) needs beyond what the reader's device already
// knows: the catalogue record, which of the reader's buckets hold it (the
// bookmark's filled state and the Add-to-bucket sheet's ticks), and which
// friends have read it. The reader's own position and the rooms reading it
// are already on the device, so they aren't repeated here.
func (h *DiscoverHandler) GetBookDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.ExtractUserID(w, r, h.Config.JWTSecret)
	if !ok {
		return
	}
	bookID := mux.Vars(r)["bookId"]

	var detail models.BookDetail
	var desc, g, sg sql.NullString
	err := database.DB.QueryRow(
		`SELECT book_id, title, description, author_name, page_count, genre, subgenre,
		        cover_image_url, manuscript_url
		   FROM books WHERE book_id = $1`,
		bookID,
	).Scan(
		&detail.Book.BookID, &detail.Book.Title, &desc, &detail.Book.AuthorName,
		&detail.Book.PageCount, &g, &sg, &detail.Book.CoverImageURL, &detail.Book.ManuscriptURL,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Book not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to get book"}`, http.StatusInternalServerError)
		return
	}
	detail.Book.Description = desc.String
	detail.Book.Genre, detail.Book.Subgenre = g.String, sg.String

	detail.InBuckets, err = bucketsHolding(userID, bookID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get book buckets"}`, http.StatusInternalServerError)
		return
	}
	detail.FriendsRead, err = friendsWhoRead(userID, bookID)
	if err != nil {
		http.Error(w, `{"error": "Failed to get friends"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(detail)
}

func bucketsHolding(userID, bookID string) ([]string, error) {
	rows, err := database.DB.Query(
		`SELECT ub.id
		   FROM user_bucket_books ubb
		   JOIN user_buckets ub ON ub.id = ubb.bucket_id
		  WHERE ub.user_id = $1 AND ubb.book_id = $2
		  ORDER BY ub.created_at`,
		userID, bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// friendsWhoRead counts every room-mate with a position in the book (any
// entry counts, per the 8c spec) and names the most recent few for the
// avatar stack.
func friendsWhoRead(userID, bookID string) (models.FriendsRead, error) {
	result := models.FriendsRead{Friends: []models.FriendReader{}}
	rows, err := database.DB.Query(
		`WITH `+friendsCTE+`
		SELECT u.uuid, u.username, COUNT(*) OVER ()
		  FROM reading_progress p
		  JOIN friends f ON f.user_id = p.user_id
		  JOIN users u ON u.uuid = p.user_id
		 WHERE p.book_id = $2 AND p.user_id <> $1
		 ORDER BY p.last_read_at DESC
		 LIMIT $3`,
		userID, bookID, friendsShown,
	)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.FriendReader
		if err := rows.Scan(&f.UserID, &f.Username, &result.Count); err != nil {
			return result, err
		}
		result.Friends = append(result.Friends, f)
	}
	return result, rows.Err()
}
