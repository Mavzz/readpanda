package notify

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/testdb"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// pushed records what sendPush was asked to deliver.
type pushed struct {
	mu     sync.Mutex
	msgs   map[string]utils.PushMessage // the latest, by device token
	counts map[string]int               // how many, by device token
}

func (p *pushed) count(token string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.counts[token]
}

func (p *pushed) get(token string) (utils.PushMessage, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.msgs[token]
	return m, ok
}

// fakePush replaces sendPush for one test. respond decides each send's
// result; nil means success.
func fakePush(t *testing.T, respond func(ctx context.Context, token string) error) *pushed {
	t.Helper()
	p := &pushed{msgs: map[string]utils.PushMessage{}, counts: map[string]int{}}
	original := sendPush
	sendPush = func(ctx context.Context, token string, msg utils.PushMessage) error {
		p.mu.Lock()
		p.msgs[token] = msg
		p.counts[token]++
		p.mu.Unlock()
		if respond != nil {
			return respond(ctx, token)
		}
		return nil
	}
	t.Cleanup(func() { sendPush = original })
	return p
}

// reader inserts a user with one device and returns their id and token.
func reader(t *testing.T, db *sql.DB, name string) (string, string) {
	t.Helper()
	id := uuid.New().String()
	token := "tok-" + name
	if _, err := db.Exec(`INSERT INTO users (uuid, username, email) VALUES ($1, $2, $3)`, id, name, name+"@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO device_tokens (token, user_id, platform) VALUES ($1, $2, 'ios')`, token, id); err != nil {
		t.Fatal(err)
	}
	return id, token
}

func inboxCount(t *testing.T, db *sql.DB, userID, typ string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM notifications WHERE user_id = $1 AND type = $2`, userID, typ).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSendWritesAndPushesBeforeReturning(t *testing.T) {
	db := testdb.Require(t)
	p := fakePush(t, nil)
	ann, annTok := reader(t, db, "ann")
	if _, err := db.Exec(`INSERT INTO rooms (id, name, admin_id) VALUES ('rm_1', 'AI Learning', $1)`, ann); err != nil {
		t.Fatal(err)
	}

	err := Send([]string{ann, ann}, Notification{
		Type: models.NotificationTypeRoomJoin, Title: "AI Learning", Message: "bob joined the room",
		RoomID: "rm_1", RoomName: "AI Learning",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Synchronous: the push has happened by the time Send returns.
	msg, ok := p.get(annTok)
	if !ok {
		t.Fatal("no push sent before Send returned")
	}
	if msg.Title != "AI Learning" || msg.Body != "bob joined the room" {
		t.Errorf("push text: %+v", msg)
	}
	if msg.Data["type"] != "ROOM_JOIN" || msg.Data["room_id"] != "rm_1" || msg.Data["room_name"] != "AI Learning" || msg.Data["notification_id"] == "" {
		t.Errorf("push data: %+v", msg.Data)
	}
	if _, has := msg.Data["book_id"]; has {
		t.Errorf("book_id sent when there's no book: %+v", msg.Data)
	}

	// One row, even though ann was listed twice, and it names the room.
	var roomID string
	var count int
	if err := db.QueryRow(`SELECT room_id, count(*) OVER () FROM notifications WHERE user_id = $1`, ann).Scan(&roomID, &count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || roomID != "rm_1" {
		t.Errorf("inbox: %d rows, room %q", count, roomID)
	}
}

func TestSendSkipsReadersWhoSwitchedTheTypeOff(t *testing.T) {
	db := testdb.Require(t)
	p := fakePush(t, nil)
	on, onTok := reader(t, db, "on")
	off, offTok := reader(t, db, "off")
	if _, err := db.Exec(`INSERT INTO notification_settings (user_id, replies) VALUES ($1, false)`, off); err != nil {
		t.Fatal(err)
	}

	if err := Send([]string{on, off}, Notification{Type: models.NotificationTypeReply, Title: "x replied", Message: "hi"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.get(onTok); !ok {
		t.Error("reader with replies on got no push")
	}
	if _, ok := p.get(offTok); ok {
		t.Error("reader with replies off was pushed")
	}
	if inboxCount(t, db, off, "REPLY") != 0 {
		t.Error("reader with replies off got an inbox row")
	}

	// Another type still reaches them.
	if err := Send([]string{off}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatal(err)
	}
	if inboxCount(t, db, off, "ROOM_JOIN") != 1 {
		t.Error("switching replies off stopped joins too")
	}
}

func TestSendGivesUpAtTheTimeout(t *testing.T) {
	db := testdb.Require(t)
	original := pushTimeout
	pushTimeout = 100 * time.Millisecond
	t.Cleanup(func() { pushTimeout = original })
	// FCM that never answers: each send waits for the deadline.
	fakePush(t, func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	})
	a, _ := reader(t, db, "a")
	b, _ := reader(t, db, "b")

	start := time.Now()
	if err := Send([]string{a, b}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatalf("a push timeout must not fail Send: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Send took %s; it should stop at the 100ms timeout", elapsed)
	}
	// The inbox rows are there regardless.
	if inboxCount(t, db, a, "ROOM_JOIN") != 1 || inboxCount(t, db, b, "ROOM_JOIN") != 1 {
		t.Error("inbox rows missing after a push timeout")
	}
}

func TestSendRemovesStaleTokens(t *testing.T) {
	db := testdb.Require(t)
	fakePush(t, func(context.Context, string) error { return utils.ErrStaleToken })
	a, tok := reader(t, db, "a")

	if err := Send([]string{a}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM device_tokens WHERE token = $1`, tok).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("stale token was kept")
	}
}

