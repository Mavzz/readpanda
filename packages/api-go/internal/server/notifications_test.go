package server_test

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/notify"
)

const settingsPath = "/users/me/notification-settings"

func TestNotificationSettingsDefaultToAllOn(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")

	var got notify.Settings
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("a reader who never changed anything: got %+v, want all on", got)
	}
}

func TestNotificationSettingsUpdateASubset(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")
	bob := a.user("bob")

	var got notify.Settings
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"room_comments": false}, &got)
	want := notify.DefaultSettings()
	want.RoomComments = false
	if got != want {
		t.Fatalf("after turning room comments off: got %+v, want %+v", got, want)
	}

	// A second change leaves the first alone.
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"new_books": false}, &got)
	want.NewBooks = false
	if got != want {
		t.Fatalf("after turning new books off too: got %+v, want %+v", got, want)
	}

	// And it's what GET reads back.
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != want {
		t.Fatalf("GET after updates: got %+v, want %+v", got, want)
	}

	// Turning one back on.
	a.must(http.StatusOK, "PUT", settingsPath, &ann, map[string]bool{"room_comments": true}, &got)
	want.RoomComments = true
	if got != want {
		t.Fatalf("after turning room comments back on: got %+v, want %+v", got, want)
	}

	// Someone else's settings are untouched.
	a.must(http.StatusOK, "GET", settingsPath, &bob, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("bob's settings changed: %+v", got)
	}
}

func TestNotificationSettingsRejectsBadBodies(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")

	for name, body := range map[string]any{
		"unknown key":   map[string]bool{"room_comment": false},
		"not a boolean": map[string]string{"replies": "no"},
		"not an object": []bool{false},
	} {
		if code := a.do("PUT", settingsPath, &ann, body, nil); code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, code)
		}
	}

	// Nothing was saved by the rejected requests.
	var got notify.Settings
	a.must(http.StatusOK, "GET", settingsPath, &ann, nil, &got)
	if got != notify.DefaultSettings() {
		t.Fatalf("a rejected request changed settings: %+v", got)
	}
}

func TestNotificationSettingsNeedSignIn(t *testing.T) {
	a := newAPI(t)
	if code := a.do("GET", settingsPath, nil, nil, nil); code != http.StatusUnauthorized {
		t.Errorf("GET without a token: got %d, want 401", code)
	}
	if code := a.do("PUT", settingsPath, nil, map[string]bool{"replies": false}, nil); code != http.StatusUnauthorized {
		t.Errorf("PUT without a token: got %d, want 401", code)
	}
}

func TestNotificationSettingsForADeletedAccount(t *testing.T) {
	a := newAPI(t)
	ann := a.user("ann")
	a.exec(`DELETE FROM users WHERE uuid = $1`, ann.ID)

	// The token is still valid, but there's no account to attach settings to.
	if code := a.do("PUT", settingsPath, &ann, map[string]bool{"replies": false}, nil); code != http.StatusNotFound {
		t.Errorf("PUT for a deleted account: got %d, want 404", code)
	}
}

// inboxRow is one notification as stored, for the delivery tests. Push isn't
// configured under test, so what reaches the inbox is what was sent; the
// pushes themselves are covered in internal/notify.
type inboxRow struct {
	Type, Title, Message string
	Count                int
}

