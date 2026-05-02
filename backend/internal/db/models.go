package db

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleMerchant Role = "merchant"
	RoleCustomer Role = "customer"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Role         Role
	StoreID      *uuid.UUID
}

type Store struct {
	ID   uuid.UUID
	Slug string
	Name string
}

type Species string

const (
	SpeciesCat  Species = "CAT"
	SpeciesDog  Species = "DOG"
	SpeciesFrog Species = "FROG"
)

type Pet struct {
	ID          uuid.UUID
	StoreID     uuid.UUID
	Name        string
	Species     Species
	Age         int
	PictureURL  string
	Description string
	CreatedAt   time.Time
}

type SoldPet struct {
	Pet
	SoldAt time.Time
}
