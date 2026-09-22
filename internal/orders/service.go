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
	ConfirmForCustomer(ctx context.Context, tokenHash []byte, expectedFingerprint string, now time.Time, params ReviewParams, customerAuthUserID string) (ConfirmResult, error)
	ListForCustomer(ctx context.Context, customerAuthUserID string) ([]AccountOrder, error)
	LatestCustomerSnapshot(ctx context.Context, customerAuthUserID string) (CustomerSnapshot, bool, error)
	Get(ctx context.Context, orderID string) (OrderPage, error)
	Track(ctx context.Context, trackingID string) (TrackingPage, error)
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
	return s.confirm(ctx, tokenHash, expectedFingerprint, "")
}
func (s *Service) ConfirmForCustomer(ctx context.Context, tokenHash []byte, expectedFingerprint, id string) (ConfirmResult, error) {
	return s.confirm(ctx, tokenHash, expectedFingerprint, id)
}
func (s *Service) confirm(ctx context.Context, tokenHash []byte, expectedFingerprint, id string) (ConfirmResult, error) {
	if s == nil || s.repository == nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if len(tokenHash) != cartdomain.HashByteLength {
		return ConfirmResult{}, ErrCartRequired
	}

	var result ConfirmResult
	var err error
	if id == "" {
		result, err = s.repository.Confirm(ctx, tokenHash, expectedFingerprint, s.now(), s.params)
	} else {
		result, err = s.repository.ConfirmForCustomer(ctx, tokenHash, expectedFingerprint, s.now(), s.params, id)
	}
	if err != nil {
		return result, normalizeCartError(err)
	}

	return result, nil
}
func (s *Service) ListForCustomer(ctx context.Context, id string) ([]AccountOrder, error) {
	if s == nil || s.repository == nil {
		return nil, ErrUnavailable
	}
	if id == "" {
		return []AccountOrder{}, nil
	}
	return s.repository.ListForCustomer(ctx, id)
}
func (s *Service) LatestCustomerSnapshot(ctx context.Context, id string) (CustomerSnapshot, bool, error) {
	if s == nil || s.repository == nil {
		return CustomerSnapshot{}, false, ErrUnavailable
	}
	if id == "" {
		return CustomerSnapshot{}, false, nil
	}
	return s.repository.LatestCustomerSnapshot(ctx, id)
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

func (s *Service) Track(ctx context.Context, trackingID string) (TrackingPage, error) {
	if s == nil || s.repository == nil {
		return TrackingPage{}, ErrUnavailable
	}
	if !ValidTrackingID(trackingID) {
		return TrackingPage{}, ErrInvalidTrackingID
	}

	return s.repository.Track(ctx, trackingID)
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
