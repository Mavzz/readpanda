-- Migration: Discover (8a) and Book detail (8c).
--
-- Book detail's meta line is "{author} · {pages} pages · {genre}". Books had
-- a genre but no author or length, so both are added here, nullable: a book
-- the catalogue doesn't know them for leaves them out of the line rather than
-- showing "Unknown".
-- Run against your PostgreSQL database BEFORE deploying the API that reads
-- these columns.

ALTER TABLE public.books ADD COLUMN IF NOT EXISTS author_name character varying(255);
ALTER TABLE public.books ADD COLUMN IF NOT EXISTS page_count integer;

-- A PDF reports its length the first time anyone opens it, and that length is
-- already stored per reader in reading_progress.total_pages. Backfill from it;
-- PutMyProgress keeps filling it in for books opened from now on.
UPDATE public.books b
   SET page_count = rp.pages
  FROM (SELECT book_id, MAX(total_pages) AS pages
          FROM public.reading_progress
         WHERE total_pages > 0
         GROUP BY book_id) rp
 WHERE b.book_id = rp.book_id
   AND b.page_count IS NULL;

-- "Popular this week" counts readers per book over the last seven days.
CREATE INDEX IF NOT EXISTS reading_progress_last_read_at_idx
    ON public.reading_progress (last_read_at);

-- Book detail asks "which of my buckets is this book in?", which looks up
-- book-first; the primary key only serves bucket-first lookups.
CREATE INDEX IF NOT EXISTS user_bucket_books_book_id_idx
    ON public.user_bucket_books (book_id);
