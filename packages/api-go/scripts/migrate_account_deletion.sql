-- Account deletion (app spec account-deletion R5.2, R5.4).
--
-- "Deleted reader" is a reserved account that a deleted reader's comments are
-- moved to, so the rooms they were part of keep the conversation (and other
-- people's replies) instead of losing it with them. Nobody can sign in as it:
-- it has no password (LoginUser can't match a NULL one) and no Google or Apple
-- id, and isactive is false.
--
-- Re-runnable. Stops instead of half-applying if the username or email is
-- already taken by a real account.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM public.users
        WHERE (LOWER(username) = 'deleted.reader' OR LOWER(email) = 'deleted@readpanda.invalid')
          AND uuid <> '00000000-0000-0000-0000-000000000000'
    ) THEN
        RAISE EXCEPTION 'deleted.reader or deleted@readpanda.invalid already belongs to a real account; rename it first';
    END IF;
END $$;

INSERT INTO public.users (uuid, username, email, password, isactive, role)
VALUES ('00000000-0000-0000-0000-000000000000', 'deleted.reader', 'deleted@readpanda.invalid', NULL, false, 'user')
ON CONFLICT (uuid) DO NOTHING;
