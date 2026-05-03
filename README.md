# Petstore

A multi-tenant pet store application:

- **Merchants** manage their store via a GraphQL API (no UI).
- **Customers** browse and purchase pets through a web UI.

## Tech stack

| Layer         | Technology                                        |
| ------------- | ------------------------------------------------- |
| Frontend      | React 18 + TypeScript + Vite + Apollo Client      |
| Backend       | Go 1.25 + `gqlgen` + `chi` + `pgx`                |
| Protocol      | GraphQL over HTTP                                 |
| Database      | PostgreSQL 16 (self-hosted in Docker)             |
| Auth          | HTTP Basic authentication (bcrypt-hashed)         |
| Orchestration | Docker Compose                                    |

## How this works

React Frontend
      ↓

GraphQL Request
      ↓

chi (routes request)
      ↓

gqlgen (parses schema + calls resolvers)
      ↓

Your resolver functions
      ↓

pgx (DB queries / transactions)
      ↓

PostgreSQL

## Quick start

Run the full stack locally:

```bash
docker compose up --build
```

Open in your browser:

| URL                                          | What you'll see                       |
| -------------------------------------------- | ------------------------------------- |
| http://localhost:5173?store=PetVerse         | Customer storefront for "PetVerse"    |
| http://localhost:5173?store=BestPets         | Customer storefront for "BestPets"    |
| http://localhost:8080/playground             | GraphiQL (merchant + customer API)    |
| http://localhost:8080/healthz                | Health check                          |

To stop everything:
```bash
docker compose down          # keep the database volume
docker compose down -v       # also wipe the database
```