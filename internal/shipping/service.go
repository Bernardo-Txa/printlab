package shipping

import (
	"context"
	"errors"
	"log"
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
	GetCustomer(ctx context.Context, cartID string) (customers.CustomerDetails, bool, error)
	GetAddress(ctx context.Context, cartID string) (customers.ShippingAddress, bool, error)
	SaveAddress(ctx context.Context, cartID string, address customers.ShippingAddress) error
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
	page.Selected = selected && page.Selected
	if page.Selected {
		page.Message = "Frete selecionado. Revisão do pedido será a próxima etapa."
	}
	return page, err
}

// DeliveryPage renders the method choice. It only quotes shipping when the
// customer explicitly chooses shipping or already has a saved shipping choice.
func (s *Service) DeliveryPage(ctx context.Context, tokenHash []byte, method string) (CheckoutShippingPage, error) {
	if method != "" && method != DeliveryMethodShipping && method != DeliveryMethodPickup {
		return CheckoutShippingPage{}, ErrInvalidDeliveryMethod
	}
	activeCart, view, _, err := s.readyCustomer(ctx, tokenHash)
	page := CheckoutShippingPage{Cart: view, ProductsSubtotalBRL: view.SubtotalBRL, PickupAvailable: true, SelectedMethod: method, AddressForm: customers.CheckoutForm{Values: customers.CheckoutInput{CountryCode: customers.CountryCodeBR}}}
	if err != nil {
		return page, err
	}
	address, addressFound, err := s.customers.GetAddress(ctx, activeCart.ID)
	if err != nil {
		return page, ErrUnavailable
	}
	if addressFound {
		page.AddressForm.Values = customers.InputFromDetails(customers.CheckoutDetails{Address: address})
		page.AddressForm.Found = true
	}
	selection, found, err := s.repository.GetSelection(ctx, activeCart.ID)
	if err != nil {
		return page, ErrUnavailable
	}
	if method == "" && found {
		page.SelectedMethod = selection.DeliveryMethod
	}
	if page.SelectedMethod == DeliveryMethodShipping {
		if !addressFound {
			return page, nil
		}
		return s.Page(ctx, tokenHash, found)
	}
	if page.SelectedMethod == DeliveryMethodPickup {
		page.ShippingPriceBRL = FormatBRL(0)
		page.PartialTotalBRL = view.SubtotalBRL
	}
	return page, nil
}

func (s *Service) SaveAddress(ctx context.Context, tokenHash []byte, input customers.CheckoutInput) (CheckoutShippingPage, error) {
	activeCart, view, _, err := s.readyCustomer(ctx, tokenHash)
	page := CheckoutShippingPage{Cart: view, ProductsSubtotalBRL: view.SubtotalBRL, PickupAvailable: true, SelectedMethod: DeliveryMethodShipping}
	if err != nil {
		return page, err
	}
	address, values, fieldErrors := customers.NormalizeShippingAddressInput(input)
	page.AddressForm = customers.CheckoutForm{Values: values, Errors: fieldErrors}
	if fieldErrors.Any() {
		return page, ErrAddressRequired
	}
	if err := s.customers.SaveAddress(ctx, activeCart.ID, address); err != nil {
		return page, ErrUnavailable
	}
	page.AddressForm.Values = customers.InputFromDetails(customers.CheckoutDetails{Address: address})
	page.AddressForm.Found = true
	page.AddressForm.Saved = true
	return page, nil
}

