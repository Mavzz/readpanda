package models

import (
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"isactive"`
	LoginType string    `json:"login_type"`
	UUID      string    `json:"uuid"`
	GoogleSub *string   `json:"google_sub,omitempty"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// UserPreferences represents user reading preferences
type UserPreferences struct {
	ID          int                    `json:"id"`
	UserID      string                 `json:"user_id"`
	Preferences map[string]interface{} `json:"preferences"`
}

// Book represents a book in the system
type Book struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Subgenre      string    `json:"subgenre"`
	Genre         string    `json:"genre"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
	ManuscriptURL *string   `json:"manuscript_url,omitempty"`
	Status        int       `json:"status"`
	Views         int       `json:"views"`
	UserID        *string   `json:"user_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Notification represents a user notification
type Notification struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// Preference represents a genre/subgenre preference
type Preference struct {
	ID       int    `json:"id"`
	Genre    string `json:"genre"`
	Subgenre string `json:"subgenre"`
}

// CuratedBucket represents an editorially curated collection
type CuratedBucket struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	SortOrder     int           `json:"sort_order"`
	IsActive      bool          `json:"is_active"`
	CoverImageURL *string       `json:"cover_image_url"`
	BookCount     int           `json:"book_count"`
	BooksPreview  []BookPreview `json:"books_preview"`
}

// BookPreview is a minimal book representation for bucket previews
type BookPreview struct {
	BookID        string  `json:"book_id"`
	Title         string  `json:"title"`
	AuthorName    string  `json:"author_name,omitempty"`
	CoverImageURL *string `json:"cover_image_url,omitempty"`
	ManuscriptURL *string `json:"manuscript_url,omitempty"`
}

// CuratedBucketDetail is the full response for a single curated bucket
type CuratedBucketDetail struct {
	ID    string             `json:"id"`
	Title string             `json:"title"`
	Books []CuratedBookEntry `json:"books"`
}

// CuratedBookEntry is a book within a curated bucket (full detail)
type CuratedBookEntry struct {
	BookID        string  `json:"book_id"`
	Title         string  `json:"title"`
	AuthorName    string  `json:"author_name,omitempty"`
	CoverImageURL *string `json:"cover_image_url,omitempty"`
	ManuscriptURL *string `json:"manuscript_url,omitempty"`
	Genre         string  `json:"genre"`
	Rating        float64 `json:"rating"`
}

// UserBucket represents a user-created collection
type UserBucket struct {
	ID           string        `json:"id"`
	UserID       string        `json:"-"`
	Name         string        `json:"name"`
	BookCount    int           `json:"book_count,omitempty"`
	BooksPreview []BookPreview `json:"books_preview,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

// Room represents a reading room
type Room struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	InviteCode  *string   `json:"invite_code,omitempty"`
	AdminID     string    `json:"admin_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoomBook represents a book in a reading room
type RoomBook struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	BookID    string    `json:"book_id"`
	AddedAt   time.Time `json:"added_at"`
	AddedByID string    `json:"added_by_id"`
}

// RoomMember represents a member of a reading room
type RoomMember struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	InvitedAt time.Time `json:"invited_at"`
	JoinedAt  time.Time `json:"joined_at"`
}

// RoomMemberDetail is a member as the Room Detail screen needs them —
// the user's name and role alongside the membership row.
type RoomMemberDetail struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// RoomBucket is the reading list a room is working through. Buckets live in
// two tables, so Type ("user" | "curated") says which one ID refers to.
type RoomBucket struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Type  string        `json:"type"`
	Books []BookPreview `json:"books"`
}

// RoomDetail is the full Room Detail payload: the room, who's in it, and what
// it's reading (a standalone book, or a current book from a bucket).
type RoomDetail struct {
	Room
	CurrentBook *BookPreview       `json:"current_book,omitempty"`
	Bucket      *RoomBucket        `json:"bucket,omitempty"`
	Members     []RoomMemberDetail `json:"members"`
}

