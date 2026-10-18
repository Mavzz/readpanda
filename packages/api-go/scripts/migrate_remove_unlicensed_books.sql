-- Takes the books ReadPanda has no right to host out of the catalogue: the
-- in-copyright novels and textbooks that came in with the test library
-- (migrate_test_library.sql) and the older seed (readpanda_seed_books.sql).
-- Before App Store submission. What may go back in: books public domain in
-- the US whose author, translator and illustrator all died 70+ years ago, or
-- books under a licence that allows commercial redistribution.
--
-- Kept, because they are free to host: Dickens (David Copperfield, Great
-- Expectations), the three Constitution of India texts (Indian law lets
-- anyone reproduce it), and Python Notes for Professionals (CC BY-SA).
--
-- What goes with each book:
--   - its comments, their likes and read marks, highlights and reading
--     progress (all cascade from books);
--   - its notifications ("… is ready to read", comment and room-book notices),
--     which would otherwise stay in inboxes pointing at nothing;
--   - its places in curated and readers' buckets. A curated bucket left empty
--     is deleted (the Witcher Series), and so are readers' copies of it that
--     end up empty.
-- Rooms are kept, members and all: a room reading one of these books goes
-- back to having no book, as a new room does, and drops a bucket that no
-- longer exists. (rooms.current_book_id cascades, so deleting the books first
-- would have deleted the rooms.)
--
-- Books are matched by id or by file: the seeded books have different ids in
-- production, where migrate_test_library.sql matched them by file.
--
-- The files stay in R2 until scripts/remove_unlicensed_books_r2.sh runs.
-- Run this first, everywhere the bucket is used (local and Supabase), so no
-- database is left pointing at deleted files.
--
-- One transaction, re-runnable: a second run finds nothing and changes
-- nothing. To see what it would remove without removing it:
--   sed 's/^COMMIT;$/ROLLBACK;/' scripts/migrate_remove_unlicensed_books.sql | psql …

BEGIN;

CREATE TEMP TABLE removed_books ON COMMIT DROP AS
SELECT b.book_id, b.title, v.reason
  FROM (VALUES
    ('bk_a8a81d6e', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Fantasy/Mythic Fantasy/Art Heist Baby.pdf',
     'Harry Potter fan fiction: the fan author''s work, derived from a copyrighted series'),
    ('bk_80e398dd', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataMining.pdf',
     'Springer textbook (Aggarwal, 2015), in copyright'),
    ('bk_13d216bf', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataScienceAndPredictiveAnalyt.pdf',
     'Springer textbook (Dinov, 2018), in copyright'),
    ('bk_4e146709', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/QuantumMechanics.pdf',
     'Springer textbook (Hecht, 2000), in copyright'),
    ('bk_d153b2cf', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/RegressionModelingStrategies.pdf',
     'Springer textbook (Harrell, 2015), in copyright'),
    ('bk_23c53501', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_23c53501-Cormen_Algorithms_3rd.pdf',
     'MIT Press textbook (Cormen et al., 2009), in copyright'),
    ('bk_6a77fd96', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_6a77fd96-Foundations of Algorithms - Richard E. Neapolitan.pdf',
     'Jones & Bartlett textbook (Neapolitan, 2014), in copyright'),
    ('bk_1fa54678', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_1fa54678-computeralgorithmshorowitz-160202195329.pdf',
     'Universities Press textbook (Horowitz et al.), in copyright'),
    ('bk_dc713202', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_dc713202-2-Aurélien-Géron-Hands-On-Machine-Learning-with-Scikit-Learn-Keras-and-Tensorflow_-Concepts-Tools-and-Techniques-to-Build-Intelligent-Systems-O%u2019Reilly-Media-2019.pdf',
     'O''Reilly book (Géron, 2019), in copyright'),
    ('bk_0055eda3', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_0055eda3-2013_Book_UnderstandingStatisticsUsingR.pdf',
     'Springer textbook (2013): free to download in 2020, never free to redistribute'),
    ('bk_189ef70f', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_189ef70f-2014_Book_DataAnalysis.pdf',
     'Springer textbook (2014): free to download in 2020, never free to redistribute'),
    ('bk_c079ae13', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_c079ae13-2015_Book_DataStructuresAndAlgorithmsWit.pdf',
     'Springer textbook (2015): free to download in 2020, never free to redistribute'),
    ('bk_07595f86', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_07595f86-2016_Book_PrinciplesOfDataMining.pdf',
     'Springer textbook (2016): free to download in 2020, never free to redistribute'),
    ('bk_b9992a17', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_b9992a17-2017_Book_AnIntroductionToMachineLearnin.pdf',
     'Springer textbook (2017): free to download in 2020, never free to redistribute'),
    ('bk_95945c53', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_95945c53-2017_Book_IntroductionToArtificialIntell.pdf',
     'Springer textbook (2017): free to download in 2020, never free to redistribute'),
    ('bk_a290f1ee', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_a290f1ee-2018_Book_ABeginnerSGuideToScalaObjectOr.pdf',
     'Springer textbook (2018): free to download in 2020, never free to redistribute'),
    ('bk_704a73bd', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_704a73bd-2018_Book_IntroductionToDeepLearning.pdf',
     'Springer textbook (2018): free to download in 2020, never free to redistribute'),
    ('bk_abd50953', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_abd50953-2018_Book_NeuralNetworksAndDeepLearning.pdf',
     'Springer textbook (2018): free to download in 2020, never free to redistribute'),
    ('bk_4c70f301', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_4c70f301-2019_Book_AdvancedGuideToPython3Programm.pdf',
     'Springer textbook (2019): free to download in 2020, never free to redistribute'),
    ('bk_2d425aaf', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_2d425aaf-337752698-Barrister-Parvateesam.pdf',
     'Mokkapati Narasimha Sastry died in 1973: in copyright in India until 2033'),
    ('bk_a866e332', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_a866e332-Maha-Bharatam.pdf',
     'Chandamama magazine serial (1969–70), in copyright'),
    ('bk_07b1cda2', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_07b1cda2-తెలుగు వర్ణమాల - TELUGU LEARNING KIT MODULE.pdf',
     'Modern learning kit (C P Brown Academy), in copyright'),
    ('bk_4bd582a6', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_4bd582a6-haruki murakami - the wind-up bird chronicle.pdf',
     'Murakami (living author), in copyright'),
    ('bk_9ad97ab2', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_9ad97ab2- The Last Wish.epub',
     'Sapkowski (living author), in copyright'),
    ('bk_48ea0037', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_48ea0037- Blood of Elves.epub',
     'Sapkowski (living author), in copyright'),
    ('bk_b51e2f92', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_b51e2f92- Baptism of Fire.epub',
     'Sapkowski (living author), in copyright'),
    ('bk_fb9daf65', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_fb9daf65- The Tower of Swallows.epub',
     'Sapkowski (living author), in copyright'),
    ('bk_59b1cd1e', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_59b1cd1e- Lady of the Lake.epub',
     'Sapkowski (living author), in copyright')
  ) AS v(book_id, manuscript_url, reason)
  JOIN books b ON b.book_id = v.book_id OR b.manuscript_url = v.manuscript_url;

