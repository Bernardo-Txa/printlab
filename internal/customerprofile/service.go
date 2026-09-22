package customerprofile

import "context"

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }
func (s *Service) Get(ctx context.Context, authUserID string) (Profile, bool, error) {
	return s.repository.Get(ctx, authUserID)
}
func (s *Service) Save(ctx context.Context, p Profile) error { return s.repository.Upsert(ctx, p) }
