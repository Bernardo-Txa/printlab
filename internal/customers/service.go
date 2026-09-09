package customers

import (
	"context"
	"errors"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
)

type Repository interface {
	Get(ctx context.Context, cartID string) (CheckoutDetails, bool, error)
	Save(ctx context.Context, cartID string, details CheckoutDetails) error
}

type CartService interface {
	CheckoutCart(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, error)
	Renew(ctx context.Context, cartID string) (cartdomain.Cart, error)
}

type Service struct {
	repository Repository
	cart       CartService
}

func NewService(repository Repository, cart CartService) *Service {
	return &Service{
		repository: repository,
		cart:       cart,
	}
}

type CheckoutPage struct {
	Cart cartdomain.CartView
	Form CheckoutForm
}

type SaveResult struct {
	Page      CheckoutPage
	ExpiresAt time.Time
}

func (s *Service) Page(ctx context.Context, tokenHash []byte, saved bool) (CheckoutPage, error) {
	activeCart, cartView, err := s.readyCart(ctx, tokenHash)
	if err != nil {
		return CheckoutPage{}, err
	}

	details, found, err := s.repository.Get(ctx, activeCart.ID)
	if err != nil {
		return CheckoutPage{}, ErrUnavailable
	}

	form := CheckoutForm{
		Values: CheckoutInput{CountryCode: CountryCodeBR},
		Saved:  saved,
	}
	if found {
		form.Values = InputFromDetails(details)
		form.Saved = saved
	}

	return CheckoutPage{Cart: cartView, Form: form}, nil
}

func (s *Service) Save(ctx context.Context, tokenHash []byte, input CheckoutInput) (SaveResult, error) {
	activeCart, cartView, err := s.readyCart(ctx, tokenHash)
	if err != nil {
		return SaveResult{}, err
	}

	details, values, fieldErrors := NormalizeCheckoutInput(input)
	page := CheckoutPage{
		Cart: cartView,
		Form: CheckoutForm{
			Values: values,
			Errors: fieldErrors,
		},
	}
	if fieldErrors.Any() {
		return SaveResult{Page: page}, ErrInvalidDetails
	}

	if err := s.repository.Save(ctx, activeCart.ID, details); err != nil {
		return SaveResult{}, ErrUnavailable
	}

	renewedCart, err := s.cart.Renew(ctx, activeCart.ID)
	if err != nil {
		return SaveResult{}, ErrUnavailable
	}

	page.Form.Values = InputFromDetails(details)
	return SaveResult{Page: page, ExpiresAt: renewedCart.ExpiresAt}, nil
}

func (s *Service) readyCart(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, error) {
	if s == nil || s.repository == nil || s.cart == nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, ErrUnavailable
	}

	activeCart, cartView, err := s.cart.CheckoutCart(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, cartdomain.ErrInvalidToken) || errors.Is(err, cartdomain.ErrNotFound) {
			return cartdomain.Cart{}, cartdomain.CartView{}, ErrCartRequired
		}

		return cartdomain.Cart{}, cartdomain.CartView{}, ErrUnavailable
	}

	if cartView.IsEmpty || len(cartView.Lines) == 0 {
		return cartdomain.Cart{}, cartdomain.CartView{}, ErrEmptyCart
	}
	if cartView.HasUnavailableItems {
		return cartdomain.Cart{}, cartdomain.CartView{}, ErrUnavailableItems
	}

	return activeCart, cartView, nil
}
