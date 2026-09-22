package customerprofile

import "context"
import "github.com/Bernardo-Txa/printlab/internal/customers"

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }
func (s *Service) Get(ctx context.Context, authUserID string) (Profile, bool, error) {
	return s.repository.Get(ctx, authUserID)
}
func (s *Service) Save(ctx context.Context, p Profile) error { return s.repository.Upsert(ctx, p) }

func FromCheckout(id string, input customers.CheckoutInput) Profile {
	return Profile{AuthUserID: id, FullName: input.FullName, Phone: input.Phone, CPF: input.CPF, PostalCode: input.PostalCode, Street: input.Street, Number: input.Number, Complement: input.Complement, District: input.District, City: input.City, State: input.State, CountryCode: input.CountryCode}
}
func (p Profile) CheckoutInput(email string) customers.CheckoutInput {
	return customers.CheckoutInput{FullName: p.FullName, Email: email, Phone: p.Phone, CPF: p.CPF, PostalCode: p.PostalCode, Street: p.Street, Number: p.Number, Complement: p.Complement, District: p.District, City: p.City, State: p.State, CountryCode: p.CountryCode}
}
