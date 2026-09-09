package customers

import (
	"context"
	"errors"
	"testing"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
)

func TestPageRequiresActiveCart(t *testing.T) {
	service := NewService(newFakeCustomerRepository(), &fakeCustomerCart{err: cartdomain.ErrNotFound})

	_, err := service.Page(context.Background(), testCustomerHash(), false)
	if !errors.Is(err, ErrCartRequired) {
		t.Fatalf("expected ErrCartRequired, got %v", err)
	}
}

func TestSaveRejectsEmptyCart(t *testing.T) {
	repository := newFakeCustomerRepository()
	service := NewService(repository, &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1"},
		view: cartdomain.EmptyView(),
	})

	_, err := service.Save(context.Background(), testCustomerHash(), validCheckoutInput(nil))
	if !errors.Is(err, ErrEmptyCart) {
		t.Fatalf("expected ErrEmptyCart, got %v", err)
	}
	if repository.saveCalls != 0 {
		t.Fatal("expected empty cart not to persist details")
	}
}

func TestSaveRejectsUnavailableItems(t *testing.T) {
	repository := newFakeCustomerRepository()
	service := NewService(repository, &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1"},
		view: cartdomain.CartView{
			Lines:               []cartdomain.CartLine{{ID: "item-1", Available: false}},
			SubtotalBRL:         "R$ 0,00",
			HasUnavailableItems: true,
		},
	})

	_, err := service.Save(context.Background(), testCustomerHash(), validCheckoutInput(nil))
	if !errors.Is(err, ErrUnavailableItems) {
		t.Fatalf("expected ErrUnavailableItems, got %v", err)
	}
	if repository.saveCalls != 0 {
		t.Fatal("expected unavailable cart not to persist details")
	}
}