func TestSendWithPushDisabledStillWritesTheInbox(t *testing.T) {
	db := testdb.Require(t)
	fakePush(t, func(context.Context, string) error { return utils.ErrPushDisabled })
	a, _ := reader(t, db, "a")

	if err := Send([]string{a}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatal(err)
	}
	if inboxCount(t, db, a, "ROOM_JOIN") != 1 {
		t.Error("no inbox row with push disabled")
	}
}

func TestBroadcastSkipsReadersWithNewBooksOff(t *testing.T) {
	db := testdb.Require(t)
	p := fakePush(t, nil)
	writer, _ := reader(t, db, "writer")
	on, onTok := reader(t, db, "on")
	off, _ := reader(t, db, "off")
	if _, err := db.Exec(`INSERT INTO notification_settings (user_id, new_books) VALUES ($1, false)`, off); err != nil {
		t.Fatal(err)
	}

	if err := ToAllUsersExcept(writer, Notification{Type: models.NotificationTypeNewBook, Title: "New book added", Message: "Dune is ready to read."}); err != nil {
		t.Fatal(err)
	}
	// The broadcast pushes in the background; wait for it, so the fake isn't
	// swapped back while that goroutine is still using it.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := p.get(onTok); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("broadcast never pushed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for name, want := range map[string]struct {
		id   string
		rows int
	}{"writer": {writer, 0}, "on": {on, 1}, "off": {off, 0}} {
		if got := inboxCount(t, db, want.id, "NEW_BOOK"); got != want.rows {
			t.Errorf("%s: %d NEW_BOOK rows, want %d", name, got, want.rows)
		}
	}
}

// One reader, many devices, each slow to answer: sent one after another the
// budget runs out part-way (QA found 8 of 25 reached); sent side by side,
// every device gets it well inside the timeout.
func TestSendReachesEveryDeviceOfAManyDeviceReader(t *testing.T) {
	db := testdb.Require(t)
	original := pushTimeout
	pushTimeout = 2 * time.Second
	t.Cleanup(func() { pushTimeout = original })
	p := fakePush(t, func(ctx context.Context, _ string) error {
		select {
		case <-time.After(300 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	id, _ := reader(t, db, "many")
	var tokens []string
	for i := 0; i < 25; i++ {
		tok := fmt.Sprintf("tok-many-%02d", i)
		tokens = append(tokens, tok)
		if _, err := db.Exec(`INSERT INTO device_tokens (token, user_id, platform) VALUES ($1, $2, 'ios')`, tok, id); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Now()
	if err := Send([]string{id}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	missed := 0
	for _, tok := range tokens {
		if p.count(tok) != 1 {
			missed++
		}
	}
	if missed > 0 {
		t.Errorf("%d of %d devices missed the push (took %s)", missed, len(tokens), elapsed)
	}
	if elapsed > time.Second {
		t.Errorf("25 devices at 300ms each took %s; side by side it should be about 1s", elapsed)
	}
}

// A device whose sends fail (APNs rejecting the credential, say) doesn't stop
// the reader's other devices getting the push.
func TestAFailingDeviceDoesNotBlockTheOthers(t *testing.T) {
	db := testdb.Require(t)
	p := fakePush(t, func(_ context.Context, token string) error {
		if token == "tok-bad" {
			return errors.New("googleapi: Error 401: Invalid APNs credential")
		}
		return nil
	})
	id, good := reader(t, db, "mixed")
	if _, err := db.Exec(`INSERT INTO device_tokens (token, user_id, platform) VALUES ('tok-bad', $1, 'ios')`, id); err != nil {
		t.Fatal(err)
	}

	if err := Send([]string{id}, Notification{Type: models.NotificationTypeRoomJoin, Title: "r", Message: "m"}); err != nil {
		t.Fatal(err)
	}
	if p.count(good) != 1 {
		t.Error("the working device missed the push")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM device_tokens WHERE token = 'tok-bad'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("a rejected-credential token isn't stale and must be kept (count %d, %v)", n, err)
	}
}
