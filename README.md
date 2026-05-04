# Pet Store

A multi-tenant pet store application:

- **Merchants** manage their store via a GraphQL API (no UI).
- **Customers** browse and purchase pets through a web UI.

Each merchant has their own store; merchants cannot see or touch other stores'
inventory. Purchases are race-safe — two customers cannot end up with the
same pet, even with concurrent requests.

---

## Tech stack

| Layer         | Technology                                        |
| ------------- | ------------------------------------------------- |
| Frontend      | React 18 + TypeScript + Vite + Apollo Client      |
| Backend       | Go 1.25 + `gqlgen` + `chi` + `pgx`                |
| Protocol      | GraphQL over HTTP                                 |
| Database      | PostgreSQL 16 (self-hosted in Docker)             |
| Auth          | HTTP Basic authentication (bcrypt-hashed)         |
| Orchestration | Docker Compose                                    |

---

## How this works

The React frontend sends GraphQL mutations and queries over HTTP to the Go backend. `chi` handles routing and runs the auth middleware (HTTP Basic, bcrypt), then hands off to `gqlgen`'s generated executor which calls the resolver functions. The resolvers talk to PostgreSQL via `pgx` — using `SELECT FOR UPDATE` and transactions to make purchases race-safe.

1. React Frontend


2. GraphQL Request


3. chi (routes request)


4. gqlgen (parses schema + calls resolvers)


5. Your resolver functions


6. pgx (DB queries / transactions)


7. PostgreSQL

---

## Repository layout

```
petstore/
├── backend/                          # Go GraphQL server
│   ├── cmd/server/main.go            # HTTP entrypoint
│   ├── gqlgen.yml                    # codegen config
│   ├── graph/                        # GraphQL surface
│   │   ├── schema.graphqls           # source of truth (hand-written)
│   │   ├── generated.go              # gqlgen-generated executable schema
│   │   ├── model/models_gen.go       # gqlgen-generated typed models
│   │   ├── resolver.go               # root Resolver{Repo}
│   │   ├── schema.resolvers.go       # resolver bodies (hand-written)
│   │   └── mapping.go                # db <-> model conversion helpers
│   ├── internal/
│   │   ├── auth/                     # HTTP Basic middleware + role guards
│   │   ├── db/                       # pgx repository, models, integration tests
│   │   └── errs/                     # domain error types
│   ├── go.mod / go.sum
│   └── Dockerfile
├── frontend/                         # React + TypeScript storefront
│   ├── public/
│   │   └── seed/                     # Local pet images (cat.jpg, dog.jpg, frog.jpg)
│   ├── src/
│   │   ├── App.tsx, main.tsx, Storefront.tsx, styles.css, types.ts
│   │   ├── auth/                     # AuthProvider + Login
│   │   ├── cart/                     # CartProvider
│   │   ├── components/               # PetCard, CartDrawer, ConfirmDialog,
│   │   │                             # ToastProvider, PetCardSkeleton
│   │   └── graphql/                  # Apollo client + operations
│   ├── package.json / package-lock.json
│   ├── vite.config.ts / tsconfig.json
│   └── Dockerfile
├── db/migrations/                    # SQL schema + seed data
│   ├── 001_schema.sql
│   ├── 002_seed.sql                  # demo merchants + customer
│   └── 003_seed_pets.sql             # demo pet inventory
├── docs/USAGE.md                     # API reference with examples
├── docker-compose.yml
└── README.md
```

---

## Prerequisites

- **Docker** and **Docker Compose** v2 (`docker compose ...`).
- Ports `5173` (frontend), `8080` (backend), `5432` (postgres) free on the host.

That's it — no local Go or Node installation is required.

---

## Run the system locally

```bash
# from the repo root
docker compose up --build
```

The first build takes a couple of minutes (pulling images, fetching Go modules,
installing npm packages). Subsequent runs are fast.

When the logs settle you should see:

- Postgres: `database system is ready to accept connections`
- Backend:  `petstore backend listening on :8080`
- Frontend: `Local: http://localhost:5173/`

Open in your browser:

| URL                                          | What you'll see                       |
| -------------------------------------------- | ------------------------------------- |
| http://localhost:5173?store=PetVerse             | Customer storefront for "PetVerse"        |
| http://localhost:5173?store=BestPets         | Customer storefront for "BestPets"    |
| http://localhost:8080/playground             | GraphiQL (merchant + customer API)    |
| http://localhost:8080/healthz                | Health check                          |

To stop everything:
```bash
docker compose down          # keep the database volume
docker compose down -v       # also wipe the database
```

---

## Seeded credentials

| Role     | Username   | Password       | Scope                 |
| -------- | ---------- | -------------- | --------------------- |
| Merchant | `PetVerse`     | `PetVersepass`     | Store `PetVerse`          |
| Merchant | `BestPets` | `BestPetspass` | Store `BestPets`      |
| Customer | `astha`    | `asthapass`    | All storefronts       |

The system starts with **15 demo pets** pre-loaded across the two stores
(9 for PetVerse, 6 for BestPets — see `db/migrations/003_seed_pets.sql`),
so the storefront has content immediately. Pet images are served from
`frontend/public/seed/` (local SVGs — no external CDN). You can create more
via the merchant API at any time.

