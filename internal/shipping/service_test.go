package shipping

import (
	"context"
	"errors"
	"testing"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customers"
)

func TestServicePageRequiresCartAndCustomerDetails(t *testing.T) {
	tests := []struct {
		name       string
		cartErr    error
		detailsOK  bool
		wantErr    error
		wantCartID string
	}{
		{
			name:    "missing cart",
			cartErr: cartdomain.ErrNotFound,
			wantErr: ErrCartRequired,
		},
		{
			name:      "missing customer details",
			detailsOK: false,
			wantErr:   ErrDetailsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeShippingRepository{
				items: shippingCartItemsFixture(),
				boxes: shippingBoxesFixture(),
			}
			cart := &fakeShippingCartService{
				cart: shippingCartFixture(),
				view: shippingCartViewFixture(),
				err:  tt.cartErr,
			}
			customerRepository := &fakeShippingCustomerRepository{
				details: shippingDetailsFixture(),
				found:   tt.detailsOK,
			}
			service := NewService(repository, cart, customerRepository, shippingCalculatorFixture(), "01153000", []string{"1", "2"})

			_, err := service.Page(context.Background(), []byte("token-hash"), false)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestServicePageUnavailableWhenShippingProfileIsMissing(t *testing.T) {
	calculator := shippingCalculatorFixture()
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: []CartItem{
			{
				ID:          "item-1",
				ProductID:   "product-1",
				ProductName: "Produto sem perfil",
				Quantity:    1,
			},
		},
		boxes: shippingBoxesFixture(),
	}, calculator)

	page, err := service.Page(context.Background(), []byte("token-hash"), false)
	if err != nil {
		t.Fatalf("expected unavailable page without error, got %v", err)
	}
	if !page.Unavailable || page.Message != "Frete temporariamente indisponivel para este carrinho." {
		t.Fatalf("expected missing profile message, got %#v", page)
	}
	if len(calculator.requests) != 0 {
		t.Fatalf("expected calculator not to be called without shipping profile, got %d calls", len(calculator.requests))
	}
}

func TestServicePageUnavailableWhenNoShippingBoxExists(t *testing.T) {
	calculator := shippingCalculatorFixture()
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: shippingCartItemsFixture(),
		boxes: nil,
	}, calculator)

	page, err := service.Page(context.Background(), []byte("token-hash"), false)
	if err != nil {
		t.Fatalf("expected unavailable page without error, got %v", err)
	}
	if !page.Unavailable || page.Message != "Nao conseguimos calcular automaticamente o frete para este carrinho." {
		t.Fatalf("expected no-box message, got %#v", page)
	}
	if len(calculator.requests) != 0 {
		t.Fatalf("expected calculator not to be called without boxes, got %d calls", len(calculator.requests))
	}
}

func TestServicePageRejectsEmptyRepositoryItems(t *testing.T) {
	calculator := shippingCalculatorFixture()
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: nil,
		boxes: shippingBoxesFixture(),
	}, calculator)

	_, err := service.Page(context.Background(), []byte("token-hash"), false)
	if !errors.Is(err, ErrEmptyCart) {
		t.Fatalf("expected ErrEmptyCart, got %v", err)
	}
	if len(calculator.requests) != 0 {
		t.Fatalf("expected calculator not to be called for empty item list, got %d calls", len(calculator.requests))
	}
}

