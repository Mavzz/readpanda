-- Edit profile (app spec coming-soon R5.2): a name shown on the reader's own
-- Profile above their @username. Optional; NULL shows the username alone.
--
-- Re-runnable.

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS display_name character varying(40);
