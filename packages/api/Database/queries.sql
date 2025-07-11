CREATE TABLE IF NOT EXISTS public.users
-- Table to store user information
(
    username character varying(50) COLLATE pg_catalog."default" NOT NULL, -- Unique username
    email character varying(100) COLLATE pg_catalog."default" NOT NULL,   -- Unique email address
    password character varying(300) COLLATE pg_catalog."default",         -- Hashed password
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,     -- Account creation timestamp
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,     -- Last update timestamp
    isactive boolean,                                                    -- User active status
    login_type login_type_enum,                                          -- Enum for login type
    uuid uuid NOT NULL,                                                  -- Unique user identifier (primary key)
    google_sub text COLLATE pg_catalog."default",                        -- Google OAuth subject (if applicable)
    CONSTRAINT users_pkey PRIMARY KEY (uuid),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_username_key UNIQUE (username)
)

-- Table to store genre/subgenre preferences
CREATE TABLE IF NOT EXISTS public.preferences
(
    id integer NOT NULL DEFAULT nextval('preferences_id_seq'::regclass), -- Primary key
    subgenre character varying(50) COLLATE pg_catalog."default" NOT NULL, -- Subgenre name
    description text COLLATE pg_catalog."default",                        -- Description of the preference
    genre character varying(50) COLLATE pg_catalog."default",             -- Genre name
    CONSTRAINT preferences_pkey PRIMARY KEY (id)
)

-- Table to store user-specific preferences as JSON
CREATE TABLE IF NOT EXISTS public.user_preferences
(
    id integer NOT NULL DEFAULT nextval('user_preferences_id_seq'::regclass), -- Primary key
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,         -- Creation timestamp
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,         -- Last update timestamp
    preferences json,                                                        -- Preferences stored as JSON
    user_id uuid NOT NULL,                                                   -- Reference to users table
    CONSTRAINT user_preferences_pkey PRIMARY KEY (id),
    CONSTRAINT user_preference_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (uuid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
);

-- Table to store metadata for books; actual files are in Firebase Storage
CREATE TABLE books (
    book_id SERIAL PRIMARY KEY,                        -- Unique book identifier
    user_id INTEGER NOT NULL,                          -- Reference to user (author)
    title VARCHAR(255) NOT NULL,                       -- Book title
    description TEXT,                                  -- Book description
    genre VARCHAR(50),                                 -- Book genre
    cover_image_url TEXT NOT NULL,                     -- URL for cover image in Firebase Storage
    manuscript_url TEXT NOT NULL,                      -- URL for manuscript in Firebase Storage
    status VARCHAR(50) DEFAULT 'Draft',                -- Book status: Draft, Live, Archived, etc.
    views INTEGER DEFAULT 0,                           -- Number of views
    earnings DECIMAL(10, 2) DEFAULT 0.00,              -- Earnings from the book
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,    -- Creation timestamp
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,    -- Last update timestamp
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Enum type for login methods
CREATE TYPE public.login_type_enum AS ENUM
    ('email', 'social_google', 'ldap');

-- Sequence for books table primary key
CREATE SEQUENCE IF NOT EXISTS public.books_book_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

-- Sequence for preferences table primary key
CREATE SEQUENCE IF NOT EXISTS public.preferences_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

-- Sequence for user_preferences table primary key
CREATE SEQUENCE IF NOT EXISTS public.user_preferences_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;