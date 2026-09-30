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

INSERT INTO public.books VALUES ('bk_a8a81d6e', NULL, 'Art Heist Baby', '', 'Mythic Fantasy', 'Fantasy', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Fantasy/Mythic Fantasy/Art Heist Baby.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Fantasy/Mythic Fantasy/Art Heist Baby.pdf', 0, '2026-05-07 12:09:09.503022', '2026-05-07 12:09:09.503022', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.books VALUES ('bk_80e398dd', NULL, 'DataMining', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/DataMining.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataMining.pdf', 0, '2026-05-07 12:09:09.506775', '2026-05-07 12:09:09.506775', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.books VALUES ('bk_13d216bf', NULL, 'DataScienceAndPredictiveAnalyt', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/DataScienceAndPredictiveAnalyt.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataScienceAndPredictiveAnalyt.pdf', 0, '2026-05-07 12:09:09.507465', '2026-05-07 12:09:09.507465', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.books VALUES ('bk_dda04766', NULL, 'PythonNotesForProfessionals', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/PythonNotesForProfessionals.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/PythonNotesForProfessionals.pdf', 0, '2026-05-07 12:09:09.508052', '2026-05-07 12:09:09.508052', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.books VALUES ('bk_4e146709', NULL, 'QuantumMechanics', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/QuantumMechanics.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/QuantumMechanics.pdf', 0, '2026-05-07 12:09:09.5086', '2026-05-07 12:09:09.5086', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.books VALUES ('bk_d153b2cf', NULL, 'RegressionModelingStrategies', '', 'Science', 'Non-Fiction', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/Non-Fiction/Science/RegressionModelingStrategies.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/RegressionModelingStrategies.pdf', 0, '2026-05-07 12:09:09.509423', '2026-05-07 12:09:09.509423', 1, 0) ON CONFLICT DO NOTHING;


--
-- Data for Name: curated_buckets; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.curated_buckets VALUES ('cb_99dc5546', 'DataScience', 0, NULL, true, '2026-05-07 07:18:40.680999', '2026-05-07 07:18:40.680999', 4) ON CONFLICT DO NOTHING;
INSERT INTO public.curated_buckets VALUES ('cb_82b24907', 'QuantumPhysics', 1, NULL, true, '2026-05-07 07:19:01.728185', '2026-05-07 13:18:34.876226', 1) ON CONFLICT DO NOTHING;


--
-- Data for Name: curated_bucket_books; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.curated_bucket_books VALUES ('cb_99dc5546', 'bk_80e398dd', 0, '2026-05-07 12:48:40.683874') ON CONFLICT DO NOTHING;
INSERT INTO public.curated_bucket_books VALUES ('cb_99dc5546', 'bk_13d216bf', 1, '2026-05-07 12:48:40.686958') ON CONFLICT DO NOTHING;
INSERT INTO public.curated_bucket_books VALUES ('cb_99dc5546', 'bk_dda04766', 2, '2026-05-07 12:48:40.687604') ON CONFLICT DO NOTHING;
INSERT INTO public.curated_bucket_books VALUES ('cb_99dc5546', 'bk_d153b2cf', 3, '2026-05-07 12:48:40.688203') ON CONFLICT DO NOTHING;
INSERT INTO public.curated_bucket_books VALUES ('cb_82b24907', 'bk_4e146709', 0, '2026-05-07 12:49:01.729095') ON CONFLICT DO NOTHING;


--
-- PostgreSQL database dump complete
--


