package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (a *api) bucketBookIDs(as user, bucketID string) []string {
	a.t.Helper()
	var page struct {
		Books []struct {
			BookID string `json:"book_id"`
		} `json:"books"`
	}
	a.must(http.StatusOK, "GET", "/users/me/buckets/"+bucketID+"/books", &as, nil, &page)
	ids := make([]string, len(page.Books))
	for i, b := range page.Books {
		ids[i] = b.BookID
	}
	return ids
}

// coming-soon R4.2: the books land after the ones already there, and one
// that's already in the bucket isn't added twice.
func TestAddBooksToBucketAppendsInOrder(t *testing.T) {
	a := newAPI(t)
	me := a.user("ada")
	one, two, three := a.book("One"), a.book("Two"), a.book("Three")

	var created struct {
		Bucket struct {
			ID string `json:"id"`
		} `json:"bucket"`
	}
	a.must(http.StatusCreated, "POST", "/users/me/buckets", &me,
		map[string]any{"Name": "Sea", "book_ids": []string{one}}, &created)
	id := created.Bucket.ID

	var added struct {
		BookCount int `json:"book_count"`
	}
	a.must(http.StatusOK, "POST", "/users/me/buckets/"+id+"/books", &me,
		map[string]any{"book_ids": []string{three, two, one}}, &added)

	if added.BookCount != 3 {
		t.Fatalf("book_count = %d, want 3", added.BookCount)
	}
	want := []string{one, three, two}
	if got := a.bucketBookIDs(me, id); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("bucket books = %v, want %v", got, want)
	}
}

// A book that can't be added is a failure, not a quiet 200 (R4.4).
func TestAddBooksToBucketReportsAFailedInsert(t *testing.T) {
	a := newAPI(t)
	me := a.user("ada")
	a.exec(`INSERT INTO user_buckets (id, user_id, name) VALUES ('b1', $1, 'Mine')`, me.ID)

	a.must(http.StatusInternalServerError, "POST", "/users/me/buckets/b1/books", &me,
		map[string]any{"book_ids": []string{"no-such-book"}}, nil)
}
