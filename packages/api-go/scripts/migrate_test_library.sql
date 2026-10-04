-- Copies the test library set up locally on 2026-10-03/04 into production
-- (Supabase). Book files are already in R2 — production uses the same
-- bucket — so this only adds and updates rows.
--
-- One transaction, safe to re-run: rows are matched on natural keys
-- (genre + subgenre, user email, manuscript_url), never on local ids.
-- Deploy the API with the metadata fix first, or its startup job may rename
-- the uploaded books.

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

-- 3. Details for the six older seeded books (matched by file).
UPDATE books SET title = 'Art Heist, Baby!', source_title = 'Art Heist, Baby!', author_name = 'otrtbs', author_from_lookup = false, genre = 'Romance', subgenre = 'Romantic Suspense', description = 'A Marauders-era Harry Potter fanfiction pairing James Potter and Regulus Black ("Jegulus"), centred on white-collar crime and fine art theft.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Fantasy/Mythic Fantasy/Art Heist Baby.pdf';
UPDATE books SET title = 'Data Mining: The Textbook', source_title = 'Data Mining: The Textbook', author_name = 'Charu C. Aggarwal', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Data Science', description = 'A comprehensive textbook on data mining: clustering, classification, association patterns, outlier analysis, and mining text, graphs and streams.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataMining.pdf';
UPDATE books SET title = 'Data Science and Predictive Analytics', source_title = 'Data Science and Predictive Analytics', author_name = 'Ivo D. Dinov', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Data Science', description = 'Data science and predictive analytics with R, applied to biomedical and health data.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/DataScienceAndPredictiveAnalyt.pdf';
UPDATE books SET title = 'Python Notes for Professionals', source_title = 'Python Notes for Professionals', author_name = 'GoalKicker.com', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Computer Science', description = 'Over 700 pages of Python tips and examples compiled from Stack Overflow Documentation.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/PythonNotesForProfessionals.pdf';
UPDATE books SET title = 'Quantum Mechanics', source_title = 'Quantum Mechanics', author_name = 'K. T. Hecht', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Science', description = 'A graduate text in quantum mechanics covering angular momentum, perturbation theory, scattering and many-body systems.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/QuantumMechanics.pdf';
UPDATE books SET title = 'Regression Modeling Strategies (2nd Edition)', source_title = 'Regression Modeling Strategies (2nd Edition)', author_name = 'Frank E. Harrell Jr.', author_from_lookup = false, genre = 'Non-Fiction', subgenre = 'Data Science', description = 'Applying linear models, logistic and ordinal regression, and survival analysis to real data, with an emphasis on sound modelling strategy.', metadata_checked_at = COALESCE(metadata_checked_at, NOW()), updated_at = NOW()
 WHERE manuscript_url = 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/Non-Fiction/Science/RegressionModelingStrategies.pdf';

-- 4. The 28 bulk-uploaded books.
INSERT INTO books (book_id, user_id, title, source_title, description, genre, subgenre, author_name,
                   cover_image_url, manuscript_url, status, views, created_at, updated_at)
