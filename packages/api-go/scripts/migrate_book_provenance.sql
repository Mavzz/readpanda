-- Where each imported book came from and why it was judged free to host
-- everywhere (cmd/gutenberg-import). It's the record to answer a copyright
-- complaint with: the rights statement, everyone credited with their dates,
-- and the rule they passed. (source, source_id) is what lets an import be
-- re-run without adding a book twice.
--
-- Re-runnable.

CREATE TABLE IF NOT EXISTS public.book_provenance (
    book_id character varying(50) NOT NULL,
    source character varying(32) NOT NULL,
    source_id character varying(64) NOT NULL,
    source_url text NOT NULL,
    downloaded_from text,
    rights text NOT NULL,
    contributors jsonb NOT NULL,
    rule text NOT NULL,
    checked_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT book_provenance_pkey PRIMARY KEY (book_id),
    CONSTRAINT book_provenance_source_key UNIQUE (source, source_id),
    CONSTRAINT book_provenance_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id) ON DELETE CASCADE
);

-- Closed to Supabase's Data API, like every other table (see the end of
-- readpanda_schema.sql).
ALTER TABLE public.book_provenance ENABLE ROW LEVEL SECURITY;
