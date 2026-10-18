package notify

import (
	"sort"
	"strings"
	"testing"

	"github.com/Mavzz/readpanda/api-go/internal/models"
	"github.com/Mavzz/readpanda/api-go/internal/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func TestEveryRoomTypeHasASwitch(t *testing.T) {
	for _, typ := range []string{
		models.NotificationTypeReply,
		models.NotificationTypeRoomComments,
		models.NotificationTypeRoomJoin,
		models.NotificationTypeRoomBook,
		models.NotificationTypeNewBook,
	} {
		if _, ok := switchColumn[typ]; !ok {
			t.Errorf("%s has no switch", typ)
		}
	}
	if got := switchedOn(models.NotificationTypeSystem, "u.uuid"); got != "TRUE" {
		t.Errorf("SYSTEM should always go out, got %q", got)
	}
}

// Runs each type's condition against real rows: a reader with no settings
// row, one with only Room comments off, and one with everything off.
func TestSwitchedOnFiltersRecipients(t *testing.T) {
	db := testdb.Require(t)
	if _, err := db.Exec(`
		INSERT INTO users (uuid, username, email) VALUES
		  ('a0000000-0000-0000-0000-000000000001', 'defaults', 'a@example.com'),
		  ('b0000000-0000-0000-0000-000000000002', 'nocomments', 'b@example.com'),
		  ('c0000000-0000-0000-0000-000000000003', 'alloff', 'c@example.com');
		INSERT INTO notification_settings (user_id, room_comments) VALUES
		  ('b0000000-0000-0000-0000-000000000002', false);
		INSERT INTO notification_settings (user_id, replies, room_comments, room_joins, room_book, new_books) VALUES
		  ('c0000000-0000-0000-0000-000000000003', false, false, false, false, false);
	`); err != nil {
		t.Fatal(err)
	}

	recipients := func(typ string) string {
		rows, err := db.Query(`SELECT username FROM users u WHERE ` + switchedOn(typ, "u.uuid"))
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatal(err)
			}
			names = append(names, n)
		}
		sort.Strings(names)
		return strings.Join(names, ",")
	}

	for typ, want := range map[string]string{
		models.NotificationTypeRoomComments: "defaults",
		models.NotificationTypeReply:        "defaults,nocomments",
		models.NotificationTypeRoomJoin:     "defaults,nocomments",
		models.NotificationTypeRoomBook:     "defaults,nocomments",
		models.NotificationTypeNewBook:      "defaults,nocomments",
		models.NotificationTypeSystem:       "alloff,defaults,nocomments",
	} {
		if got := recipients(typ); got != want {
			t.Errorf("%s: got %q, want %q", typ, got, want)
		}
	}
}
