--
-- PostgreSQL database dump
--


-- Dumped from database version 18.3 (Homebrew)
-- Dumped by pg_dump version 18.3 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: login_type_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.login_type_enum AS ENUM (
    'email',
    'social_google',
    'ldap'
);


--
-- Name: room_role_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.room_role_enum AS ENUM (
    'admin',
    'reader'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: book_comment_likes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.book_comment_likes (
    comment_id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: book_comment_reads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.book_comment_reads (
    user_id uuid NOT NULL,
    comment_id character varying(50) NOT NULL,
    read_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: book_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.book_comments (
    id character varying(50) NOT NULL,
    room_id character varying(50) NOT NULL,
    book_id character varying(50) NOT NULL,
    file_hash character varying(64),
    user_id uuid NOT NULL,
    page integer DEFAULT 0 NOT NULL,
    anchor_text text,
    anchor_bounds jsonb,
    anchor_key character varying(64) NOT NULL,
    body text NOT NULL,
    parent_id character varying(50),
    client_id character varying(64),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT book_comments_anchor_text_check CHECK (((anchor_text IS NULL) OR (char_length(anchor_text) <= 1000))),
    CONSTRAINT book_comments_body_check CHECK (((char_length(body) >= 1) AND (char_length(body) <= 2000))),
    CONSTRAINT book_comments_page_check CHECK ((page >= 0))
);


--
-- Name: book_highlights; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.book_highlights (
    id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    book_id character varying(50) NOT NULL,
    file_hash character varying(64),
    page integer DEFAULT 0 NOT NULL,
    anchor_text text NOT NULL,
    anchor_bounds jsonb,
    client_id character varying(64),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT book_highlights_anchor_text_check CHECK (((char_length(anchor_text) >= 1) AND (char_length(anchor_text) <= 1000))),
    CONSTRAINT book_highlights_page_check CHECK ((page >= 0))
);


--
-- Name: books; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.books (
    book_id character varying(50) NOT NULL,
    user_id uuid,
    title character varying(255),
    description text,
    subgenre character varying(50),
    genre character varying(50),
    cover_image_url text,
    manuscript_url text,
    views integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    status integer DEFAULT 0,
    earnings numeric DEFAULT 0,
    source_title character varying(255),
    metadata_checked_at timestamp without time zone,
    author_name character varying(255),
    page_count integer,
    author_from_lookup boolean DEFAULT false NOT NULL,
    pages_from_lookup boolean DEFAULT false NOT NULL
);


--
-- Name: curated_bucket_books; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.curated_bucket_books (
    bucket_id character varying(50) NOT NULL,
    book_id character varying(50) NOT NULL,
    sort_order integer DEFAULT 0,
    added_at timestamp without time zone DEFAULT now()
);


--
-- Name: curated_buckets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.curated_buckets (
    id character varying(50) NOT NULL,
    title character varying(100) NOT NULL,
    sort_order integer DEFAULT 0,
    cover_image_url text,
    is_active boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    book_count integer,
    description text,
    genre_tags text[] DEFAULT '{}'::text[] NOT NULL
);


--
-- Name: device_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_tokens (
    token text NOT NULL,
    user_id uuid NOT NULL,
    platform character varying(16) DEFAULT 'unknown'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id integer NOT NULL,
    user_id uuid NOT NULL,
    is_read boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    message text,
    type character varying(32) DEFAULT 'SYSTEM'::character varying NOT NULL,
    title text,
    book_id character varying(50)
);


--
-- Name: notifications_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.notifications ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.notifications_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.preferences (
    id integer NOT NULL,
    subgenre character varying(50) NOT NULL,
    description text,
    genre character varying(50)
);


--
-- Name: preferences_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.preferences ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.preferences_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: reading_progress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reading_progress (
    user_id uuid NOT NULL,
    book_id character varying(50) NOT NULL,
    current_page integer DEFAULT 0 NOT NULL,
    total_pages integer DEFAULT 0 NOT NULL,
    last_read_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    furthest_page integer DEFAULT 0 NOT NULL,
    CONSTRAINT reading_progress_furthest_page_check CHECK ((furthest_page >= 0)),
    CONSTRAINT reading_progress_pages_check CHECK (((current_page >= 0) AND (total_pages >= 0)))
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id integer NOT NULL,
    user_id uuid NOT NULL,
    token text NOT NULL,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: refresh_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.refresh_tokens ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.refresh_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: room_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.room_members (
    room_id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    joined_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    role public.room_role_enum DEFAULT 'reader'::public.room_role_enum
);


--
-- Name: rooms; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rooms (
    id character varying(50) NOT NULL,
    name character varying(50) NOT NULL,
    description text,
    invite_code character varying(6),
    is_private boolean,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    admin_id uuid NOT NULL,
    current_book_id character varying(50),
    current_bucket_id character varying(50),
    current_bucket_type character varying(10),
    CONSTRAINT rooms_bucket_pair_check CHECK (((current_bucket_id IS NULL) = (current_bucket_type IS NULL))),
    CONSTRAINT rooms_current_bucket_type_check CHECK (((current_bucket_type IS NULL) OR ((current_bucket_type)::text = ANY (ARRAY[('user'::character varying)::text, ('curated'::character varying)::text]))))
);


--
-- Name: user_bucket_books; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_bucket_books (
    bucket_id character varying(50) NOT NULL,
    book_id character varying(50) NOT NULL,
    added_at timestamp without time zone DEFAULT now(),
    sort_order integer DEFAULT 0
);


--
-- Name: user_buckets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_buckets (
    id character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    book_count integer,
    source_curated_id character varying(50)
);


--
-- Name: user_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_preferences (
    id integer NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    preferences json,
    user_id uuid NOT NULL
);


--
-- Name: user_preferences_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.user_preferences ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.user_preferences_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    uuid uuid NOT NULL,
    username character varying(50) NOT NULL,
    email character varying(100) NOT NULL,
    password character varying(300),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    isactive boolean DEFAULT true,
    login_type public.login_type_enum,
    google_sub text,
    role character varying(20) DEFAULT 'user'::character varying NOT NULL
);


--
-- Name: book_comment_likes book_comment_likes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_likes
    ADD CONSTRAINT book_comment_likes_pkey PRIMARY KEY (comment_id, user_id);


--
-- Name: book_comment_reads book_comment_reads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_reads
    ADD CONSTRAINT book_comment_reads_pkey PRIMARY KEY (user_id, comment_id);


--
-- Name: book_comments book_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comments
    ADD CONSTRAINT book_comments_pkey PRIMARY KEY (id);


--
-- Name: book_highlights book_highlights_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_highlights
    ADD CONSTRAINT book_highlights_pkey PRIMARY KEY (id);


--
-- Name: books books_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.books
    ADD CONSTRAINT books_pkey PRIMARY KEY (book_id);


--
-- Name: curated_bucket_books curated_bucket_books_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.curated_bucket_books
    ADD CONSTRAINT curated_bucket_books_pkey PRIMARY KEY (bucket_id, book_id);


--
-- Name: curated_buckets curated_buckets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.curated_buckets
    ADD CONSTRAINT curated_buckets_pkey PRIMARY KEY (id);


--
-- Name: device_tokens device_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT device_tokens_pkey PRIMARY KEY (token);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: preferences preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.preferences
    ADD CONSTRAINT preferences_pkey PRIMARY KEY (id);


--
-- Name: reading_progress reading_progress_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reading_progress
    ADD CONSTRAINT reading_progress_pkey PRIMARY KEY (user_id, book_id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: room_members room_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.room_members
    ADD CONSTRAINT room_members_pkey PRIMARY KEY (room_id, user_id);


--
-- Name: rooms rooms_invite_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rooms
    ADD CONSTRAINT rooms_invite_code_key UNIQUE (invite_code);


--
-- Name: rooms rooms_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rooms
    ADD CONSTRAINT rooms_pkey PRIMARY KEY (id);


--
-- Name: user_bucket_books user_bucket_books_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_bucket_books
    ADD CONSTRAINT user_bucket_books_pkey PRIMARY KEY (bucket_id, book_id);


--
-- Name: user_buckets user_buckets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_buckets
    ADD CONSTRAINT user_buckets_pkey PRIMARY KEY (id);


--
-- Name: user_preferences user_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_preferences
    ADD CONSTRAINT user_preferences_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (uuid);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: book_comments_parent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX book_comments_parent_idx ON public.book_comments USING btree (parent_id) WHERE (parent_id IS NOT NULL);


--
-- Name: book_comments_room_book_anchor_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX book_comments_room_book_anchor_idx ON public.book_comments USING btree (room_id, book_id, page, anchor_key);


--
-- Name: book_comments_user_client_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX book_comments_user_client_idx ON public.book_comments USING btree (user_id, client_id) WHERE (client_id IS NOT NULL);


--
-- Name: book_highlights_user_book_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX book_highlights_user_book_idx ON public.book_highlights USING btree (user_id, book_id, page);


--
-- Name: book_highlights_user_client_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX book_highlights_user_client_idx ON public.book_highlights USING btree (user_id, client_id) WHERE (client_id IS NOT NULL);


--
-- Name: device_tokens_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_tokens_user_id_idx ON public.device_tokens USING btree (user_id);


--
-- Name: idx_notifications_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_id ON public.notifications USING btree (user_id);


--
-- Name: idx_refresh_tokens_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_expires_at ON public.refresh_tokens USING btree (expires_at);


--
-- Name: idx_refresh_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_user_id ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_user_buckets_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_buckets_user_id ON public.user_buckets USING btree (user_id);


--
-- Name: idx_userpreferences_userid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_userpreferences_userid ON public.user_preferences USING btree (user_id);


--
-- Name: notifications_user_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_user_created_idx ON public.notifications USING btree (user_id, created_at DESC);


--
-- Name: reading_progress_book_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reading_progress_book_id_idx ON public.reading_progress USING btree (book_id);


--
-- Name: reading_progress_last_read_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reading_progress_last_read_at_idx ON public.reading_progress USING btree (last_read_at);


--
-- Name: user_bucket_books_book_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_bucket_books_book_id_idx ON public.user_bucket_books USING btree (book_id);


--
-- Name: user_buckets_source_curated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_buckets_source_curated_idx ON public.user_buckets USING btree (user_id, source_curated_id) WHERE (source_curated_id IS NOT NULL);


--
-- Name: book_comment_likes book_comment_likes_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_likes
    ADD CONSTRAINT book_comment_likes_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.book_comments(id) ON DELETE CASCADE;


--
-- Name: book_comment_likes book_comment_likes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_likes
    ADD CONSTRAINT book_comment_likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: book_comment_reads book_comment_reads_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_reads
    ADD CONSTRAINT book_comment_reads_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.book_comments(id) ON DELETE CASCADE;


--
-- Name: book_comment_reads book_comment_reads_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comment_reads
    ADD CONSTRAINT book_comment_reads_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: book_comments book_comments_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comments
    ADD CONSTRAINT book_comments_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id) ON DELETE CASCADE;


--
-- Name: book_comments book_comments_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comments
    ADD CONSTRAINT book_comments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.book_comments(id) ON DELETE CASCADE;


--
-- Name: book_comments book_comments_room_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comments
    ADD CONSTRAINT book_comments_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.rooms(id) ON DELETE CASCADE;


--
-- Name: book_comments book_comments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_comments
    ADD CONSTRAINT book_comments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: book_highlights book_highlights_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_highlights
    ADD CONSTRAINT book_highlights_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id) ON DELETE CASCADE;


--
-- Name: book_highlights book_highlights_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.book_highlights
    ADD CONSTRAINT book_highlights_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: curated_bucket_books curated_bucket_books_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.curated_bucket_books
    ADD CONSTRAINT curated_bucket_books_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id);


--
-- Name: curated_bucket_books curated_bucket_books_bucket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.curated_bucket_books
    ADD CONSTRAINT curated_bucket_books_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES public.curated_buckets(id) ON DELETE CASCADE;


--
-- Name: device_tokens device_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_tokens
    ADD CONSTRAINT device_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: notifications notifications_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id) ON DELETE SET NULL;