func TestServicePageUnavailableWhenNoBoxFitsPlanningPackage(t *testing.T) {
	calculator := &fakeShippingCalculator{
		responses: [][]SuperFreteQuote{
			{
				{
					ServiceCode: "1",
					ServiceName: "PAC",
					PriceCents:  999,
					Package: &SuperFreteReturnedPackage{
						HeightMM: 200,
						WidthMM:  300,
						LengthMM: 400,
					},
				},
			},
		},
	}
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: shippingCartItemsFixture(),
		boxes: []ShippingBox{{
			ID:               "box-small",
			Name:             "Caixa Pequena",
			Internal:         DimensionsMM{Height: 120, Width: 160, Length: 240},
			External:         DimensionsMM{Height: 130, Width: 170, Length: 250},
			PackagingWeightG: 100,
		}},
	}, calculator)

	page, err := service.Page(context.Background(), []byte("token-hash"), false)
	if err != nil {
		t.Fatalf("expected unavailable page without error, got %v", err)
	}
	if !page.Unavailable || page.Message != "Nao conseguimos calcular automaticamente o frete para este carrinho." {
		t.Fatalf("expected no-fitting-box message, got %#v", page)
	}
	if len(calculator.requests) != 1 {
		t.Fatalf("expected only planning request, got %d calls", len(calculator.requests))
	}
}

func TestServicePageUsesPlanningPackageThenFinalRealBoxQuote(t *testing.T) {
	calculator := shippingCalculatorFixture()
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: shippingCartItemsFixture(),
		boxes: shippingBoxesFixture(),
	}, calculator)

	page, err := service.Page(context.Background(), []byte("token-hash"), false)
	if err != nil {
		t.Fatalf("expected shipping page, got %v", err)
	}
	if page.Unavailable {
		t.Fatalf("expected available quotes, got %#v", page)
	}
	if len(calculator.requests) != 2 {
		t.Fatalf("expected planning and final requests, got %d", len(calculator.requests))
	}

	planning := calculator.requests[0]
	if planning.Package != nil || len(planning.Products) != 1 {
		t.Fatalf("expected planning request with products only, got %#v", planning)
	}
	if planning.Products[0].Quantity != 2 || planning.Products[0].WeightKG != 0.285 || planning.Products[0].HeightCM != 21 {
		t.Fatalf("expected product profile converted to SuperFrete units, got %#v", planning.Products[0])
	}

	final := calculator.requests[1]
	if len(final.Products) != 0 || final.Package == nil {
		t.Fatalf("expected final request with package only, got %#v", final)
	}
	if final.Package.WeightKG != 0.69 || final.Package.HeightCM != 14 || final.Package.WidthCM != 18 || final.Package.LengthCM != 26 {
		t.Fatalf("expected final package to use real external box and packaging weight, got %#v", final.Package)
	}

	if len(page.Quotes) != 2 {
		t.Fatalf("expected two final quotes, got %#v", page.Quotes)
	}
	if page.Quotes[0].ServiceCode != "1" || page.Quotes[0].PriceCents != 1890 || page.Quotes[0].PriceBRL != "R$ 18,90" {
		t.Fatalf("expected final PAC quote, got %#v", page.Quotes[0])
	}
	if page.Quotes[0].PriceCents == 999 {
		t.Fatal("expected planning price not to be presented to customer")
	}
	if page.ProductsSubtotalBRL != "R$ 79,80" || page.ShippingPriceBRL != "" || page.PartialTotalBRL != "" {
		t.Fatalf("expected products subtotal only before selection, got %#v", page)
	}
}

