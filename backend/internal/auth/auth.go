// Package auth provides HTTP Basic authentication and a request-scoped context
// for the authenticated user.
package auth

import (
	"context"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/petstore/backend/internal/db"
	"github.com/petstore/backend/internal/errs"
)

type ctxKey struct{}

// Middleware authenticates the request via HTTP Basic and stores the resolved user on the request context. 
// Anonymous requests are allowed through so that the GraphQL resolver layer can produce per-field auth errors.
func Middleware(repo *db.Repo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			user, err := repo.UserByUsername(r.Context(), username)
			if err != nil {
				challenge(w)
				return
			}
			if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
				challenge(w)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func challenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="petstore"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

// FromContext returns the authenticated user, or nil for anonymous requests.
func FromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(ctxKey{}).(*db.User)
	return u
}

// RequireMerchant returns the user iff they are an authenticated merchant.
func RequireMerchant(ctx context.Context) (*db.User, error) {
	u := FromContext(ctx)
	if u == nil {
		return nil, errs.ErrUnauthorized
	}
	if u.Role != db.RoleMerchant || u.StoreID == nil {
		return nil, errs.ErrForbidden
	}
	return u, nil
}

// RequireCustomer returns the user iff they are an authenticated customer.
func RequireCustomer(ctx context.Context) (*db.User, error) {
	u := FromContext(ctx)
	if u == nil {
		return nil, errs.ErrUnauthorized
	}
	if u.Role != db.RoleCustomer {
		return nil, errs.ErrForbidden
	}
	return u, nil
}
