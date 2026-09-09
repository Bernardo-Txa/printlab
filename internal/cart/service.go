package cart

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

type Repository interface {
	FindActiveCart(ctx context.Context, tokenHash []byte, now time.Time) (Cart, error)
	CreateCart(ctx context.Context, tokenHash []byte, expiresAt time.Time) (Cart, error)
	RenewCart(ctx context.Context, cartID string, expiresAt time.Time) (Cart, error)
	ProductForAddBySlug(ctx context.Context, slug string) (ProductForAdd, error)
	ListItems(ctx context.Context, cartID string) ([]StoredItem, error)
	AddItem(ctx context.Context, cartID string, productID string, variantID *string, quantity int) error
	UpdateItemQuantity(ctx context.Context, cartID string, itemID string, quantity int) (bool, error)
	RemoveItem(ctx context.Context, cartID string, itemID string) (bool, error)
}

type Service struct {
	repository  Repository
	now         func() time.Time
	supabaseURL string
}

type ServiceOption func(*Service)

func WithClock(now func() time.Time) ServiceOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func WithSupabaseURL(supabaseURL string) ServiceOption {
	return func(service *Service) {
		service.supabaseURL = strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	}
}

func NewService(repository Repository, options ...ServiceOption) *Service {
	service := &Service{
		repository: repository,
		now:        time.Now,
	}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) View(ctx context.Context, tokenHash []byte) (CartView, error) {
	if s == nil || s.repository == nil {
		return CartView{}, ErrUnavailable
	}

	if !validHash(tokenHash) {
		return EmptyView(), nil
	}

	activeCart, err := s.repository.FindActiveCart(ctx, tokenHash, s.now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return EmptyView(), nil
		}

		return CartView{}, ErrUnavailable
	}

	items, err := s.repository.ListItems(ctx, activeCart.ID)
	if err != nil {
		return CartView{}, ErrUnavailable
	}

	return s.prepareView(items)
}

func (s *Service) Add(ctx context.Context, tokenHash []byte, input AddItemInput) (Cart, error) {
	if s == nil || s.repository == nil {
		return Cart{}, ErrUnavailable
	}

	if !validHash(tokenHash) {
		return Cart{}, ErrInvalidToken
	}

	if !validQuantity(input.Quantity) {
		return Cart{}, ErrInvalidQuantity
	}

	productSlug := strings.TrimSpace(input.ProductSlug)
	variantSlug := strings.TrimSpace(input.VariantSlug)
	if !products.ValidSlug(productSlug) {
		return Cart{}, ErrInvalidProduct
	}
	if variantSlug != "" && !products.ValidSlug(variantSlug) {
		return Cart{}, ErrInvalidVariant
	}

	product, err := s.repository.ProductForAddBySlug(ctx, productSlug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Cart{}, ErrInvalidProduct
		}

		return Cart{}, ErrUnavailable
	}
	if !product.IsActive {
		return Cart{}, ErrProductUnavailable
	}

	variantID, err := resolveVariantID(product, variantSlug)
	if err != nil {
		return Cart{}, err
	}

	now := s.now()
	activeCart, err := s.repository.FindActiveCart(ctx, tokenHash, now)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			activeCart, err = s.repository.CreateCart(ctx, tokenHash, now.Add(TTL))
			if err != nil {
				return Cart{}, ErrUnavailable
			}
		} else {
			return Cart{}, ErrUnavailable
		}
	}

	if err := s.repository.AddItem(ctx, activeCart.ID, product.ID, variantID, input.Quantity); err != nil {
		if errors.Is(err, ErrQuantityLimit) {
			return Cart{}, ErrQuantityLimit
		}

		return Cart{}, ErrUnavailable
	}

	renewed, err := s.repository.RenewCart(ctx, activeCart.ID, now.Add(TTL))
	if err != nil {
		return Cart{}, ErrUnavailable
	}

	return renewed, nil
}

func (s *Service) UpdateQuantity(ctx context.Context, tokenHash []byte, itemID string, quantity int) (*Cart, error) {
	if s == nil || s.repository == nil {
		return nil, ErrUnavailable
	}

	if !validHash(tokenHash) {
		return nil, ErrInvalidToken
	}

	if !validQuantity(quantity) {
		return nil, ErrInvalidQuantity
	}

	activeCart, err := s.repository.FindActiveCart(ctx, tokenHash, s.now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}

		return nil, ErrUnavailable
	}

	updated, err := s.repository.UpdateItemQuantity(ctx, activeCart.ID, itemID, quantity)
	if err != nil {
		return nil, ErrUnavailable
	}
	if !updated {
		return nil, nil
	}

	renewed, err := s.repository.RenewCart(ctx, activeCart.ID, s.now().Add(TTL))
	if err != nil {
		return nil, ErrUnavailable
	}

	return &renewed, nil
}

