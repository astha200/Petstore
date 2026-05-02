-- Pet store schema. Loaded by the postgres container on first boot.
SET client_min_messages = WARNING;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Stores (one per merchant tenant).
CREATE TABLE stores (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users: merchants are scoped to a store; customers have store_id NULL.
CREATE TYPE user_role AS ENUM ('merchant', 'customer');

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    role           user_role NOT NULL,
    store_id       UUID REFERENCES stores(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT merchant_has_store CHECK (
        (role = 'merchant' AND store_id IS NOT NULL) OR
        (role = 'customer' AND store_id IS NULL)
    )
);

CREATE TYPE pet_species AS ENUM ('CAT', 'DOG', 'FROG');

CREATE TABLE pets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id    UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    species     pet_species NOT NULL,
    age         INTEGER NOT NULL CHECK (age >= 0),
    picture_url TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_pets_store ON pets (store_id) WHERE deleted_at IS NULL;

-- Purchases: a pet can only be purchased once. The UNIQUE constraint on pet_id
-- is the last line of defense against double-buy race conditions.
CREATE TABLE purchases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pet_id       UUID NOT NULL UNIQUE REFERENCES pets(id) ON DELETE CASCADE,
    customer_id  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_purchases_purchased_at ON purchases (purchased_at);
