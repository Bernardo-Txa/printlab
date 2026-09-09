package shipping

import (
	"context"
	"errors"
	"slices"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customers"
)

type Repository interface {
	ListCartItems(ctx context.Context, cartID string) ([]CartItem, error)
	ListActiveBoxes(ctx context.Context) ([]ShippingBox, error)
	GetSelection(ctx context.Context, cartID string) (ShippingSelection, bool, error)
	SaveSelection(ctx context.Context, cartID string, selection ShippingSelection) error
}

type CartService interface {
	CheckoutCart(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, error)
}

type CustomerRepository interface {
	Get(ctx context.Context, cartID string) (customers.CheckoutDetails, bool, error)
}

type Calculator interface {
	Calculate(ctx context.Context, request SuperFreteCalculatorRequest) ([]SuperFreteQuote, error)
}

type Service struct {
	repository     Repository
	cart           CartService
	customers      CustomerRepository
	calculator     Calculator
	originCEP      string
	serviceCodes   []string
	serviceList    string
	now            func() time.Time
	quoteExpiresIn time.Duration
}

type ServiceOption func(*Service)

func WithClock(now func() time.Time) ServiceOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func WithQuoteTTL(ttl time.Duration) ServiceOption {
	return func(service *Service) {
		if ttl > 0 {
			service.quoteExpiresIn = ttl
		}
	}
}

