// Package db is the data-access layer. All SQL lives here.
//
// Race-condition strategy:
//   - Purchase locks the pet row with SELECT ... FOR UPDATE.
//   - purchases.pet_id has a UNIQUE constraint as a final safeguard.
//   - Checkout locks requested pets in sorted order to avoid deadlocks,
//     validates availability, then inserts purchases in one transaction.
package db

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/petstore/backend/internal/errs"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// Users

func (r *Repo) UserByUsername(ctx context.Context, username string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, store_id FROM users WHERE username = $1`,
		username,
	)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.StoreID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrUnauthorized
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) StoreBySlug(ctx context.Context, slug string) (*Store, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, slug, name FROM stores WHERE slug = $1`, slug)
	var s Store
	if err := row.Scan(&s.ID, &s.Slug, &s.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrStoreUnknown
		}
		return nil, err
	}
	return &s, nil
}

// merchant

func (r *Repo) CreatePet(ctx context.Context, storeID uuid.UUID, name string, species Species, age int, pictureURL, description string) (*Pet, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO pets (store_id, name, species, age, picture_url, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, store_id, name, species, age, picture_url, description, created_at
	`, storeID, name, species, age, pictureURL, description)

	var p Pet
	if err := row.Scan(&p.ID, &p.StoreID, &p.Name, &p.Species, &p.Age, &p.PictureURL, &p.Description, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePet soft-deletes the pet. Fails if the pet was already purchased or belongs to a different store. 
// Locks the pet row to be safe against a concurrent purchase.
func (r *Repo) DeletePet(ctx context.Context, storeID, petID uuid.UUID) error {
	return r.tx(ctx, func(tx pgx.Tx) error {
		var ownerID uuid.UUID
		var deletedAt *time.Time
		err := tx.QueryRow(ctx, `
			SELECT store_id, deleted_at FROM pets WHERE id = $1 FOR UPDATE
		`, petID).Scan(&ownerID, &deletedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errs.ErrPetNotFound
			}
			return err
		}
		// Treat cross-tenant access the same as not-found so we don't leak whether
		// a pet exists in another store.
		if ownerID != storeID {
			return errs.ErrPetNotFound
		}
		if deletedAt != nil {
			return errs.ErrPetNotFound
		}

		var purchased bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM purchases WHERE pet_id = $1)`, petID,
		).Scan(&purchased); err != nil {
			return err
		}
		if purchased {
			return &errs.PetUnavailableError{}
		}

		_, err = tx.Exec(ctx, `UPDATE pets SET deleted_at = now() WHERE id = $1`, petID)
		return err
	})
}

func (r *Repo) UnsoldPets(ctx context.Context, storeID uuid.UUID) ([]Pet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, store_id, name, species, age, picture_url, description, created_at
		FROM pets
		WHERE store_id = $1
		  AND deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM purchases WHERE purchases.pet_id = pets.id)
		ORDER BY created_at DESC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPets(rows)
}

func (r *Repo) SoldPets(ctx context.Context, storeID uuid.UUID, start, end time.Time) ([]SoldPet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.store_id, p.name, p.species, p.age, p.picture_url, p.description, p.created_at, pu.purchased_at
		FROM pets p
		JOIN purchases pu ON pu.pet_id = p.id
		WHERE p.store_id = $1
		  AND pu.purchased_at >= $2
		  AND pu.purchased_at <= $3
		ORDER BY pu.purchased_at DESC
	`, storeID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SoldPet, 0)
	for rows.Next() {
		var s SoldPet
		if err := rows.Scan(&s.ID, &s.StoreID, &s.Name, &s.Species, &s.Age, &s.PictureURL, &s.Description, &s.CreatedAt, &s.SoldAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// customer

func (r *Repo) AvailablePetsByStoreSlug(ctx context.Context, storeSlug string) ([]Pet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.store_id, p.name, p.species, p.age, p.picture_url, p.description, p.created_at
		FROM pets p
		JOIN stores s ON s.id = p.store_id
		WHERE s.slug = $1
		  AND p.deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM purchases WHERE purchases.pet_id = p.id)
		ORDER BY p.created_at DESC
	`, storeSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPets(rows)
}

