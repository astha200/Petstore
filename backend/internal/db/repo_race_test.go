package db_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/petstore/backend/internal/db"
	"github.com/petstore/backend/internal/errs"
)

// Ensures only one concurrent purchase succeeds for the same pet.
// All other attempts should fail with PetUnavailableError.
//
// Run with:
//
//	docker compose up -d db
//	go test ./internal/db/... -run TestPurchasePet_OnlyOneWinsUnderRace -v
func TestPurchasePet_OnlyOneWinsUnderRace(t *testing.T) {
	pool := openTestDB(t)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := db.New(pool)

	// Use seeded data from the local dev setup.
	var aliceID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID); err != nil {
		t.Fatalf("seeded customer 'alice' not found (run docker compose first): %v", err)
	}

	var storeID uuid.UUID
	var storeSlug string
	if err := pool.QueryRow(ctx, `SELECT id, slug FROM stores ORDER BY created_at LIMIT 1`).
		Scan(&storeID, &storeSlug); err != nil {
		t.Fatalf("no seeded stores: %v", err)
	}

	// Create a new pet so the test does not modify demo pets.
	pet, err := repo.CreatePet(ctx, storeID, "race-test-"+uuid.NewString()[:8],
		db.SpeciesCat, 1,
		"https://example.com/p.jpg",
		"transient pet for the race-condition test")
	if err != nil {
		t.Fatalf("create pet: %v", err)
	}
	t.Cleanup(func() {
		// Best-effort cleanup in case the DB is still running.
		_, _ = pool.Exec(context.Background(), `DELETE FROM purchases WHERE pet_id = $1`, pet.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM pets WHERE id = $1`, pet.ID)
	})

	const concurrency = 50

	var (
		wg          sync.WaitGroup
		successes   atomic.Int32
		unavailable atomic.Int32
		other       atomic.Int32
	)

	// Start all goroutines at roughly the same time.
	start := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.PurchasePet(ctx, aliceID, storeSlug, pet.ID)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.As(err, new(*errs.PetUnavailableError)):
				unavailable.Add(1)
			default:
				other.Add(1)
				t.Logf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 successful purchase, got %d", got)
	}
	if got := unavailable.Load(); got != concurrency-1 {
		t.Errorf("expected %d PetUnavailableError, got %d", concurrency-1, got)
	}
	if got := other.Load(); got != 0 {
		t.Errorf("got %d unexpected errors (see test logs)", got)
	}

	// Sanity check: only one purchase row should exist.
	var rowCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM purchases WHERE pet_id = $1`, pet.ID).
		Scan(&rowCount); err != nil {
		t.Fatalf("count purchases: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("expected 1 purchase row, got %d", rowCount)
	}
}

// Opens the local test database, or skips if it is not running.
func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://petstore:petstore@localhost:5432/petstore?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot create pool (%v)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: database not reachable (%v)", err)
	}
	return pool
}
