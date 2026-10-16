package notify

import (
	"fmt"

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
