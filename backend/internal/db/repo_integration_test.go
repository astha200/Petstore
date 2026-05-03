package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/petstore/backend/internal/db"
	"github.com/petstore/backend/internal/errs"
)

// Ensures a merchant cannot delete a pet from another store.
// We return ErrPetNotFound to avoid leaking cross-tenant data.
func TestDeletePet_RejectsCrossTenant(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()
	ctx := context.Background()
	repo := db.New(pool)

	_, storeAID := freshStore(t, pool)
	_, storeBID := freshStore(t, pool)

	pet, err := repo.CreatePet(ctx, storeBID, "victim", db.SpeciesCat, 1,
		"https://example.com/p.jpg", "owned by store B")
	if err != nil {
		t.Fatalf("create pet: %v", err)
	}

	// Merchant A tries to delete a pet that belongs to store B.
	err = repo.DeletePet(ctx, storeAID, pet.ID)
	if !errors.Is(err, errs.ErrPetNotFound) {
		t.Fatalf("expected ErrPetNotFound, got %v", err)
	}

	// And the pet must still be present in its real store.
	pets, err := repo.UnsoldPets(ctx, storeBID)
	if err != nil {
		t.Fatalf("unsold pets: %v", err)
	}
	if len(pets) != 1 || pets[0].ID != pet.ID {
		t.Errorf("pet should still belong to store B; got %d pets back", len(pets))
	}
}

// Ensures UnsoldPets only returns pets for the given store.
func TestUnsoldPets_ScopedToCallersStore(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()
	ctx := context.Background()
	repo := db.New(pool)

	_, storeAID := freshStore(t, pool)
	_, storeBID := freshStore(t, pool)

	petA, err := repo.CreatePet(ctx, storeAID, "alpha", db.SpeciesCat, 1, "u", "d")
	if err != nil {
		t.Fatalf("create pet A: %v", err)
	}
	petB, err := repo.CreatePet(ctx, storeBID, "beta", db.SpeciesDog, 2, "u", "d")
	if err != nil {
		t.Fatalf("create pet B: %v", err)
	}

	aPets, err := repo.UnsoldPets(ctx, storeAID)
	if err != nil {
		t.Fatalf("unsold pets A: %v", err)
	}
	if len(aPets) != 1 || aPets[0].ID != petA.ID {
		t.Errorf("store A should see only its own pet; got %d", len(aPets))
	}

	bPets, err := repo.UnsoldPets(ctx, storeBID)
	if err != nil {
		t.Fatalf("unsold pets B: %v", err)
	}
	if len(bPets) != 1 || bPets[0].ID != petB.ID {
		t.Errorf("store B should see only its own pet; got %d", len(bPets))
	}
}