func (s *Service) RemoveItem(ctx context.Context, tokenHash []byte, itemID string) (*Cart, error) {
	if s == nil || s.repository == nil {
		return nil, ErrUnavailable
	}

	if !validHash(tokenHash) {
		return nil, ErrInvalidToken
	}

	activeCart, err := s.repository.FindActiveCart(ctx, tokenHash, s.now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}

		return nil, ErrUnavailable
	}

	removed, err := s.repository.RemoveItem(ctx, activeCart.ID, itemID)
	if err != nil {
		return nil, ErrUnavailable
	}
	if !removed {
		return nil, nil
	}

	renewed, err := s.repository.RenewCart(ctx, activeCart.ID, s.now().Add(TTL))
	if err != nil {
		return nil, ErrUnavailable
	}

	return &renewed, nil
}

func (s *Service) prepareView(items []StoredItem) (CartView, error) {
	if len(items) == 0 {
		return EmptyView(), nil
	}

	view := CartView{
		Lines:       make([]CartLine, 0, len(items)),
		SubtotalBRL: formatBRL(0),
	}

	for _, item := range items {
		line, err := s.prepareLine(item)
		if err != nil {
			return CartView{}, err
		}

		view.Lines = append(view.Lines, line)
		if !line.Available {
			view.HasUnavailableItems = true
			continue
		}

		view.SubtotalCents, err = addCents(view.SubtotalCents, line.SubtotalCents)
		if err != nil {
			return CartView{}, err
		}
		view.SubtotalBRL = formatBRL(view.SubtotalCents)
	}

	return view, nil
}

func (s *Service) prepareLine(item StoredItem) (CartLine, error) {
	unitPrice := item.Product.PriceCents
	if item.Variant != nil && item.Variant.PriceCents != nil {
		unitPrice = *item.Variant.PriceCents
	}

	line := CartLine{
		ID:             item.ID,
		ProductName:    item.Product.Name,
		ProductSlug:    item.Product.Slug,
		Quantity:       item.Quantity,
		UnitPriceCents: unitPrice,
		UnitPriceBRL:   formatBRL(unitPrice),
		Available:      true,
		Image:          s.prepareImage(selectedImage(item)),
	}

	if item.Variant != nil {
		line.HasVariant = true
		line.VariantName = item.Variant.Name
		line.VariantSlug = item.Variant.Slug
	}

	switch {
	case !item.Product.IsActive:
		line.Available = false
		line.ProductAvailable = false
		line.UnavailableReason = "Produto indisponivel."
	case item.Variant == nil && item.ProductHasActiveVariants:
		line.Available = false
		line.ProductAvailable = true
		line.RequiresVariantPick = true
		line.UnavailableReason = "Escolha uma variante novamente."
	case item.Variant != nil && (!item.Variant.IsActive || item.Variant.ProductID != item.Product.ID):
		line.Available = false
		line.ProductAvailable = true
		line.VariantAvailable = false
		line.UnavailableReason = "Variante indisponivel."
	default:
		line.ProductAvailable = true
		line.VariantAvailable = item.Variant == nil || item.Variant.IsActive
	}

	if !line.Available {
		line.SubtotalBRL = formatBRL(0)
		return line, nil
	}

	subtotal, err := lineSubtotalCents(line.UnitPriceCents, line.Quantity)
	if err != nil {
		return CartLine{}, err
	}

	line.SubtotalCents = subtotal
	line.SubtotalBRL = formatBRL(subtotal)

	return line, nil
}

func selectedImage(item StoredItem) *products.ProductImage {
	if item.VariantImage != nil {
		return item.VariantImage
	}

	return item.ProductImage
}

func (s *Service) prepareImage(image *products.ProductImage) *products.ProductImage {
	if image == nil {
		return nil
	}

	prepared := *image
	prepared.URL = products.PublicProductImageURL(s.supabaseURL, image.StoragePath)
	return &prepared
}

func resolveVariantID(product ProductForAdd, variantSlug string) (*string, error) {
	hasActiveVariants := false
	for _, variant := range product.Variants {
		if variant.IsActive && variant.ProductID == product.ID {
			hasActiveVariants = true
			break
		}
	}

	if variantSlug == "" {
		if hasActiveVariants {
			return nil, ErrVariantRequired
		}

		return nil, nil
	}

	for _, variant := range product.Variants {
		if variant.Slug != variantSlug {
			continue
		}

		if variant.ProductID != product.ID {
			return nil, ErrInvalidVariant
		}

		if !variant.IsActive {
			return nil, ErrVariantUnavailable
		}

		variantID := variant.ID
		return &variantID, nil
	}

	return nil, ErrInvalidVariant
}

func validQuantity(quantity int) bool {
	return quantity >= MinQuantity && quantity <= MaxQuantity
}

func validHash(tokenHash []byte) bool {
	return len(tokenHash) == HashByteLength
}