func TestSaveValidDetailsNormalizesPersistsAndRenewsCart(t *testing.T) {
	repository := newFakeCustomerRepository()
	cartService := &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(time.Hour)},
		view: availableCartView(),
	}
	service := NewService(repository, cartService)

	result, err := service.Save(context.Background(), testCustomerHash(), validCheckoutInput(func(input *CheckoutInput) {
		input.Email = " JOAO@example.COM "
		input.Phone = "+55 27 99999-9999"
		input.CPF = "529.982.247-25"
		input.PostalCode = "29100-000"
		input.State = "es"
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.saveCalls != 1 || repository.lastCartID != "cart-1" {
		t.Fatalf("expected one save scoped by cart, got calls=%d cart=%q", repository.saveCalls, repository.lastCartID)
	}
	if repository.lastDetails.Customer.Email != "joao@example.com" {
		t.Fatalf("expected normalized email, got %q", repository.lastDetails.Customer.Email)
	}
	if repository.lastDetails.Customer.Phone != "+5527999999999" || repository.lastDetails.Customer.CPF != "52998224725" {
		t.Fatalf("expected normalized Brazilian identifiers, got %#v", repository.lastDetails.Customer)
	}
	if repository.lastDetails.Address.PostalCode != "29100000" || repository.lastDetails.Address.State != "ES" {
		t.Fatalf("expected normalized address, got %#v", repository.lastDetails.Address)
	}
	if cartService.renewCalls != 1 || cartService.lastRenewCartID != "cart-1" {
		t.Fatalf("expected cart renewal, got calls=%d cart=%q", cartService.renewCalls, cartService.lastRenewCartID)
	}
	if result.ExpiresAt.IsZero() {
		t.Fatal("expected renewed expiration")
	}
	if result.Page.Form.HasErrors() {
		t.Fatalf("expected returned page without errors, got %#v", result.Page.Form.Errors)
	}
}

func TestSaveInvalidDetailsReturnsFieldErrorsWithoutPersistence(t *testing.T) {
	repository := newFakeCustomerRepository()
	service := NewService(repository, &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1"},
		view: availableCartView(),
	})

	result, err := service.Save(context.Background(), testCustomerHash(), validCheckoutInput(func(input *CheckoutInput) {
		input.CPF = "529.982.247-24"
		input.PostalCode = "2910A000"
	}))
	if !errors.Is(err, ErrInvalidDetails) {
		t.Fatalf("expected ErrInvalidDetails, got %v", err)
	}
	if result.Page.Form.Errors.Message("cpf") == "" || result.Page.Form.Errors.Message("postal_code") == "" {
		t.Fatalf("expected field errors for CPF and postal code, got %#v", result.Page.Form.Errors)
	}
	if repository.saveCalls != 0 {
		t.Fatal("expected invalid details not to persist")
	}
}

func TestPagePrefillsSavedDetails(t *testing.T) {
	repository := newFakeCustomerRepository()
	complement := "Apto 302"
	repository.details = CheckoutDetails{
		Customer: CustomerDetails{
			FullName: "Joao Silva",
			Email:    "joao@example.com",
			Phone:    "+5527999999999",
			CPF:      "52998224725",
		},
		Address: ShippingAddress{
			PostalCode:  "29100000",
			Street:      "Rua Um",
			Number:      "12A",
			Complement:  &complement,
			District:    "Centro",
			City:        "Vila Velha",
			State:       "ES",
			CountryCode: CountryCodeBR,
		},
	}
	repository.found = true
	service := NewService(repository, &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1"},
		view: availableCartView(),
	})

	page, err := service.Page(context.Background(), testCustomerHash(), true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !page.Form.Saved {
		t.Fatal("expected saved flag")
	}
	if page.Form.Values.CPF != "52998224725" || page.Form.Values.Complement != "Apto 302" {
		t.Fatalf("expected prefilled values, got %#v", page.Form.Values)
	}
}

func TestSaveRepositoryErrorIsGeneric(t *testing.T) {
	repository := newFakeCustomerRepository()
	repository.saveErr = errors.New("postgres password=secret")
	service := NewService(repository, &fakeCustomerCart{
		cart: cartdomain.Cart{ID: "cart-1"},
		view: availableCartView(),
	})

	_, err := service.Save(context.Background(), testCustomerHash(), validCheckoutInput(nil))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

type fakeCustomerRepository struct {
	details CheckoutDetails
	found   bool
	getErr  error
	saveErr error

	saveCalls   int
	lastCartID  string
	lastDetails CheckoutDetails
}

func newFakeCustomerRepository() *fakeCustomerRepository {
	return &fakeCustomerRepository{}
}

func (r *fakeCustomerRepository) Get(_ context.Context, _ string) (CheckoutDetails, bool, error) {
	if r.getErr != nil {
		return CheckoutDetails{}, false, r.getErr
	}

	return r.details, r.found, nil
}

func (r *fakeCustomerRepository) Save(_ context.Context, cartID string, details CheckoutDetails) error {
	if r.saveErr != nil {
		return r.saveErr
	}

	r.saveCalls++
	r.lastCartID = cartID
	r.lastDetails = details
	return nil
}

type fakeCustomerCart struct {
	cart cartdomain.Cart
	view cartdomain.CartView
	err  error

	renewCalls      int
	lastRenewCartID string
}

func (c *fakeCustomerCart) CheckoutCart(_ context.Context, _ []byte) (cartdomain.Cart, cartdomain.CartView, error) {
	if c.err != nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, c.err
	}

	return c.cart, c.view, nil
}

func (c *fakeCustomerCart) Renew(_ context.Context, cartID string) (cartdomain.Cart, error) {
	c.renewCalls++
	c.lastRenewCartID = cartID
	if c.cart.ID == "" {
		c.cart.ID = cartID
	}

	return c.cart, nil
}

func availableCartView() cartdomain.CartView {
	return cartdomain.CartView{
		Lines: []cartdomain.CartLine{
			{
				ID:            "item-1",
				ProductName:   "Produto Real",
				Quantity:      2,
				SubtotalBRL:   "R$ 79,80",
				SubtotalCents: 7980,
				Available:     true,
			},
		},
		SubtotalCents: 7980,
		SubtotalBRL:   "R$ 79,80",
	}
}

func testCustomerHash() []byte {
	return make([]byte, cartdomain.HashByteLength)
}
