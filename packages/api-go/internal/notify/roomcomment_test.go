package notify

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/testdb"
)

// room is a room reading one book, with its members' devices and progress.
type room struct {
	db     *sql.DB
	id     string
	bookID string
	ids    map[string]string // member name → user id
	toks   map[string]string // member name → device token
}

// newRoom creates "AI Learning" reading bk_1, with ann as creator and each of
// members at the furthest page given (-1: no progress row).
func newRoom(t *testing.T, members map[string]int) *room {
	t.Helper()
	db := testdb.Require(t)
	r := &room{db: db, id: "rm_1", bookID: "bk_1", ids: map[string]string{}, toks: map[string]string{}}
	if _, err := db.Exec(`INSERT INTO books (book_id, title, status) VALUES ('bk_1', 'Data Mining', 1)`); err != nil {
		t.Fatal(err)
	}
	r.ids["ann"], r.toks["ann"] = reader(t, db, "ann")
	if _, err := db.Exec(`INSERT INTO rooms (id, name, admin_id) VALUES ('rm_1', 'AI Learning', $1)`, r.ids["ann"]); err != nil {
		t.Fatal(err)
	}
	members["ann"] = 0
	for name, furthest := range members {
		if name != "ann" {
			r.ids[name], r.toks[name] = reader(t, db, name)
		}
		if _, err := db.Exec(`INSERT INTO room_members (room_id, user_id) VALUES ('rm_1', $1)`, r.ids[name]); err != nil {
			t.Fatal(err)
		}
		if furthest >= 0 {
			if _, err := db.Exec(`INSERT INTO reading_progress (user_id, book_id, current_page, total_pages, furthest_page)
			                      VALUES ($1, 'bk_1', $2, 200, $2)`, r.ids[name], furthest); err != nil {
				t.Fatal(err)
			}
		}
	}
	return r
}

// comment is a comment by ann.
func (r *room) comment(t *testing.T, page int, body string, exclude ...string) {
	t.Helper()
	var ids []string
	for _, name := range exclude {
		ids = append(ids, r.ids[name])
	}
	err := RoomComment(RoomCommentEvent{
		RoomID: r.id, RoomName: "AI Learning", BookID: r.bookID,
		AuthorID: r.ids["ann"], AuthorName: "ann",
		Page: page, Body: body, Exclude: ids, Window: 30 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// inbox is name's room-comments rows, newest first.
func (r *room) inbox(t *testing.T, name string) []string {
	t.Helper()
	rows, err := r.db.Query(`SELECT message || ' (' || event_count || ')' FROM notifications
	                          WHERE user_id = $1 AND type = 'ROOM_COMMENTS' ORDER BY id DESC`, r.ids[name])
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

func TestRoomCommentTextFollowsEachReadersProgress(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"ahead": 50, "atPage": 10, "behind": 5, "never": -1})

	r.comment(t, 10, "Clustering gets wild here")

	for name, want := range map[string]string{
		"ahead":  "ann: Clustering gets wild here",
		"atPage": "ann: Clustering gets wild here", // page <= furthest, like the comment layer
		"behind": "ann commented on p. 11",
		"never":  "ann commented on p. 11",
	} {
		msg, ok := p.get(r.toks[name])
		if !ok {
			t.Errorf("%s: no push", name)
			continue
		}
		if msg.Body != want || msg.Title != "AI Learning" {
			t.Errorf("%s: pushed %q / %q, want AI Learning / %q", name, msg.Title, msg.Body, want)
		}
		if msg.Data["type"] != "ROOM_COMMENTS" || msg.Data["room_id"] != "rm_1" || msg.Data["book_id"] != "bk_1" || msg.Data["room_name"] != "AI Learning" {
			t.Errorf("%s: data %+v", name, msg.Data)
		}
		if got := r.inbox(t, name); len(got) != 1 || got[0] != want+" (1)" {
			t.Errorf("%s: inbox %q", name, got)
		}
	}
	if p.count(r.toks["ann"]) != 0 || len(r.inbox(t, "ann")) != 0 {
		t.Error("the commenter was notified")
	}
}

func TestRoomCommentPageZeroIsUnlockedForEveryone(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"never": -1})
	r.comment(t, 0, "First page thoughts")
	if msg, _ := p.get(r.toks["never"]); msg.Body != "ann: First page thoughts" {
		t.Errorf("a page-0 comment for someone who hasn't opened the book: %q", msg.Body)
	}
}

