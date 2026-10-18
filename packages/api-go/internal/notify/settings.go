package notify

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/models"
)

// Settings is a reader's switch per kind of notification. A switched-off kind
// isn't written to their inbox or pushed. A reader with no
// notification_settings row has everything on.
type Settings struct {
	Replies      bool `json:"replies"`
	RoomComments bool `json:"room_comments"`
	RoomJoins    bool `json:"room_joins"`
	RoomBook     bool `json:"room_book"`
	NewBooks     bool `json:"new_books"`
}

// DefaultSettings is what a reader who has never changed a switch gets.
func DefaultSettings() Settings {
	return Settings{Replies: true, RoomComments: true, RoomJoins: true, RoomBook: true, NewBooks: true}
}

// SettingsPatch is a partial update: nil leaves a switch as it is.
type SettingsPatch struct {
	Replies      *bool `json:"replies"`
	RoomComments *bool `json:"room_comments"`
	RoomJoins    *bool `json:"room_joins"`
	RoomBook     *bool `json:"room_book"`
	NewBooks     *bool `json:"new_books"`
}

// LoadSettings returns userID's switches, all on if they've never changed one.
func LoadSettings(userID string) (Settings, error) {
	var s Settings
	err := database.DB.QueryRow(
		`SELECT replies, room_comments, room_joins, room_book, new_books
		   FROM notification_settings WHERE user_id = $1`,
		userID,
	).Scan(&s.Replies, &s.RoomComments, &s.RoomJoins, &s.RoomBook, &s.NewBooks)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultSettings(), nil
	}
	return s, err
}

// UpdateSettings applies p and returns the full set. The first change creates
// the row, with every switch p doesn't mention left on.
func UpdateSettings(userID string, p SettingsPatch) (Settings, error) {
	var s Settings
	err := database.DB.QueryRow(
		`INSERT INTO notification_settings
		    (user_id, replies, room_comments, room_joins, room_book, new_books)
		 VALUES ($1, COALESCE($2, true), COALESCE($3, true), COALESCE($4, true),
		         COALESCE($5, true), COALESCE($6, true))
		 ON CONFLICT (user_id) DO UPDATE SET
		    replies       = COALESCE($2, notification_settings.replies),
		    room_comments = COALESCE($3, notification_settings.room_comments),
		    room_joins    = COALESCE($4, notification_settings.room_joins),
		    room_book     = COALESCE($5, notification_settings.room_book),
		    new_books     = COALESCE($6, notification_settings.new_books),
		    updated_at    = CURRENT_TIMESTAMP
		 RETURNING replies, room_comments, room_joins, room_book, new_books`,
		userID, p.Replies, p.RoomComments, p.RoomJoins, p.RoomBook, p.NewBooks,
	).Scan(&s.Replies, &s.RoomComments, &s.RoomJoins, &s.RoomBook, &s.NewBooks)
	return s, err
}

// switchColumn is the notification_settings column that turns each type off.
// SYSTEM has none: it always goes out.
var switchColumn = map[string]string{
	models.NotificationTypeReply:        "replies",
	models.NotificationTypeRoomComments: "room_comments",
	models.NotificationTypeRoomJoin:     "room_joins",
	models.NotificationTypeRoomBook:     "room_book",
	models.NotificationTypeNewBook:      "new_books",
}

// switchedOn is a SQL condition, for a recipient query's WHERE, that keeps
// only readers who haven't switched notifType off. userExpr is the column
// holding the recipient's id. Filtering in SQL keeps every fan-out one
// statement, and means a switched-off reader gets no inbox row either.
//
// The column name comes from switchColumn, never from input, so it's safe to
// format into the query.
func switchedOn(notifType, userExpr string) string {
	column, ok := switchColumn[notifType]
	if !ok {
		return "TRUE"
	}
	return fmt.Sprintf(
		"NOT EXISTS (SELECT 1 FROM notification_settings ns WHERE ns.user_id = %s AND NOT ns.%s)",
		userExpr, column,
	)
}
