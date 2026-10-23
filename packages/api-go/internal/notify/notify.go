// Package notify writes notifications to users' inboxes and pushes them to
// their devices. The inbox row is the record; the push is a best-effort nudge
// to go and look at it, so a failed or unconfigured push never loses one.
// A reader who has switched a kind of notification off gets neither.
package notify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// Notification is what gets written — once per recipient.
type Notification struct {
	Type    string
	Title   string
	Message string
	BookID  string // optional
	RoomID  string // optional
	// Only carried in the push, so a tap can open the room before the app has
	// loaded its rooms. The inbox row has the room id.
	RoomName string
	// Whose action this is (the replier, the joiner), if anyone's. Recipients
	// who have blocked them aren't told (report-block R5.5).
	ActorID string
}

// broadcastTimeout bounds the NEW_BOOK fan-out, which runs after the upload
// has answered; it only stops a stuck FCM call from leaking forever.
const broadcastTimeout = 2 * time.Minute

// pushTimeout bounds the pushes Send waits for. Cloud Run throttles CPU once
// a response is sent, so room events push before their request answers; this
// keeps a slow FCM from holding that request up for long. A var so tests can
// shorten it.
var pushTimeout = 5 * time.Second

// pushWorkers is how many sends to one notification's devices are in flight
// at once.
const pushWorkers = 10

// sendPush is utils.SendPush, swapped out in tests.
var sendPush = utils.SendPush

// Send stores n for each of userIDs who hasn't switched n.Type off, then
// pushes it to their devices before returning, giving up after pushTimeout.
// An error means the inbox rows weren't written; push failures are only
// logged.
func Send(userIDs []string, n Notification) error {
	if len(userIDs) == 0 {
		return nil
	}
	notBlocking := ""
	if n.ActorID != "" {
		if _, err := uuid.Parse(n.ActorID); err != nil {
			return fmt.Errorf("actor %q: %w", n.ActorID, err)
		}
		notBlocking = ` AND NOT EXISTS (SELECT 1 FROM user_blocks b WHERE b.blocker_id = u AND b.blocked_id = '` + n.ActorID + `')`
	}
	byUser, err := insert(
		`INSERT INTO notifications (user_id, type, title, message, book_id, room_id)
		 SELECT DISTINCT u, $2, $3, $4, $5, $6 FROM unnest($1::uuid[]) AS u
		 WHERE `+switchedOn(typeOrSystem(n.Type), "u")+notBlocking+`
		 RETURNING id, user_id`,
		pq.Array(userIDs), n,
	)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()
	pushToDevices(ctx, byUser, n)
	return nil
}

// ToAllUsersExcept is the broadcast form — a new book is news to everyone but
// the person who published it. The fan-out stays in SQL so a large user base
// is one statement, not one query per user. Returns once the rows are
// written; pushes continue in the background.
func ToAllUsersExcept(excludeUserID string, n Notification) error {
	byUser, err := insert(
		`INSERT INTO notifications (user_id, type, title, message, book_id, room_id)
		 SELECT uuid, $2, $3, $4, $5, $6 FROM users
		 WHERE uuid <> $1::uuid AND `+switchedOn(typeOrSystem(n.Type), "users.uuid")+`
		 RETURNING id, user_id`,
		excludeUserID, n,
	)
	if err != nil {
		return err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
		defer cancel()
		pushToDevices(ctx, byUser, n)
	}()
	return nil
}

func typeOrSystem(t string) string {
	if t == "" {
		return models.NotificationTypeSystem
	}
	return t
}

// delivery is one recipient's push: the inbox row it names (so a tapped push
// can mark it read) and the text, which can differ per reader — a room
// comment is locked for some and not others.
type delivery struct {
	notificationID int
	title, message string
}

