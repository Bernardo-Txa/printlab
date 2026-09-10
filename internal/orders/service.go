package orders

import (
	"context"
	"errors"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
)

type Repository interface {
	Review(ctx context.Context, tokenHash []byte, now time.Time, params ReviewParams) (ReviewPage, error)
	Confirm(ctx context.Context, tokenHash []byte, expectedFingerprint string, now time.Time, params ReviewParams) (ConfirmResult, error)
	Get(ctx context.Context, orderID string) (OrderPage, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
	params     ReviewParams
}

type ServiceOption func(*Service)

func WithClock(now func() time.Time) ServiceOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func NewService(repository Repository, originPostalCode string, serviceCodes []string, options ...ServiceOption) *Service {
	service := &Service{
		repository: repository,
		now:        time.Now,
		params: ReviewParams{
			OriginPostalCode: originPostalCode,
			ServiceCodes:     append([]string(nil), serviceCodes...),
		},
	}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Review(ctx context.Context, tokenHash []byte, stale bool) (ReviewPage, error) {
	if s == nil || s.repository == nil {
		return ReviewPage{}, ErrUnavailable
	}
	if len(tokenHash) != cartdomain.HashByteLength {
		return ReviewPage{}, ErrCartRequired
	}

	page, err := s.repository.Review(ctx, tokenHash, s.now(), s.params)
	if err != nil {
		return ReviewPage{}, normalizeCartError(err)
	}
	if stale {
		page.Message = StaleReviewMessage
	}

	return page, nil
}

func (s *Service) Confirm(ctx context.Context, tokenHash []byte, expectedFingerprint string) (ConfirmResult, error) {
	if s == nil || s.repository == nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if len(tokenHash) != cartdomain.HashByteLength {
		return ConfirmResult{}, ErrCartRequired
	}

	result, err := s.repository.Confirm(ctx, tokenHash, expectedFingerprint, s.now(), s.params)
	if err != nil {
		return result, normalizeCartError(err)
	}

	return result, nil
}

func (s *Service) Get(ctx context.Context, orderID string) (OrderPage, error) {
	if s == nil || s.repository == nil {
		return OrderPage{}, ErrUnavailable
	}
	if !ValidOrderID(orderID) {
		return OrderPage{}, ErrInvalidOrderID
	}

	return s.repository.Get(ctx, orderID)
}

func normalizeCartError(err error) error {
	switch {
	case errors.Is(err, cartdomain.ErrInvalidToken),
		errors.Is(err, cartdomain.ErrNotFound):
		return ErrCartRequired
	default:
		return err
	}
}