func (a *api) inbox(u user, typ string) []inboxRow {
	a.t.Helper()
	rows, err := database.DB.Query(
		`SELECT type, COALESCE(title, ''), message, event_count FROM notifications
		  WHERE user_id = $1 AND type = $2 ORDER BY id`, u.ID, typ)
	if err != nil {
		a.t.Fatal(err)
	}
	defer rows.Close()
	var out []inboxRow
	for rows.Next() {
		var r inboxRow
		if err := rows.Scan(&r.Type, &r.Title, &r.Message, &r.Count); err != nil {
			a.t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// threeReaders is "Club", created by alice, with bob and cat in it, all well
// into "Dune".
func threeReaders(t *testing.T) (a *api, alice, bob, cat user, r room, book string) {
	a, alice, bob, r, book = commentsSetup(t)
	cat = a.user("cat")
	a.joinRoom(cat, r)
	for _, u := range []user{alice, bob, cat} {
		a.putProgress(u, book, 100, 300)
	}
	return
}

func TestCommentNotifiesTheRestOfTheRoom(t *testing.T) {
	a, alice, bob, cat, r, book := threeReaders(t)

	a.comment(bob, r, book, map[string]any{"page": 5, "body": "Paul is  growing\non me"})

	want := []inboxRow{{"ROOM_COMMENTS", "Club", "bob: Paul is growing on me", 1}}
	for _, u := range []user{alice, cat} {
		if got := a.inbox(u, "ROOM_COMMENTS"); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %+v, want %+v", u.Name, got, want)
		}
	}
	if got := a.inbox(bob, "ROOM_COMMENTS"); len(got) != 0 {
		t.Errorf("the commenter was notified: %+v", got)
	}
}

func TestReplyNotifiesTheThreadStarterOnce(t *testing.T) {
	a, alice, bob, cat, r, book := threeReaders(t)
	root := a.comment(bob, r, book, map[string]any{"page": 5, "body": "Paul is growing on me"})

	a.comment(cat, r, book, map[string]any{"page": 5, "parent_id": root.ID, "body": "Same, after the desert"})

	// bob hears it as a reply, and only as a reply.
	if got := a.inbox(bob, "REPLY"); !reflect.DeepEqual(got, []inboxRow{{"REPLY", "cat replied", "Same, after the desert", 1}}) {
		t.Errorf("bob's replies: %+v", got)
	}
	if got := a.inbox(bob, "ROOM_COMMENTS"); len(got) != 0 {
		t.Errorf("bob also got a room notification: %+v", got)
	}
	// alice's unread row from bob's comment now counts both.
	if got := a.inbox(alice, "ROOM_COMMENTS"); len(got) != 1 || got[0].Message != "2 new comments" || got[0].Count != 2 {
		t.Errorf("alice: %+v", got)
	}
	if got := a.inbox(cat, "REPLY"); len(got) != 0 {
		t.Errorf("the replier got a reply notification: %+v", got)
	}
}

func TestReplyToYourOwnCommentIsNotAReply(t *testing.T) {
	a, alice, bob, _, r, book := threeReaders(t)
	root := a.comment(bob, r, book, map[string]any{"page": 5, "body": "first"})

	a.comment(bob, r, book, map[string]any{"page": 5, "parent_id": root.ID, "body": "and another thing"})

	if got := a.inbox(bob, "REPLY"); len(got) != 0 {
		t.Errorf("self-reply notified: %+v", got)
	}
	if got := a.inbox(alice, "ROOM_COMMENTS"); len(got) != 1 || got[0].Count != 2 {
		t.Errorf("the room should still hear about it: %+v", got)
	}
}

func TestReplyWithRepliesSwitchedOffCountsAsARoomComment(t *testing.T) {
	a, _, bob, cat, r, book := threeReaders(t)
	root := a.comment(bob, r, book, map[string]any{"page": 5, "body": "first"})
	a.must(http.StatusOK, "PUT", settingsPath, &bob, map[string]bool{"replies": false}, nil)

	a.comment(cat, r, book, map[string]any{"page": 5, "parent_id": root.ID, "body": "reply"})

	if got := a.inbox(bob, "REPLY"); len(got) != 0 {
		t.Errorf("reply sent with replies off: %+v", got)
	}
	if got := a.inbox(bob, "ROOM_COMMENTS"); len(got) != 1 || got[0].Message != "cat: reply" {
		t.Errorf("bob should get it as a room comment instead: %+v", got)
	}
}

func TestReplyAfterTheThreadStarterLeft(t *testing.T) {
	a, _, bob, cat, r, book := threeReaders(t)
	root := a.comment(bob, r, book, map[string]any{"page": 5, "body": "first"})
	a.must(http.StatusNoContent, "DELETE", "/room/"+r.ID+"/members/me", &bob, nil, nil)

	a.comment(cat, r, book, map[string]any{"page": 5, "parent_id": root.ID, "body": "reply"})

	if got := a.inbox(bob, "REPLY"); len(got) != 0 {
		t.Errorf("a reader who left got a reply notification: %+v", got)
	}
	if got := a.inbox(bob, "ROOM_COMMENTS"); len(got) != 0 {
		t.Errorf("a reader who left got a room notification: %+v", got)
	}
}

func TestReplyIsLockedAtItsRootsPage(t *testing.T) {
	a, alice, bob, _, r, book := threeReaders(t)
	root := a.comment(bob, r, book, map[string]any{"page": 150, "body": "the ending!"})
	dan := a.user("dan")
	a.joinRoom(dan, r)
	a.putProgress(dan, book, 10, 300)

	// The client says page 0; the server puts a reply on its root's page.
	a.comment(alice, r, book, map[string]any{"page": 0, "parent_id": root.ID, "body": "no way"})

	if got := a.inbox(dan, "ROOM_COMMENTS"); len(got) != 1 || got[0].Message != "alice commented on p. 151" {
		t.Errorf("dan, 140 pages behind: %+v", got)
	}
}

func TestRetriedCommentDoesNotNotifyAgain(t *testing.T) {
	a, alice, bob, _, r, book := threeReaders(t)
	body := map[string]any{"page": 5, "body": "sent twice", "client_id": "c-123"}

	first := a.comment(bob, r, book, body)
	second := a.comment(bob, r, book, body)

	if first.ID != second.ID {
		t.Fatalf("the retry stored a second comment: %s, %s", first.ID, second.ID)
	}
	if got := a.inbox(alice, "ROOM_COMMENTS"); len(got) != 1 || got[0].Count != 1 {
		t.Errorf("a retried POST notified again: %+v", got)
	}
}

func TestJoinNotifiesOnlyTheCreator(t *testing.T) {
	a := newAPI(t)
	alice, bob, cat := a.user("alice"), a.user("bob"), a.user("cat")
	r := a.newRoom(alice, "Club")

	a.joinRoom(bob, r)
	a.joinRoom(cat, r)

	want := []inboxRow{
		{"ROOM_JOIN", "Club", "bob joined the room", 1},
		{"ROOM_JOIN", "Club", "cat joined the room", 1},
	}
	if got := a.inbox(alice, "ROOM_JOIN"); !reflect.DeepEqual(got, want) {
		t.Errorf("creator: %+v, want %+v", got, want)
	}
	for _, u := range []user{bob, cat} {
		if got := a.inbox(u, "ROOM_JOIN"); len(got) != 0 {
			t.Errorf("member %s was told about a join: %+v", u.Name, got)
		}
	}
}

func TestFailedJoinsDoNotNotify(t *testing.T) {
	a := newAPI(t)
	alice, bob := a.user("alice"), a.user("bob")
	r := a.newRoom(alice, "Club")
	a.joinRoom(bob, r)

	a.must(http.StatusConflict, "POST", "/room/join", &bob, map[string]string{"invite_code": r.InviteCode}, nil)
	a.must(http.StatusNotFound, "POST", "/room/join", &bob, map[string]string{"invite_code": "ZZZZZZ"}, nil)
	// The creator "joining" their own room is the same 409.
	a.must(http.StatusConflict, "POST", "/room/join", &alice, map[string]string{"invite_code": r.InviteCode}, nil)

	if got := a.inbox(alice, "ROOM_JOIN"); len(got) != 1 {
		t.Errorf("creator got %d join notifications, want only bob's first join: %+v", len(got), got)
	}
}

func TestJoinWithNewMembersSwitchedOff(t *testing.T) {
	a := newAPI(t)
	alice, bob := a.user("alice"), a.user("bob")
	r := a.newRoom(alice, "Club")
	a.must(http.StatusOK, "PUT", settingsPath, &alice, map[string]bool{"room_joins": false}, nil)

	a.joinRoom(bob, r)

	if got := a.inbox(alice, "ROOM_JOIN"); len(got) != 0 {
		t.Errorf("join notified with New members off: %+v", got)
	}
}

func TestRacingJoinsNotifyOnce(t *testing.T) {
	a := newAPI(t)
	alice, bob := a.user("alice"), a.user("bob")
	r := a.newRoom(alice, "Club")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.do("POST", "/room/join", &bob, map[string]string{"invite_code": r.InviteCode}, nil)
		}()
	}
	wg.Wait()

	if got := a.inbox(alice, "ROOM_JOIN"); len(got) != 1 {
		t.Errorf("five simultaneous joins notified %d times, want 1", len(got))
	}
}

