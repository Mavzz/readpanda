-- Adds a role column to users for the new RequireAdmin authorization check
-- (previously every admin curated-bucket endpoint only checked "is logged in",
-- not "is an admin"). Existing users default to 'user'; promote specific
-- accounts manually, e.g.:
--   UPDATE users SET role = 'admin' WHERE username = '<admin-username>';

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS role character varying(20) NOT NULL DEFAULT 'user';