func TestServiceSelectRevalidatesAndPersistsCurrentQuote(t *testing.T) {
	now := time.Date(2026, 9, 9, 18, 0, 0, 0, time.UTC)
	repository := &fakeShippingRepository{
		items: shippingCartItemsFixture(),
		boxes: shippingBoxesFixture(),
	}
	calculator := &fakeShippingCalculator{
		responses: [][]SuperFreteQuote{
			planningSuperFreteQuotesFixture(),
			{
				{
					ServiceCode:      "1",
					ServiceName:      "PAC",
					CarrierName:      "Correios",
					PriceCents:       2090,
					DeliveryTimeDays: intPointer(5),
				},
			},
		},
	}
	service := shippingServiceFixture(t, repository, calculator, WithClock(func() time.Time {
		return now
	}))

	_, err := service.Select(context.Background(), []byte("token-hash"), "1")
	if err != nil {
		t.Fatalf("expected valid selection, got %v", err)
	}
	if len(repository.savedSelections) != 1 {
		t.Fatalf("expected one persisted selection, got %d", len(repository.savedSelections))
	}

	saved := repository.savedSelections[0]
	if saved.ServiceCode != "1" || saved.ServiceName != "PAC" || saved.CarrierName != "Correios" || saved.PriceCents != 2090 {
		t.Fatalf("expected current final quote to be persisted, got %#v", saved)
	}
	if saved.ShippingBoxID != "box-medium" || saved.PackageWeightG != 690 || saved.PackageHeightMM != 140 || saved.PackageWidthMM != 180 || saved.PackageLengthMM != 260 {
		t.Fatalf("expected real package snapshot, got %#v", saved)
	}
	if len(saved.InputHash) != 32 {
		t.Fatalf("expected SHA-256 input hash, got %d bytes", len(saved.InputHash))
	}
	if !saved.QuotedAt.Equal(now) || !saved.ExpiresAt.Equal(now.Add(QuoteTTL)) {
		t.Fatalf("expected 30 minute validity, got quoted_at=%s expires_at=%s", saved.QuotedAt, saved.ExpiresAt)
	}
	if len(calculator.requests) != 2 || calculator.requests[1].Package == nil {
		t.Fatalf("expected selection to re-run final package quote, got %#v", calculator.requests)
	}
}

func TestServiceSelectRejectsUnavailableServiceWithoutPersisting(t *testing.T) {
	repository := &fakeShippingRepository{
		items: shippingCartItemsFixture(),
		boxes: shippingBoxesFixture(),
	}
	service := shippingServiceFixture(t, repository, shippingCalculatorFixture())

	result, err := service.Select(context.Background(), []byte("token-hash"), "33")
	if !errors.Is(err, ErrInvalidService) {
		t.Fatalf("expected invalid service, got %v", err)
	}
	if !result.Page.Unavailable && result.Page.Message == "" {
		t.Fatalf("expected page with selection message, got %#v", result.Page)
	}
	if len(repository.savedSelections) != 0 {
		t.Fatalf("expected no persisted selection, got %#v", repository.savedSelections)
	}
}

func TestServicePageMarksOnlyValidCurrentSelection(t *testing.T) {
	now := time.Date(2026, 9, 9, 18, 30, 0, 0, time.UTC)
	boxes := shippingBoxesFixture()
	currentHash := shippingInputHashFixture(t, boxes[1])

	tests := []struct {
		name      string
		selection ShippingSelection
		want      bool
	}{
		{
			name: "valid selection",
			selection: ShippingSelection{
				Provider:    ProviderSuperFrete,
				ServiceCode: "1",
				InputHash:   currentHash,
				ExpiresAt:   now.Add(time.Minute),
			},
			want: true,
		},
		{
			name: "expired selection",
			selection: ShippingSelection{
				Provider:    ProviderSuperFrete,
				ServiceCode: "1",
				InputHash:   currentHash,
				ExpiresAt:   now.Add(-time.Minute),
			},
			want: false,
		},
		{
			name: "hash changed",
			selection: ShippingSelection{
				Provider:    ProviderSuperFrete,
				ServiceCode: "1",
				InputHash:   bytes32("changed"),
				ExpiresAt:   now.Add(time.Minute),
			},
			want: false,
		},
		{
			name: "service no longer quoted",
			selection: ShippingSelection{
				Provider:    ProviderSuperFrete,
				ServiceCode: "33",
				InputHash:   currentHash,
				ExpiresAt:   now.Add(time.Minute),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeShippingRepository{
				items:          shippingCartItemsFixture(),
				boxes:          boxes,
				selection:      tt.selection,
				selectionFound: true,
			}
			service := shippingServiceFixture(t, repository, shippingCalculatorFixture(), WithClock(func() time.Time {
				return now
			}))

			page, err := service.Page(context.Background(), []byte("token-hash"), true)
			if err != nil {
				t.Fatalf("expected shipping page, got %v", err)
			}
			if page.Selected != tt.want {
				t.Fatalf("expected selected=%t, got page %#v", tt.want, page)
			}
			if tt.want && (page.ShippingPriceBRL != "R$ 18,90" || page.PartialTotalBRL != "R$ 98,70") {
				t.Fatalf("expected selected totals, got %#v", page)
			}
		})
	}
}