---

## Try it end-to-end

1. **Open the GraphiQL playground:** http://localhost:8080/playground
2. **Add an `Authorization` header** in the bottom panel, set to:
   ```
   Authorization: Basic UGV0VmVyc2U6UGV0VmVyc2VwYXNz
   ```
   (That's `PetVerse:PetVersepass` base64-encoded. Compute your own with `echo -n 'user:pass' | base64`.)
3. **Create a pet:**
   ```graphql
   mutation {
     createPet(input: {
       name: "Whiskers"
       species: CAT
       age: 3
       pictureUrl: "https://images.unsplash.com/photo-1574158622682-e40e69881006?w=400&h=400&fit=crop"
       description: "A friendly tabby."
     }) {
       id name createdAt
     }
   }
   ```
4. **Open the storefront:** http://localhost:5173?store=PetVerse — sign in as
   `astha / asthapass` and you'll see Whiskers. Click **Buy now** or **Add to
   cart** → **Checkout**.
5. **Query sold pets** as the merchant:
   ```graphql
   query {
     soldPets(startDate: "2026-01-01T00:00:00Z", endDate: "2026-12-31T23:59:59Z") {
       id name soldAt
     }
   }
   ```

See [docs/USAGE.md](docs/USAGE.md) for the full API reference.

---

## How requirements are satisfied

### Merchant

| Requirement                                       | Where                                       |
| ------------------------------------------------- | ------------------------------------------- |
| Create pet (name/species/age/picture/desc/createdAt) | `createPet` mutation                     |
| Remove pet (only before purchase)                 | `deletePet` mutation; locks row + checks `purchases` |
| Query sold pets by inclusive date range           | `soldPets(startDate, endDate)` query        |
| Query unsold pets                                 | `unsoldPets` query                          |

### Customer

| Requirement                                       | Where                                       |
| ------------------------------------------------- | ------------------------------------------- |
| Public store URL with available pets              | `/?store=<slug>` storefront                 |
| Buy now (instant purchase, error if unavailable)  | `purchasePet` mutation; UI shows the human-readable error |
| Add to cart + checkout (with names of unavailable pets in error) | `checkout` mutation; UI surfaces names |

### Cross-cutting

- **Multi-tenant isolation:** every merchant resolver pulls `store_id` from
  the authenticated user — there is no way to pass a `store_id` argument
  from the client.
- **Role separation:** `RequireMerchant` / `RequireCustomer` reject the wrong
  role with a 403-equivalent GraphQL error.
- **Race conditions:**
  - `purchasePet` runs `SELECT … FOR UPDATE` on the pet row, then inserts
    into `purchases`. The `purchases.pet_id UNIQUE` constraint is the
    backstop — if two requests slip past the lock, exactly one INSERT wins
    and the other receives a "no longer available" error.
  - `checkout` locks all selected pet rows in a single transaction (sorted
    by id to avoid deadlocks), validates atomically, then inserts. If any
    pet was already sold the whole transaction rolls back and the user gets
    an error listing the offending names.
  - `deletePet` locks the pet row and checks `purchases` before soft-deleting.

---

## Development without Docker

If you want hot reload outside Docker:

```bash
# Database
docker compose up db

# Backend (requires Go 1.25+)
cd backend
DATABASE_URL='postgres://petstore:petstore@localhost:5432/petstore?sslmode=disable' \
  go run ./cmd/server

# Frontend (requires Node 20)
cd frontend
npm install
npm run dev
```

---

## Running the test suite

The backend ships with integration tests in `backend/internal/db/`
covering multi-tenant isolation, atomicity, the duplicate-IDs fix, and a
50-goroutine race-condition test that proves exactly-once purchase semantics.

```bash
docker compose up -d db
cd backend
go test ./internal/db/... -v
```

If the database isn't reachable, every test calls `t.Skip()` so plain
`go test ./...` still passes.

> macOS note: if a native Postgres is already listening on port 5432, the test
> binary on the host will hit the wrong server. Either stop the native Postgres,
> or run the tests inside the docker network (`docker run --rm
> --network petstore_claude_default -v "$PWD/backend":/src -w /src
> -e TEST_DATABASE_URL='postgres://petstore:petstore@db:5432/petstore?sslmode=disable'
> golang:1.25-alpine sh -c "apk add --no-cache git >/dev/null && go test ./internal/db/... -v"`).
> Linux users should not encounter this conflict.

---

## Troubleshooting

**Backend can't reach Postgres on first boot.** The backend retries for 30s;
if Postgres is slow, just `docker compose restart backend`.

**Storefront shows "Could not reach the server."** Confirm the backend is up
(`curl http://localhost:8080/healthz`) and that you visited the URL on
`localhost` (CORS is locked to `http://localhost:5173`).

**Want a clean slate?** `docker compose down -v && docker compose up --build`
recreates the database from scratch.

---

## Submission notes

- Built and tested on macOS (Apple Silicon).
- Verified with Docker Desktop 4.x and Docker Compose v2.
- All Docker images used (`postgres:16-alpine`, `golang:1.25-alpine`,
  `node:20-alpine`) are multi-architecture, so the same `docker compose up
  --build` command works on Linux x86_64 graders' machines without changes.
- No external services; everything runs locally.
