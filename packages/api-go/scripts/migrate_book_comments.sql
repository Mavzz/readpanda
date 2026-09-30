-- Migration: passage-anchored, room-scoped comments (handoff 6a/6b).
--
-- The reader had no way to say anything about what it was showing, and the
-- Room Book screen rendered a fixture. Comments key on the ROOM and the BOOK
-- together: the same book read in two rooms is two conversations, and a
-- comment must never follow a reader into a room its author never joined.
--
-- The anchor is a page plus the selected text. Books here are opaque PDFs
-- with no chapters and no extracted text, so the page is the only positional
-- fact both ends can agree on; the text is what lets the highlight be found
-- again in the document. anchor_bounds is the fallback for when the text has
-- moved. page is 0-based, the same convention as reading_progress.current_page
-- and the mobile reader's own position.
--
-- Whether a comment is visible is decided server-side against the reader's
-- furthest_page, and locked comments are returned as a bare count — never a
-- position, a preview, or a per-page tally. The spoiler rule is not a filter
-- the client applies to data it was already handed.
-- Run against your PostgreSQL database.

CREATE TABLE IF NOT EXISTS public.book_comments (
    id character varying(50) NOT NULL PRIMARY KEY,
    room_id character varying(50) NOT NULL,
    book_id character varying(50) NOT NULL,
    -- Which edition the anchor was taken from. Two PDFs can share a book_id;
    -- an anchor from one must not be drawn onto the other, so the reader
    -- compares this against the hash of the file it actually opened.
    file_hash character varying(64),
    user_id uuid NOT NULL,
    page integer NOT NULL DEFAULT 0,
    -- NULL for a page-level comment: the reader commented on the page without
    -- selecting anything, and 6b omits the quote block for those.
    anchor_text text,
    -- Per-line rects in normalized page space, used only when the text can no
    -- longer be found. Opaque to the server.
    anchor_bounds jsonb,
    -- Everything sharing an anchor is one thread and one gutter dot. Computed
    -- server-side from page + normalized anchor_text so two clients selecting
    -- the same sentence land on the same thread; never accepted from the body.
    anchor_key character varying(64) NOT NULL,
    body text NOT NULL,
    -- One level only: a reply's parent is always a root. Enforced in Go,
    -- since a CHECK constraint cannot look at another row.
    parent_id character varying(50),
    -- Minted by the client before the request leaves the phone. The API client
    -- retries failed POSTs, so without this a flaky network posts the same
    -- comment several times.
    client_id character varying(64),
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_room_id_fkey;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_room_id_fkey FOREIGN KEY (room_id)
    REFERENCES public.rooms(id) ON DELETE CASCADE;

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_book_id_fkey;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_book_id_fkey FOREIGN KEY (book_id)
    REFERENCES public.books(book_id) ON DELETE CASCADE;

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_user_id_fkey;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

-- A deleted root takes its replies with it: a reply with no thread to sit in
-- has nothing to anchor to.
ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_parent_id_fkey;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_parent_id_fkey FOREIGN KEY (parent_id)
    REFERENCES public.book_comments(id) ON DELETE CASCADE;

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_page_check;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_page_check CHECK (page >= 0);

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_body_check;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_body_check
    CHECK (char_length(body) BETWEEN 1 AND 2000);

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_anchor_text_check;
ALTER TABLE public.book_comments
    ADD CONSTRAINT book_comments_anchor_text_check
    CHECK (anchor_text IS NULL OR char_length(anchor_text) <= 1000);

-- The feed reads one room's comments on one book, ordered up the manuscript
-- and grouped into threads.
CREATE INDEX IF NOT EXISTS book_comments_room_book_anchor_idx
    ON public.book_comments (room_id, book_id, page, anchor_key);

CREATE INDEX IF NOT EXISTS book_comments_parent_idx
    ON public.book_comments (parent_id) WHERE parent_id IS NOT NULL;

-- Scoped to the author so one client can't claim another's key. Partial, so
-- rows written without a client id don't all collide on NULL.
CREATE UNIQUE INDEX IF NOT EXISTS book_comments_user_client_idx
    ON public.book_comments (user_id, client_id) WHERE client_id IS NOT NULL;


-- Likes. One row per person per comment, so the count is an aggregate and
-- "did I like this" is a lookup — neither can drift the way a counter column
-- would.
CREATE TABLE IF NOT EXISTS public.book_comment_likes (
    comment_id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (comment_id, user_id)
);

ALTER TABLE public.book_comment_likes DROP CONSTRAINT IF EXISTS book_comment_likes_comment_id_fkey;
ALTER TABLE public.book_comment_likes
    ADD CONSTRAINT book_comment_likes_comment_id_fkey FOREIGN KEY (comment_id)
    REFERENCES public.book_comments(id) ON DELETE CASCADE;

ALTER TABLE public.book_comment_likes DROP CONSTRAINT IF EXISTS book_comment_likes_user_id_fkey;
ALTER TABLE public.book_comment_likes
    ADD CONSTRAINT book_comment_likes_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;


-- Read receipts. Per comment rather than a per-thread watermark, because
-- opening a thread clears the dots for exactly the comments it contains —
-- a later reply to the same thread has to come back unread.
CREATE TABLE IF NOT EXISTS public.book_comment_reads (
    user_id uuid NOT NULL,
    comment_id character varying(50) NOT NULL,
    read_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, comment_id)
);

ALTER TABLE public.book_comment_reads DROP CONSTRAINT IF EXISTS book_comment_reads_user_id_fkey;
ALTER TABLE public.book_comment_reads
    ADD CONSTRAINT book_comment_reads_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

ALTER TABLE public.book_comment_reads DROP CONSTRAINT IF EXISTS book_comment_reads_comment_id_fkey;
ALTER TABLE public.book_comment_reads
    ADD CONSTRAINT book_comment_reads_comment_id_fkey FOREIGN KEY (comment_id)
    REFERENCES public.book_comments(id) ON DELETE CASCADE;


-- The spoiler rule needs the furthest page a reader has ever reached, not the
-- page they are on: flipping back to re-read an earlier passage must not
-- re-lock comments they have already been shown.
ALTER TABLE public.reading_progress
    ADD COLUMN IF NOT EXISTS furthest_page integer NOT NULL DEFAULT 0;

ALTER TABLE public.reading_progress DROP CONSTRAINT IF EXISTS reading_progress_furthest_page_check;
ALTER TABLE public.reading_progress
    ADD CONSTRAINT reading_progress_furthest_page_check CHECK (furthest_page >= 0);

-- Seed the new column from the positions already stored. Idempotent by
-- construction: GREATEST leaves an already-seeded row alone.
UPDATE public.reading_progress
   SET furthest_page = GREATEST(furthest_page, current_page)
 WHERE furthest_page < current_page;
