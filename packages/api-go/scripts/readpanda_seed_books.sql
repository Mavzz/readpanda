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
-- Data for Name: books; Type: TABLE DATA; Schema: public; Owner: -
--
-- Only the seeded book ReadPanda may host. The other five (in-copyright
-- textbooks and a fan fiction) were taken out by
-- migrate_remove_unlicensed_books.sql, and are left out here so a new
-- database doesn't bring them back.
--

INSERT INTO public.books VALUES ('bk_dda04766', NULL, 'PythonNotesForProfessionals', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/PythonNotesForProfessionals.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/PythonNotesForProfessionals.pdf', 0, '2026-05-07 12:09:09.508052', '2026-05-07 12:09:09.508052', 1, 0) ON CONFLICT DO NOTHING;


--
-- Data for Name: curated_buckets; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.curated_buckets VALUES ('cb_99dc5546', 'DataScience', 0, NULL, true, '2026-05-07 07:18:40.680999', '2026-05-07 07:18:40.680999', 1) ON CONFLICT DO NOTHING;


--
-- Data for Name: curated_bucket_books; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.curated_bucket_books VALUES ('cb_99dc5546', 'bk_dda04766', 2, '2026-05-07 12:48:40.687604') ON CONFLICT DO NOTHING;


--
-- PostgreSQL database dump complete
--


