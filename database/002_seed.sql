-- Seed data: two demo merchant stores and one demo customer.
-- Passwords (bcrypt cost 10):
--   PetVerse   -> PetVersepass
--   BestPets   -> BestPetspass
--   alice      -> alicepass

WITH new_stores AS (
    INSERT INTO stores (slug, name) VALUES
        ('PetVerse', 'PetVerse'),
        ('BestPets', 'BestPets')
    RETURNING id, slug
)
INSERT INTO users (username, password_hash, role, store_id)
SELECT 'PetVerse', '$2b$10$a7c5IutPxRaowqMYNzp12e5dqmaig2GrggemNP2fHGN3v.h38372i', 'merchant'::user_role, id FROM new_stores WHERE slug = 'PetVerse'
UNION ALL
SELECT 'BestPets', '$2b$10$bHyEAbj8qh/TBkrVshB6/.9gVoJ22Gatt6boGa2zQ0uNt8nnT6Pv2', 'merchant'::user_role, id FROM new_stores WHERE slug = 'BestPets';

INSERT INTO users (username, password_hash, role, store_id)
VALUES ('alice', '$2b$10$YU1HaRdStSLbu3rj.Z2WuOt74HtF4ZZ2gNL9ouDEGxDb2O619P39u', 'customer'::user_role, NULL);
