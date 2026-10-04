-- Migration: bucket detail screens (9a My bucket, 9b Curated bucket) and the
-- book metadata lookup. Run after migrate_discover.sql, before deploying the
-- API that reads these columns. Safe to re-run.

-- ── Book metadata ──────────────────────────────────────────────
-- Seeded titles are file names ("DataScienceAndPredictiveAnalyt"). The
-- metadata job rewrites `title` in place with the resolved one, so every
-- endpoint gets a readable title without each query having to choose. The
-- original is kept in source_title. metadata_checked_at marks a book the job
-- has already looked up, found or not, so it isn't asked about again.
ALTER TABLE public.books ALTER COLUMN title TYPE character varying(255);
ALTER TABLE public.books ADD COLUMN IF NOT EXISTS source_title character varying(255);
ALTER TABLE public.books ADD COLUMN IF NOT EXISTS metadata_checked_at timestamp without time zone;
UPDATE public.books SET source_title = title WHERE source_title IS NULL;

-- ── Curated buckets ────────────────────────────────────────────
-- description: the editorial line under 9b's name (written in the CMS).
-- genre_tags: editor-set subgenre tags. When empty, tags are derived from
-- the books (see discover.go curatedTags), so a bucket isn't untagged just
-- because nobody has tagged it yet.
ALTER TABLE public.curated_buckets ADD COLUMN IF NOT EXISTS description text;
ALTER TABLE public.curated_buckets ADD COLUMN IF NOT EXISTS genre_tags text[] NOT NULL DEFAULT '{}';

-- ── User buckets ───────────────────────────────────────────────
-- source_curated_id: set when a user bucket is a "Save to My Books" copy of a
-- curated one, so 9b can show "Saved" and not make a second copy.
ALTER TABLE public.user_buckets ADD COLUMN IF NOT EXISTS source_curated_id character varying(50);
CREATE UNIQUE INDEX IF NOT EXISTS user_buckets_source_curated_idx
    ON public.user_buckets (user_id, source_curated_id)
    WHERE source_curated_id IS NOT NULL;

-- sort_order: 9a's manual order (edit-mode reorder). Existing buckets keep
-- the order books were added in.
ALTER TABLE public.user_bucket_books ADD COLUMN IF NOT EXISTS sort_order integer;
UPDATE public.user_bucket_books ubb
   SET sort_order = o.pos
  FROM (SELECT bucket_id, book_id,
               ROW_NUMBER() OVER (PARTITION BY bucket_id ORDER BY added_at, book_id) - 1 AS pos
          FROM public.user_bucket_books) o
 WHERE ubb.bucket_id = o.bucket_id AND ubb.book_id = o.book_id
   AND ubb.sort_order IS NULL;
ALTER TABLE public.user_bucket_books ALTER COLUMN sort_order SET DEFAULT 0;