--
-- Name: notifications notifications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: reading_progress reading_progress_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reading_progress
    ADD CONSTRAINT reading_progress_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: room_members room_members_room_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.room_members
    ADD CONSTRAINT room_members_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.rooms(id) ON DELETE CASCADE;


--
-- Name: room_members room_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.room_members
    ADD CONSTRAINT room_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: rooms rooms_admin_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rooms
    ADD CONSTRAINT rooms_admin_id_fkey FOREIGN KEY (admin_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: rooms rooms_current_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rooms
    ADD CONSTRAINT rooms_current_book_id_fkey FOREIGN KEY (current_book_id) REFERENCES public.books(book_id) ON DELETE CASCADE;


--
-- Name: books user_books_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.books
    ADD CONSTRAINT user_books_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid);


--
-- Name: user_bucket_books user_bucket_books_book_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_bucket_books
    ADD CONSTRAINT user_bucket_books_book_id_fkey FOREIGN KEY (book_id) REFERENCES public.books(book_id);


--
-- Name: user_bucket_books user_bucket_books_bucket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_bucket_books
    ADD CONSTRAINT user_bucket_books_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES public.user_buckets(id) ON DELETE CASCADE;


--
-- Name: user_buckets user_buckets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_buckets
    ADD CONSTRAINT user_buckets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Name: user_preferences user_preference_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_preferences
    ADD CONSTRAINT user_preference_fkey FOREIGN KEY (user_id) REFERENCES public.users(uuid) ON DELETE CASCADE;


--
-- Supabase: close the auto-generated Data API. Supabase exposes every public
-- table over REST to anyone holding the project's anon key; with RLS on and
-- no policies, that API sees nothing. The Go server connects as the table
-- owner, which bypasses RLS, so the app is unaffected.
--

DO $$
DECLARE t record;
BEGIN
    FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t.tablename);
    END LOOP;
END $$;


--
-- PostgreSQL database dump complete
--


