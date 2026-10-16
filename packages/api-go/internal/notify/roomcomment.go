package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
)

// snippetLength is how much of a comment a push quotes.
const snippetLength = 120

// RoomCommentEvent is one new comment (or reply) in a room.
type RoomCommentEvent struct {
	RoomID, RoomName string
	BookID           string
	AuthorID         string
	AuthorName       string
	Page             int // 0-based, as stored; a reply's is its root's
	Body             string
	Exclude          []string      // members told some other way (the replied-to author)
	Window           time.Duration // how often one reader's row may push
}

// RoomComment tells every other member of the room about a comment. Each
// reader has at most one unread ROOM_COMMENTS row per room: the first comment
// creates it and pushes it, later ones fold into it ("3 new comments") and
// only push once the row hasn't been pushed for Window. Reading the row ends
// the bundle; the next comment starts a new one.
//
// What a reader is told depends on how far they've read, by the same rule the
// comment layer uses (page <= furthest_page): the text if they've reached the
// page, only the page number if they haven't. A bundled row never has text.
func RoomComment(e RoomCommentEvent) error {
	unlocked := e.AuthorName + ": " + Snippet(e.Body)
	// The reader shows pages 1-based.
	locked := fmt.Sprintf("%s commented on p. %d", e.AuthorName, e.Page+1)
	exclude := e.Exclude
	if exclude == nil {
		exclude = []string{}
	}

	// One statement for every recipient. ON CONFLICT targets the partial
	// unique index on unread ROOM_COMMENTS rows, so two comments at once fold
	// into the same row; the second waits for the first's row lock and sees
	// its last_pushed_at, so only one of them pushes. LOCALTIMESTAMP matches
	// the CURRENT_TIMESTAMP defaults the other rows get, and is the same for
	// the whole statement, which is how RETURNING tells which rows to push.
	rows, err := database.DB.Query(
		`WITH recipients AS (
		   SELECT m.user_id, COALESCE(rp.furthest_page, 0) AS furthest
		     FROM room_members m
		     LEFT JOIN reading_progress rp ON rp.user_id = m.user_id AND rp.book_id = $3
		    WHERE m.room_id = $1
		      AND m.user_id <> $4::uuid
		      AND NOT (m.user_id = ANY($5::uuid[]))
		      AND `+switchedOn(models.NotificationTypeRoomComments, "m.user_id")+`
		 )
		 INSERT INTO notifications
		     (user_id, type, title, message, book_id, room_id, event_count, last_pushed_at, updated_at)
		 SELECT user_id, 'ROOM_COMMENTS', $2,
		        CASE WHEN $6 <= furthest THEN $7 ELSE $8 END,
		        $3, $1, 1, LOCALTIMESTAMP, LOCALTIMESTAMP
		   FROM recipients
		 ON CONFLICT (user_id, room_id) WHERE type = 'ROOM_COMMENTS' AND is_read IS NOT TRUE
		 DO UPDATE SET
		     event_count = notifications.event_count + 1,
		     message = (notifications.event_count + 1)::text || ' new comments',
		     title = EXCLUDED.title,
		     book_id = EXCLUDED.book_id,
		     updated_at = LOCALTIMESTAMP,
		     last_pushed_at = CASE
		         WHEN notifications.last_pushed_at IS NULL
		           OR notifications.last_pushed_at <= LOCALTIMESTAMP - make_interval(secs => $9)
		         THEN LOCALTIMESTAMP
		         ELSE notifications.last_pushed_at
		     END
		 RETURNING id, user_id, title, message, last_pushed_at = LOCALTIMESTAMP`,
		e.RoomID, e.RoomName, e.BookID, e.AuthorID, pq.Array(exclude),
		e.Page, unlocked, locked, e.Window.Seconds(),
	)
	if err != nil {
		return fmt.Errorf("storing room comment notifications: %w", err)
	}
	toPush := map[string]delivery{}
	for rows.Next() {
		var d delivery
		var userID string
		var push bool
		if err := rows.Scan(&d.notificationID, &userID, &d.title, &d.message, &push); err != nil {
			rows.Close()
			return fmt.Errorf("reading room comment notifications: %w", err)
		}
		if push {
			toPush[userID] = d
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading room comment notifications: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()
	pushToDevices(ctx, toPush, Notification{
		Type:     models.NotificationTypeRoomComments,
		BookID:   e.BookID,
		RoomID:   e.RoomID,
		RoomName: e.RoomName,
	})
	return nil
}

// Snippet is a comment as one line, cut to snippetLength characters.
func Snippet(body string) string {
	line := strings.Join(strings.Fields(body), " ")
	runes := []rune(line)
	if len(runes) <= snippetLength {
		return line
	}
	return strings.TrimRight(string(runes[:snippetLength]), " ") + "…"
}