// insert runs one of the INSERT … RETURNING id, user_id statements above and
// returns a delivery per user, all with n's text.
func insert(query string, recipients interface{}, n Notification) (map[string]delivery, error) {
	rows, err := database.DB.Query(query, recipients, typeOrSystem(n.Type), n.Title, n.Message,
		nullIfEmpty(n.BookID), nullIfEmpty(n.RoomID))
	if err != nil {
		return nil, fmt.Errorf("storing notifications: %w", err)
	}
	defer rows.Close()
	byUser := map[string]delivery{}
	for rows.Next() {
		var id int
		var userID string
		if err := rows.Scan(&id, &userID); err != nil {
			return nil, fmt.Errorf("reading stored notifications: %w", err)
		}
		byUser[userID] = delivery{notificationID: id, title: n.Title, message: n.Message}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading stored notifications: %w", err)
	}
	return byUser, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// pushToDevices sends each user in byUser their delivery, to every device they
// have, and returns when they've all been tried or ctx is done. n supplies
// the type and the ids the app opens.
func pushToDevices(ctx context.Context, byUser map[string]delivery, n Notification) {
	if len(byUser) == 0 {
		return
	}
	userIDs := make([]string, 0, len(byUser))
	for id := range byUser {
		userIDs = append(userIDs, id)
	}

	rows, err := database.DB.QueryContext(ctx,
		`SELECT token, user_id FROM device_tokens WHERE user_id = ANY($1::uuid[])`,
		pq.Array(userIDs),
	)
	if err != nil {
		log.Printf("notify: loading device tokens: %v", err)
		return
	}
	type device struct{ token, userID string }
	var devices []device
	for rows.Next() {
		var d device
		if err := rows.Scan(&d.token, &d.userID); err != nil {
			log.Printf("notify: reading device token: %v", err)
			continue
		}
		devices = append(devices, d)
	}
	rows.Close()

	typ := typeOrSystem(n.Type)
	var (
		mu                  sync.Mutex
		sent, stale, failed int
		disabled            bool
	)
	send := func(d device) {
		data := map[string]string{
			"type":            typ,
			"notification_id": strconv.Itoa(byUser[d.userID].notificationID),
		}
		for key, value := range map[string]string{"book_id": n.BookID, "room_id": n.RoomID, "room_name": n.RoomName} {
			if value != "" {
				data[key] = value
			}
		}

		msg := byUser[d.userID]
		err := sendPush(ctx, d.token, utils.PushMessage{Title: msg.title, Body: msg.message, Data: data})
		switch {
		case err == nil:
			mu.Lock()
			sent++
			mu.Unlock()
		case errors.Is(err, utils.ErrPushDisabled):
			// Inbox rows are written; there's just no FCM to deliver through.
			mu.Lock()
			disabled = true
			mu.Unlock()
		case errors.Is(err, utils.ErrStaleToken):
			mu.Lock()
			stale++
			mu.Unlock()
			if _, err := database.DB.Exec(`DELETE FROM device_tokens WHERE token = $1`, d.token); err != nil {
				log.Printf("notify: removing stale device token: %v", err)
			}
		default:
			mu.Lock()
			failed++
			mu.Unlock()
			log.Printf("notify: push to %s failed: %v", d.userID, err)
		}
	}

	// Side by side, a few at a time: one after another, a reader with many
	// devices (old installs, tokens APNs rejects) used up the whole timeout
	// before reaching the phone they were holding.
	jobs := make(chan device)
	var wg sync.WaitGroup
	for i := 0; i < pushWorkers && i < len(devices); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range jobs {
				send(d)
			}
		}()
	}
feed:
	for _, d := range devices {
		mu.Lock()
		stop := disabled
		mu.Unlock()
		if stop {
			break
		}
		select {
		case jobs <- d:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	if disabled {
		return
	}
	if ctx.Err() != nil && sent+stale+failed < len(devices) {
		log.Printf("notify: %s — gave up after %d of %d devices: %v", typ, sent+stale+failed, len(devices), ctx.Err())
		return
	}
	log.Printf("notify: %s — %d pushed, %d stale tokens removed, %d devices", typ, sent, stale, len(devices))
}
