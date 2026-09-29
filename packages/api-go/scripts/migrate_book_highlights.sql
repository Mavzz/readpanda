-- Migration: personal highlights.
--
-- A highlight is a passage the reader marked for themselves, with no comment
-- attached. Unlike book_comments it keys on the READER and the BOOK only — no
-- room. Highlights are private: nobody else ever sees them, and they follow
-- the reader into every room and into solo reading of the same book.
--
-- The anchor has the same shape as a comment's, for the same reasons: the page
-- is the only positional fact both ends agree on for an opaque PDF, the text
-- is what lets the passage be found again, and anchor_bounds is the fallback
-- for when the text has moved. page is 0-based.
-- Run against your PostgreSQL database.

CREATE TABLE IF NOT EXISTS public.book_highlights (
    id character varying(50) NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL,
    book_id character varying(50) NOT NULL,
    -- Which edition the anchor was taken from; the reader leaves highlights
    -- from a different file undrawn rather than marking the wrong words.
    file_hash character varying(64),
    page integer NOT NULL DEFAULT 0,
    anchor_text text NOT NULL,
    -- Per-line rects in normalized page space. Opaque to the server.
    anchor_bounds jsonb,
    -- Minted on the phone before the request leaves it, so a retried POST
    -- returns the first write instead of adding a duplicate.
    client_id character varying(64),
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.book_highlights DROP CONSTRAINT IF EXISTS book_highlights_user_id_fkey;
ALTER TABLE public.book_highlights
    ADD CONSTRAINT book_highlights_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

ALTER TABLE public.book_highlights DROP CONSTRAINT IF EXISTS book_highlights_book_id_fkey;
ALTER TABLE public.book_highlights
    ADD CONSTRAINT book_highlights_book_id_fkey FOREIGN KEY (book_id)
    REFERENCES public.books(book_id) ON DELETE CASCADE;

ALTER TABLE public.book_highlights DROP CONSTRAINT IF EXISTS book_highlights_page_check;
ALTER TABLE public.book_highlights
    ADD CONSTRAINT book_highlights_page_check CHECK (page >= 0);

ALTER TABLE public.book_highlights DROP CONSTRAINT IF EXISTS book_highlights_anchor_text_check;
ALTER TABLE public.book_highlights
    ADD CONSTRAINT book_highlights_anchor_text_check
    CHECK (char_length(anchor_text) BETWEEN 1 AND 1000);

-- The reader loads one person's highlights on one book, in page order.
CREATE INDEX IF NOT EXISTS book_highlights_user_book_idx
    ON public.book_highlights (user_id, book_id, page);

-- Scoped to the author, and partial so rows without a client id don't all
-- collide on NULL.
CREATE UNIQUE INDEX IF NOT EXISTS book_highlights_user_client_idx
    ON public.book_highlights (user_id, client_id) WHERE client_id IS NOT NULL;
