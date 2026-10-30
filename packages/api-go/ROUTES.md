# ReadPanda API Routes Documentation

Base URL: `{host}:{port}{API_VERSION_PREFIX}`

All routes (except where noted) require a **Bearer token** in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

---

## Authentication

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/signup` | No | Register a new user. Body: `{ username, email, password }` (plaintext over TLS). Username: 3–20 letters, numbers, `.` or `_`, stored lowercase; password ≥ 8 characters. Returns access token, refresh token, and user preferences. Errors carry the field: `400 { error, field }` for a rule broken (`username` / `email` / `password`), `409 { error, field }` for a taken `username` or `email`. |
| GET | `/signup/available?username=` | No | Whether a username is free: `{ available, suggestion? }` (a nearby free name when taken). `400 { error, field: "username" }` if the name breaks the rules. Rate-limited to 30 requests a minute per client (`429`). |
| POST | `/auth/login` | No | Login with email/username and password (username matched case-insensitively, exact match first). Returns access token, refresh token, and user data, including `user_id`, `display_name` and `login_type` (as every sign-in and `/token/refresh` now does). |
| POST | `/auth/google` | No | Authenticate via Google ID token. Body: `{ token }`. Creates account on first login, with a username made from the Google name to fit the sign-up rule (as for Apple). Returns access/refresh tokens. |
| POST | `/auth/apple` | No | Sign in with Apple. Body: `{ identity_token, authorization_code?, full_name? }`. Verifies the token against Apple's keys (audience `APPLE_BUNDLE_ID`); finds the account by Apple id, else links one with the same verified email, else creates one (`201`, with preferences). With `APPLE_TEAM_ID`/`APPLE_KEY_ID`/`APPLE_PRIVATE_KEY` set, also stores Apple's refresh token so deleting the account can revoke it. `401` invalid token; `400` if Apple shared no email for a new account. |
| POST | `/auth/logout` | Yes | Logout the current user. |
| POST | `/token/refresh` | No | Refresh an expired access token. Body: `{ refreshToken }`. Returns `{ accessToken, refreshToken }`: the refresh token is rotated, so the one sent is spent. Each sign-in has its own refresh token, so refreshing on one device leaves the account's other devices signed in. `401` for a spent, expired or signed-out token. |

**Blocked words.** A username, room name or description, or comment containing a word on the list in `internal/moderation/wordlist.go` is refused with `400 { error: "That includes words we don't allow.", code: "blocked_words", field }`. Matching is on whole words, ignoring case, accents and look-alike characters ("Scunthorpe" passes). Apple and Google sign-up, which make the username from the reader's name, fall back to `reader` instead of refusing.

**Suspended accounts** (`users.isactive = false`) get `403 { error, code: "account_suspended" }` from `/auth/login` (after the password check), `/auth/apple`, `/auth/google` and `/token/refresh`, from posting comments, creating, joining or changing rooms, `/reports`, and the block routes, and from the two reads the app refreshes with, `/notifications/unread/count` and `/room/my-rooms`. Other reads keep working until the access token expires; the app signs out on whichever refusal comes first.

---

## Users

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/users` | Admin | List all users (admin role required; `401` without a token, `403` for non-admins). Returns `[ { username, email, isactive, login_type, uuid, created_at } ]`. |
| GET | `/users/me/blocks` | Yes | Readers the caller has blocked, newest first: `[ { user_id, username, blocked_at } ]`. |
| POST | `/users/me/blocks` | Yes | Block a reader. Body: `{ user_id }`. `204`, also when already blocked. `400` for yourself or the Deleted reader, `404` for an unknown reader. They aren't told. See **Blocks** under Comments for what it hides. |
| DELETE | `/users/me/blocks/{userId}` | Yes | Unblock. `204` whether or not they were blocked. |
| DELETE | `/users/me` | Yes | Delete the caller's account for good, in one transaction. Rooms they created pass to the earliest-joined member (who becomes admin), or are deleted if nobody else is in them; their comments stay, moved to the reserved "Deleted reader" account (`deleted.reader`, id `00000000-…`; see `scripts/migrate_account_deletion.sql`), so replies survive; books they uploaded stay with no uploader. Everything else of theirs (progress, buckets, highlights, tokens, devices, notifications, memberships, blocks either way) is deleted. Reports they made stay, without a reporter. Then, if they signed in with Apple, revokes Apple's token (best effort; logged on failure). `204`, also when the account is already gone. |
| GET | `/users/me` | Yes | The caller's account: `{ user_id, username, display_name, email, login_type }`. Edit profile reads it on opening, so a session from before sign-ins carried these fields gets them. |
| PATCH | `/users/me` | Yes | Edit profile. Body: any of `{ display_name, username }`. `display_name` is trimmed, up to 40 characters, and empty clears it. `username` follows the sign-up rule (stored lowercase); its own name in another case is no change; taken (including the reserved name, or taken between the app's check and the save) is `409 { error, field: "username", suggestion }`. Returns `{ user_id, username, display_name, email, login_type }`. Comments and rooms read usernames at query time, so they follow a rename. Requires `scripts/migrate_profile.sql`. |
| PUT | `/users/me/email` | Yes | Change email. Body: `{ email, current_password }`. Email accounts only: Apple and Google accounts, or a wrong password, get `403 { field: "current_password" }`. `400` for an invalid address, `409` when another account has it. Returns the profile as PATCH does. There's no confirmation email yet. |
| PUT | `/users/me/password` | Yes | Change password. Body: `{ current_password, new_password }` (8+ characters, else `400`). `403` as for email. `204`. Other signed-in devices stay signed in. |

---

## Books

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/books/upload-urls` | Yes | Start a publish. Body `{ manuscript: { name, size }, cover?: { name, size } }`. Returns `{ book_id, manuscript: { key, url, content_type }, cover? }`: PUT each file to its `url` with that `Content-Type` header (URLs last 1 hour). Manuscript: PDF or EPUB, up to 500 MB. Cover: JPEG, PNG, WebP or GIF, up to 20 MB. |
| POST | `/books/upload` | Yes | Publish a new book. **JSON** (after `/books/upload-urls`): `book_id`, `title`, `description`, `genre`, `subgenre`, `author_name`, `notify` (default true), `manuscript_key`, `cover_key?`. Checks the files are in storage within the limits. Still accepts the old **multipart form** with `cover` and `manuscript` files, but Cloud Run rejects those bodies over 32 MB. |
| GET | `/books` | Yes | Get all books for the authenticated user. Returns `{ books: [...] }`. |
| GET | `/books/all` | Yes | Get all books in the system. Returns `{ books: [...] }`. |
| GET | `/books/search?q=` | Yes | Search books by title or author (case-insensitive; `%` and `_` match literally). Returns `{ books: [...] }` in `/books/all`'s shape, up to 50: title starting with `q` first, then title containing it, then author containing it, each by title. `q` is trimmed; under 2 characters is `400`. |
| POST | `/books/seed` | Yes | Seed books from object storage (R2/MinIO). Scans the storage bucket and inserts missing books into the database. |
| GET | `/books/{bookId}` | Yes | Book detail (8c). Returns `{ book: { book_id, title, description, author_name, page_count, genre, subgenre, cover_image_url, manuscript_url }, in_buckets: [bucketId], friends_read: { count, friends: [{ user_id, username }] } }`. `in_buckets` lists the caller's own buckets holding the book. `friends_read` counts room-mates (anyone sharing a room with the caller) with a reading position in the book, naming the 3 most recent. `author_name` / `page_count` are null when unknown. |

---

## Discover

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/admin/books/enrich?limit=50&recheck=false` | Admin | Run the book metadata lookup now (Open Library, then Google Books) over books not yet checked: readable title from the file name, then title/author/page count from the lookup. Also runs at startup and after a seed or upload. `recheck=true` re-queues every book. Returns `{ checked, resolved }`. Set `METADATA_LOOKUP=off` to skip the network. |
| GET | `/discover?genre={subgenre}` | Yes | Discover tab (8a). Returns `{ genre, genres: [{ value, label, liked }], curated: [CuratedBucket], popular: [{ ...book, added_at, readers_this_week, friends_read }] }` (`added_at` is the book's catalogue date, for See all's Newest sort). Without `genre` this is the **For you** feed: curated buckets ranked by how many of their books are in the caller's liked subgenres, and Popular ranked by readers in the last 7 days + 2× room-mates who've read it + 3 if the book is in a liked subgenre. With `genre`, both sections are filtered to that subgenre and Popular is ranked by readers this week. `genres` lists the caller's liked subgenres first, then every other subgenre with at least one book. Requires `scripts/migrate_discover.sql`. |

---

## User Preferences

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/user/preferences?username={username}` | Yes | Get preferences for a user. Returns the preferences JSON object. |
| POST | `/user/preferences?username={username}` | Yes | Update preferences for a user. Body: `{ preferences: {...} }`. |

---

## Genres & Subgenres

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/genres` | Yes | List all distinct genres. Returns `{ genre: [{ value, label }] }`. |
| GET | `/subgenres` | Yes | List all subgenres. Returns `{ subgenre: [{ value, label }] }`. |

---

## Notifications

Every route here acts on the caller, taken from the access token. The old `?username=` parameter is ignored.

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/notifications` | Yes | The caller's notifications, most recently updated first. Returns `[ { id, user_id, type, title, message, book_id, room_id, count, is_read, created_at, updated_at } ]`, or `[]` when empty. `type` is `SYSTEM`, `NEW_BOOK`, `REPLY`, `ROOM_COMMENTS`, `ROOM_JOIN` or `ROOM_BOOK`. `book_id` names the book to open (null if it was deleted); `room_id` the room, for the room types (the row is deleted with its room). `count` is how many comments a `ROOM_COMMENTS` row stands for, 1 otherwise; `updated_at` moves forward when another comment folds into it. |
| GET | `/notifications/unread/count` | Yes | Returns `{ unread_count: <int> }`. |
| PUT | `/notifications/{id}/read` | Yes | Mark one notification read. Idempotent. `204`, or `404` if it isn't the caller's. |
| POST | `/users/me/devices` | Yes | Register an FCM token for push. Body: `{ token, platform }`, where `platform` is `ios` or `android`. Upserts on the token, so a phone that switches accounts moves to the new one. `204`. |
| DELETE | `/users/me/devices/{token}` | Yes | Unregister a device (URL-encode the token). Only removes the caller's own token. Idempotent. `204`. |
| GET | `/users/me/notification-settings` | Yes | The caller's switch per kind of notification: `{ replies, room_comments, room_joins, room_book, new_books }`, all `true` until they change one. |
| PUT | `/users/me/notification-settings` | Yes | Change any subset of the five, e.g. `{ "room_comments": false }`; the others keep their value. Returns the full set. `400` for an unknown key or a non-boolean value; `404` if the account no longer exists. A switched-off kind is neither written to the caller's inbox nor pushed to them. |

### Where notifications come from

`internal/notify` writes the inbox row, then pushes to the recipient's registered devices with the default iOS sound. The row is the record: if push is unconfigured or a send fails, the notification is still in the inbox. Tokens that FCM reports as unregistered are deleted. Readers who switched a kind off (`/users/me/notification-settings`) get neither the row nor the push, and nobody hears about a reply, comment or join by a reader they've blocked.

`notify.Send`, for room events, pushes before the triggering request answers, giving up after 5 seconds: Cloud Run throttles CPU once a response is sent. The `NEW_BOOK` broadcast (`notify.ToAllUsersExcept`) still pushes in the background after the upload answers.

| Trigger | Recipients | Notification |
|---------|------------|--------------|
| `POST /reports` (a new report, not a repeat) | Every account with the `admin` role | `SYSTEM`, "New report", "{reason} · comment in {room} \| reader {username} \| room {room}". Never the reported text. The report is saved even if this fails. |
| `POST /books/upload` with a manuscript | Every user except the publisher, and anyone with New books off | `NEW_BOOK`, "New book added", `book_id` set |
| `POST /room/{id}/book/{bookId}/comments` with `parent_id` | The thread's author, if it isn't the replier, they're still a member, and Replies is on | `REPLY`, "{username} replied", the reply cut to 120 characters; `room_id`, `book_id` set |
| `PATCH /room/{id}/reading` to a different, non-null book | Every member except the creator, if Room picks a book is on | `ROOM_BOOK`, title the room name, "Now reading {Book title}"; `room_id`, `book_id` set. Clearing the book, or sending the one it already has, notifies nobody. |
| `POST /room/join` or `POST /room/{id}/join` (a new member, not a 403, 404 or 409) | The room's creator, if New members is on | `ROOM_JOIN`, title the room name, "{username} joined the room"; `room_id` set |
| `POST /room/{id}/book/{bookId}/comments` | Every other member, except a reply's thread author who got `REPLY` | `ROOM_COMMENTS`, title the room name. "{username}: {text}" if the reader has reached the page (`page <= furthest_page`), otherwise "{username} commented on p. {page+1}". One unread row per reader per room: later comments update it to "{n} new comments" (no text) and it pushes at most once per `NOTIFY_ROOM_COMMENT_WINDOW`. A retried POST (same `client_id`) notifies nobody. |

Push payload `data`: `{ type, notification_id, book_id?, room_id?, room_name? }`.

Push is enabled when `FIREBASE_PROJECT_ID` (or a service account file with a `project_id`) and credentials are set; see `.env.local.example`. Without them the server logs `Push notifications disabled` at startup and everything else still works.

---

## User Buckets (Collections)

Users can create up to **20 personal buckets** to organize books.

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/users/me/buckets` | Yes | List all buckets for the authenticated user. Each bucket includes `book_count` and a `books_preview` (first 2 books). |
| POST | `/users/me/buckets` | Yes | Create a new bucket. Body: `{ name: string, book_ids?: int[] }`. Name must be 1–40 chars. Returns the created bucket. |
| PUT | `/users/me/buckets/{id}` | Yes | Rename a bucket. Body: `{ name: string }`. |
| DELETE | `/users/me/buckets/{id}` | Yes | Delete a bucket and its book associations. |
| POST | `/users/me/buckets/{id}/books` | Yes | Add books to a bucket. Body: `{ book_ids: int[] }`. |
| DELETE | `/users/me/buckets/{id}/books/{bookId}` | Yes | Remove a single book from a bucket. |
| PUT | `/users/me/buckets/{id}/order` | Yes | Save a bucket's manual order (9a edit mode). Body: `{ book_ids: [...] }`. Unlisted books keep their relative order after the listed ones. Returns 204. |

---

## Curated "Our Picks" (Admin)

Curated buckets shown on the home screen. Admin routes are portal-only.

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/home/our-picks` | Yes | Get curated buckets. Mobile clients receive only **active** buckets; portal clients (`X-Application-Type: portal`) receive all buckets including inactive ones. |
| POST | `/home/our-picks` | Yes | (Admin) Create a new curated bucket. |
| PUT | `/home/our-picks/{bucketId}` | Yes | (Admin) Update a curated bucket (name, active status, sort order). |
| DELETE | `/home/our-picks/{bucketId}` | Yes | (Admin) Delete a curated bucket. |
| GET | `/home/our-picks/{bucketId}/books` | Yes | Get all books in a curated bucket. |
| POST | `/home/our-picks/{bucketId}/books` | Yes | (Admin) Add books to a curated bucket. |
| DELETE | `/home/our-picks/{bucketId}/books/{bookId}` | Yes | (Admin) Remove a book from a curated bucket. |

---

## Admin Data Browser (Admin)

Generic view over every table in the `public` schema, used by the portal's Users and Database pages. All routes require the `admin` role. Table and column names are checked against the catalog; values are always bind parameters. Values come back as text, and `password` is redacted.

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/admin/tables` | Yes | (Admin) List tables with row counts. |
| GET | `/admin/tables/{table}` | Yes | (Admin) Table schema: columns (type, nullable, default, primary key, enum values) and primary key. |
| GET | `/admin/tables/{table}/rows` | Yes | (Admin) Page through rows. Query: `limit` (≤500, default 50), `offset`, `sort`, `dir` (`asc`/`desc`), `search` (matches the whole row as text). |
| POST | `/admin/tables/{table}/rows` | Yes | (Admin) Insert a row. Body: `{ "<column>": value }`; omitted columns get their default. Returns the inserted row. |
| DELETE | `/admin/tables/{table}/rows` | Yes | (Admin) Delete one row by primary key. Body: `{ "key": { "<pk column>": value } }`. |
| GET | `/admin/users/{uuid}` | Yes | (Admin) A user plus their rows in every table with a foreign key to `users` (first 100 per table). |
| PATCH | `/admin/users/{uuid}` | Yes | (Admin) Suspend or reinstate. Body: `{ isactive }`. Suspending deletes their refresh tokens, so the app signs out at its next refresh. `400` for the admin themselves or the Deleted reader; `404` unknown user. |

### Moderation (Admin)

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/admin/reports?status=open&limit=&offset=` | Yes | (Admin) Reports, newest first. `status` is `open` (default), `actioned`, `dismissed` or `all`; `limit` ≤ 100, default 50. Returns `{ reports[], has_more, open_count }`. Each report: `{ id, target_type, target_id, reason, details, snapshot, status, action, created_at, resolved_at, resolved_by, reporter, target_user, room, comment_state }`. `reporter` and `target_user` are `{ id, username, suspended }`, or null once the account is deleted; `room` is `{ id, name, state }` with `state` `open`, `invite_only` or `gone`; `comment_state` is `live`, `deleted`, `removed` or `gone`. |
| POST | `/admin/reports/{id}/resolve` | Yes | (Admin) Act on a report. Body: `{ action }`: `remove_comment` (comment reports; shows as removed by ReadPanda, replies kept), `close_room` or `delete_room` (room reports), `suspend_user` (any report about someone), or `dismiss`. Closes this report and every other open one on the same target, recording the action, the admin and the time. Then tells the people it affects with a `SYSTEM` notification giving the report's reason, never the reporter: the comment's author for `remove_comment` (unless they'd already deleted it), the creator for `close_room` (if the room was open) and `delete_room`, and a deleted room's other members ("A room was removed", no reason). Returns `{ status, action, closed }`. `400` for an action that doesn't fit; `404` unknown report; `409` already closed. |

---

## Rooms

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/room/create` | Yes | Create a room. Body: `{ name, description, is_private }`. `is_private` defaults to `true` (Invite only) when left out. Every room gets an invite code, and the creator is enrolled as an `admin` member. |
| GET | `/room/my-rooms` | Yes | Rooms the user created or joined. Returns a bare array. |
| POST | `/room/join` | Yes | Join a room by code. Body: `{ invite_code }`. `404` unknown code, `409` already a member. |
| GET | `/room/open?book_id=&limit=&offset=` | Yes | Open rooms (`is_private = false`) the caller didn't make and isn't in, optionally only those reading `book_id`. Rooms reading a book first, then more members, then newer. `limit` defaults to 20, at most 50. Leaves out rooms made by someone the caller blocked. Returns `{ rooms: [{ id, name, description, current_book: { book_id, title, cover_image_url } \| null, member_count, creator_username, created_at }], has_more }`; no invite code or member list. Requires `scripts/migrate_open_rooms.sql`. |
| POST | `/room/{id}/join` | Yes | Join an open room without a code. Returns the Room Detail, and tells the creator as `/room/join` does. `404` unknown room, `403` invite-only, `409` already a member (even if the room has since gone invite-only). |
| GET | `/room/{id}` | Yes | Full Room Detail: the room, `members[]` (`user_id`, `username`, `role`, `joined_at`, `blocked`), `current_book`, and `bucket` (`{ id, name, type, books[] }`). `blocked` is whether the caller has blocked that member. `403` for non-members. |
| PATCH | `/room/{id}` | Yes | Change who can join. Body: `{ is_private }`; fields left out are unchanged. Creator-only (`403`). Members and the invite code stay. Returns the Room Detail. |
| DELETE | `/room/{id}` | Yes | Delete a room. Creator-only (`403`); `room_members` rows cascade. Returns `204`. |
| DELETE | `/room/{id}/members/me` | Yes | Leave a room. The creator can't leave (`403`) — they delete it instead. Returns `204`. |
| PATCH | `/room/{id}/reading` | Yes | Set what the room reads. Body: `{ current_book_id, bucket_id, bucket_type }` (`bucket_type` is `user` or `curated`; send `null`s to clear). Creator-only (`403`); with a bucket set, `current_book_id` must belong to it (`400`). Returns the updated Room Detail. |

A room reads **either** a standalone book **or** a bucket (a shared reading
list) with a current book chosen from it. Buckets live in two tables
(`user_buckets` / `curated_buckets`), so `rooms.current_bucket_id` is paired
with `current_bucket_type` rather than a single-table foreign key — see
`scripts/migrate_room_reading.sql`.

---

## Comments

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/room/{id}/book/{bookId}/comments` | Yes | The comment layer for one book in one room. Returns `{ room_id, book_id, furthest_page, threads[], locked_count, unlocked_unread_count }`. Each thread is `{ anchor_key, page, anchor_text, anchor_bounds, file_hash, comments[], unread_count }`, and each comment carries `likes`, `liked_by_me`, `read` and its `replies[]`. A first comment kept only for its replies has `deleted: "author" \| "moderator"` or, for one the caller reported, `hidden: "reported"`, with no `user_id`, `username`, `body` or likes. `403` for non-members. |
| POST | `/room/{id}/book/{bookId}/comments` | Yes | Create a comment. Body: `{ page, anchor_text, anchor_bounds, parent_id, body, client_id, file_hash }`. `page` is 0-based; omit `anchor_text` for a page-level comment. Sending `parent_id` makes it a reply, which inherits the root's page, anchor and room — values in the body are ignored. Also advances the author's `furthest_page` to `GREATEST(existing, page)`. `201`; `400` nested reply, bad body, or blocked words; `403` non-member; `404` unknown room, book or parent, or a parent that was deleted. |
| POST | `/room/{id}/book/{bookId}/comments/read` | Yes | Mark comments read. Body: `{ comment_ids: [] }`; ids that aren't comments on this book in this room are dropped. Returns `204`. |
| DELETE | `/comments/{commentId}` | Yes | Delete your own comment. With replies, it stays as a placeholder (`deleted: "author"`, no body, author or likes); without, it's removed, along with a placeholder it was the last reply under. `204`; `403` not the author; `404` unknown or already deleted. |
| POST | `/comments/{commentId}/like` | Yes | Like a comment. Idempotent. Membership is checked through the comment's own room. Returns `204`; `404` for a deleted placeholder. |
| DELETE | `/comments/{commentId}/like` | Yes | Remove a like. Idempotent. Returns `204`. |

Comments key on the **room and the book together**: the same book read in two
rooms is two conversations, and a comment never follows a reader into a room
its author didn't join. Everything sharing an `anchor_key` — derived
server-side from the page plus the normalized selected text — is one thread and
one gutter dot, so two people highlighting the same sentence land in the same
conversation. Replies are one level deep; a reply to a reply is a `400`.

Visibility is decided here, not in the client. A comment is unlocked when its
page is at or before the caller's `reading_progress.furthest_page`, or when the
caller wrote it. Everything still ahead of them contributes only to
`locked_count` — no id, no page, no preview — so the response holds nothing to
read ahead with. `furthest_page` is used rather than `current_page` so that
flipping back to re-read an earlier passage can't re-lock what a reader has
already been shown. See `scripts/migrate_book_comments.sql`.

**Blocks and reports.** Nothing by someone the caller blocked is sent to them: not their threads (with every reply), not their replies elsewhere, and not in `locked_count` or the unread counts (nor in `newly_unlocked` on `PUT /progress/{bookId}`). A comment the caller reported is gone for them too: a reply entirely, a first comment down to its `hidden: "reported"` placeholder while it has replies. A placeholder is never unread or locked. See `scripts/migrate_report_block.sql`.

`file_hash` is the fingerprint of the PDF the anchor was taken from. Two files
can share a `book_id`, so the reader compares this against the document it
actually opened and declines to draw anchors from a different edition.

---

## Reports

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/reports` | Yes | Report a comment, reader or room. Body: `{ target_type: "comment" \| "user" \| "room", target_id, room_id?, reason, details? }`. `reason` is `harassment`, `hate`, `sexual`, `violence`, `self_harm`, `spam` or `other`; `details` (≤ 500 characters) is required for `other`. The target must be something the caller can see: a live comment in their room, a reader they share a room with (`room_id`, if sent, one of those rooms), a room they're in or an open room. Stores a copy of the text (the comment, the username, or the room's name and description) that outlives it. `201 { id }`; `200 { status: "already_reported" }` if the caller already has an open report on it (no second alert); `400` bad reason or details, or your own content, yourself or the Deleted reader; `404` not visible; `429` after 30 in 24 hours. Every admin is notified (see below). |

