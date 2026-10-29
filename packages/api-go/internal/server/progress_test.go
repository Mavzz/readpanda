package server_test

import (
	"net/http"
	"testing"
	"time"
)

type progress struct {
	CurrentPage   int `json:"current_page"`
	TotalPages    int `json:"total_pages"`
	FurthestPage  int `json:"furthest_page"`
	ProgressPct   int `json:"progress_pct"`
	NewlyUnlocked int `json:"newly_unlocked"`
}

func (a *api) putProgress(as user, bookID string, current, total int) progress {
	a.t.Helper()
	var p progress
	a.must(http.StatusOK, "PUT", "/progress/"+bookID, &as,
		map[string]int{"current_page": current, "total_pages": total}, &p)
	return p
}

func TestPutProgress(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	book := a.book("Dune")

	p := a.putProgress(u, book, 40, 200)
	if p.CurrentPage != 40 || p.FurthestPage != 40 || p.ProgressPct != 21 {
		t.Errorf("first save = %+v", p)
	}

	// Flipping back to re-read must not pull the spoiler line back with it.
	p = a.putProgress(u, book, 10, 200)
	if p.CurrentPage != 10 || p.FurthestPage != 40 {
		t.Errorf("after going back: current=%d furthest=%d, want 10 and 40", p.CurrentPage, p.FurthestPage)
	}

	// Leaving the reader before the PDF reports its length sends 0 pages.
	p = a.putProgress(u, book, 12, 0)
	if p.TotalPages != 200 {
		t.Errorf("total_pages = %d, a 0 must not erase the known length", p.TotalPages)
	}

	var mine []struct {
		BookID string `json:"book_id"`
	}
	a.must(http.StatusOK, "GET", "/progress", &u, nil, &mine)
	if len(mine) != 1 || mine[0].BookID != book {
		t.Errorf("GET /progress = %+v", mine)
	}
}

// A phone that was offline syncs a day late, after the iPad moved on. Its
// older page must not replace the newer one, but the pages it really read
// still count toward the spoiler line.
func TestPutProgressLateSave(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	book := a.book("Dune")

	put := func(current, total int, readAt string) progress {
		t.Helper()
		var p progress
		a.must(http.StatusOK, "PUT", "/progress/"+book, &u,
			map[string]any{"current_page": current, "total_pages": total, "read_at": readAt}, &p)
		return p
	}
	monday := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	tuesday := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)

	put(30, 200, monday)
	put(90, 200, tuesday)

	// The phone's Monday save, arriving late: the page stays at 90.
	p := put(40, 300, monday)
	if p.CurrentPage != 90 || p.TotalPages != 200 {
		t.Errorf("late save: current=%d total=%d, want the stored 90 and 200", p.CurrentPage, p.TotalPages)
	}

	// A late save that read further than anyone still moves the spoiler line.
	p = put(120, 200, monday)
	if p.CurrentPage != 90 || p.FurthestPage != 120 {
		t.Errorf("late save past the line: current=%d furthest=%d, want 90 and 120", p.CurrentPage, p.FurthestPage)
	}

	// Older apps send no read_at: read now, so it wins as it always has.
	p = a.putProgress(u, book, 50, 200)
	if p.CurrentPage != 50 {
		t.Errorf("save without read_at: current=%d, want 50", p.CurrentPage)
	}

	// A clock set ahead counts as now, so the next real save can still win.
	put(60, 200, time.Now().UTC().Add(240*time.Hour).Format(time.RFC3339))
	p = a.putProgress(u, book, 70, 200)
	if p.CurrentPage != 70 {
		t.Errorf("save after a future read_at: current=%d, want 70", p.CurrentPage)
	}
}

func TestPutProgressRejects(t *testing.T) {
	a := newAPI(t)
	u := a.user("reader")
	book := a.book("Dune")

	a.must(http.StatusBadRequest, "PUT", "/progress/"+book, &u, map[string]int{"current_page": -1, "total_pages": 10}, nil)
	a.must(http.StatusNotFound, "PUT", "/progress/bk_missing", &u, map[string]int{"current_page": 1, "total_pages": 10}, nil)
}

func TestRoomProgress(t *testing.T) {
	a := newAPI(t)
	owner, reader, outsider := a.user("owner"), a.user("reader"), a.user("outsider")
	book := a.book("Dune")
	r := a.newRoom(owner, "Club")
	a.joinRoom(reader, r)

	type track struct {
		BookID  string `json:"book_id"`
		Members []struct {
			Username    string `json:"username"`
			CurrentPage int    `json:"current_page"`
		} `json:"members"`
	}

	// Nothing chosen yet: an empty track, not an error.
	var empty track
	a.must(http.StatusOK, "GET", "/room/"+r.ID+"/progress", &owner, nil, &empty)
	if empty.BookID != "" || len(empty.Members) != 0 {
		t.Errorf("room with no book = %+v", empty)
	}

	a.exec(`UPDATE rooms SET current_book_id = $1 WHERE id = $2`, book, r.ID)
	a.putProgress(reader, book, 30, 100)

	var got track
	a.must(http.StatusOK, "GET", "/room/"+r.ID+"/progress", &owner, nil, &got)
	if len(got.Members) != 2 {
		t.Fatalf("members = %+v, want both, including the owner who hasn't opened it", got.Members)
	}
	// Admin first, at page 0; then the reader where they are.
	if got.Members[0].Username != "owner" || got.Members[0].CurrentPage != 0 ||
		got.Members[1].Username != "reader" || got.Members[1].CurrentPage != 30 {
		t.Errorf("members = %+v", got.Members)
	}

	a.must(http.StatusForbidden, "GET", "/room/"+r.ID+"/progress", &outsider, nil, nil)
}