func (s *Service) SelectDelivery(ctx context.Context, tokenHash []byte, method, serviceCode string) (SelectResult, error) {
	switch method {
	case DeliveryMethodPickup:
		return s.selectPickup(ctx, tokenHash)
	case DeliveryMethodShipping:
		return s.Select(ctx, tokenHash, serviceCode)
	default:
		return SelectResult{}, ErrInvalidDeliveryMethod
	}
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
		page.Message = "A opção selecionada não está mais disponível. Escolha uma cotação atual."
		return SelectResult{Page: page}, ErrInvalidService
	}

	now := s.now()
	selection := ShippingSelection{
		DeliveryMethod:   DeliveryMethodShipping,
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

func (s *Service) selectPickup(ctx context.Context, tokenHash []byte) (SelectResult, error) {
	activeCart, _, _, err := s.readyCustomer(ctx, tokenHash)
	if err != nil {
		return SelectResult{}, err
	}
	now := s.now()
	selection := ShippingSelection{
		CartID:          activeCart.ID,
		DeliveryMethod:  DeliveryMethodPickup,
		Provider:        "",
		ServiceCode:     "",
		ServiceName:     "",
		PriceCents:      0,
		PackageWeightG:  0,
		PackageHeightMM: 0,
		PackageWidthMM:  0,
		PackageLengthMM: 0,
		InputHash:       make([]byte, 32),
		QuotedAt:        now,
		ExpiresAt:       now.Add(s.quoteExpiresIn),
	}
	if err := s.repository.SaveSelection(ctx, activeCart.ID, selection); err != nil {
		return SelectResult{}, ErrUnavailable
	}
	return SelectResult{Page: CheckoutShippingPage{Selected: true, SelectedMethod: DeliveryMethodPickup, PickupAvailable: true, ShippingPriceBRL: FormatBRL(0)}}, nil
}

func (s *Service) prepareQuotes(ctx context.Context, tokenHash []byte) (PreparedQuote, CheckoutShippingPage, error) {
	activeCart, cartView, details, items, err := s.readyCheckout(ctx, tokenHash)
	page := CheckoutShippingPage{
		Cart: cartView, ProductsSubtotalBRL: cartView.SubtotalBRL, PickupAvailable: true, SelectedMethod: DeliveryMethodShipping,
		AddressForm: customers.CheckoutForm{
			Found:  true,
			Values: customers.InputFromDetails(customers.CheckoutDetails{Address: details.Address}),
		},
	}
	if err != nil {
		return PreparedQuote{}, page, err
	}

	preparedItems, err := prepareQuoteProducts(items)
	if err != nil {
		page.Unavailable = true
		page.Message = "Frete temporariamente indisponível para este carrinho."
		return PreparedQuote{}, page, nil
	}

	boxes, err := s.repository.ListActiveBoxes(ctx)
	if err != nil {
		return PreparedQuote{}, page, ErrUnavailable
	}
	if s.calculator == nil || s.originCEP == "" || s.serviceList == "" {
		logShippingQuoteUnavailable("config", "shipping_not_configured", nil)
		page.Unavailable = true
		page.Message = "Cotação de frete temporariamente indisponível."
		return PreparedQuote{}, page, nil
	}

	logPackagingRequest(preparedItems, boxes)
	logPackagingCandidates(preparedItems, boxes)
	packageSnapshot, err := s.packageForQuote(preparedItems, boxes)
	if err != nil {
		logShippingQuoteUnavailable("packaging", "no_fitting_box", nil)
		logNoFittingBoxDiagnostics(preparedItems, boxes)
		page.Unavailable = true
		page.Message = "Não conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	inputHash, err := BuildInputHash(quoteFingerprint(s.originCEP, details.Address.PostalCode, s.serviceCodes, preparedItems, packageSnapshot))
	if err != nil {
		return PreparedQuote{}, page, ErrUnavailable
	}
	packageSnapshot.InputHash = inputHash
	logShippingQuoteRequest("final", nil, packageSnapshot.WeightG, packageSnapshot.Dimensions, len(s.serviceCodes))
	finalQuotes, err := s.calculator.Calculate(ctx, SuperFreteCalculatorRequest{
		FromPostalCode: s.originCEP,
		ToPostalCode:   details.Address.PostalCode,
		Services:       s.serviceList,
		Package: &SuperFretePackage{
			WeightKG: GramsToKilograms(packageSnapshot.WeightG),
			HeightCM: MillimetersToCentimeters(packageSnapshot.Dimensions.Height),
			WidthCM:  MillimetersToCentimeters(packageSnapshot.Dimensions.Width),
			LengthCM: MillimetersToCentimeters(packageSnapshot.Dimensions.Length),
		},
	})
	if err != nil {
		logShippingQuoteUnavailable("final", "final_request_failed", err)
		page.Unavailable = true
		page.Message = "Não conseguimos calcular automaticamente o frete para este carrinho."
		return PreparedQuote{}, page, nil
	}

	quotes := shippingQuotes(finalQuotes)
	log.Printf("shipping quote response stage=final final_valid_quotes=%d", len(quotes))
	if len(quotes) == 0 {
		logShippingQuoteUnavailable("final", "final_no_valid_quotes", nil)
		page.Unavailable = true
		page.Message = "Não conseguimos calcular automaticamente o frete para este carrinho."
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
		page.SelectedMethod = selection.DeliveryMethod
		if page.SelectedMethod == "" {
			page.SelectedMethod = DeliveryMethodShipping
		}
	}

	page.Quotes = prepared.Quotes
	page.Selected = prepared.Selection != nil
	page.ShippingPriceBRL, page.PartialTotalBRL = selectionSummary(cartView.SubtotalCents, page.Quotes)

	return prepared, page, nil
}

func (s *Service) packageForQuote(items []QuoteProduct, boxes []ShippingBox) (ShippingPackage, error) {
	box, err := SelectShippingBoxForProducts(items, boxes)
	if err == nil {
		logPackagingSelected(PackagingSourceRealBox, ShippingPackage{Box: box, PackagingSource: PackagingSourceRealBox, PackagingWeightG: box.PackagingWeightG, Dimensions: box.External}, boxes, 0)
		totalWeightG, err := TotalPackageWeightG(items, box.PackagingWeightG)
		if err != nil {
			return ShippingPackage{}, err
		}
		return ShippingPackage{Box: box, PackagingSource: PackagingSourceRealBox, PackagingWeightG: box.PackagingWeightG, WeightG: totalWeightG, Dimensions: box.External}, nil
	}
	if !errors.Is(err, ErrNoFittingBox) {
		return ShippingPackage{}, err
	}
	fallbackReason := "no_fitting_box"
	if len(boxes) == 0 {
		fallbackReason = "no_active_boxes"
	}
	log.Printf("shipping packaging fallback reason=%s", fallbackReason)
	fallback, err := FallbackPackageForProducts(items)
	if err != nil {
		return ShippingPackage{}, err
	}
	units := 0
	for _, item := range items {
		units += item.Quantity
	}
	logPackagingSelected(PackagingSourceFallback, fallback, boxes, units)
	return fallback, nil
}

func (s *Service) readyCustomer(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, customers.CheckoutDetails, error) {
	if s == nil || s.repository == nil || s.cart == nil || s.customers == nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, ErrUnavailable
	}

	activeCart, cartView, err := s.cart.CheckoutCart(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, cartdomain.ErrInvalidToken) || errors.Is(err, cartdomain.ErrNotFound) {
			return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, ErrCartRequired
		}

		return cartdomain.Cart{}, cartdomain.CartView{}, customers.CheckoutDetails{}, ErrUnavailable
	}
	if cartView.IsEmpty || len(cartView.Lines) == 0 {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, ErrEmptyCart
	}
	if cartView.HasUnavailableItems {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, ErrUnavailableItems
	}

	customer, found, err := s.customers.GetCustomer(ctx, activeCart.ID)
	if err != nil {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, ErrUnavailable
	}
	if !found {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, ErrDetailsRequired
	}

	return activeCart, cartView, customers.CheckoutDetails{Customer: customer}, nil
}