---

## Reading progress

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/progress` | Yes | Every book the caller has a position in, most recently read first, with the book's title, cover and manuscript url. |
| PUT | `/progress/{bookId}` | Yes | Publish where the caller is in a book. Body: `{ current_page, total_pages, read_at }`. `current_page` is 0-based. A `total_pages` of 0 keeps the length already known. `read_at` (RFC 3339, optional) is when the page was read; the app sends it because saves made offline arrive late. A save read before the stored `last_read_at` keeps the stored page, length and time. Missing, unparseable or future `read_at` means now. `furthest_page` only climbs, late saves included. Returns the stored row with `progress_pct` and `newly_unlocked` (comments the save carried the reader past). `200`; `400` negative pages; `404` unknown book. |
| GET | `/room/{id}/progress` | Yes | Every member's place in the room's current book, including members who haven't opened it (page 0). Admin first. `403` non-member. |

## Highlights

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/books/{bookId}/highlights` | Yes | The caller's own highlights on one book, ordered by page. Returns an array of `{ id, book_id, page, anchor_text, anchor_bounds, file_hash, client_id, created_at }`; `[]` when there are none. |
| POST | `/books/{bookId}/highlights` | Yes | Create a highlight. Body: `{ page, anchor_text, anchor_bounds, file_hash, client_id }`. `page` is 0-based and `anchor_text` is required (trimmed, capped at 1000 chars). Idempotent on `client_id`: a retry returns the first row. `201`; `400` empty text or negative page; `404` unknown book. |
| DELETE | `/highlights/{highlightId}` | Yes | Delete one of the caller's highlights. `204`; `404` if it doesn't exist or belongs to someone else. |

