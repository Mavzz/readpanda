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

CREATE TABLE IF NOT EXISTS public.preferences
(
    id integer NOT NULL DEFAULT nextval('preferences_id_seq'::regclass),
    subgenre character varying(50) COLLATE pg_catalog."default" NOT NULL,
    description text COLLATE pg_catalog."default",
    genre character varying(50) COLLATE pg_catalog."default",
    CONSTRAINT preferences_pkey PRIMARY KEY (id)
)

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
);

--books table used to store the metadata for the books in the database the actual book is stored in firebase storage 
CREATE TABLE books (
    book_id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(50),
    cover_image_url TEXT NOT NULL,       -- URL for the cover image in Firebase Storage
    manuscript_url TEXT NOT NULL,        -- URL for the book manuscript in Firebase Storage
    status VARCHAR(50) DEFAULT 'Draft',  -- e.g., 'Draft', 'Live', 'Archived' (matches mockBooks status)
    views INTEGER DEFAULT 0,             -- Initial views count (matches mockBooks views)
    earnings DECIMAL(10, 2) DEFAULT 0.00,-- Initial earnings (matches mockBooks earnings)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);