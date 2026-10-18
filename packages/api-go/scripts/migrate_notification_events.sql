-- Migration: notifications from things that happen in rooms, and a switch per
-- kind of notification.
--
-- Replies, room comments, joins and a room picking its book all notify (see
-- specs/room-notifications in readpanda-mobile). Room comments are bundled:
-- each reader has at most one unread row per room, updated as comments come
-- in, and that row also remembers when it was last pushed, so the push
-- throttle needs no scheduler. Readers can switch each kind off.
-- Run against your PostgreSQL database. Safe to re-run.

-- ── notifications ──────────────────────────────────────────────────────────

-- updated_at is what the inbox sorts on once a bundled row can change after
-- it's written. Rows that existed before keep their order: the first run
-- copies created_at across instead of stamping them all with now().
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'public' AND table_name = 'notifications'
                     AND column_name = 'updated_at') THEN
        ALTER TABLE public.notifications
            ADD COLUMN updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP;
        UPDATE public.notifications SET updated_at = created_at;
    END IF;
END $$;

-- room_id: the room a row is about (NULL for NEW_BOOK and SYSTEM).
-- event_count: how many comments a ROOM_COMMENTS row stands for; 1 otherwise.
-- last_pushed_at: when the row was last pushed, for the bundling window.
ALTER TABLE public.notifications
    ADD COLUMN IF NOT EXISTS room_id character varying(50),
    ADD COLUMN IF NOT EXISTS event_count integer NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS last_pushed_at timestamp without time zone;

-- A deleted room takes its notifications with it: there's nothing left for a
-- tap to open.
ALTER TABLE public.notifications DROP CONSTRAINT IF EXISTS notifications_room_id_fkey;
ALTER TABLE public.notifications
    ADD CONSTRAINT notifications_room_id_fkey FOREIGN KEY (room_id)
    REFERENCES public.rooms(id) ON DELETE CASCADE;

-- At most one unread room-comments row per reader per room. Two comments
-- arriving together upsert the same row instead of racing to create two.
-- is_read is nullable, hence IS NOT TRUE rather than NOT is_read.
CREATE UNIQUE INDEX IF NOT EXISTS notifications_unread_room_comments_key
    ON public.notifications (user_id, room_id)
    WHERE type = 'ROOM_COMMENTS' AND is_read IS NOT TRUE;

CREATE INDEX IF NOT EXISTS notifications_user_updated_idx
    ON public.notifications (user_id, updated_at DESC);

-- ── notification_settings ──────────────────────────────────────────────────

-- One row per reader who has changed a switch; no row means everything is on.
-- A switched-off kind is neither written to the inbox nor pushed.
CREATE TABLE IF NOT EXISTS public.notification_settings (
    user_id uuid PRIMARY KEY,
    replies boolean NOT NULL DEFAULT true,
    room_comments boolean NOT NULL DEFAULT true,
    room_joins boolean NOT NULL DEFAULT true,
    room_book boolean NOT NULL DEFAULT true,
    new_books boolean NOT NULL DEFAULT true,
    updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.notification_settings DROP CONSTRAINT IF EXISTS notification_settings_user_id_fkey;
ALTER TABLE public.notification_settings
    ADD CONSTRAINT notification_settings_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

-- Supabase serves every public table over its REST API to anyone with the
-- anon key. RLS with no policies closes that, as readpanda_schema.sql does
-- for every table; the Go server owns the tables, so it isn't affected.
ALTER TABLE public.notification_settings ENABLE ROW LEVEL SECURITY;
