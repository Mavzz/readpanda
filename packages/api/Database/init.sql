-- Type: login_type_enum

-- DROP TYPE IF EXISTS public.login_type_enum;

CREATE TYPE public.login_type_enum AS ENUM
    ('email', 'social_google', 'ldap');

ALTER TYPE public.login_type_enum
    OWNER TO readpanda;

-- SEQUENCE: public.books_book_id_seq

-- DROP SEQUENCE IF EXISTS public.books_book_id_seq;

CREATE SEQUENCE IF NOT EXISTS public.books_book_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

ALTER SEQUENCE public.books_book_id_seq
    OWNED BY public.books.book_id;

ALTER SEQUENCE public.books_book_id_seq
    OWNER TO readpanda;

-- SEQUENCE: public.preferences_id_seq

-- DROP SEQUENCE IF EXISTS public.preferences_id_seq;

CREATE SEQUENCE IF NOT EXISTS public.preferences_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

ALTER SEQUENCE public.preferences_id_seq
    OWNED BY public.preferences.id;

ALTER SEQUENCE public.preferences_id_seq
    OWNER TO readpanda;

-- SEQUENCE: public.user_preferences_id_seq

-- DROP SEQUENCE IF EXISTS public.user_preferences_id_seq;

CREATE SEQUENCE IF NOT EXISTS public.user_preferences_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

ALTER SEQUENCE public.user_preferences_id_seq
    OWNED BY public.user_preferences.id;

ALTER SEQUENCE public.user_preferences_id_seq
    OWNER TO readpanda;


-- Table: public.books

-- DROP TABLE IF EXISTS public.books;

