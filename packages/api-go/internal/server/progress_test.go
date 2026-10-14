package server_test

import (
	"net/http"
	"testing"
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
