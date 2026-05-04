-- Seed data: two demo merchant stores and one demo customer.
-- Passwords (bcrypt cost 10):
--   PetVerse   -> PetVersepass
--   BestPets   -> BestPetspass
--   astha      -> asthapass

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
VALUES ('astha', '$2a$10$K8pLp.iWy7qlIGTLF5Fyhu/eIZjQH6qwoY4ss25tyiAuqjNQpyuWS', 'customer'::user_role, NULL);
