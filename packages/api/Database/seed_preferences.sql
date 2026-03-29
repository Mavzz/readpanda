-- Seed default reading preferences.
-- Idempotent: re-running this file does not duplicate existing genre/subgenre pairs.

INSERT INTO public.preferences (genre, subgenre, description)
SELECT v.genre, v.subgenre, v.description
FROM (
  VALUES
    ('Fantasy', 'Epic Fantasy', 'Large-scale stories with rich world-building.'),
    ('Fantasy', 'Urban Fantasy', 'Fantasy elements grounded in modern settings.'),
    ('Fantasy', 'Dark Fantasy', 'Fantasy with grim themes and moral ambiguity.'),
    ('Fantasy', 'Mythic Fantasy', 'Stories inspired by folklore and mythology.'),

    ('Science Fiction', 'Dystopian', 'Speculative futures shaped by social collapse.'),
    ('Science Fiction', 'Space Opera', 'Adventure-driven stories across space and planets.'),
    ('Science Fiction', 'Cyberpunk', 'High-tech worlds with powerful institutions.'),
    ('Science Fiction', 'Time Travel', 'Plots driven by temporal shifts and paradoxes.'),

    ('Mystery', 'Cozy Mystery', 'Light-toned mysteries focused on puzzle solving.'),
    ('Mystery', 'Detective', 'Investigative stories led by a central sleuth.'),
    ('Mystery', 'Police Procedural', 'Crime-solving centered around police process.'),
    ('Mystery', 'Thriller', 'High-stakes tension with urgent pacing.'),

    ('Romance', 'Contemporary Romance', 'Modern-day relationships and emotional arcs.'),
    ('Romance', 'Historical Romance', 'Love stories set in past eras.'),
    ('Romance', 'Romantic Suspense', 'Romance combined with danger and mystery.'),

    ('Horror', 'Psychological Horror', 'Fear rooted in the mind and perception.'),
    ('Horror', 'Supernatural Horror', 'Terror involving paranormal forces.'),

    ('Non-Fiction', 'Biography', 'Life stories of real people.'),
    ('Non-Fiction', 'History', 'Narratives and analysis of historical events.'),
    ('Non-Fiction', 'Self-Help', 'Practical guidance for personal growth.'),
    ('Non-Fiction', 'Science', 'Accessible writing on scientific ideas and discovery.'),

    ('Young Adult', 'YA Fantasy', 'Fantasy stories tailored to teen and YA readers.'),
    ('Young Adult', 'YA Romance', 'Coming-of-age romantic stories for YA readers.')
) AS v(genre, subgenre, description)
WHERE NOT EXISTS (
  SELECT 1
  FROM public.preferences p
  WHERE p.subgenre = v.subgenre
    AND p.genre IS NOT DISTINCT FROM v.genre
);