func shippingServiceFixture(t *testing.T, repository *fakeShippingRepository, calculator *fakeShippingCalculator, options ...ServiceOption) *Service {
	t.Helper()

	return NewService(
		repository,
		&fakeShippingCartService{cart: shippingCartFixture(), view: shippingCartViewFixture()},
		&fakeShippingCustomerRepository{details: shippingDetailsFixture(), found: true},
		calculator,
		"01153000",
		[]string{"1", "2"},
		options...,
	)
}

func shippingCartFixture() cartdomain.Cart {
	return cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}
}

func shippingCartViewFixture() cartdomain.CartView {
	return cartdomain.CartView{
		Lines: []cartdomain.CartLine{
			{
				ID:            "item-1",
				ProductName:   "Produto Real",
				VariantName:   "Padrao",
				HasVariant:    true,
				Quantity:      2,
				SubtotalCents: 7980,
				SubtotalBRL:   "R$ 79,80",
				Available:     true,
			},
		},
		SubtotalCents: 7980,
		SubtotalBRL:   "R$ 79,80",
	}
}

func shippingDetailsFixture() customers.CheckoutDetails {
	return customers.CheckoutDetails{
		Customer: customers.CustomerDetails{
			FullName: "Joao Silva",
			Email:    "joao@example.com",
			Phone:    "27999999999",
			CPF:      "52998224725",
		},
		Address: customers.ShippingAddress{
			PostalCode:  "20020050",
			Street:      "Rua Um",
			Number:      "12",
			District:    "Centro",
			City:        "Rio de Janeiro",
			State:       "RJ",
			CountryCode: customers.CountryCodeBR,
		},
	}
}

func shippingCartItemsFixture() []CartItem {
	return []CartItem{
		{
			ID:          "item-1",
			ProductID:   "product-1",
			ProductName: "Produto Real",
			VariantID:   "variant-1",
			VariantName: "Padrao",
			Quantity:    2,
			ProductProfile: &ShippingProfile{
				WeightG: 285,
				Dimensions: DimensionsMM{
					Height: 210,
					Width:  105,
					Length: 90,
				},
			},
		},
	}
}

func shippingQuoteProductsFixture() []QuoteProduct {
	products, err := prepareQuoteProducts(shippingCartItemsFixture())
	if err != nil {
		panic(err)
	}

	return products
}

func shippingBoxesFixture() []ShippingBox {
	return []ShippingBox{
		{
			ID:               "box-small",
			Name:             "Caixa Pequena",
			Slug:             "caixa-pequena",
			Internal:         DimensionsMM{Height: 110, Width: 150, Length: 230},
			External:         DimensionsMM{Height: 120, Width: 160, Length: 240},
			PackagingWeightG: 100,
			SortOrder:        1,
		},
		{
			ID:               "box-medium",
			Name:             "Caixa Media",
			Slug:             "caixa-media",
			Internal:         DimensionsMM{Height: 130, Width: 170, Length: 250},
			External:         DimensionsMM{Height: 140, Width: 180, Length: 260},
			PackagingWeightG: 120,
			SortOrder:        2,
		},
		{
			ID:               "box-large",
			Name:             "Caixa Grande",
			Slug:             "caixa-grande",
			Internal:         DimensionsMM{Height: 180, Width: 250, Length: 350},
			External:         DimensionsMM{Height: 190, Width: 260, Length: 360},
			PackagingWeightG: 220,
			SortOrder:        3,
		},
	}
}

func planningSuperFreteQuotesFixture() []SuperFreteQuote {
	return []SuperFreteQuote{
		{
			ServiceCode: "1",
			ServiceName: "PAC",
			PriceCents:  999,
			Package: &SuperFreteReturnedPackage{
				HeightMM: 120,
				WidthMM:  160,
				LengthMM: 240,
			},
		},
	}
}