// ReadingProgress is one reader's place in one book. Progress is personal and
// keys on the book rather than the room, so the same position follows a reader
// across every room reading that book — and survives the room going away.
type ReadingProgress struct {
	UserID      string `json:"user_id"`
	BookID      string `json:"book_id"`
	CurrentPage int    `json:"current_page"`
	TotalPages  int    `json:"total_pages"`
	// The furthest page ever reached, which only climbs. Comments unlock
	// against this rather than CurrentPage, so re-reading an earlier passage
	// can't hide a comment the reader has already been shown.
	FurthestPage int       `json:"furthest_page"`
	ProgressPct  int       `json:"progress_pct"`
	LastReadAt   time.Time `json:"last_read_at"`
}

// MemberProgress is one member on a room's pace track. A member who hasn't
// opened the book yet still appears, at page 0 with a nil LastReadAt — the
// track shows who is behind, so it can't leave people out.
type MemberProgress struct {
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	CurrentPage int        `json:"current_page"`
	TotalPages  int        `json:"total_pages"`
	ProgressPct int        `json:"progress_pct"`
	LastReadAt  *time.Time `json:"last_read_at"`
}

// RoomProgress is the pace track for whatever a room is currently reading.
// A room that hasn't chosen a book yet returns an empty BookID and no members:
// there is no book to be paced against.
type RoomProgress struct {
	RoomID  string           `json:"room_id"`
	BookID  string           `json:"book_id"`
	Members []MemberProgress `json:"members"`
}

// CommentAnchor is where in the manuscript a comment is attached. Text is the
// passage the author selected, and is empty for a page-level comment — 6b
// omits the quote block for those. Bounds is the fallback for redrawing the
// highlight when the text can no longer be found in the document.
type CommentAnchor struct {
	Page   int             `json:"page"`
	Text   string          `json:"text"`
	Bounds json.RawMessage `json:"bounds,omitempty"`
	Key    string          `json:"key"`
}

// BookComment is one comment on a passage of a book, inside a room. Replies
// hang off their root and inherit its anchor and room — a reply is part of the
// conversation about a passage, not a new place in the book.
type BookComment struct {
	ID        string        `json:"id"`
	RoomID    string        `json:"room_id"`
	BookID    string        `json:"book_id"`
	UserID    string        `json:"user_id"`
	Username  string        `json:"username"`
	Page      int           `json:"page"`
	Anchor    CommentAnchor `json:"anchor"`
	Body      string        `json:"body"`
	ParentID  string        `json:"parent_id,omitempty"`
	Likes     int           `json:"likes"`
	LikedByMe bool          `json:"liked_by_me"`
	Read      bool          `json:"read"`
	CreatedAt time.Time     `json:"created_at"`
	Replies   []BookComment `json:"replies"`
}

// BookCommentThread is every comment sharing one anchor — one gutter dot in
// the reader, one sheet in 6b.
type BookCommentThread struct {
	AnchorKey    string          `json:"anchor_key"`
	Page         int             `json:"page"`
	AnchorText   string          `json:"anchor_text"`
	AnchorBounds json.RawMessage `json:"anchor_bounds,omitempty"`
	FileHash     string          `json:"file_hash,omitempty"`
	Comments     []BookComment   `json:"comments"`
	UnreadCount  int             `json:"unread_count"`
}

// BookCommentsResponse is the comment layer for one book in one room.
//
// LockedCount is deliberately a bare integer. A comment the reader hasn't
// reached yet contributes nothing else to this response — no id, no page, no
// preview — so there is nothing in the payload to read ahead with.
type BookCommentsResponse struct {
	RoomID              string              `json:"room_id"`
	BookID              string              `json:"book_id"`
	FurthestPage        int                 `json:"furthest_page"`
	Threads             []BookCommentThread `json:"threads"`
	LockedCount         int                 `json:"locked_count"`
	UnlockedUnreadCount int                 `json:"unlocked_unread_count"`
}

// LoginType constants
const (
	LoginTypeEmail        = "email"
	LoginTypeSocialGoogle = "social_google"
	LoginTypeLDAP         = "ldap"
)

// Role constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// SignupRequest represents the signup request body
type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse represents the token response
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	Picture      string `json:"picture,omitempty"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// GoogleAuthPayload represents Google OAuth token payload
type GoogleAuthPayload struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Sub     string `json:"sub"`
	Picture string `json:"picture"`
}
