// Package main is the entrypoint for the pet store backend.
//
// Exposes:
//   - POST /query       GraphQL API (Basic auth)
//   - GET  /playground  GraphQL playground
//   - GET  /healthz     health check
//
// Configuration comes from environment variables.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/petstore/backend/graph"
	"github.com/petstore/backend/internal/auth"
	"github.com/petstore/backend/internal/db"
)

func main() {
	cfg := loadConfig()

	pool, err := waitForDB(cfg.DSN, 30*time.Second)
	if err != nil {
		log.Fatalf("database not ready: %v", err)
	}
	defer pool.Close()

	repo := db.New(pool)
	gql := newGraphQLServer(repo)
	router := newRouter(repo, gql, cfg.CORSOrigin)

	runWithGracefulShutdown(router, ":"+cfg.Port)
}

// configuration

type config struct {
	DSN        string
	Port       string
	CORSOrigin string
}

func loadConfig() config {
	return config{
		DSN:        envOr("DATABASE_URL", "postgres://petstore:petstore@localhost:5432/petstore?sslmode=disable"),
		Port:       envOr("PORT", "8080"),
		CORSOrigin: envOr("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// HTTP routing

// newRouter wires the chi router with our middleware stack and routes.
func newRouter(repo *db.Repo, gql http.Handler, corsOrigin string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Logger)
	r.Use(corsHandler(corsOrigin))

	// Liveness probe; unauthenticated so load balancers can reach it.
	r.Get("/healthz", health)

	// GraphQL API; Basic auth resolves the caller and stores them on the request context.
	r.With(auth.Middleware(repo)).Handle("/query", gql)

	// GraphiQL UI; unauthenticated. Users supply an Authorization header from inside the playground.
	r.Handle("/playground", playground.Handler("Pet Store · GraphiQL", "/query"))

	return r
}

func corsHandler(origin string) func(http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{origin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// GraphQL server

// Sets up the GraphQL server with transports, caching, and introspection.
func newGraphQLServer(repo *db.Repo) *handler.Server {
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Repo: repo},
	}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
	return srv
}

// Starts the server and shuts it down gracefully on SIGINT/SIGTERM.
func runWithGracefulShutdown(handler http.Handler, addr string) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("petstore backend listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatalf("listen: %v", err)
	case <-stop:
		log.Println("shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// Waits for the database to become reachable before starting the server.
func waitForDB(dsn string, timeout time.Duration) (*pgxpool.Pool, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pool, err := pgxpool.New(context.Background(), dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return nil, lastErr
}