// Ensures duplicate pet IDs in checkout are handled correctly.
// Only one purchase should be recorded.
func TestCheckout_DedupesPetIDs(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()
	ctx := context.Background()
	repo := db.New(pool)

	storeSlug, storeID := freshStore(t, pool)
	customerID := freshCustomer(t, pool)

	pet, err := repo.CreatePet(ctx, storeID, "dedupe-target", db.SpeciesFrog, 1,
		"https://example.com/p.jpg", "test pet")
	if err != nil {
		t.Fatalf("create pet: %v", err)
	}

	purchased, err := repo.Checkout(ctx, customerID, storeSlug,
		[]uuid.UUID{pet.ID, pet.ID, pet.ID})
	if err != nil {
		t.Fatalf("checkout with duplicate IDs should succeed; got: %v", err)
	}
	if len(purchased) != 1 {
		t.Errorf("expected 1 purchased pet, got %d", len(purchased))
	}

	var rows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM purchases WHERE pet_id = $1`, pet.ID,
	).Scan(&rows); err != nil {
		t.Fatalf("count purchases: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 purchase row, got %d", rows)
	}
}

// Ensures checkout fails if any pet is unavailable.
// No pets should be purchased and the error should list the unavailable one.
func TestCheckout_AtomicOnPartialUnavailability(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()
	ctx := context.Background()
	repo := db.New(pool)

	storeSlug, storeID := freshStore(t, pool)
	earlyBuyer := freshCustomer(t, pool)
	cartCustomer := freshCustomer(t, pool)

	apple, err := repo.CreatePet(ctx, storeID, "Apple", db.SpeciesCat, 1, "u", "d")
	if err != nil {
		t.Fatalf("create Apple: %v", err)
	}
	banana, err := repo.CreatePet(ctx, storeID, "Banana", db.SpeciesDog, 2, "u", "d")
	if err != nil {
		t.Fatalf("create Banana: %v", err)
	}
	cherry, err := repo.CreatePet(ctx, storeID, "Cherry", db.SpeciesFrog, 1, "u", "d")
	if err != nil {
		t.Fatalf("create Cherry: %v", err)
	}

	// Banana sells out from under cartCustomer before they check out.
	if _, err := repo.PurchasePet(ctx, earlyBuyer, storeSlug, banana.ID); err != nil {
		t.Fatalf("early buy of Banana: %v", err)
	}

	// cartCustomer tries to check out all three; should fail naming Banana.
	_, err = repo.Checkout(ctx, cartCustomer, storeSlug,
		[]uuid.UUID{apple.ID, banana.ID, cherry.ID})
	if err == nil {
		t.Fatal("checkout should have failed because Banana is sold")
	}

	var unavail *errs.PetUnavailableError
	if !errors.As(err, &unavail) {
		t.Fatalf("expected *PetUnavailableError, got %T: %v", err, err)
	}
	if len(unavail.Names) != 1 || unavail.Names[0] != "Banana" {
		t.Errorf("expected error to name only 'Banana', got %v", unavail.Names)
	}

	// Atomicity: Apple and Cherry must NOT have been purchased.
	for _, p := range []struct {
		name string
		id   uuid.UUID
	}{{"Apple", apple.ID}, {"Cherry", cherry.ID}} {
		var rows int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM purchases WHERE pet_id = $1`, p.id,
		).Scan(&rows); err != nil {
			t.Fatalf("count %s: %v", p.name, err)
		}
		if rows != 0 {
			t.Errorf("atomicity violated: %s was purchased despite checkout failing", p.name)
		}
	}
}

// Ensures looking up an unknown store returns ErrStoreUnknown so the
// frontend can show a distinct "store not found" message
func TestStoreBySlug_UnknownReturnsErrStoreUnknown(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()
	repo := db.New(pool)

	_, err := repo.StoreBySlug(context.Background(), "definitely-not-a-real-store-"+uuid.NewString()[:8])
	if !errors.Is(err, errs.ErrStoreUnknown) {
		t.Fatalf("expected ErrStoreUnknown, got %v", err)
	}
}

// Creates a test store and cleans it up after the test.
func freshStore(t *testing.T, pool *pgxpool.Pool) (slug string, id uuid.UUID) {
	t.Helper()
	slug = "test-store-" + uuid.NewString()[:8]
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO stores (slug, name) VALUES ($1, $2) RETURNING id`,
		slug, slug,
	).Scan(&id); err != nil {
		t.Fatalf("create test store: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE on pets.store_id and purchases.pet_id wipes everything.
		_, _ = pool.Exec(context.Background(), `DELETE FROM stores WHERE id = $1`, id)
	})
	return slug, id
}

// Creates a test customer and cleans up related data after the test.
func freshCustomer(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	username := "test-customer-" + uuid.NewString()[:8]
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash, role, store_id)
		 VALUES ($1, 'no-real-hash', 'customer'::user_role, NULL)
		 RETURNING id`,
		username,
	).Scan(&id); err != nil {
		t.Fatalf("create test customer: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM purchases WHERE customer_id = $1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// docker run --rm --network petstore_default -v "$PWD/backend":/src -w /src -e TEST_DATABASE_URL='postgres://petstore:petstore@db:5432/petstore?sslmode=disable' golang:1.25-alpine sh -c "apk add --no-cache git >/dev/null && go test ./internal/db/... -v"