func finalSuperFreteQuotesFixture() []SuperFreteQuote {
	return []SuperFreteQuote{
		{
			ServiceCode:      "1",
			ServiceName:      "PAC",
			CarrierName:      "Correios",
			PriceCents:       1890,
			DeliveryTimeDays: intPointer(5),
		},
		{
			ServiceCode:      "2",
			ServiceName:      "SEDEX",
			CarrierName:      "Correios",
			PriceCents:       3140,
			DeliveryTimeDays: intPointer(2),
		},
	}
}

func shippingCalculatorFixture() *fakeShippingCalculator {
	return &fakeShippingCalculator{
		responses: [][]SuperFreteQuote{
			planningSuperFreteQuotesFixture(),
			finalSuperFreteQuotesFixture(),
		},
	}
}

func shippingInputHashFixture(t *testing.T, box ShippingBox) []byte {
	t.Helper()

	hash, err := BuildInputHash(quoteFingerprint(
		"01153000",
		"20020050",
		[]string{"1", "2"},
		shippingQuoteProductsFixture(),
		box,
	))
	if err != nil {
		t.Fatalf("expected input hash, got %v", err)
	}

	return hash
}

func intPointer(value int) *int {
	return &value
}

type fakeShippingRepository struct {
	items []CartItem
	boxes []ShippingBox

	selection      ShippingSelection
	selectionFound bool

	listCartItemsErr error
	listBoxesErr     error
	getSelectionErr  error
	saveErr          error

	savedSelections []ShippingSelection
}

func (r *fakeShippingRepository) ListCartItems(_ context.Context, _ string) ([]CartItem, error) {
	if r.listCartItemsErr != nil {
		return nil, r.listCartItemsErr
	}

	return r.items, nil
}

func (r *fakeShippingRepository) ListActiveBoxes(_ context.Context) ([]ShippingBox, error) {
	if r.listBoxesErr != nil {
		return nil, r.listBoxesErr
	}

	return r.boxes, nil
}

func (r *fakeShippingRepository) GetSelection(_ context.Context, _ string) (ShippingSelection, bool, error) {
	if r.getSelectionErr != nil {
		return ShippingSelection{}, false, r.getSelectionErr
	}

	return r.selection, r.selectionFound, nil
}

func (r *fakeShippingRepository) SaveSelection(_ context.Context, _ string, selection ShippingSelection) error {
	if r.saveErr != nil {
		return r.saveErr
	}

	r.savedSelections = append(r.savedSelections, selection)
	return nil
}

type fakeShippingCartService struct {
	cart cartdomain.Cart
	view cartdomain.CartView
	err  error
}

func (s *fakeShippingCartService) CheckoutCart(_ context.Context, _ []byte) (cartdomain.Cart, cartdomain.CartView, error) {
	if s.err != nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, s.err
	}

	return s.cart, s.view, nil
}

type fakeShippingCustomerRepository struct {
	details customers.CheckoutDetails
	found   bool
	err     error
}

func (r *fakeShippingCustomerRepository) Get(_ context.Context, _ string) (customers.CheckoutDetails, bool, error) {
	if r.err != nil {
		return customers.CheckoutDetails{}, false, r.err
	}

	return r.details, r.found, nil
}

type fakeShippingCalculator struct {
	responses [][]SuperFreteQuote
	errs      []error
	requests  []SuperFreteCalculatorRequest
}

func (c *fakeShippingCalculator) Calculate(_ context.Context, request SuperFreteCalculatorRequest) ([]SuperFreteQuote, error) {
	c.requests = append(c.requests, request)
	call := len(c.requests) - 1
	if call < len(c.errs) && c.errs[call] != nil {
		return nil, c.errs[call]
	}
	if call >= len(c.responses) {
		return nil, ErrUnavailable
	}

	return c.responses[call], nil
}
