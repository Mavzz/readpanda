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
| POST | `/signup` | No | Register a new user. Body: `{ username, email, password }` (password is AES-encrypted from frontend). Returns access token, refresh token, and user preferences. |
| POST | `/auth/login` | No | Login with email/username and password. Returns access token, refresh token, and user data. |
| POST | `/auth/google` | No | Authenticate via Google ID token. Body: `{ token }`. Creates account on first login. Returns access/refresh tokens. |
| POST | `/auth/logout` | Yes | Logout the current user. |
| POST | `/token/refresh` | No | Refresh an expired access token. Body: `{ refresh_token }`. Returns a new access token. |

---

## Users

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/users` | No | List all users. Returns `[ { id, username, email, isactive, login_type, uuid, created_at } ]`. |

---

## Books

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/books/upload-urls` | Yes | Start a publish. Body `{ manuscript: { name, size }, cover?: { name, size } }`. Returns `{ book_id, manuscript: { key, url, content_type }, cover? }`: PUT each file to its `url` with that `Content-Type` header (URLs last 1 hour). Manuscript: PDF or EPUB, up to 500 MB. Cover: JPEG, PNG, WebP or GIF, up to 20 MB. |
| POST | `/books/upload` | Yes | Publish a new book. **JSON** (after `/books/upload-urls`): `book_id`, `title`, `description`, `genre`, `subgenre`, `author_name`, `notify` (default true), `manuscript_key`, `cover_key?`. Checks the files are in storage within the limits. Still accepts the old **multipart form** with `cover` and `manuscript` files, but Cloud Run rejects those bodies over 32 MB. |
| GET | `/books` | Yes | Get all books for the authenticated user. Returns `{ books: [...] }`. |
| GET | `/books/all` | Yes | Get all books in the system. Returns `{ books: [...] }`. |
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
| GET | `/notifications` | Yes | The caller's notifications, newest first. Returns `[ { id, user_id, type, title, message, book_id, is_read, created_at } ]`, or `[]` when empty. `type` is `SYSTEM` or `NEW_BOOK`. For `NEW_BOOK`, `book_id` names the book to open (null if it was deleted). |
| GET | `/notifications/unread/count` | Yes | Returns `{ unread_count: <int> }`. |
| PUT | `/notifications/{id}/read` | Yes | Mark one notification read. Idempotent. `204`, or `404` if it isn't the caller's. |
| POST | `/users/me/devices` | Yes | Register an FCM token for push. Body: `{ token, platform }`, where `platform` is `ios` or `android`. Upserts on the token, so a phone that switches accounts moves to the new one. `204`. |
| DELETE | `/users/me/devices/{token}` | Yes | Unregister a device (URL-encode the token). Only removes the caller's own token. Idempotent. `204`. |

### Where notifications come from

`internal/notify` writes the inbox row, then pushes to the recipient's registered devices in the background. The row is the record: if push is unconfigured or a send fails, the notification is still in the inbox. Tokens that FCM reports as unregistered are deleted.

| Trigger | Recipients | Notification |
|---------|------------|--------------|
| `POST /books/upload` with a manuscript | Every user except the publisher | `NEW_BOOK`, "New book added", `book_id` set |

Push payload `data`: `{ type, notification_id, book_id? }`.

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

---

## Rooms

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/room/create` | Yes | Create a room. Body: `{ name, description, is_private }`. Every room gets an invite code, and the creator is enrolled as an `admin` member. |
| GET | `/room/my-rooms` | Yes | Rooms the user created or joined. Returns a bare array. |
| POST | `/room/join` | Yes | Join a room by code. Body: `{ invite_code }`. `404` unknown code, `409` already a member. |
| GET | `/room/{id}` | Yes | Full Room Detail: the room, `members[]` (`user_id`, `username`, `role`, `joined_at`), `current_book`, and `bucket` (`{ id, name, type, books[] }`). `403` for non-members. |
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
| GET | `/room/{id}/book/{bookId}/comments` | Yes | The comment layer for one book in one room. Returns `{ room_id, book_id, furthest_page, threads[], locked_count, unlocked_unread_count }`. Each thread is `{ anchor_key, page, anchor_text, anchor_bounds, file_hash, comments[], unread_count }`, and each comment carries `likes`, `liked_by_me`, `read` and its `replies[]`. `403` for non-members. |
| POST | `/room/{id}/book/{bookId}/comments` | Yes | Create a comment. Body: `{ page, anchor_text, anchor_bounds, parent_id, body, client_id, file_hash }`. `page` is 0-based; omit `anchor_text` for a page-level comment. Sending `parent_id` makes it a reply, which inherits the root's page, anchor and room — values in the body are ignored. Also advances the author's `furthest_page` to `GREATEST(existing, page)`. `201`; `400` nested reply or bad body; `403` non-member; `404` unknown room, book or parent. |
| POST | `/room/{id}/book/{bookId}/comments/read` | Yes | Mark comments read. Body: `{ comment_ids: [] }`; ids that aren't comments on this book in this room are dropped. Returns `204`. |
| POST | `/comments/{commentId}/like` | Yes | Like a comment. Idempotent. Membership is checked through the comment's own room. Returns `204`. |
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

`file_hash` is the fingerprint of the PDF the anchor was taken from. Two files
can share a `book_id`, so the reader compares this against the document it
actually opened and declines to draw anchors from a different edition.

---

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
