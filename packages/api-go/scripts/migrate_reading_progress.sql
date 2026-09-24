-- Migration: per-member reading progress.
--
-- The pace track on Home, Reading and Room Book ("you're on page 40, Priya is
-- on 70") had no backing table, so the mobile app rendered a fixture. Progress
-- is personal and keys on the BOOK, not the room: the same position follows a
-- reader whether they read the book alone, in one room, or in several.
-- A room's pace track is therefore a join, not its own store.
-- Run against your PostgreSQL database.

CREATE TABLE IF NOT EXISTS public.reading_progress (
    user_id uuid NOT NULL,
    book_id character varying(50) NOT NULL,
    -- The page the reader is ON (0-based), matching the mobile reader's
    -- position, and how many pages the book turned out to have. total_pages
    -- is 0 until the reader opens the book and the PDF reports its length.
    current_page integer NOT NULL DEFAULT 0,
    total_pages integer NOT NULL DEFAULT 0,
    last_read_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);

ALTER TABLE public.reading_progress DROP CONSTRAINT IF EXISTS reading_progress_user_id_fkey;
ALTER TABLE public.reading_progress
    ADD CONSTRAINT reading_progress_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

ALTER TABLE public.reading_progress DROP CONSTRAINT IF EXISTS reading_progress_book_id_fkey;
ALTER TABLE public.reading_progress
    ADD CONSTRAINT reading_progress_book_id_fkey FOREIGN KEY (book_id)
    REFERENCES public.books(book_id) ON DELETE CASCADE;

ALTER TABLE public.reading_progress DROP CONSTRAINT IF EXISTS reading_progress_pages_check;
ALTER TABLE public.reading_progress
    ADD CONSTRAINT reading_progress_pages_check
    CHECK (current_page >= 0 AND total_pages >= 0);

-- The room pace track reads every member's row for one book, so the lookup
-- runs book-first; the primary key only serves user-first lookups.
CREATE INDEX IF NOT EXISTS reading_progress_book_id_idx
    ON public.reading_progress (book_id);
