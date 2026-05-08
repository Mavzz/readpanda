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
| POST | `/books/upload` | Yes | Publish a new book. **Multipart form** with fields: `title`, `description`, `genre`, `subgenre`, and files: `cover` (image), `manuscript`. Max 32 MB. |
| GET | `/books` | Yes | Get all books for the authenticated user. Returns `{ books: [...] }`. |
| GET | `/books/all` | Yes | Get all books in the system. Returns `{ books: [...] }`. |
| POST | `/books/seed` | Yes | Seed books from object storage (R2/MinIO). Scans the storage bucket and inserts missing books into the database. |

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

| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| GET | `/notifications?username={username}` | Yes | Get all notifications for a user, ordered by newest first. Returns `[ { id, user_id, message, is_read, created_at } ]`. |
| GET | `/notifications/unread/count?username={username}` | Yes | Get the count of unread notifications. Returns `{ unread_count: <int> }`. |

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
internal/utils/              — JWT, hashing, Firebase, R2 storage helpers
```
