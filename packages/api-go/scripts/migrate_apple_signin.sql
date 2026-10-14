-- Sign in with Apple (app spec signup-redesign R8.1, R8.5).
--
-- apple_sub is Apple's stable user id ("sub" in the identity token) — the
-- way a returning Apple sign-in finds its account, since Apple shares the
-- email only as the user chose (possibly a private relay address).
-- apple_refresh_token is the token from exchanging the sign-in's
-- authorization code; deleting the account revokes it (App Store 5.1.1(v)).
--
-- Re-runnable.

ALTER TYPE public.login_type_enum ADD VALUE IF NOT EXISTS 'social_apple';

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS apple_sub text,
    ADD COLUMN IF NOT EXISTS apple_refresh_token text;

CREATE UNIQUE INDEX IF NOT EXISTS users_apple_sub_key
    ON public.users (apple_sub) WHERE apple_sub IS NOT NULL;