-- What is about to go, for the person running this.
SELECT r.book_id, r.title, r.reason,
       (SELECT COUNT(*) FROM rooms x WHERE x.current_book_id = r.book_id) AS rooms_reading,
       (SELECT COUNT(*) FROM book_comments x WHERE x.book_id = r.book_id) AS comments,
       (SELECT COUNT(*) FROM reading_progress x WHERE x.book_id = r.book_id) AS readers
  FROM removed_books r
 ORDER BY r.title;

-- 1. Rooms: keep them, without the book.
UPDATE rooms
   SET current_book_id = NULL, updated_at = NOW()
 WHERE current_book_id IN (SELECT book_id FROM removed_books);

-- 2. Notifications about the books. Done before the books go, since the
--    foreign key would only blank book_id and leave the message behind.
DELETE FROM notifications WHERE book_id IN (SELECT book_id FROM removed_books);

-- 3. Curated buckets: take the books out, then delete the buckets left empty
--    by it (not ones that were already empty) along with readers' copies of
--    them that are empty too, and recount the rest.
CREATE TEMP TABLE touched_curated ON COMMIT DROP AS
SELECT DISTINCT bucket_id FROM curated_bucket_books
 WHERE book_id IN (SELECT book_id FROM removed_books);

DELETE FROM curated_bucket_books WHERE book_id IN (SELECT book_id FROM removed_books);

CREATE TEMP TABLE emptied_curated ON COMMIT DROP AS
SELECT t.bucket_id FROM touched_curated t
 WHERE NOT EXISTS (SELECT 1 FROM curated_bucket_books x WHERE x.bucket_id = t.bucket_id);

UPDATE curated_buckets c
   SET book_count = (SELECT COUNT(*) FROM curated_bucket_books x WHERE x.bucket_id = c.id),
       updated_at = NOW()
 WHERE c.id IN (SELECT bucket_id FROM touched_curated);

-- A cover borrowed from one of the books.
UPDATE curated_buckets c
   SET cover_image_url = NULL, updated_at = NOW()
  FROM books b
 WHERE b.book_id IN (SELECT book_id FROM removed_books)
   AND c.cover_image_url = b.cover_image_url;

-- 4. Readers' buckets: same, except a reader's own bucket is theirs to delete;
--    only an emptied copy of a curated bucket that is going away goes with it.
CREATE TEMP TABLE touched_user ON COMMIT DROP AS
SELECT DISTINCT bucket_id FROM user_bucket_books
 WHERE book_id IN (SELECT book_id FROM removed_books);

DELETE FROM user_bucket_books WHERE book_id IN (SELECT book_id FROM removed_books);

UPDATE user_buckets u
   SET book_count = (SELECT COUNT(*) FROM user_bucket_books x WHERE x.bucket_id = u.id),
       updated_at = NOW()
 WHERE u.id IN (SELECT bucket_id FROM touched_user);

CREATE TEMP TABLE deleted_buckets ON COMMIT DROP AS
SELECT u.id AS bucket_id, 'user'::text AS bucket_type FROM user_buckets u
 WHERE u.source_curated_id IN (SELECT bucket_id FROM emptied_curated)
   AND NOT EXISTS (SELECT 1 FROM user_bucket_books x WHERE x.bucket_id = u.id)
UNION ALL
SELECT bucket_id, 'curated' FROM emptied_curated;

-- Rooms reading from a bucket that is about to go drop it too.
UPDATE rooms r
   SET current_bucket_id = NULL, current_bucket_type = NULL, updated_at = NOW()
  FROM deleted_buckets d
 WHERE r.current_bucket_id = d.bucket_id AND r.current_bucket_type = d.bucket_type;

DELETE FROM user_buckets
 WHERE id IN (SELECT bucket_id FROM deleted_buckets WHERE bucket_type = 'user');
DELETE FROM curated_buckets
 WHERE id IN (SELECT bucket_id FROM deleted_buckets WHERE bucket_type = 'curated');

-- 5. The books. Comments (with their likes and read marks), highlights and
--    reading progress cascade.
DELETE FROM books WHERE book_id IN (SELECT book_id FROM removed_books);

COMMIT;
