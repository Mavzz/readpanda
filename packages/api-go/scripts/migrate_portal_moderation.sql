-- Migration: Portal moderation (readpanda-mobile specs/portal-moderation).
-- Run against your PostgreSQL database before deploying the API that keeps
-- suspension history.
--
-- Re-runnable: a second run adds nothing (R1.5).

-- Every suspension and its reinstatement (R1). The history goes with the
-- reader's account (R1.7); an admin's deletion only blanks who acted.
-- suspended_at is NULL for a suspension from before history began, and
-- ended without reinstated_at is one that's over but we don't know when.
CREATE TABLE IF NOT EXISTS public.user_suspensions (
    id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    suspended_at timestamp without time zone,
    suspended_by uuid,
    report_id character varying(50),
    note text,
    ended boolean DEFAULT false NOT NULL,
    reinstated_at timestamp without time zone,
    reinstated_by uuid,
    reinstate_note text,
    source character varying(10) NOT NULL,
    CONSTRAINT user_suspensions_pkey PRIMARY KEY (id),
    CONSTRAINT user_suspensions_note_check CHECK (((note IS NULL) OR (char_length(note) <= 500))),
    CONSTRAINT user_suspensions_reinstate_note_check CHECK (((reinstate_note IS NULL) OR (char_length(reinstate_note) <= 500))),
    CONSTRAINT user_suspensions_source_check CHECK (((source)::text = ANY (ARRAY['report'::text, 'users'::text, 'backfill'::text]))),
    CONSTRAINT user_suspensions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE,
    CONSTRAINT user_suspensions_suspended_by_fkey FOREIGN KEY (suspended_by) REFERENCES public.users(uuid) ON DELETE SET NULL,
    CONSTRAINT user_suspensions_reinstated_by_fkey FOREIGN KEY (reinstated_by) REFERENCES public.users(uuid) ON DELETE SET NULL,
    CONSTRAINT user_suspensions_report_id_fkey FOREIGN KEY (report_id) REFERENCES public.content_reports(id) ON DELETE SET NULL
);

-- At most one open suspension per reader, so suspending twice records once
-- (R1.3) and "suspended" always means "has an open one" (R1.8).
CREATE UNIQUE INDEX IF NOT EXISTS user_suspensions_open_once ON public.user_suspensions USING btree (user_id) WHERE (NOT ended);
CREATE INDEX IF NOT EXISTS user_suspensions_user_idx ON public.user_suspensions USING btree (user_id, suspended_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS user_suspensions_suspended_at_idx ON public.user_suspensions USING btree (suspended_at);
CREATE INDEX IF NOT EXISTS user_suspensions_reinstated_at_idx ON public.user_suspensions USING btree (reinstated_at);

-- The Readers and Rooms tabs group reports by who and what they're about,
-- and the Stats tab counts them by when they closed.
CREATE INDEX IF NOT EXISTS content_reports_target_user_idx ON public.content_reports USING btree (target_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS content_reports_room_target_idx ON public.content_reports USING btree (target_id) WHERE ((target_type)::text = 'room'::text);
CREATE INDEX IF NOT EXISTS content_reports_resolved_at_idx ON public.content_reports USING btree (resolved_at);

-- Closed to Supabase's Data API, like every other table.
ALTER TABLE public.user_suspensions ENABLE ROW LEVEL SECURITY;

-- Backfill (R1.5). Before this, a suspension was only users.isactive, so
-- this is what can still be known. Everything goes in closed first, because
-- a reader can have several and only one may be open; step 3 reopens the
-- one that's still in force.

-- 1. Each past "Suspend" on a report. Acting on one report closes every
-- open report on the same thing at the same moment by the same admin, so a
-- group of those is one suspension, credited to its earliest report. A
-- group already recorded (by an earlier run, or live since) is skipped.
INSERT INTO public.user_suspensions (id, user_id, suspended_at, suspended_by, report_id, ended, source)
SELECT 'su_' || gen_random_uuid(), g.target_user_id, g.resolved_at, g.resolved_by, g.report_id, true, 'backfill'
  FROM (
    SELECT DISTINCT ON (cr.target_user_id, cr.resolved_at, cr.resolved_by)
           cr.target_user_id, cr.resolved_at, cr.resolved_by, cr.id AS report_id
      FROM public.content_reports cr
      JOIN public.users u ON u.uuid = cr.target_user_id
     WHERE cr.action = 'suspend_user'
       AND cr.resolved_at IS NOT NULL
       AND u.username <> 'deleted.reader'
     ORDER BY cr.target_user_id, cr.resolved_at, cr.resolved_by, cr.created_at, cr.id
  ) g
 WHERE NOT EXISTS (
    SELECT 1
      FROM public.user_suspensions s
      JOIN public.content_reports c2 ON c2.id = s.report_id
     WHERE c2.target_user_id = g.target_user_id
       AND c2.resolved_at = g.resolved_at
       AND c2.resolved_by IS NOT DISTINCT FROM g.resolved_by
 );

-- 2. Anyone suspended now with nothing on record was suspended from the
-- Users page, at a time we don't know. The Deleted reader is inactive by
-- design and has no history (R1.8).
INSERT INTO public.user_suspensions (id, user_id, suspended_at, ended, source)
SELECT 'su_' || gen_random_uuid(), u.uuid, NULL, true, 'backfill'
  FROM public.users u
 WHERE u.isactive IS FALSE
   AND u.username <> 'deleted.reader'
   AND NOT EXISTS (SELECT 1 FROM public.user_suspensions s WHERE s.user_id = u.uuid);

-- 3. A reader suspended now with no open suspension gets their latest one
-- reopened: the newest, or the undated one step 2 added when that's all
-- they have.
UPDATE public.user_suspensions s
   SET ended = false
  FROM (
    SELECT DISTINCT ON (s2.user_id) s2.id
      FROM public.user_suspensions s2
      JOIN public.users u ON u.uuid = s2.user_id
     WHERE u.isactive IS FALSE
       AND u.username <> 'deleted.reader'
       AND NOT EXISTS (SELECT 1 FROM public.user_suspensions o WHERE o.user_id = s2.user_id AND NOT o.ended)
     ORDER BY s2.user_id, s2.suspended_at DESC NULLS FIRST, s2.id
  ) latest
 WHERE s.id = latest.id;
