# Deploying the Go API

Production runs on **Google Cloud Run**, with the database on **Supabase** and
book files on **Cloudflare R2**. This covers redeploying, first-time setup,
schema changes and troubleshooting.

| | Value |
|---|---|
| Service URL | https://readpanda-backend-439290157125.asia-south1.run.app |
| GCP project | `reanpanda-d3beb` (also the Firebase project) |
| Cloud Run service / region | `readpanda-backend` / `asia-south1` (Mumbai) |
| Database | Supabase, `ap-south-1`, via the session pooler |
| Firebase key | Secret Manager secret `firebase-sa` |
| Env vars | `env.yaml`, local only, never committed |

The API and the database are both in Mumbai on purpose: every request makes
several database round trips, and keeping them in one region keeps each one to
a millisecond or two.

## Redeploy

You need `gcloud` logged in with access to the project, and a filled-in
`env.yaml` in this directory (ask a maintainer, or build one from the template
under [env.yaml](#envyaml) below).

```bash
cd packages/api-go
gcloud config set project reanpanda-d3beb

gcloud run deploy readpanda-backend \
  --source . \
  --region asia-south1 \
  --allow-unauthenticated \
  --set-build-env-vars GOOGLE_BUILDABLE=./cmd/server \
  --env-vars-file env.yaml \
  --set-secrets=/secrets/firebase/sa.json=firebase-sa:latest
```

Cloud Run builds the code in Google Cloud (you don't need Docker), then
switches traffic to the new revision once it starts. The first build takes a
few minutes; later ones are faster.

What each flag is for:

- `--source .` uploads this directory. **`.gcloudignore` decides what's left
  out**: secrets, local binaries and `scripts/`. Without it gcloud would fall
  back to `.gitignore`, whose `server` pattern also matches `cmd/server/`, and
  the build would fail.
- `GOOGLE_BUILDABLE=./cmd/server` tells the Go buildpack where `main` is.
- `--env-vars-file env.yaml` replaces **all** env vars on each deploy. A value
  missing from the file is gone from the service.
- `--set-secrets=...` mounts the Firebase service account as a file at
  `/secrets/firebase/sa.json`, which `FIREBASE_SERVICE_ACCOUNT_PATH` points to.

### Check the deploy

```bash
URL=https://readpanda-backend-439290157125.asia-south1.run.app/api/v1
curl -s -o /dev/null -w "%{http_code}\n" $URL/genres              # 401: up, needs login
curl -s -X POST $URL/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"nobody","password":"x"}'                        # "Invalid username or password": database reachable
```

Then check the startup lines in the logs (below): `Connected to PostgreSQL`,
`Object storage (R2/MinIO) initialised` and `Push notifications enabled`.

## Logs

```bash
# Follow live
gcloud run services logs tail readpanda-backend --region asia-south1

# Recent failures only
gcloud logging read 'resource.labels.service_name="readpanda-backend" AND textPayload:"ERROR"' \
  --limit 20 --freshness 1d --format='value(timestamp,textPayload)'
```

Or open Cloud Run console, `readpanda-backend`, **Logs**. Request lines look
like `POST /api/v1/auth/google -> 201 (275ms)`. Server errors start with
`ERROR` and include the cause.

The `run.googleapis.com/requests` log only has status codes. The cause is in
the service's own output, `stderr`.

## env.yaml

YAML, not `.env` syntax, and every value quoted (Cloud Run only accepts
strings). Leave out `PORT`, which Cloud Run sets.

```yaml
API_VERSION: "/api/v1"

# Supabase: Connect button, "Session pooler"
PG_HOST: "aws-0-ap-south-1.pooler.supabase.com"
PG_PORT: "5432"
PG_USER: "postgres.<project-ref>"
PG_PASSWORD: "..."
PG_DB: "postgres"
PG_SSLMODE: "require"

# Random, and different from local: openssl rand -base64 48
JWT_SECRET: "..."
JWT_REFRESH_SECRET: "..."

GOOGLE_CLIENT_ID: "....apps.googleusercontent.com"
GOOGLE_IOS_CLIENT_ID: "....apps.googleusercontent.com"

FIREBASE_SERVICE_ACCOUNT_PATH: "/secrets/firebase/sa.json"

R2_ENDPOINT: "https://<account-id>.r2.cloudflarestorage.com"
R2_ACCESS_KEY_ID: "..."
R2_SECRET_ACCESS_KEY: "..."
R2_BUCKET_NAME: "readpanda"
R2_PUBLIC_URL: "https://pub-<id>.r2.dev"
```

Changing `JWT_SECRET` or `JWT_REFRESH_SECRET` signs every user out. Nothing
else happens, so rotate them if they might have leaked.

## Schema changes

Supabase doesn't run migrations for you. For a change that new code depends on:

1. Run the new `scripts/migrate_<what>.sql` in the Supabase **SQL Editor**
   first. Migrations are written to be re-runnable, and additive changes are
   safe for the code already running.
2. Then deploy the code.
3. Regenerate `scripts/readpanda_schema.sql` as described in the
   [README](README.md#changing-the-schema).

The SQL Editor only accepts plain SQL. `psql` commands (`\restrict`,
`COPY ... FROM stdin`) fail there. To move data, dump with
`pg_dump --data-only --inserts --on-conflict-do-nothing`.

## First-time setup

For rebuilding the environment from scratch, such as a staging copy or a new
project. The steps are in dependency order.

1. **Google Cloud project.** Reuse the Firebase project or create one
   (`gcloud projects create`), link a billing account in the console, then:
   ```bash
   gcloud services enable run.googleapis.com cloudbuild.googleapis.com \
     artifactregistry.googleapis.com secretmanager.googleapis.com compute.googleapis.com
   ```
2. **Database.** Create a Supabase project in the region you'll deploy to.
   In the SQL Editor, run `scripts/readpanda_schema.sql`, then
   `scripts/readpanda_seed_preferences.sql`, and optionally
   `scripts/readpanda_seed_books.sql`. Table Editor should list 18 tables, each
   marked "RLS enabled".
3. **Object storage.** Create an R2 bucket, enable its public `r2.dev` URL, and
   create an API token with read and write access to it.
4. **Firebase key.** Store the service account JSON, and let Cloud Run's
   runtime account read it:
   ```bash
   gcloud secrets create firebase-sa --data-file=serviceAccountKey.json
   PROJECT_NUMBER=$(gcloud projects describe $(gcloud config get project) --format='value(projectNumber)')
   gcloud secrets add-iam-policy-binding firebase-sa \
     --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
     --role=roles/secretmanager.secretAccessor
   ```
5. **env.yaml.** Fill it in as above.
6. **Deploy** with the command under [Redeploy](#redeploy). Answer yes if it
   asks to create an Artifact Registry repository.
7. **Point the app at it.** In the mobile repo, set the `BACKEND_URL` GitHub
   secret to the service URL, with no `/api/v1`, then run its build workflow.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Build fails with no Go files or `cmd/server` not found | `.gcloudignore` is missing, so `.gitignore`'s `server` rule excluded `cmd/server/`. |
| `Failed to connect to database` at startup | Using Supabase's direct host (`db.<ref>.supabase.co`), which is IPv6-only and unreachable from Cloud Run. Use the session pooler host, `postgres.<ref>` as the user, and `PG_SSLMODE: "require"`. |
| `--env-vars-file` rejects the file | It's in `KEY=value` form or has unquoted numbers. Use `KEY: "value"`. |
| `service account file not found` | The `--set-secrets` flag was left off, or the runtime account lacks `secretAccessor` on `firebase-sa`. |
| `Push notifications disabled` in the logs | Same as the row above. |
| Any 5xx from the app | Search the logs for `ERROR`; the line has the cause. |
| Everything fails after a quiet week | Free Supabase projects pause after about 7 days without activity. Restore it from the Supabase dashboard. |
