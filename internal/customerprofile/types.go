package customerprofile

import (
	"context"
	"time"
)

type Profile struct {
	AuthUserID                                                                 string
	FullName, Phone, CPF                                                       string
	PostalCode, Street, Number, Complement, District, City, State, CountryCode string
	CreatedAt, UpdatedAt                                                       time.Time
}

type Repository interface {
	Get(ctx context.Context, authUserID string) (Profile, bool, error)
	Upsert(ctx context.Context, profile Profile) error
}
