-- Migration: book metadata must describe the book's own file (Turn 11 data fix).
-- Run after migrate_buckets_9.sql, before deploying the API that reads these
-- columns. Safe to re-run, but the reset below redoes every lookup each time.
--
-- The first metadata job took the first search result whose title matched,
-- so a generic title got some other book's author: a Hecht "Quantum
-- Mechanics" cover was captioned "Leonard Susskind", a Charu C. Aggarwal
-- "Data Mining" cover "Mehmed Kantardzic". The job now only takes an author
-- it can tie to the file (see internal/metadata pick), and these columns
-- record which values it wrote, so a recheck can clear them without touching
-- an uploader's author or the page count a reader's PDF reported.

ALTER TABLE public.books ADD COLUMN IF NOT EXISTS author_from_lookup boolean NOT NULL DEFAULT false;
ALTER TABLE public.books ADD COLUMN IF NOT EXISTS pages_from_lookup boolean NOT NULL DEFAULT false;

-- ── Backfill: what did the old job write? ──────────────────────
-- Seeded books (no uploader) had no author until the lookup ran, so any
-- author on a checked seeded book is the lookup's.
UPDATE public.books
   SET author_from_lookup = true
 WHERE user_id IS NULL
   AND metadata_checked_at IS NOT NULL
   AND author_name IS NOT NULL;

-- A page count is the file's own when a reader's PDF reported it. Prefer
-- that; otherwise a checked book's count came from the lookup.
UPDATE public.books b
   SET page_count = rp.pages, pages_from_lookup = false
  FROM (SELECT book_id, MAX(total_pages) AS pages
          FROM public.reading_progress
         WHERE total_pages > 0
         GROUP BY book_id) rp
 WHERE rp.book_id = b.book_id;

UPDATE public.books b
   SET pages_from_lookup = true
 WHERE b.metadata_checked_at IS NOT NULL
   AND b.page_count IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM public.reading_progress p
                    WHERE p.book_id = b.book_id AND p.total_pages > 0);

-- ── Redo every lookup under the stricter matcher ───────────────
-- Titles are left as they are (the job rewrites them from source_title).
UPDATE public.books
   SET author_name = CASE WHEN author_from_lookup THEN NULL ELSE author_name END,
       author_from_lookup = false,
       page_count = CASE WHEN pages_from_lookup THEN NULL ELSE page_count END,
       pages_from_lookup = false,
       metadata_checked_at = NULL;
