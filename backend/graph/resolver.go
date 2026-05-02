package graph

// Resolver is the dependency container shared by all resolvers. The generated
// schema.resolvers.go references it via embedded receiver types.

import "github.com/petstore/backend/internal/db"

type Resolver struct {
	Repo *db.Repo
}
