-- Demo pets so a fresh `docker compose up` shows a populated storefront.
-- Pictures are served from the frontend's own static assets (frontend/public/seed/)

INSERT INTO pets (store_id, name, species, age, picture_url, description)
SELECT s.id, p.name, p.species::pet_species, p.age, p.picture_url, p.description
FROM stores s
JOIN (VALUES
    -- PetVerse store (9 pets)
    ('PetVerse', 'Whiskers', 'CAT',  3, 'http://localhost:5173/seed/whiskers.jpg', 'A friendly tabby who loves windowsills.'),
    ('PetVerse', 'Nori',     'DOG',  2, 'http://localhost:5173/seed/nori.jpg', 'A playful golden retriever puppy.'),
    ('PetVerse', 'Hopper',   'FROG', 1, 'http://localhost:5173/seed/hopper.jpg', 'A bright green tree frog.'),
    ('PetVerse', 'Mango',    'CAT',  2, 'http://localhost:5173/seed/mango.jpg', 'An adventurous orange tabby.'),
    ('PetVerse', 'Pepper',   'DOG',  5, 'http://localhost:5173/seed/pepper.jpg', 'A calm, well-trained border collie.'),
    ('PetVerse', 'Bubbles',  'FROG', 1, 'http://localhost:5173/seed/bubbles.jpg', 'A tiny blue poison dart frog — look, don''t touch!'),
    ('PetVerse', 'Cleo',     'CAT',  6, 'http://localhost:5173/seed/cleo.jpg', 'A regal Egyptian Mau with striking spots.'),
    ('PetVerse', 'Rex',      'DOG',  3, 'http://localhost:5173/seed/rex.jpg', 'A boisterous Labrador who never skips fetch.'),
    ('PetVerse', 'Jade',     'FROG', 2, 'http://localhost:5173/seed/jade.jpg', 'A jade-green tree frog fond of evening rain.'),
    -- BestPets store (6 pets)
    ('BestPets', 'Biscuit',  'DOG',  4, 'http://localhost:5173/seed/biscuit.jpg', 'A loyal corgi who loves long walks.'),
    ('BestPets', 'Luna',     'CAT',  1, 'http://localhost:5173/seed/luna.jpg', 'A curious kitten just out of her shell.'),
    ('BestPets', 'Kermit',   'FROG', 3, 'http://localhost:5173/seed/golden.jpg', 'A cheerful bullfrog with an enormous smile.'),
    ('BestPets', 'Rusty',    'DOG',  7, 'http://localhost:5173/seed/rusty.jpg', 'A gentle old-timer who loves afternoon naps.'),
    ('BestPets', 'Mittens',  'CAT',  4, 'http://localhost:5173/seed/kitten.jpg', 'A polydactyl cat with extra-wide paws.'),
    ('BestPets', 'Dart',     'FROG', 1, 'http://localhost:5173/seed/golden.jpg', 'A speedy little red-eyed tree frog.')
) AS p(slug, name, species, age, picture_url, description) ON s.slug = p.slug;