func (s *Service) readyCheckout(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, customers.CheckoutDetails, []CartItem, error) {
	activeCart, cartView, customer, err := s.readyCustomer(ctx, tokenHash)
	if err != nil {
		return activeCart, cartView, customer, nil, err
	}
	address, found, err := s.customers.GetAddress(ctx, activeCart.ID)
	if err != nil {
		return cartdomain.Cart{}, cartView, customers.CheckoutDetails{}, nil, ErrUnavailable
	}
	if !found {
		return activeCart, cartView, customer, nil, ErrAddressRequired
	}
	details := customers.CheckoutDetails{Customer: customer.Customer, Address: address}

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

func logShippingQuoteUnavailable(stage string, reason string, err error) {
	if err == nil {
		log.Printf("shipping quote unavailable stage=%s reason=%s", stage, reason)
		return
	}

	var clientErr *SuperFreteClientError
	if errors.As(err, &clientErr) {
		if clientErr.StatusCode > 0 {
			log.Printf("shipping quote unavailable stage=%s reason=%s category=%s status=%d", stage, reason, clientErr.Category, clientErr.StatusCode)
			return
		}

		log.Printf("shipping quote unavailable stage=%s reason=%s category=%s", stage, reason, clientErr.Category)
		return
	}

	log.Printf("shipping quote unavailable stage=%s reason=%s", stage, reason)
}

func logShippingQuoteRequest(stage string, _ []QuoteProduct, packageWeightG int64, packageDimensions DimensionsMM, services int) {
	log.Printf("shipping quote request stage=%s package_weight_g=%d package_h_mm=%d package_w_mm=%d package_l_mm=%d services=%d", stage, packageWeightG, packageDimensions.Height, packageDimensions.Width, packageDimensions.Length, services)
}

func logPackagingRequest(items []QuoteProduct, boxes []ShippingBox) {
	units := 0
	for _, item := range items {
		units += item.Quantity
	}
	log.Printf("shipping packaging request product_lines=%d units=%d candidate_boxes=%d", len(items), units, len(boxes))
	for i, item := range items {
		log.Printf("shipping packaging line=%d quantity=%d weight_g=%d height_mm=%d width_mm=%d length_mm=%d", i, item.Quantity, item.Profile.WeightG, item.Profile.Dimensions.Height, item.Profile.Dimensions.Width, item.Profile.Dimensions.Length)
	}
}

func logPackagingCandidates(items []QuoteProduct, boxes []ShippingBox) {
	candidates := append([]ShippingBox(nil), boxes...)
	sortShippingBoxes(candidates)
	for i, box := range candidates {
		log.Printf("shipping packaging candidate index=%d fits=%t internal_h_mm=%d internal_w_mm=%d internal_l_mm=%d", i, FitsProductsInBox(items, box.Internal), box.Internal.Height, box.Internal.Width, box.Internal.Length)
	}
}

func logPackagingSelected(source string, packageSnapshot ShippingPackage, boxes []ShippingBox, units int) {
	if source == PackagingSourceFallback {
		log.Printf("shipping packaging selected source=fallback package_weight_g=%d package_h_mm=%d package_w_mm=%d package_l_mm=%d units=%d", packageSnapshot.WeightG, packageSnapshot.Dimensions.Height, packageSnapshot.Dimensions.Width, packageSnapshot.Dimensions.Length, units)
		return
	}
	box := packageSnapshot.Box
	index := 0
	candidates := append([]ShippingBox(nil), boxes...)
	sortShippingBoxes(candidates)
	for i, candidate := range candidates {
		if candidate.ID == box.ID {
			index = i
			break
		}
	}
	log.Printf("shipping packaging selected source=real_box box_index=%d internal_h_mm=%d internal_w_mm=%d internal_l_mm=%d external_h_mm=%d external_w_mm=%d external_l_mm=%d packaging_weight_g=%d", index, box.Internal.Height, box.Internal.Width, box.Internal.Length, box.External.Height, box.External.Width, box.External.Length, box.PackagingWeightG)
}

func shippingQuotes(superFreteQuotes []SuperFreteQuote) []ShippingQuote {
	quotes := make([]ShippingQuote, 0, len(superFreteQuotes))
	for _, externalQuote := range superFreteQuotes {
		if externalQuote.PriceCents < 0 || externalQuote.ServiceCode == "" || externalQuote.ServiceName == "" {
			log.Printf("shipping quote discarded service_code=%q service_name=%q reason=invalid_final_quote", safeSuperFreteLogValue(externalQuote.ServiceCode), safeSuperFreteLogValue(externalQuote.ServiceName))
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

func quoteFingerprint(originCEP string, destinationCEP string, services []string, items []QuoteProduct, packageSnapshot ShippingPackage) QuoteFingerprint {
	fingerprint := QuoteFingerprint{
		OriginPostalCode:      originCEP,
		DestinationPostalCode: destinationCEP,
		Services:              append([]string(nil), services...),
		Options: QuoteOptions{
			OwnHand:           false,
			Receipt:           false,
			UseInsuranceValue: false,
		},
		Packaging: QuotePackageFingerprint{
			Source:           packageSnapshot.PackagingSource,
			ID:               packageSnapshot.Box.ID,
			ExternalHeightMM: packageSnapshot.Dimensions.Height,
			ExternalWidthMM:  packageSnapshot.Dimensions.Width,
			ExternalLengthMM: packageSnapshot.Dimensions.Length,
			PackagingWeightG: packageSnapshot.PackagingWeightG,
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
