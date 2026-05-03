// Package auth provides HTTP Basic authentication and a request-scoped context for the authenticated user.
package auth

import (
	"context"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/petstore/backend/internal/db"
	"github.com/petstore/backend/internal/errs"
)

type ctxKey struct{}

// dummyHash is used to make the unknown-username code path pay the same bcrypt cost as a real comparison. 
// Without this, an attacker can enumerate valid usernames purely by request-timing.
var dummyHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	if err != nil {
		// DefaultCost is valid, so this should never fail.
		panic(err)
	}
	return h
}()

// Middleware authenticates Basic Auth credentials and stores the user in context.
// Requests without credentials continue so resolvers can return field-level auth errors.
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
				// Burn an equivalent bcrypt cycle so an attacker cannot distinguish "user does not exist" from "wrong password" by timing.
				_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
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
