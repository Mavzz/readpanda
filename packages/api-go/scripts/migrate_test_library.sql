-- Copies the test library set up locally on 2026-10-03/04 into production
-- (Supabase). Book files are already in R2 — production uses the same
-- bucket — so this only adds and updates rows.
--
-- One transaction, safe to re-run: rows are matched on natural keys
-- (genre + subgenre, user email, manuscript_url), never on local ids.
-- Deploy the API with the metadata fix first, or its startup job may rename
-- the uploaded books.
--
-- Trimmed on 2026-10-18 to the books ReadPanda may host: the in-copyright
-- ones, and the Witcher Series bucket that held some of them, were taken
-- out by migrate_remove_unlicensed_books.sql and are left out here so
-- re-running this doesn't bring them back.

BEGIN;

-- 0. The uploader must already exist in production (sign in once first).
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM users WHERE email = 'venkataramaaditya.nimmagadda.prx@gmail.com') THEN
    RAISE EXCEPTION 'User venkataramaaditya.nimmagadda.prx@gmail.com not found: sign in to production once, then re-run';
  END IF;
END $$;

-- 1. New genres / subgenres.
INSERT INTO preferences (subgenre, description, genre)
SELECT v.subgenre, v.description, v.genre
  FROM (VALUES
    ('Classics', 'Enduring novels from the literary canon.', 'Fiction'),
    ('Humor', 'Comic stories written to make you laugh.', 'Fiction'),
    ('Literary Fiction', 'Character-driven fiction with a focus on style and theme.', 'Fiction'),
    ('Computer Science', 'Algorithms, programming and the foundations of computing.', 'Non-Fiction'),
    ('Data Science', 'Statistics, machine learning and working with data.', 'Non-Fiction'),
    ('Law', 'Constitutions, legal texts and how the law works.', 'Non-Fiction'),
    ('Language Learning', 'Learn to read, write and speak a new language.', 'Non-Fiction')
  ) AS v(subgenre, description, genre)
 WHERE NOT EXISTS (SELECT 1 FROM preferences p WHERE p.genre = v.genre AND p.subgenre = v.subgenre);

-- 2. Admin access for curated buckets.
UPDATE users SET role = 'admin' WHERE email = 'venkataramaaditya.nimmagadda.prx@gmail.com';

-- 3. Details for the older seeded book ReadPanda kept (matched by file).
UPDATE books SET title = 'Python Notes for Professionals', source_title = 'Python Notes for Professionals', author_name = 'GoalKicker.com', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Computer Science', description = 'Over 700 pages of Python tips and examples compiled from Stack Overflow Documentation.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/PythonNotesForProfessionals.pdf';

-- 4. The bulk-uploaded books ReadPanda may host.
INSERT INTO books (book_id, user_id, title, source_title, description, genre, subgenre, author_name,
                   cover_image_url, manuscript_url, status, views, created_at, updated_at)
SELECT v.book_id, (SELECT uuid FROM users WHERE email = 'venkataramaaditya.nimmagadda.prx@gmail.com'), v.title, v.title, v.description,
       v.genre, v.subgenre, v.author_name, v.cover_image_url, v.manuscript_url, 1, 0, v.created_at, v.created_at
  FROM (VALUES
    ('bk_467a849d', 'David Copperfield', $q$Dickens's semi-autobiographical novel follows David from a hard childhood through work, love and loss to life as a writer. Scanned 19th-century edition.$q$, 'Fiction', 'Classics', 'Charles Dickens', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_467a849d-David_Copperfield.pdf', '2026-10-03 22:42:59.093695'::timestamp),
    ('bk_d5900ca7', 'Great Expectations', 'The orphan Pip comes into a fortune from an unknown benefactor and learns what gentility costs. Scanned 1861 Tauchnitz edition, Volume I.', 'Fiction', 'Classics', 'Charles Dickens', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_d5900ca7-Great_Expectations.pdf', '2026-10-03 22:43:00.204866'::timestamp),
    ('bk_52f5a794', 'The Constitution of India', 'The full text of the Constitution of India: the Preamble and Parts I to XXII, with amendments.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_52f5a794-COI.pdf', '2026-10-03 22:43:00.516404'::timestamp),
    ('bk_6c9e0d90', 'The Constitution of India: Schedules 1–12', 'The First to Twelfth Schedules of the Constitution of India, from the States and territories to the powers of municipalities.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_6c9e0d90-coi-eng-schedules_1-12.pdf', '2026-10-03 22:43:00.999841'::timestamp),
    ('bk_2969504e', 'The Constitution of India: Appendix', 'Appendices to the Constitution of India, including the Constitution (Application to Jammu and Kashmir) Order, 1954.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_2969504e-coi_appendix.pdf', '2026-10-03 22:43:01.630512'::timestamp)
  ) AS v(book_id, title, description, genre, subgenre, author_name, cover_image_url, manuscript_url, created_at)
 WHERE NOT EXISTS (SELECT 1 FROM books b WHERE b.manuscript_url = v.manuscript_url OR b.book_id = v.book_id);

COMMIT;
