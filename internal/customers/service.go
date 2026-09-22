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
	GetCustomer(ctx context.Context, cartID string) (CustomerDetails, bool, error)
	SaveCustomer(ctx context.Context, cartID string, customer CustomerDetails) error
	GetAddress(ctx context.Context, cartID string) (ShippingAddress, bool, error)
	SaveAddress(ctx context.Context, cartID string, address ShippingAddress) error
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

	customer, customerFound, err := s.repository.GetCustomer(ctx, activeCart.ID)
	if err != nil {
		return CheckoutPage{}, ErrUnavailable
	}
	address, addressFound, err := s.repository.GetAddress(ctx, activeCart.ID)
	if err != nil {
		return CheckoutPage{}, ErrUnavailable
	}

	form := CheckoutForm{
		Values: CheckoutInput{CountryCode: CountryCodeBR},
		Saved:  saved,
	}
	if customerFound {
		form.Values = InputFromDetails(CheckoutDetails{Customer: customer, Address: address})
		form.Saved = saved
		form.Found = true
		if !addressFound {
			form.Values.CountryCode = CountryCodeBR
		}
	}

	return CheckoutPage{Cart: cartView, Form: form}, nil
}

func (s *Service) Save(ctx context.Context, tokenHash []byte, input CheckoutInput) (SaveResult, error) {
	activeCart, cartView, err := s.readyCart(ctx, tokenHash)
	if err != nil {
		return SaveResult{}, err
	}

	customer, values, fieldErrors := NormalizeCustomerInput(input)
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

	if err := s.repository.SaveCustomer(ctx, activeCart.ID, customer); err != nil {
		return SaveResult{}, ErrUnavailable
	}

	renewedCart, err := s.cart.Renew(ctx, activeCart.ID)
	if err != nil {
		return SaveResult{}, ErrUnavailable
	}

	page.Form.Values = InputFromDetails(CheckoutDetails{Customer: customer})
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