CREATE TABLE IF NOT EXISTS public.books
(
    book_id integer NOT NULL DEFAULT nextval('books_book_id_seq'::regclass),
    user_id uuid,
    title character varying(50) COLLATE pg_catalog."default",
    description text COLLATE pg_catalog."default",
    subgenre character varying(50) COLLATE pg_catalog."default",
    genre character varying(50) COLLATE pg_catalog."default",
    cover_image_url text COLLATE pg_catalog."default",
    manuscript_url text COLLATE pg_catalog."default",
    views integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    status integer DEFAULT 0,
    earnings numeric DEFAULT 0,
    CONSTRAINT books_pkey PRIMARY KEY (book_id),
    CONSTRAINT user_books_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.books
    OWNER to readpanda;



-- Table: public.preferences

-- DROP TABLE IF EXISTS public.preferences;

CREATE TABLE IF NOT EXISTS public.preferences
(
    id integer NOT NULL DEFAULT nextval('preferences_id_seq'::regclass),
    subgenre character varying(50) COLLATE pg_catalog."default" NOT NULL,
    description text COLLATE pg_catalog."default",
    genre character varying(50) COLLATE pg_catalog."default",
    CONSTRAINT preferences_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.preferences
    OWNER to readpanda;



-- Table: public.user_preferences

-- DROP TABLE IF EXISTS public.user_preferences;

CREATE TABLE IF NOT EXISTS public.user_preferences
(
    id integer NOT NULL DEFAULT nextval('user_preferences_id_seq'::regclass),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    preferences json,
    user_id uuid NOT NULL,
    CONSTRAINT user_preferences_pkey PRIMARY KEY (id),
    CONSTRAINT user_preference_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.user_preferences
    OWNER to readpanda;



-- Table: public.users

-- DROP TABLE IF EXISTS public.users;

CREATE TABLE IF NOT EXISTS public.users
(
    username character varying(50) COLLATE pg_catalog."default" NOT NULL,
    email character varying(100) COLLATE pg_catalog."default" NOT NULL,
    password character varying(300) COLLATE pg_catalog."default",
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    isactive boolean,
    login_type login_type_enum,
    uuid uuid NOT NULL,
    google_sub text COLLATE pg_catalog."default",
    CONSTRAINT users_pkey PRIMARY KEY (uuid),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_username_key UNIQUE (username)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.users
    OWNER to readpanda;


-- Table: public.user_preferences

-- DROP TABLE IF EXISTS public.user_preferences;

CREATE TABLE IF NOT EXISTS public.user_preferences
(
    id integer NOT NULL DEFAULT nextval('user_preferences_id_seq'::regclass),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    preferences json,
    user_id uuid NOT NULL,
    CONSTRAINT user_preferences_pkey PRIMARY KEY (id),
    CONSTRAINT user_preference_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.user_preferences
    OWNER to readpanda;


-- Table: public.user_notifications

-- DROP TABLE IF EXISTS public.user_notifications;

CREATE TABLE IF NOT EXISTS public.user_notifications
(
    id integer NOT NULL DEFAULT nextval('user_notifications_id_seq'::regclass),
    user_id uuid NOT NULL,
    is_read boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    message json,
    CONSTRAINT user_notifications_pkey PRIMARY KEY (id),
    CONSTRAINT user_notifications_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.user_notifications
    OWNER to readpanda;


-- Table: public.refresh_tokens

-- DROP TABLE IF EXISTS public.refresh_tokens;

CREATE TABLE IF NOT EXISTS public.refresh_tokens
(
    id integer NOT NULL DEFAULT nextval('refresh_tokens_id_seq'::regclass),
    user_id uuid NOT NULL,
    token text COLLATE pg_catalog."default" NOT NULL,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.refresh_tokens
    OWNER to readpanda;
    


-- Index: public.idx_user_email
-- DROP INDEX IF EXISTS public.idx_user_email;
CREATE INDEX IF NOT EXISTS idx_user_email
    ON public.users USING btree
    (email COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;
ALTER INDEX IF EXISTS public.idx_user_email
    OWNER TO readpanda;
-- Index: public.idx_user_username
-- DROP INDEX IF EXISTS public.idx_user_username;
CREATE INDEX IF NOT EXISTS idx_user_username
    ON public.users USING btree
    (username COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;
ALTER INDEX IF EXISTS public.idx_user_username
    OWNER TO readpanda;
-- Index: public.idx_userpreferences_userid
-- DROP INDEX IF EXISTS public.idx_userpreferences_userid;
CREATE INDEX IF NOT EXISTS idx_userpreferences_userid
    ON public.user_preferences USING btree
    (user_id ASC NULLS LAST)
    TABLESPACE pg_default;
ALTER INDEX IF EXISTS public.idx_userpreferences_userid
    OWNER TO readpanda;
-- Index: public.idx_books_userid
-- DROP INDEX IF EXISTS public.idx_books_userid;
CREATE INDEX IF NOT EXISTS idx_books_userid
    ON public.books USING btree
    (user_id ASC NULLS LAST)
    TABLESPACE pg_default;
ALTER INDEX IF EXISTS public.idx_books_userid
    OWNER TO readpanda;

-- Index: idx_userpreferences_userid

-- DROP INDEX IF EXISTS public.idx_userpreferences_userid;

CREATE INDEX IF NOT EXISTS idx_userpreferences_userid
    ON public.user_preferences USING btree
    (user_id ASC NULLS LAST)
    TABLESPACE pg_default;

ALTER INDEX IF EXISTS public.idx_userpreferences_userid
    OWNER TO readpanda;


-- Index: idx_refresh_tokens_expires_at

-- DROP INDEX IF EXISTS public.idx_refresh_tokens_expires_at;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
    ON public.refresh_tokens USING btree
    (expires_at ASC NULLS LAST)
    TABLESPACE pg_default;
-- Index: idx_refresh_tokens_user_id

-- DROP INDEX IF EXISTS public.idx_refresh_tokens_user_id;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
    ON public.refresh_tokens USING btree
    (user_id ASC NULLS LAST)
    TABLESPACE pg_default;