SELECT v.book_id, (SELECT uuid FROM users WHERE email = 'venkataramaaditya.nimmagadda.prx@gmail.com'), v.title, v.title, v.description,
       v.genre, v.subgenre, v.author_name, v.cover_image_url, v.manuscript_url, 1, 0, v.created_at, v.created_at
  FROM (VALUES
    ('bk_467a849d', 'David Copperfield', $q$Dickens's semi-autobiographical novel follows David from a hard childhood through work, love and loss to life as a writer. Scanned 19th-century edition.$q$, 'Fiction', 'Classics', 'Charles Dickens', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_467a849d-David_Copperfield.pdf', '2026-10-03 22:42:59.093695'::timestamp),
    ('bk_d5900ca7', 'Great Expectations', 'The orphan Pip comes into a fortune from an unknown benefactor and learns what gentility costs. Scanned 1861 Tauchnitz edition, Volume I.', 'Fiction', 'Classics', 'Charles Dickens', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_d5900ca7-Great_Expectations.pdf', '2026-10-03 22:43:00.204866'::timestamp),
    ('bk_52f5a794', 'The Constitution of India', 'The full text of the Constitution of India: the Preamble and Parts I to XXII, with amendments.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_52f5a794-COI.pdf', '2026-10-03 22:43:00.516404'::timestamp),
    ('bk_6c9e0d90', 'The Constitution of India: Schedules 1–12', 'The First to Twelfth Schedules of the Constitution of India, from the States and territories to the powers of municipalities.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_6c9e0d90-coi-eng-schedules_1-12.pdf', '2026-10-03 22:43:00.999841'::timestamp),
    ('bk_2969504e', 'The Constitution of India: Appendix', 'Appendices to the Constitution of India, including the Constitution (Application to Jammu and Kashmir) Order, 1954.', 'Non-Fiction', 'Law', 'Government of India', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_2969504e-coi_appendix.pdf', '2026-10-03 22:43:01.630512'::timestamp),
    ('bk_23c53501', 'Introduction to Algorithms (3rd Edition)', 'The standard reference on algorithms: sorting, data structures, graph algorithms, dynamic programming and NP-completeness, with rigorous analysis.', 'Non-Fiction', 'Computer Science', 'Thomas H. Cormen, Charles E. Leiserson, Ronald L. Rivest, Clifford Stein', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_23c53501-introduction-to-algorithms-3rd-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_23c53501-Cormen_Algorithms_3rd.pdf', '2026-10-03 22:43:02.648078'::timestamp),
    ('bk_6a77fd96', 'Foundations of Algorithms (5th Edition)', 'An accessible introduction to designing and analysing algorithms, from divide-and-conquer and dynamic programming to computational complexity.', 'Non-Fiction', 'Computer Science', 'Richard E. Neapolitan', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_6a77fd96-foundations-of-algorithms-5th-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_6a77fd96-Foundations of Algorithms - Richard E. Neapolitan.pdf', '2026-10-03 22:43:06.0329'::timestamp),
    ('bk_1fa54678', 'Computer Algorithms', 'A classic textbook on algorithm design techniques: divide-and-conquer, greedy, dynamic programming, backtracking and branch-and-bound.', 'Non-Fiction', 'Computer Science', 'Ellis Horowitz, Sartaj Sahni, Sanguthevar Rajasekaran', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_1fa54678-computer-algorithms.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_1fa54678-computeralgorithmshorowitz-160202195329.pdf', '2026-10-03 22:43:09.41091'::timestamp),
    ('bk_dc713202', 'Hands-On Machine Learning with Scikit-Learn, Keras, and TensorFlow', 'A practical guide to building machine-learning systems, from classic models with Scikit-Learn to deep neural networks with Keras and TensorFlow. Second edition, early release.', 'Non-Fiction', 'Data Science', 'Aurélien Géron', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_dc713202-hands-on-machine-learning-with-scikit-learn-keras-and-tensor.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_dc713202-2-Aurélien-Géron-Hands-On-Machine-Learning-with-Scikit-Learn-Keras-and-Tensorflow_-Concepts-Tools-and-Techniques-to-Build-Intelligent-Systems-O%u2019Reilly-Media-2019.pdf', '2026-10-03 22:43:13.199787'::timestamp),
    ('bk_0055eda3', 'Understanding Statistics Using R', 'An introduction to statistical concepts taught through R programs that simulate and visualise each idea.', 'Non-Fiction', 'Data Science', 'Randall Schumacker, Sara Tomek', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_0055eda3-understanding-statistics-using-r.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_0055eda3-2013_Book_UnderstandingStatisticsUsingR.pdf', '2026-10-03 22:43:14.110732'::timestamp),
    ('bk_189ef70f', 'Data Analysis (4th Edition)', 'Statistical and computational methods for scientists and engineers, from probability and sampling to least squares and minimisation.', 'Non-Fiction', 'Data Science', 'Siegmund Brandt', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_189ef70f-data-analysis-4th-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_189ef70f-2014_Book_DataAnalysis.pdf', '2026-10-03 22:43:15.76174'::timestamp),
    ('bk_c079ae13', 'Data Structures and Algorithms with Python', 'Core data structures and algorithms taught in Python, from recursion and sequences to trees, graphs and heuristic search.', 'Non-Fiction', 'Computer Science', 'Kent D. Lee, Steve Hubbard', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_c079ae13-data-structures-and-algorithms-with-python.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_c079ae13-2015_Book_DataStructuresAndAlgorithmsWit.pdf', '2026-10-03 22:43:17.585287'::timestamp),
    ('bk_07595f86', 'Principles of Data Mining (3rd Edition)', 'A clear introduction to the algorithms behind data mining, including decision trees, rule induction, clustering and text mining.', 'Non-Fiction', 'Data Science', 'Max Bramer', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_07595f86-principles-of-data-mining-3rd-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_07595f86-2016_Book_PrinciplesOfDataMining.pdf', '2026-10-03 22:43:18.475767'::timestamp),
    ('bk_b9992a17', 'An Introduction to Machine Learning (2nd Edition)', 'An introductory machine-learning text covering Bayesian classifiers, nearest neighbours, decision trees, neural networks and performance evaluation.', 'Non-Fiction', 'Data Science', 'Miroslav Kubat', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_b9992a17-an-introduction-to-machine-learning-2nd-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_b9992a17-2017_Book_AnIntroductionToMachineLearnin.pdf', '2026-10-03 22:43:19.632504'::timestamp),
    ('bk_95945c53', 'Introduction to Artificial Intelligence (2nd Edition)', 'A broad undergraduate introduction to AI: logic, search, reasoning under uncertainty, machine learning and reinforcement learning.', 'Non-Fiction', 'Computer Science', 'Wolfgang Ertel', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_95945c53-introduction-to-artificial-intelligence-2nd-edition.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_95945c53-2017_Book_IntroductionToArtificialIntell.pdf', '2026-10-03 22:43:22.519743'::timestamp),
    ('bk_a290f1ee', $q$A Beginner's Guide to Scala, Object Orientation and Functional Programming$q$, 'Learn Scala from the ground up, along with the object-oriented and functional programming ideas it combines. Second edition.', 'Non-Fiction', 'Computer Science', 'John Hunt', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_a290f1ee-a-beginner-s-guide-to-scala-object-orientation-and-functiona.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_a290f1ee-2018_Book_ABeginnerSGuideToScalaObjectOr.pdf', '2026-10-03 22:43:27.417554'::timestamp),
    ('bk_704a73bd', 'Introduction to Deep Learning', 'A concise introduction to deep learning, from logical calculus and perceptrons to convolutional and recurrent networks.', 'Non-Fiction', 'Data Science', 'Sandro Skansi', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_704a73bd-introduction-to-deep-learning.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_704a73bd-2018_Book_IntroductionToDeepLearning.pdf', '2026-10-03 22:43:28.158686'::timestamp),
    ('bk_abd50953', 'Neural Networks and Deep Learning: A Textbook', 'The theory and algorithms of neural networks and deep learning, covering classical models as well as modern architectures.', 'Non-Fiction', 'Data Science', 'Charu C. Aggarwal', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_abd50953-neural-networks-and-deep-learning-a-textbook.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_abd50953-2018_Book_NeuralNetworksAndDeepLearning.pdf', '2026-10-03 22:43:29.669156'::timestamp),
    ('bk_4c70f301', 'Advanced Guide to Python 3 Programming', 'Advanced Python 3 topics: graphics, games, testing, file I/O, databases, logging, concurrency, networking and more.', 'Non-Fiction', 'Computer Science', 'John Hunt', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_4c70f301-advanced-guide-to-python-3-programming.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_4c70f301-2019_Book_AdvancedGuideToPython3Programm.pdf', '2026-10-03 22:43:32.31878'::timestamp),
    ('bk_2d425aaf', 'Barrister Parvateesam', 'A classic Telugu comic novel about Parvateesam, a village boy from Mogalturru, and his misadventures on the way to becoming a barrister.', 'Fiction', 'Humor', 'Mokkapati Narasimha Sastry', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_2d425aaf-337752698-Barrister-Parvateesam.pdf', '2026-10-03 22:43:48.118748'::timestamp),
    ('bk_a866e332', 'Mahabharatam (Chandamama)', 'The Mahabharata retold in Telugu as an illustrated serial in Chandamama magazine, collected from the 1969–70 issues.', 'Fantasy', 'Mythic Fantasy', NULL, 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_a866e332-mahabharatam-chandamama.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_a866e332-Maha-Bharatam.pdf', '2026-10-03 22:43:55.585598'::timestamp),
    ('bk_07b1cda2', 'Telugu Varnamala: Telugu Learning Kit, Module 1', 'Learn to read and write the Telugu alphabet through Roman script, with vowels, consonants and their combinations.', 'Non-Fiction', 'Language Learning', 'C P Brown Academy', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_07b1cda2-telugu-varnamala-telugu-learning-kit-module-1.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_07b1cda2-తెలుగు వర్ణమాల - TELUGU LEARNING KIT MODULE.pdf', '2026-10-03 22:43:56.800869'::timestamp),
    ('bk_4bd582a6', 'The Wind-Up Bird Chronicle', $q$Toru Okada's search for his missing cat, then his missing wife, leads him into a hidden world beneath Tokyo. Translated by Jay Rubin.$q$, 'Fiction', 'Literary Fiction', 'Haruki Murakami', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_4bd582a6-the-wind-up-bird-chronicle.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_4bd582a6-haruki murakami - the wind-up bird chronicle.pdf', '2026-10-03 22:43:57.658642'::timestamp),
    ('bk_9ad97ab2', 'The Last Wish', 'The first Witcher stories: Geralt of Rivia, a monster hunter for hire, in tales that reshape familiar fairy tales.', 'Fantasy', 'Dark Fantasy', 'Andrzej Sapkowski', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_9ad97ab2-the-last-wish.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_9ad97ab2- The Last Wish.epub', '2026-10-03 22:43:58.263975'::timestamp),
    ('bk_48ea0037', 'Blood of Elves', $q$Geralt takes Ciri, the young princess of Cintra, to the witchers' fortress as war gathers across the Continent.$q$, 'Fantasy', 'Dark Fantasy', 'Andrzej Sapkowski', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_48ea0037-blood-of-elves.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_48ea0037- Blood of Elves.epub', '2026-10-03 22:43:58.826669'::timestamp),
    ('bk_b51e2f92', 'Baptism of Fire', 'Wounded and outlawed, Geralt sets off with an unlikely company to find Ciri.', 'Fantasy', 'Dark Fantasy', 'Andrzej Sapkowski', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_b51e2f92-baptism-of-fire.jpeg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_b51e2f92- Baptism of Fire.epub', '2026-10-03 22:43:59.443356'::timestamp),
    ('bk_fb9daf65', 'The Tower of Swallows', $q$Hunted by bounty hunters and sorcerers, Ciri seeks a way out while Geralt's company closes in.$q$, 'Fantasy', 'Dark Fantasy', 'Andrzej Sapkowski', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_fb9daf65-the-tower-of-swallows.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_fb9daf65- The Tower of Swallows.epub', '2026-10-03 22:44:01.052734'::timestamp),
    ('bk_59b1cd1e', 'Lady of the Lake', $q$The final Witcher saga novel: Ciri crosses worlds and Geralt's journey reaches its end.$q$, 'Fantasy', 'Dark Fantasy', 'Andrzej Sapkowski', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/covers/bk_59b1cd1e-lady-of-the-lake.jpg', 'https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_59b1cd1e- Lady of the Lake.epub', '2026-10-03 22:44:03.183791'::timestamp)
  ) AS v(book_id, title, description, genre, subgenre, author_name, cover_image_url, manuscript_url, created_at)
 WHERE NOT EXISTS (SELECT 1 FROM books b WHERE b.manuscript_url = v.manuscript_url OR b.book_id = v.book_id);

-- 5. The "Witcher Series" curated bucket and its books.
INSERT INTO curated_buckets (id, title, sort_order, is_active, description, genre_tags)
SELECT 'cb_d5108467', 'Witcher Series', 2, 't', NULL, '{}'
 WHERE NOT EXISTS (SELECT 1 FROM curated_buckets WHERE title = 'Witcher Series');

INSERT INTO curated_bucket_books (bucket_id, book_id, sort_order)
SELECT cb.id, b.book_id, v.sort_order
  FROM (VALUES
    ('https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_9ad97ab2- The Last Wish.epub', 0),
    ('https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_48ea0037- Blood of Elves.epub', 1),
    ('https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_b51e2f92- Baptism of Fire.epub', 2),
    ('https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_fb9daf65- The Tower of Swallows.epub', 3),
    ('https://pub-baceedd91d1140ceb367750da255e3f7.r2.dev/books/manuscripts/bk_59b1cd1e- Lady of the Lake.epub', 4)
  ) AS v(manuscript_url, sort_order)
  JOIN books b ON b.manuscript_url = v.manuscript_url
  JOIN curated_buckets cb ON cb.title = 'Witcher Series'
 WHERE NOT EXISTS (SELECT 1 FROM curated_bucket_books x WHERE x.bucket_id = cb.id AND x.book_id = b.book_id);

UPDATE curated_buckets cb
   SET book_count = (SELECT COUNT(*) FROM curated_bucket_books x WHERE x.bucket_id = cb.id)
 WHERE cb.title = 'Witcher Series';

COMMIT;