func NewService(repository Repository, cart CartService, customers CustomerRepository, calculator Calculator, originCEP string, serviceCodes []string, options ...ServiceOption) *Service {
	service := &Service{
		repository:     repository,
		cart:           cart,
		customers:      customers,
		calculator:     calculator,
		originCEP:      originCEP,
		serviceCodes:   append([]string(nil), serviceCodes...),
		serviceList:    joinServiceCodes(serviceCodes),
		now:            time.Now,
		quoteExpiresIn: QuoteTTL,
	}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Page(ctx context.Context, tokenHash []byte, selected bool) (CheckoutShippingPage, error) {
	_, page, err := s.prepareQuotes(ctx, tokenHash)
	if err != nil {
		return page, err
	}

	page.Selected = selected && page.Selected
	if page.Selected {
		page.Message = "Frete selecionado. Revisao do pedido sera a proxima etapa."
	}

	return page, nil
}

func (s *Service) Select(ctx context.Context, tokenHash []byte, serviceCode string) (SelectResult, error) {
	prepared, page, err := s.prepareQuotes(ctx, tokenHash)
	if err != nil {
		return SelectResult{Page: page}, err
	}
	if page.Unavailable || len(prepared.Quotes) == 0 {
		return SelectResult{Page: page}, ErrNoQuotes
	}

	var chosen ShippingQuote
	found := false
	for _, quote := range prepared.Quotes {
		if quote.ServiceCode == serviceCode {
			chosen = quote
			found = true
			break
		}
	}
	if !found {
		page.Message = "A opcao selecionada nao esta mais disponivel. Escolha uma cotacao atual."
		return SelectResult{Page: page}, ErrInvalidService
	}

	now := s.now()
	selection := ShippingSelection{
		CartID:           prepared.Cart.ID,
		ShippingBoxID:    prepared.Package.Box.ID,
		Provider:         chosen.Provider,
		ServiceCode:      chosen.ServiceCode,
		ServiceName:      chosen.ServiceName,
		CarrierName:      chosen.CarrierName,
		PriceCents:       chosen.PriceCents,
		DeliveryTimeDays: chosen.DeliveryTimeDays,
		PackageWeightG:   prepared.Package.WeightG,
		PackageHeightMM:  prepared.Package.Dimensions.Height,
		PackageWidthMM:   prepared.Package.Dimensions.Width,
		PackageLengthMM:  prepared.Package.Dimensions.Length,
		InputHash:        prepared.Package.InputHash,
		QuotedAt:         now,
		ExpiresAt:        now.Add(s.quoteExpiresIn),
	}

	if err := s.repository.SaveSelection(ctx, prepared.Cart.ID, selection); err != nil {
		return SelectResult{}, ErrUnavailable
	}

	return SelectResult{Page: page}, nil
}

func (s *Service) prepareQuotes(ctx context.Context, tokenHash []byte) (PreparedQuote, CheckoutShippingPage, error) {
	activeCart, cartView, details, items, err := s.readyCheckout(ctx, tokenHash)
	page := CheckoutShippingPage{
		Cart:                cartView,
		ProductsSubtotalBRL: cartView.SubtotalBRL,
	}
	if err != nil {
		return PreparedQuote{}, page, err
	}

	preparedItems, err := prepareQuoteProducts(items)
	if err != nil {
		page.Unavailable = true
		page.Message = "Frete temporariamente indisponivel para este carrinho."
		return PreparedQuote{}, page, nil
	}

	boxes, err := s.repository.ListActiveBoxes(ctx)
	if err != nil {
		return PreparedQuote{}, page, ErrUnavailable
	}
	if len(boxes) == 0 {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	if s.calculator == nil || s.originCEP == "" || s.serviceList == "" {
		page.Unavailable = true
		page.Message = "Cotacao de frete temporariamente indisponivel."
		return PreparedQuote{}, page, nil
	}

	planningQuotes, err := s.calculator.Calculate(ctx, SuperFreteCalculatorRequest{
		FromPostalCode: s.originCEP,
		ToPostalCode:   details.Address.PostalCode,
		Services:       s.serviceList,
		Products:       superFreteProducts(preparedItems),
	})
	if err != nil {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	idealPackage, ok := firstReturnedPackage(planningQuotes)
	if !ok {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	box, err := SelectSmallestBox(DimensionsMM{
		Height: idealPackage.HeightMM,
		Width:  idealPackage.WidthMM,
		Length: idealPackage.LengthMM,
	}, boxes)
	if err != nil {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	totalWeightG, err := TotalPackageWeightG(preparedItems, box.PackagingWeightG)
	if err != nil {
		return PreparedQuote{}, page, err
	}

	inputHash, err := BuildInputHash(quoteFingerprint(s.originCEP, details.Address.PostalCode, s.serviceCodes, preparedItems, box))
	if err != nil {
		return PreparedQuote{}, page, ErrUnavailable
	}

	packageSnapshot := ShippingPackage{
		Box:        box,
		WeightG:    totalWeightG,
		Dimensions: box.External,
		InputHash:  inputHash,
	}
	finalQuotes, err := s.calculator.Calculate(ctx, SuperFreteCalculatorRequest{
		FromPostalCode: s.originCEP,
		ToPostalCode:   details.Address.PostalCode,
		Services:       s.serviceList,
		Package: &SuperFretePackage{
			WeightKG: GramsToKilograms(totalWeightG),
			HeightCM: MillimetersToCentimeters(box.External.Height),
			WidthCM:  MillimetersToCentimeters(box.External.Width),
			LengthCM: MillimetersToCentimeters(box.External.Length),
		},
	})
	if err != nil {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	quotes := shippingQuotes(finalQuotes)
	if len(quotes) == 0 {
		page.Unavailable = true
		page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	prepared := PreparedQuote{
		Cart:     activeCart,
		CartView: cartView,
		Package:  packageSnapshot,
		Quotes:   quotes,
	}
	selection, found, err := s.repository.GetSelection(ctx, activeCart.ID)
	if err != nil {
		return PreparedQuote{}, page, ErrUnavailable
	}
	if found && validSelection(selection, packageSnapshot.InputHash, s.now()) && markSelectedQuote(prepared.Quotes, selection.ServiceCode) {
		prepared.Selection = &selection
	}

	page.Quotes = prepared.Quotes
	page.Selected = prepared.Selection != nil
	page.ShippingPriceBRL, page.PartialTotalBRL = selectionSummary(cartView.SubtotalCents, page.Quotes)

	return prepared, page, nil
}

func (s *Service) readyCheckout(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, customers.CheckoutDetails, []CartItem, error) {
	if s == nil || s.repository == nil || s.cart == nil || s.customers == nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, nil, ErrUnavailable
	}

	activeCart, cartView, err := s.cart.CheckoutCart(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, cartdomain.ErrInvalidToken) || errors.Is(err, cartdomain.ErrNotFound) {
			return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, nil, ErrCartRequired
		}

		return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, nil, ErrUnavailable
	}
	if cartView.IsEmpty || len(cartView.Lines) == 0 {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrEmptyCart
	}
	if cartView.HasUnavailableItems {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrUnavailableItems
	}

	details, found, err := s.customers.Get(ctx, activeCart.ID)
	if err != nil {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrUnavailable
	}
	if !found {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrDetailsRequired
	}

	items, err := s.repository.ListCartItems(ctx, activeCart.ID)
	if err != nil {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrUnavailable
	}
	if len(items) == 0 {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrEmptyCart
	}

	return activeCart, cartView, details, items, nil
}

func prepareQuoteProducts(items []CartItem) ([]QuoteProduct, error) {
	products := make([]QuoteProduct, 0, len(items))
	for _, item := range items {
		profile, ok := EffectiveShippingProfile(item)
		if !ok {
			return nil, ErrMissingShippingProfile
		}
		products = append(products, QuoteProduct{
			ID:        item.ID,
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Profile:   profile,
		})
	}

	return products, nil
}

func superFreteProducts(items []QuoteProduct) []SuperFreteProduct {
	products := make([]SuperFreteProduct, 0, len(items))
	for _, item := range items {
		products = append(products, SuperFreteProduct{
			Quantity: item.Quantity,
			WeightKG: GramsToKilograms(item.Profile.WeightG),
			HeightCM: MillimetersToCentimeters(item.Profile.Dimensions.Height),
			WidthCM:  MillimetersToCentimeters(item.Profile.Dimensions.Width),
			LengthCM: MillimetersToCentimeters(item.Profile.Dimensions.Length),
		})
	}

	return products
}

func firstReturnedPackage(quotes []SuperFreteQuote) (SuperFreteReturnedPackage, bool) {
	for _, quote := range quotes {
		if quote.Package != nil {
			return *quote.Package, true
		}
	}

	return SuperFreteReturnedPackage{}, false
}

func shippingQuotes(superFreteQuotes []SuperFreteQuote) []ShippingQuote {
	quotes := make([]ShippingQuote, 0, len(superFreteQuotes))
	for _, externalQuote := range superFreteQuotes {
		if externalQuote.PriceCents < 0 || externalQuote.ServiceCode == "" || externalQuote.ServiceName == "" {
			continue
		}

		quote := ShippingQuote{
			Provider:         ProviderSuperFrete,
			ServiceCode:      externalQuote.ServiceCode,
			ServiceName:      externalQuote.ServiceName,
			CarrierName:      externalQuote.CarrierName,
			PriceCents:       externalQuote.PriceCents,
			PriceBRL:         FormatBRL(externalQuote.PriceCents),
			DeliveryTimeDays: externalQuote.DeliveryTimeDays,
			DeliveryTime:     DeliveryTimeLabel(externalQuote.DeliveryTimeDays),
		}
		quotes = append(quotes, quote)
	}

	return quotes
}

func quoteFingerprint(originCEP string, destinationCEP string, services []string, items []QuoteProduct, box ShippingBox) QuoteFingerprint {
	fingerprint := QuoteFingerprint{
		OriginPostalCode:      originCEP,
		DestinationPostalCode: destinationCEP,
		Services:              append([]string(nil), services...),
		Options: QuoteOptions{
			OwnHand:           false,
			Receipt:           false,
			UseInsuranceValue: false,
		},
		Box: QuoteBoxFingerprint{
			ID:               box.ID,
			ExternalHeightMM: box.External.Height,
			ExternalWidthMM:  box.External.Width,
			ExternalLengthMM: box.External.Length,
			PackagingWeightG: box.PackagingWeightG,
		},
	}
	for _, item := range items {
		fingerprint.Products = append(fingerprint.Products, QuoteProductFingerprint{
			ID:        item.ID,
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			WeightG:   item.Profile.WeightG,
			HeightMM:  item.Profile.Dimensions.Height,
			WidthMM:   item.Profile.Dimensions.Width,
			LengthMM:  item.Profile.Dimensions.Length,
		})
	}

	return fingerprint
}

func validSelection(selection ShippingSelection, currentHash []byte, now time.Time) bool {
	return selection.Provider == ProviderSuperFrete &&
		selection.ExpiresAt.After(now) &&
		inputHashEqual(selection.InputHash, currentHash)
}

func markSelectedQuote(quotes []ShippingQuote, serviceCode string) bool {
	for i := range quotes {
		if quotes[i].ServiceCode == serviceCode {
			quotes[i].Selected = true
			return true
		}
	}

	return false
}

func selectionSummary(productsSubtotalCents int64, quotes []ShippingQuote) (string, string) {
	for _, quote := range quotes {
		if !quote.Selected {
			continue
		}

		total, err := PartialTotalBRL(productsSubtotalCents, quote.PriceCents)
		if err != nil {
			return "", ""
		}

		return quote.PriceBRL, total
	}

	return "", ""
}

func joinServiceCodes(services []string) string {
	values := append([]string(nil), services...)
	slices.SortFunc(values, func(left string, right string) int {
		return serviceSortValue(left) - serviceSortValue(right)
	})

	return stringsJoin(values, ",")
}

func stringsJoin(values []string, separator string) string {
	if len(values) == 0 {
		return ""
	}

	result := values[0]
	for _, value := range values[1:] {
		result += separator + value
	}

	return result
}
