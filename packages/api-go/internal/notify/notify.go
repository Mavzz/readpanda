// Package notify writes notifications to users' inboxes and pushes them to
// their devices. The inbox row is the record; the push is a best-effort nudge
// to go and look at it, so a failed or unconfigured push never loses one.
package notify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

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
}

// sendTimeout bounds one fan-out. Pushes run after the triggering request has
// already answered, so this only stops a stuck FCM call from leaking forever.
const sendTimeout = 2 * time.Minute

// ToUsers stores n in each user's inbox and pushes it to their devices.
// Returns once the rows are written; pushes continue in the background.
func ToUsers(userIDs []string, n Notification) error {
	if len(userIDs) == 0 {
		return nil
	}
	return insertAndPush(
		`INSERT INTO notifications (user_id, type, title, message, book_id)
		 SELECT u, $2, $3, $4, $5 FROM unnest($1::uuid[]) AS u
		 RETURNING id, user_id`,
		pq.Array(userIDs), n,
	)
}

// ToAllUsersExcept is the broadcast form — a new book is news to everyone but
// the person who published it. The fan-out stays in SQL so a large user base
// is one statement, not one query per user.
func ToAllUsersExcept(excludeUserID string, n Notification) error {
	return insertAndPush(
		`INSERT INTO notifications (user_id, type, title, message, book_id)
		 SELECT uuid, $2, $3, $4, $5 FROM users WHERE uuid <> $1::uuid
		 RETURNING id, user_id`,
		excludeUserID, n,
	)
}

func insertAndPush(query string, recipients interface{}, n Notification) error {
	if n.Type == "" {
		n.Type = models.NotificationTypeSystem
	}
	var bookID interface{}
	if n.BookID != "" {
		bookID = n.BookID
	}

	rows, err := database.DB.Query(query, recipients, n.Type, n.Title, n.Message, bookID)
	if err != nil {
		return fmt.Errorf("storing notifications: %w", err)
	}
	// notification id per user, so a tapped push can name the row it is about.
	byUser := map[string]int{}
	for rows.Next() {
		var id int
		var userID string
		if err := rows.Scan(&id, &userID); err != nil {
			rows.Close()
			return fmt.Errorf("reading stored notifications: %w", err)
		}
		byUser[userID] = id
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading stored notifications: %w", err)
	}

	go pushToDevices(byUser, n)
	return nil
}

func pushToDevices(byUser map[string]int, n Notification) {
	if len(byUser) == 0 {
		return
	}
	userIDs := make([]string, 0, len(byUser))
	for id := range byUser {
		userIDs = append(userIDs, id)
	}

	rows, err := database.DB.Query(
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

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	sent, stale := 0, 0
	for _, d := range devices {
		data := map[string]string{
			"type":            n.Type,
			"notification_id": strconv.Itoa(byUser[d.userID]),
		}
		if n.BookID != "" {
			data["book_id"] = n.BookID
		}

		err := utils.SendPush(ctx, d.token, utils.PushMessage{Title: n.Title, Body: n.Message, Data: data})
		switch {
		case err == nil:
			sent++
		case errors.Is(err, utils.ErrPushDisabled):
			// Inbox rows are written; there's just no FCM to deliver through.
			return
		case errors.Is(err, utils.ErrStaleToken):
			stale++
			if _, err := database.DB.Exec(`DELETE FROM device_tokens WHERE token = $1`, d.token); err != nil {
				log.Printf("notify: removing stale device token: %v", err)
			}
		default:
			log.Printf("notify: push to %s failed: %v", d.userID, err)
		}
	}
	log.Printf("notify: %s — %d pushed, %d stale tokens removed, %d devices", n.Type, sent, stale, len(devices))
}