func (a *api) setReading(as user, r room, bookID *string) int {
	a.t.Helper()
	return a.do("PATCH", "/room/"+r.ID+"/reading", &as, map[string]*string{"current_book_id": bookID}, nil)
}

func TestRoomPickingItsBookNotifiesMembers(t *testing.T) {
	a := newAPI(t)
	alice, bob, cat := a.user("alice"), a.user("bob"), a.user("cat")
	dune, foundation := a.book("Dune"), a.book("Foundation")
	r := a.newRoom(alice, "Club")
	a.joinRoom(bob, r)
	a.joinRoom(cat, r)

	var detail struct {
		ID string `json:"id"`
	}
	a.must(http.StatusOK, "PATCH", "/room/"+r.ID+"/reading", &alice, map[string]string{"current_book_id": dune}, &detail)
	if detail.ID != r.ID {
		t.Errorf("response is no longer the room detail: %+v", detail)
	}

	a.setReading(alice, r, &dune)       // the same book again
	a.setReading(alice, r, &foundation) // a change
	a.setReading(alice, r, nil)         // cleared
	a.setReading(alice, r, &dune)       // picked again after clearing

	want := []inboxRow{
		{"ROOM_BOOK", "Club", "Now reading Dune", 1},
		{"ROOM_BOOK", "Club", "Now reading Foundation", 1},
		{"ROOM_BOOK", "Club", "Now reading Dune", 1},
	}
	for _, u := range []user{bob, cat} {
		if got := a.inbox(u, "ROOM_BOOK"); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %+v, want %+v", u.Name, got, want)
		}
	}
	if got := a.inbox(alice, "ROOM_BOOK"); len(got) != 0 {
		t.Errorf("the creator was told about their own pick: %+v", got)
	}
}

