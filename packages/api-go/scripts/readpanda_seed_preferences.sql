-- Genre/subgenre catalogue shown during onboarding. Run after
-- readpanda_schema.sql. Safe to re-run.

INSERT INTO public.preferences (id, subgenre, description, genre) VALUES
    (1,  'Dark Fantasy',         'Fantasy with grim themes and moral ambiguity.',         'Fantasy'),
    (2,  'Self-Help',            'Practical guidance for personal growth.',               'Non-Fiction'),
    (3,  'Thriller',             'High-stakes tension with urgent pacing.',               'Mystery'),
    (4,  'YA Romance',           'Coming-of-age romantic stories for YA readers.',        'Young Adult'),
    (5,  'Police Procedural',    'Crime-solving centered around police process.',         'Mystery'),
    (6,  'Mythic Fantasy',       'Stories inspired by folklore and mythology.',           'Fantasy'),
    (7,  'Contemporary Romance', 'Modern-day relationships and emotional arcs.',          'Romance'),
    (8,  'YA Fantasy',           'Fantasy stories tailored to teen and YA readers.',      'Young Adult'),
    (9,  'Cyberpunk',            'High-tech worlds with powerful institutions.',          'Science Fiction'),
    (10, 'History',              'Narratives and analysis of historical events.',         'Non-Fiction'),
    (11, 'Science',              'Accessible writing on scientific ideas and discovery.', 'Non-Fiction'),
    (12, 'Biography',            'Life stories of real people.',                          'Non-Fiction'),
    (13, 'Space Opera',          'Adventure-driven stories across space and planets.',    'Science Fiction'),
    (14, 'Supernatural Horror',  'Terror involving paranormal forces.',                   'Horror'),
    (15, 'Historical Romance',   'Love stories set in past eras.',                        'Romance'),
    (16, 'Urban Fantasy',        'Fantasy elements grounded in modern settings.',         'Fantasy'),
    (17, 'Time Travel',          'Plots driven by temporal shifts and paradoxes.',        'Science Fiction'),
    (18, 'Epic Fantasy',         'Large-scale stories with rich world-building.',         'Fantasy'),
    (19, 'Psychological Horror', 'Fear rooted in the mind and perception.',               'Horror'),
    (20, 'Romantic Suspense',    'Romance combined with danger and mystery.',             'Romance'),
    (21, 'Cozy Mystery',         'Light-toned mysteries focused on puzzle solving.',      'Mystery'),
    (22, 'Detective',            'Investigative stories led by a central sleuth.',        'Mystery'),
    (23, 'Dystopian',            'Speculative futures shaped by social collapse.',        'Science Fiction')
ON CONFLICT (id) DO NOTHING;

-- The rows above set ids explicitly, so move the identity sequence past them.
SELECT setval('public.preferences_id_seq', (SELECT max(id) FROM public.preferences));