func TestRoomCommentSkipsExcludedAndSwitchedOff(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"replied": 50, "muted": 50, "other": 50})
	if _, err := r.db.Exec(`INSERT INTO notification_settings (user_id, room_comments) VALUES ($1, false)`, r.ids["muted"]); err != nil {
		t.Fatal(err)
	}

	r.comment(t, 3, "hi", "replied")

	for _, name := range []string{"replied", "muted"} {
		if p.count(r.toks[name]) != 0 || len(r.inbox(t, name)) != 0 {
			t.Errorf("%s was notified", name)
		}
	}
	if p.count(r.toks["other"]) != 1 {
		t.Error("other member wasn't pushed")
	}
}

func TestRoomCommentsBundleAndThrottle(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"bob": 50})
	tok := r.toks["bob"]

	r.comment(t, 3, "one")
	r.comment(t, 4, "two")
	r.comment(t, 99, "three, and past bob's page")

	// One row, counting up, with no comment text; only the first pushed.
	if got := r.inbox(t, "bob"); len(got) != 1 || got[0] != "3 new comments (3)" {
		t.Fatalf("inbox after three comments: %q", got)
	}
	if p.count(tok) != 1 {
		t.Fatalf("pushed %d times inside the window, want 1", p.count(tok))
	}

	// Once the window has passed, the next comment pushes the row as it stands.
	if _, err := r.db.Exec(`UPDATE notifications SET last_pushed_at = last_pushed_at - interval '31 minutes'
	                        WHERE user_id = $1`, r.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	r.comment(t, 5, "four")
	if p.count(tok) != 2 {
		t.Fatalf("pushed %d times after the window, want 2", p.count(tok))
	}
	if msg, _ := p.get(tok); msg.Body != "4 new comments" {
		t.Errorf("bundled push: %q", msg.Body)
	}
	// And the window starts again from that push.
	r.comment(t, 6, "five")
	if p.count(tok) != 2 {
		t.Errorf("pushed again straight after a bundled push")
	}

	// Reading the row ends the bundle: the next comment is a new row, pushed
	// at once, with its text.
	if _, err := r.db.Exec(`UPDATE notifications SET is_read = true WHERE user_id = $1`, r.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	r.comment(t, 7, "six")
	if got := r.inbox(t, "bob"); len(got) != 2 || got[0] != "ann: six (1)" {
		t.Errorf("inbox after reading: %q", got)
	}
	if p.count(tok) != 3 {
		t.Errorf("a comment after reading pushed %d times in total, want 3", p.count(tok))
	}
}

func TestRoomCommentWindowComesFromTheCaller(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"bob": 50})
	for i := 0; i < 3; i++ {
		if err := RoomComment(RoomCommentEvent{
			RoomID: r.id, RoomName: "AI Learning", BookID: r.bookID,
			AuthorID: r.ids["ann"], AuthorName: "ann", Page: 1, Body: "x",
			Window: time.Nanosecond, // already passed by the next comment
		}); err != nil {
			t.Fatal(err)
		}
	}
	if p.count(r.toks["bob"]) != 3 {
		t.Errorf("with a 1ns window, pushed %d of 3", p.count(r.toks["bob"]))
	}
}

func TestRoomCommentsArrivingTogetherMakeOneRowAndOnePush(t *testing.T) {
	p := fakePush(t, nil)
	r := newRoom(t, map[string]int{"bob": 50})

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := RoomComment(RoomCommentEvent{
				RoomID: r.id, RoomName: "AI Learning", BookID: r.bookID,
				AuthorID: r.ids["ann"], AuthorName: "ann", Page: 1, Body: "x", Window: 30 * time.Minute,
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if got := r.inbox(t, "bob"); len(got) != 1 || got[0] != "5 new comments (5)" {
		t.Errorf("inbox: %q", got)
	}
	if p.count(r.toks["bob"]) != 1 {
		t.Errorf("pushed %d times, want 1", p.count(r.toks["bob"]))
	}
}

func TestSnippet(t *testing.T) {
	long := strings.Repeat("ab ", 60) // 180 characters
	got := snippet(long)
	if !strings.HasSuffix(got, "…") || len([]rune(got)) > snippetLength+1 {
		t.Errorf("long comment: %q (%d runes)", got, len([]rune(got)))
	}
	if got := snippet("  two\n\nlines  "); got != "two lines" {
		t.Errorf("whitespace: %q", got)
	}
	// Cut by character, not byte, so a multi-byte letter isn't split.
	if got := snippet(strings.Repeat("é", 130)); got != strings.Repeat("é", 120)+"…" {
		t.Errorf("multi-byte: %q", got)
	}
	if got := snippet("short"); got != "short" {
		t.Errorf("short: %q", got)
	}
}
