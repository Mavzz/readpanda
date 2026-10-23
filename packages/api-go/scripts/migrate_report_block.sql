-- Migration: Report & block (readpanda-mobile specs/report-block).
-- Run against your PostgreSQL database before deploying the API that takes
-- reports and blocks.
--
-- Re-runnable.

-- What readers report (R1–R3). A report outlives whatever it's about, so
-- room_id has no foreign key, and the reporter's and target's ids go NULL
-- when their accounts are deleted (R3.3). The snapshot is the reported text
-- as it was, so a moderator sees it even after an edit or delete (R3.2).
CREATE TABLE IF NOT EXISTS public.content_reports (
    id character varying(50) NOT NULL,
    reporter_id uuid,
    target_type character varying(10) NOT NULL,
    target_id character varying(50) NOT NULL,
    target_user_id uuid,
    room_id character varying(50),
    reason character varying(20) NOT NULL,
    details text,
    snapshot text NOT NULL,
    status character varying(10) DEFAULT 'open' NOT NULL,
    action character varying(20),
    resolved_by uuid,
    resolved_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT content_reports_pkey PRIMARY KEY (id),
    CONSTRAINT content_reports_target_type_check CHECK (((target_type)::text = ANY (ARRAY['comment'::text, 'user'::text, 'room'::text]))),
    CONSTRAINT content_reports_reason_check CHECK (((reason)::text = ANY (ARRAY['harassment'::text, 'hate'::text, 'sexual'::text, 'violence'::text, 'self_harm'::text, 'spam'::text, 'other'::text]))),
    CONSTRAINT content_reports_details_check CHECK (((details IS NULL) OR (char_length(details) <= 500))),
    CONSTRAINT content_reports_status_check CHECK (((status)::text = ANY (ARRAY['open'::text, 'actioned'::text, 'dismissed'::text]))),
    CONSTRAINT content_reports_reporter_id_fkey FOREIGN KEY (reporter_id) REFERENCES public.users(uuid) ON DELETE SET NULL,
    CONSTRAINT content_reports_target_user_id_fkey FOREIGN KEY (target_user_id) REFERENCES public.users(uuid) ON DELETE SET NULL,
    CONSTRAINT content_reports_resolved_by_fkey FOREIGN KEY (resolved_by) REFERENCES public.users(uuid) ON DELETE SET NULL
);

-- One open report per reader per thing (R1.7); the moderation queue reads
-- by status, newest first.
CREATE UNIQUE INDEX IF NOT EXISTS content_reports_open_once ON public.content_reports USING btree (reporter_id, target_type, target_id) WHERE ((status)::text = 'open'::text);
CREATE INDEX IF NOT EXISTS content_reports_status_idx ON public.content_reports USING btree (status, created_at DESC);

-- Who has blocked whom (R5). Gone with either account.
CREATE TABLE IF NOT EXISTS public.user_blocks (
    blocker_id uuid NOT NULL,
    blocked_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT user_blocks_pkey PRIMARY KEY (blocker_id, blocked_id),
    CONSTRAINT user_blocks_check CHECK ((blocker_id <> blocked_id)),
    CONSTRAINT user_blocks_blocker_id_fkey FOREIGN KEY (blocker_id) REFERENCES public.users(uuid) ON DELETE CASCADE,
    CONSTRAINT user_blocks_blocked_id_fkey FOREIGN KEY (blocked_id) REFERENCES public.users(uuid) ON DELETE CASCADE
);

-- Closed to Supabase's Data API, like every other table (see the end of
-- readpanda_schema.sql).
ALTER TABLE public.content_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_blocks ENABLE ROW LEVEL SECURITY;

-- A deleted comment that has replies stays as a placeholder (R7.3, R9.2):
-- its body is emptied and these say when and by whom.
ALTER TABLE public.book_comments ADD COLUMN IF NOT EXISTS deleted_at timestamp without time zone;
ALTER TABLE public.book_comments ADD COLUMN IF NOT EXISTS deleted_by character varying(10);

ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_deleted_by_check;
ALTER TABLE public.book_comments ADD CONSTRAINT book_comments_deleted_by_check
    CHECK (((deleted_by IS NULL) OR ((deleted_by)::text = ANY (ARRAY['author'::text, 'moderator'::text]))));

-- An empty body is allowed only on a deleted placeholder.
ALTER TABLE public.book_comments DROP CONSTRAINT IF EXISTS book_comments_body_check;
ALTER TABLE public.book_comments ADD CONSTRAINT book_comments_body_check
    CHECK ((((deleted_at IS NOT NULL) AND (body = ''::text)) OR ((char_length(body) >= 1) AND (char_length(body) <= 2000))));