func TestRoomBookNotChangedDoesNotNotify(t *testing.T) {
	a := newAPI(t)
	alice, bob := a.user("alice"), a.user("bob")
	dune := a.book("Dune")
	r := a.newRoom(alice, "Club")
	a.joinRoom(bob, r)

	// Only the creator can change it.
	if code := a.setReading(bob, r, &dune); code != http.StatusForbidden {
		t.Fatalf("member changing the book: %d, want 403", code)
	}
	// With Room picks a book off, nothing.
	a.must(http.StatusOK, "PUT", settingsPath, &bob, map[string]bool{"room_book": false}, nil)
	a.setReading(alice, r, &dune)

	if got := a.inbox(bob, "ROOM_BOOK"); len(got) != 0 {
		t.Errorf("bob: %+v", got)
	}
}

func TestInboxReturnsRoomFieldsAndSortsOnUpdates(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(alice, book, 100, 300)

	a.comment(bob, r, book, map[string]any{"page": 5, "body": "first"}) // alice: room row
	cat := a.user("cat")
	a.joinRoom(cat, r)                                                   // alice: join row, newer
	a.comment(bob, r, book, map[string]any{"page": 6, "body": "second"}) // folds into the room row

	var inbox []struct {
		Type      string    `json:"type"`
		Message   string    `json:"message"`
		RoomID    *string   `json:"room_id"`
		BookID    *string   `json:"book_id"`
		Count     int       `json:"count"`
		IsRead    bool      `json:"is_read"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	a.must(http.StatusOK, "GET", "/notifications", &alice, nil, &inbox)

	// alice also has a join row from bob joining in commentsSetup.
	if len(inbox) != 3 {
		t.Fatalf("inbox: %+v", inbox)
	}
	top := inbox[0]
	if top.Type != "ROOM_COMMENTS" || top.Count != 2 || top.Message != "2 new comments" {
		t.Errorf("the updated room row should be first: %+v", inbox)
	}
	if top.RoomID == nil || *top.RoomID != r.ID || top.BookID == nil || *top.BookID != book {
		t.Errorf("room row ids: room %v, book %v", top.RoomID, top.BookID)
	}
	if !top.UpdatedAt.After(top.CreatedAt) {
		t.Errorf("updated_at %s should be after created_at %s", top.UpdatedAt, top.CreatedAt)
	}
	if inbox[1].Type != "ROOM_JOIN" || inbox[1].Count != 1 || inbox[1].RoomID == nil {
		t.Errorf("join row: %+v", inbox[1])
	}
}

// lastPushed is when u's unread room-comments row was last pushed (or
// would have been: push isn't configured under test, but the row records the
// decision either way).
func (a *api) lastPushed(u user) time.Time {
	a.t.Helper()
	var at time.Time
	if err := database.DB.QueryRow(
		`SELECT last_pushed_at FROM notifications
		  WHERE user_id = $1 AND type = 'ROOM_COMMENTS' AND is_read IS NOT TRUE`, u.ID).Scan(&at); err != nil {
		a.t.Fatal(err)
	}
	return at
}

func TestCommentsUseTheConfiguredWindow(t *testing.T) {
	for name, tc := range map[string]struct {
		window      time.Duration
		pushesTwice bool
	}{
		"30 minutes":   {30 * time.Minute, false},
		"1 nanosecond": {time.Nanosecond, true},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := *testConfig
			cfg.RoomCommentWindow = tc.window
			a := newAPIWith(t, &cfg)
			alice, bob := a.user("alice"), a.user("bob")
			book := a.book("Dune")
			r := a.newRoom(alice, "Club")
			a.joinRoom(bob, r)

			a.comment(bob, r, book, map[string]any{"page": 0, "body": "one"})
			first := a.lastPushed(alice)
			a.comment(bob, r, book, map[string]any{"page": 0, "body": "two"})
			second := a.lastPushed(alice)

			if pushed := second.After(first); pushed != tc.pushesTwice {
				t.Errorf("second comment pushed = %v, want %v (window %s)", pushed, tc.pushesTwice, tc.window)
			}
		})
	}
}

func TestReadingTheRoomRowStartsANewOne(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	a.putProgress(alice, book, 100, 300)
	a.comment(bob, r, book, map[string]any{"page": 5, "body": "one"})
	a.comment(bob, r, book, map[string]any{"page": 6, "body": "two"})

	var inbox []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	}
	a.must(http.StatusOK, "GET", "/notifications", &alice, nil, &inbox)
	for _, n := range inbox {
		if n.Type == "ROOM_COMMENTS" {
			a.must(http.StatusNoContent, "PUT", fmt.Sprintf("/notifications/%d/read", n.ID), &alice, nil, nil)
		}
	}
	a.comment(bob, r, book, map[string]any{"page": 7, "body": "three"})

	want := []inboxRow{
		{"ROOM_COMMENTS", "Club", "2 new comments", 2},
		{"ROOM_COMMENTS", "Club", "bob: three", 1},
	}
	if got := a.inbox(alice, "ROOM_COMMENTS"); !reflect.DeepEqual(got, want) {
		t.Errorf("%+v, want %+v", got, want)
	}
}

func TestDeletingARoomDeletesItsNotifications(t *testing.T) {
	a, alice, bob, r, book := commentsSetup(t)
	other := a.newRoom(alice, "Other")
	a.joinRoom(bob, other)
	a.comment(alice, r, book, map[string]any{"page": 0, "body": "in Club"})
	a.comment(alice, other, book, map[string]any{"page": 0, "body": "in Other"})

	a.must(http.StatusNoContent, "DELETE", "/room/"+r.ID, &alice, nil, nil)

	var rooms []string
	rows, err := database.DB.Query(`SELECT room_id FROM notifications WHERE user_id = $1 AND room_id IS NOT NULL`, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		rooms = append(rooms, id)
	}
	if len(rooms) != 1 || rooms[0] != other.ID {
		t.Errorf("bob's notifications after Club was deleted are about %v, want only %s", rooms, other.ID)
	}
}
