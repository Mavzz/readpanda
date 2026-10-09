# ReadPanda Go API

The backend for the ReadPanda mobile app and the writer portal. A single Go
service (`cmd/server`) on PostgreSQL, with book files in S3-compatible object
storage (Cloudflare R2 in production, MinIO locally) and push notifications
through Firebase Cloud Messaging.

| | Local | Production |
|---|---|---|
| API | `go run` on your Mac, port 3000 | Google Cloud Run, `asia-south1` |
| Database | PostgreSQL (Docker or Homebrew) | Supabase, session pooler |
| Files | MinIO (Docker) or the shared R2 bucket | Cloudflare R2 |
| Push | Optional | Firebase service account from Secret Manager |

- Every route: [ROUTES.md](ROUTES.md)
- Deploying and operating production: [DEPLOYMENT.md](DEPLOYMENT.md)
- The mobile app: [readpanda-mobile](https://github.com/Mavzz/readpanda-mobile), setup in its `docs/RUN.md`

## Run it locally

### 1. Prerequisites

- Go 1.24+ (`brew install go`)
- PostgreSQL client tools, for `psql` (`brew install libpq`, or a full `brew install postgresql`)
- Docker, if you want Postgres and MinIO in containers

### 2. Start Postgres (and MinIO)

The compose file in `packages/api` runs both, with credentials that match
`.env.local.example`:

```bash
cd ../api && docker compose up -d postgres minio
```

This gives you Postgres at `localhost:5432` (user `readpanda`, password
`readpandapostgres`, database `ReadPanda`) and MinIO at `localhost:9000`, with
its console at http://localhost:9001 (`minioadmin` / `minioadmin`). In the
console, create a bucket named `readpanda-books` and set its access policy to
public, so the app can load covers and manuscripts from it.

A Homebrew Postgres works too; put its user, password and database in
`.env.local` instead.

### 3. Create the schema

`scripts/readpanda_schema.sql` is the complete, current schema. Run it, then
the two seed files, in this order:

```bash
export PGPASSWORD=readpandapostgres
psql -h localhost -U readpanda -d ReadPanda -v ON_ERROR_STOP=1 -f scripts/readpanda_schema.sql
psql -h localhost -U readpanda -d ReadPanda -v ON_ERROR_STOP=1 -f scripts/readpanda_seed_preferences.sql
psql -h localhost -U readpanda -d ReadPanda -v ON_ERROR_STOP=1 -f scripts/readpanda_seed_books.sql   # optional
```

- `readpanda_seed_preferences.sql` holds the genre and subgenre list. Without
  it the onboarding interest picker is empty.
- `readpanda_seed_books.sql` adds six sample books and two "Our Picks"
  collections. Their files live in the production R2 bucket, so they only open
  when your `R2_*` settings point there. With a local MinIO, skip it and upload
  books through the portal or `POST /books/upload` instead.
- The `migrate_*.sql` files and `init.sql` are the history of how the schema
  got here. They are already folded into `readpanda_schema.sql`; don't run
  them on a new database.

### 4. Configure

```bash
cp .env.local.example .env.local
```

The defaults work against the Docker setup above. The server refuses to start
without `JWT_SECRET` and `JWT_REFRESH_SECRET`; any string will do locally.

Ask a maintainer for the values you can't make yourself:

- `GOOGLE_CLIENT_ID` / `GOOGLE_IOS_CLIENT_ID`: needed for Google sign-in. Email
  sign-up works without them.
- `serviceAccountKey.json` plus `FIREBASE_SERVICE_ACCOUNT_PATH=./serviceAccountKey.json`:
  needed for push. Without it the server logs `Push notifications disabled`
  and everything else works.

`.env.local`, `serviceAccountKey.json` and `env.yaml` are gitignored. Never
commit them.

### 5. Run

```bash
go run ./cmd/server
```

A healthy start logs:

```
Connecting to database ReadPanda at localhost:5432 (sslmode=disable)
Connected to PostgreSQL
Object storage (R2/MinIO) initialised — endpoint: http://localhost:9000, bucket: readpanda-books
App running on 0.0.0.0:3000 HTTP and port 3000...
```

Check it responds (a 401 is correct: the route needs a login):

```bash
curl -i http://localhost:3000/api/v1/genres
```

The iOS Simulator shares your Mac's network, so a simulator build of the app
reaches this server at `localhost:3000`. See the mobile repo's `docs/RUN.md`.

## Working on the code

```
cmd/server/main.go     Entry point: config, connections, start the server
internal/config        Environment variables (.env.local locally)
internal/database      PostgreSQL connection
internal/handlers      One file per area: users, books, rooms, comments, ...
internal/middleware    CORS, request logging, auth
internal/models        Request and response types
internal/notify        Inbox writes and push fan-out
internal/server        The router: every route and its middleware
internal/testdb        Throwaway Postgres databases for tests
internal/utils         JWT, passwords, R2, Firebase, FCM
scripts/               Schema, seeds and migration history
```

Before you push:

```bash
gofmt -l .        # should print nothing
go vet ./...
go test ./...
go build ./...
```

### Tests

Unit tests sit next to the code they cover (`*_test.go`). The handler tests in
`internal/server` drive the real router against a real PostgreSQL: each run
creates a fresh database from `scripts/readpanda_schema.sql`, empties every
table between tests, and drops the database at the end. Point them at any
server you can create databases on:

```bash
TEST_DATABASE_URL="postgres://localhost:5432/postgres?sslmode=disable" go test ./...
```

Without `TEST_DATABASE_URL` those tests are skipped and the rest still run.
Add `-v` to see the request log. CI (`.github/workflows/ci.yml`) runs the full
suite on every pull request, and the deploy workflow won't ship if it fails —
so a migration that isn't folded into `readpanda_schema.sql` shows up as a
failing test rather than a broken fresh database.

### Changing the schema

1. Write the change as a new, re-runnable `scripts/migrate_<what>.sql` (use
   `IF NOT EXISTS` and similar guards).
2. Run it on your local database, and on Supabase before deploying code that
   depends on it (see [DEPLOYMENT.md](DEPLOYMENT.md#schema-changes)).
3. Regenerate the full schema so new databases get the change:
   ```bash
   pg_dump -h localhost -U readpanda -d ReadPanda --schema-only --no-owner --no-privileges -n public \
     -f scripts/readpanda_schema.sql
   ```
   Then remove the `\restrict`/`\unrestrict` lines, `CREATE SCHEMA public`,
   its `COMMENT`, and `SET transaction_timeout`, and re-add the row-level
   security block from the end of the current file. The Supabase SQL Editor
   and older Postgres versions reject the first four.

### Errors and logs

The request logger prints one line per request with its status and duration.
When a handler returns a 5xx, the line starts with `ERROR` and includes the
error message the handler sent to the client:

```
POST /api/v1/auth/google -> 201 (275ms)
ERROR POST /api/v1/auth/google -> 500 (17ms): {"error": "pq: column \"id\" does not exist"}
```

So you don't need to add logging for a failure to be visible. Return it with
`http.Error` as the handlers already do.

### Things that will bite you

- **Admin-only routes check the role themselves.** There is no router-level
  admin gate. Call `utils.RequireAdmin` at the top of the handler, as
  `GetUsers` and the `/admin/*` handlers do.
- **`users` has no `id` column.** Its key is `uuid`. `models.User.ID` exists
  but is never filled from the database.
- **`PG_SSLMODE`** defaults to `disable`. Hosted Postgres (Supabase) needs
  `require`.
- **`CRYPTO_SECRET`** is in the env templates but nothing reads it. Passwords
  are bcrypt-hashed on the server.