// PurchasePet locks the pet row, verifies it is in the right store and not yet sold, then records the purchase. 
// The UNIQUE constraint on purchases.pet_id is the last line of defense.
func (r *Repo) PurchasePet(ctx context.Context, customerID uuid.UUID, storeSlug string, petID uuid.UUID) (*Pet, error) {
	var pet Pet
	err := r.tx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT p.id, p.store_id, p.name, p.species, p.age, p.picture_url, p.description, p.created_at
			FROM pets p
			JOIN stores s ON s.id = p.store_id
			WHERE p.id = $1
			  AND s.slug = $2
			  AND p.deleted_at IS NULL
			FOR UPDATE OF p
		`, petID, storeSlug)
		if err := row.Scan(&pet.ID, &pet.StoreID, &pet.Name, &pet.Species, &pet.Age, &pet.PictureURL, &pet.Description, &pet.CreatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errs.ErrPetNotFound
			}
			return err
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO purchases (pet_id, customer_id) VALUES ($1, $2)`,
			petID, customerID,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return &errs.PetUnavailableError{Names: []string{pet.Name}}
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

// Checkout buys multiple pets atomically. If ANY pet is unavailable the entire
// transaction rolls back and an error listing the unavailable pet names is returned.
func (r *Repo) Checkout(ctx context.Context, customerID uuid.UUID, storeSlug string, petIDs []uuid.UUID) ([]Pet, error) {
	if len(petIDs) == 0 {
		return nil, errs.ErrInvalidInput
	}

	// Dedupe IDs and lock pets in a stable order to reduce deadlock risk.
	seen := make(map[uuid.UUID]struct{}, len(petIDs))
	ordered := make([]uuid.UUID, 0, len(petIDs))
	for _, id := range petIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })

	purchased := make([]Pet, 0, len(ordered))
	err := r.tx(ctx, func(tx pgx.Tx) error {
		// Lock and load all requested pets that are still on the market.
		rows, err := tx.Query(ctx, `
			SELECT p.id, p.store_id, p.name, p.species, p.age, p.picture_url, p.description, p.created_at
			FROM pets p
			JOIN stores s ON s.id = p.store_id
			WHERE p.id = ANY($1)
			  AND s.slug = $2
			  AND p.deleted_at IS NULL
			  AND NOT EXISTS (SELECT 1 FROM purchases WHERE purchases.pet_id = p.id)
			ORDER BY p.id
			FOR UPDATE OF p
		`, ordered, storeSlug)
		if err != nil {
			return err
		}
		available, err := scanPets(rows)
		rows.Close()
		if err != nil {
			return err
		}

		availableSet := make(map[uuid.UUID]Pet, len(available))
		for _, p := range available {
			availableSet[p.ID] = p
		}

		// Identify any requested pet that is no longer available.
		if len(available) != len(ordered) {
			missing := unavailableNames(ctx, tx, ordered, availableSet)
			return &errs.PetUnavailableError{Names: missing}
		}

		// UNIQUE(pet_id) protects against concurrent purchases.
		for _, p := range available {
			if _, err := tx.Exec(ctx,
				`INSERT INTO purchases (pet_id, customer_id) VALUES ($1, $2)`,
				p.ID, customerID,
			); err != nil {
				if isUniqueViolation(err) {
					return &errs.PetUnavailableError{Names: []string{p.Name}}
				}
				return err
			}
			purchased = append(purchased, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return purchased, nil
}

// ----- helpers -----

func unavailableNames(ctx context.Context, tx pgx.Tx, requested []uuid.UUID, available map[uuid.UUID]Pet) []string {
	missingIDs := make([]uuid.UUID, 0)
	for _, id := range requested {
		if _, ok := available[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT name FROM pets WHERE id = ANY($1) ORDER BY name`, missingIDs)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]string, 0, len(missingIDs))
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (r *Repo) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func scanPets(rows pgx.Rows) ([]Pet, error) {
	out := make([]Pet, 0)
	for rows.Next() {
		var p Pet
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Species, &p.Age, &p.PictureURL, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
