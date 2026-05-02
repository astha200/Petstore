package graph

// Conversion helpers between the internal db package types and the gqlgen-
// generated GraphQL model types. Keeping this in one place avoids leaking
// db-specific concepts (UUIDs, store_id, deleted_at) into the API surface.

import (
	"github.com/petstore/backend/graph/model"
	"github.com/petstore/backend/internal/db"
)

func petToModel(p db.Pet) *model.Pet {
	return &model.Pet{
		ID:          p.ID.String(),
		Name:        p.Name,
		Species:     model.Species(p.Species),
		Age:         p.Age,
		PictureURL:  p.PictureURL,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
	}
}

func petsToModel(in []db.Pet) []*model.Pet {
	out := make([]*model.Pet, len(in))
	for i, p := range in {
		out[i] = petToModel(p)
	}
	return out
}

func soldPetToModel(s db.SoldPet) *model.SoldPet {
	return &model.SoldPet{
		ID:          s.ID.String(),
		Name:        s.Name,
		Species:     model.Species(s.Species),
		Age:         s.Age,
		PictureURL:  s.PictureURL,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		SoldAt:      s.SoldAt,
	}
}

func soldPetsToModel(in []db.SoldPet) []*model.SoldPet {
	out := make([]*model.SoldPet, len(in))
	for i, s := range in {
		out[i] = soldPetToModel(s)
	}
	return out
}
