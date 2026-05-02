// Package errs defines domain-level errors that resolvers translate into
// human-readable GraphQL error messages.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrForbidden    = errors.New("not allowed")
	ErrPetNotFound  = errors.New("pet not found")
	ErrStoreUnknown = errors.New("store not found")
	ErrInvalidInput = errors.New("invalid input")
)

// PetUnavailableError signals that one or more pets could not be purchased.
// The Names slice is included verbatim in the user-facing message so the UI can show exactly which pets are no longer on the market.
type PetUnavailableError struct {
	Names []string
}

func (e *PetUnavailableError) Error() string {
	switch len(e.Names) {
	case 0:
		return "one or more pets are no longer available"
	case 1:
		return fmt.Sprintf("%q is no longer available for purchase", e.Names[0])
	default:
		quoted := make([]string, len(e.Names))
		for i, n := range e.Names {
			quoted[i] = fmt.Sprintf("%q", n)
		}
		return fmt.Sprintf("the following pets are no longer available for purchase: %s", strings.Join(quoted, ", "))
	}
}