Highlights are **personal**: they key on the reader and the book, never a
room, and are only returned to their author. The anchor has the same shape as a
comment's, so the reader draws both the same way. Schema:
`scripts/migrate_book_highlights.sql`.

---

## Middleware

All routes have the following middleware applied:

- **CORS** — Handles cross-origin requests and preflight `OPTIONS` responses.
- **Logging** — Logs each incoming request.

---

## Project Structure

```
cmd/server/main.go          — Entry point, router setup
internal/handlers/           — Route handlers by domain
internal/middleware/          — CORS and logging middleware
internal/models/             — Data models / request types
internal/config/             — Environment configuration
internal/database/           — PostgreSQL connection
internal/notify/             — Inbox writes + push fan-out
internal/utils/              — JWT, hashing, Firebase (storage, FCM), R2 storage helpers
```

---

## Bucket screens (9a / 9b)

Requires `scripts/migrate_buckets_9.sql`.

- `GET /users/me/buckets` — each bucket now also has `finished_count`, `source_curated_id`, and up to 3 `books_preview` covers in the bucket's manual order, and `updated_at` (the latest of creation, rename, or a book added; See all's Updated sort).
- `GET /users/me/buckets/{id}/books` — `{ id, name, source_curated_id, books: [{ book_id, title, author_name, cover_image_url, manuscript_url, subgenre, page_count, progress }] }` in manual order. `progress` is the caller's `{ current_page, total_pages, progress_pct, last_read_at }` or null.
- `POST /users/me/buckets` — also accepts `{ source_curated_id }` ("Save to My Books"): copies the curated bucket's books and order. Saving again returns the existing copy with `existing: true` (200).
- `GET /home/our-picks` and `/discover` curated items — add `description`, `genre_tags`, `reading_minutes` (1.5 min/page; null unless every book's length is known), `saved_bucket_id`, and under a genre filter `matching_count`.
- `GET /home/our-picks/{bucketId}/books` — `{ id, title, description, genre_tags, book_count, reading_minutes, saved_bucket_id, books: [...] }`, books shaped as above.

**Genre tags.** A curated bucket's `genre_tags` are the editor's (`curated_buckets.genre_tags`) when set; otherwise any subgenre that at least 2 books, or at least 40% of the books, share. Under `/discover?genre=X`, a bucket shows if tagged X, ranked by the share of its books in X.
