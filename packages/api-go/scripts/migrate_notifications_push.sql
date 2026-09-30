-- Migration: notifications the app can act on, and devices to push them to.
--
-- The notifications table existed, but nothing wrote to it, rows carried no
-- kind or subject (so the app couldn't open the book one was about), and the
-- server had no record of any device to push to. This adds both halves.
-- Run against your PostgreSQL database. Safe to re-run.

-- ── notifications ──────────────────────────────────────────────────────────

-- message was json, but every consumer treats it as display text. Existing
-- rows keep their text: a JSON string unwraps to its value, anything else to
-- its JSON rendering.
DO $$
BEGIN
    IF (SELECT data_type FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'notifications'
          AND column_name = 'message') = 'json' THEN
        ALTER TABLE public.notifications
            ALTER COLUMN message TYPE text USING (message #>> '{}');
    END IF;
END $$;

-- What the notification is about. The app switches on type (NEW_BOOK opens
-- the reader on book_id); SYSTEM is plain text.
ALTER TABLE public.notifications
    ADD COLUMN IF NOT EXISTS type character varying(32) NOT NULL DEFAULT 'SYSTEM',
    ADD COLUMN IF NOT EXISTS title text,
    ADD COLUMN IF NOT EXISTS book_id character varying(50);

-- A deleted book leaves the notification as plain text rather than taking it
-- with it or pointing at nothing.
ALTER TABLE public.notifications DROP CONSTRAINT IF EXISTS notifications_book_id_fkey;
ALTER TABLE public.notifications
    ADD CONSTRAINT notifications_book_id_fkey FOREIGN KEY (book_id)
    REFERENCES public.books(book_id) ON DELETE SET NULL;

-- The inbox is always "this user's, newest first".
CREATE INDEX IF NOT EXISTS notifications_user_created_idx
    ON public.notifications (user_id, created_at DESC);

-- ── device_tokens ──────────────────────────────────────────────────────────

-- One row per FCM registration token. The token is the key, not
-- (user, token): a phone that signs into a different account moves its token
-- to that account instead of pushing to both.
CREATE TABLE IF NOT EXISTS public.device_tokens (
    token text PRIMARY KEY,
    user_id uuid NOT NULL,
    platform character varying(16) NOT NULL DEFAULT 'unknown',
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.device_tokens DROP CONSTRAINT IF EXISTS device_tokens_user_id_fkey;
ALTER TABLE public.device_tokens
    ADD CONSTRAINT device_tokens_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(uuid) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS device_tokens_user_id_idx
    ON public.device_tokens (user_id);
