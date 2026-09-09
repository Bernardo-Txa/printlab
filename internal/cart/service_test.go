package cart

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

func TestViewReturnsEmptyForMissingCart(t *testing.T) {
	repository := newFakeRepository()
	service := NewService(repository, WithClock(fixedClock()))

	view, err := service.View(context.Background(), testHash(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !view.IsEmpty {
		t.Fatal("expected empty cart")
	}

	if repository.listItemsCalls != 0 {
		t.Fatal("expected missing cart not to list items")
	}
}

func TestViewReturnsEmptyForExpiredCart(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-expired", ExpiresAt: fixedNow().Add(-time.Minute)}
	service := NewService(repository, WithClock(fixedClock()))

	view, err := service.View(context.Background(), testHash(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !view.IsEmpty {
		t.Fatal("expected expired cart to be treated as empty")
	}
}

func TestAddCreatesCartOnFirstValidAdd(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:         "prod-1",
		Name:       "Produto Real",
		Slug:       "produto-real",
		PriceCents: 3990,
		IsActive:   true,
	}
	service := NewService(repository, WithClock(fixedClock()))

	activeCart, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		Quantity:    1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if activeCart.ID == "" {
		t.Fatal("expected cart to be created")
	}
	if repository.createCalls != 1 {
		t.Fatalf("expected one cart creation, got %d", repository.createCalls)
	}
	if repository.addCalls != 1 {
		t.Fatalf("expected one add call, got %d", repository.addCalls)
	}
	if repository.lastAddedVariantID != nil {
		t.Fatal("expected product without active variants to be added without variant")
	}
	if !activeCart.ExpiresAt.Equal(fixedNow().Add(TTL)) {
		t.Fatalf("expected cart expiration to be renewed to TTL, got %s", activeCart.ExpiresAt)
	}
}

func TestAddProductWithActiveVariantsRequiresVariant(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: true,
		Variants: []VariantForAdd{
			{ID: "variant-1", ProductID: "prod-1", Slug: "padrao", IsActive: true},
		},
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		Quantity:    1,
	})
	if !errors.Is(err, ErrVariantRequired) {
		t.Fatalf("expected ErrVariantRequired, got %v", err)
	}
	if repository.createCalls != 0 || repository.addCalls != 0 {
		t.Fatal("expected invalid add not to create cart or item")
	}
}

func TestAddValidActiveVariant(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: true,
		Variants: []VariantForAdd{
			{ID: "variant-1", ProductID: "prod-1", Slug: "padrao", IsActive: true},
		},
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		VariantSlug: "padrao",
		Quantity:    2,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.lastAddedVariantID == nil || *repository.lastAddedVariantID != "variant-1" {
		t.Fatalf("expected selected variant to be added, got %#v", repository.lastAddedVariantID)
	}
}

func TestAddRejectsInactiveVariant(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: true,
		Variants: []VariantForAdd{
			{ID: "variant-1", ProductID: "prod-1", Slug: "padrao", IsActive: false},
		},
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		VariantSlug: "padrao",
		Quantity:    1,
	})
	if !errors.Is(err, ErrVariantUnavailable) {
		t.Fatalf("expected ErrVariantUnavailable, got %v", err)
	}
}

func TestAddRejectsVariantFromAnotherProduct(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: true,
		Variants: []VariantForAdd{
			{ID: "variant-1", ProductID: "prod-2", Slug: "padrao", IsActive: true},
		},
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		VariantSlug: "padrao",
		Quantity:    1,
	})
	if !errors.Is(err, ErrInvalidVariant) {
		t.Fatalf("expected ErrInvalidVariant, got %v", err)
	}
}

func TestAddRejectsInactiveProduct(t *testing.T) {
	repository := newFakeRepository()
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: false,
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "produto-real",
		Quantity:    1,
	})
	if !errors.Is(err, ErrProductUnavailable) {
		t.Fatalf("expected ErrProductUnavailable, got %v", err)
	}
}

func TestAddRejectsMissingProduct(t *testing.T) {
	service := NewService(newFakeRepository(), WithClock(fixedClock()))

	_, err := service.Add(context.Background(), testHash(1), AddItemInput{
		ProductSlug: "nao-existe",
		Quantity:    1,
	})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestAddIncrementsExistingItemAndEnforcesLimit(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-1", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.products["produto-real"] = ProductForAdd{
		ID:       "prod-1",
		Slug:     "produto-real",
		IsActive: true,
	}
	service := NewService(repository, WithClock(fixedClock()))

	for _, quantity := range []int{2, 3} {
		if _, err := service.Add(context.Background(), testHash(1), AddItemInput{ProductSlug: "produto-real", Quantity: quantity}); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	key := itemKey("cart-1", "prod-1", nil)
	if got := repository.quantitiesByKey[key]; got != 5 {
		t.Fatalf("expected quantity 5, got %d", got)
	}

	repository.quantitiesByKey[key] = 98
	_, err := service.Add(context.Background(), testHash(1), AddItemInput{ProductSlug: "produto-real", Quantity: 2})
	if !errors.Is(err, ErrQuantityLimit) {
		t.Fatalf("expected ErrQuantityLimit, got %v", err)
	}
	if got := repository.quantitiesByKey[key]; got != 98 {
		t.Fatalf("expected quantity to stay 98, got %d", got)
	}
}

func TestAddRejectsInvalidQuantity(t *testing.T) {
	service := NewService(newFakeRepository(), WithClock(fixedClock()))

	for _, quantity := range []int{0, 100} {
		_, err := service.Add(context.Background(), testHash(1), AddItemInput{
			ProductSlug: "produto-real",
			Quantity:    quantity,
		})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Fatalf("expected ErrInvalidQuantity for %d, got %v", quantity, err)
		}
	}
}

func TestViewCalculatesCurrentPricesAndKeepsUnavailableItems(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-1", ExpiresAt: fixedNow().Add(time.Hour)}
	override := int64(2990)
	repository.itemsByCart["cart-1"] = []StoredItem{
		{
			ID:       "item-available",
			CartID:   "cart-1",
			Quantity: 2,
			Product:  ProductSnapshot{ID: "prod-1", Name: "Produto Ativo", Slug: "produto-ativo", PriceCents: 3990, IsActive: true},
			Variant:  &VariantSnapshot{ID: "variant-1", ProductID: "prod-1", Name: "Padrao", Slug: "padrao", PriceCents: &override, IsActive: true},
			ProductImage: &products.ProductImage{
				ID:          "product-image",
				ProductID:   "prod-1",
				StoragePath: "produtos/produto.webp",
			},
			VariantImage: &products.ProductImage{
				ID:          "variant-image",
				ProductID:   "prod-1",
				VariantID:   "variant-1",
				StoragePath: "produtos/variante.webp",
			},
		},
		{
			ID:       "item-product-inactive",
			CartID:   "cart-1",
			Quantity: 1,
			Product:  ProductSnapshot{ID: "prod-2", Name: "Produto Inativo", Slug: "produto-inativo", PriceCents: 4990, IsActive: false},
		},
		{
			ID:       "item-variant-inactive",
			CartID:   "cart-1",
			Quantity: 1,
			Product:  ProductSnapshot{ID: "prod-3", Name: "Produto com Variante", Slug: "produto-variante", PriceCents: 5990, IsActive: true},
			Variant:  &VariantSnapshot{ID: "variant-3", ProductID: "prod-3", Name: "Antiga", Slug: "antiga", IsActive: false},
		},
		{
			ID:                       "item-needs-variant",
			CartID:                   "cart-1",
			Quantity:                 1,
			Product:                  ProductSnapshot{ID: "prod-4", Name: "Produto Evoluido", Slug: "produto-evoluido", PriceCents: 6990, IsActive: true},
			ProductHasActiveVariants: true,
		},
	}
	service := NewService(repository, WithClock(fixedClock()), WithSupabaseURL("https://example.supabase.co"))

	view, err := service.View(context.Background(), testHash(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if view.IsEmpty {
		t.Fatal("expected cart with lines")
	}
	if len(view.Lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(view.Lines))
	}
	if view.SubtotalCents != 5980 || view.SubtotalBRL != "R$ 59,80" {
		t.Fatalf("expected only available item in subtotal, got %d %q", view.SubtotalCents, view.SubtotalBRL)
	}
	if !view.HasUnavailableItems {
		t.Fatal("expected unavailable items flag")
	}
	if !view.Lines[0].Available || view.Lines[0].UnitPriceCents != 2990 || view.Lines[0].SubtotalCents != 5980 {
		t.Fatalf("expected first line to use current variant price, got %#v", view.Lines[0])
	}
	if view.Lines[0].Image == nil || view.Lines[0].Image.ID != "variant-image" || view.Lines[0].Image.URL == "" {
		t.Fatalf("expected variant image to be preferred and prepared, got %#v", view.Lines[0].Image)
	}
	if view.Lines[1].Available || view.Lines[1].UnavailableReason != "Produto indisponivel." {
		t.Fatalf("expected inactive product line, got %#v", view.Lines[1])
	}
	if view.Lines[2].Available || view.Lines[2].UnavailableReason != "Variante indisponivel." {
		t.Fatalf("expected inactive variant line, got %#v", view.Lines[2])
	}
	if view.Lines[3].Available || !view.Lines[3].RequiresVariantPick {
		t.Fatalf("expected line without variant to be unavailable after product gains variants, got %#v", view.Lines[3])
	}
}

func TestViewReturnsOverflowErrorInsteadOfWrongTotal(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-1", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.itemsByCart["cart-1"] = []StoredItem{
		{
			ID:       "item-overflow",
			CartID:   "cart-1",
			Quantity: 2,
			Product:  ProductSnapshot{ID: "prod-1", Name: "Produto", Slug: "produto", PriceCents: math.MaxInt64, IsActive: true},
		},
	}
	service := NewService(repository, WithClock(fixedClock()))

	_, err := service.View(context.Background(), testHash(1))
	if !errors.Is(err, ErrAmountOverflow) {
		t.Fatalf("expected ErrAmountOverflow, got %v", err)
	}
}

func TestUpdateQuantityIsScopedToCurrentCart(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-a", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.cartsByHash[string(testHash(2))] = Cart{ID: "cart-b", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.itemCartIDs["item-a"] = "cart-a"
	repository.itemQuantities["item-a"] = 1
	service := NewService(repository, WithClock(fixedClock()))

	updatedCart, err := service.UpdateQuantity(context.Background(), testHash(2), "item-a", 9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedCart != nil {
		t.Fatal("expected item from another cart not to renew cart")
	}
	if got := repository.itemQuantities["item-a"]; got != 1 {
		t.Fatalf("expected item quantity to stay 1, got %d", got)
	}

	updatedCart, err = service.UpdateQuantity(context.Background(), testHash(1), "item-a", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedCart == nil || updatedCart.ID != "cart-a" {
		t.Fatalf("expected cart-a to be renewed, got %#v", updatedCart)
	}
	if got := repository.itemQuantities["item-a"]; got != 5 {
		t.Fatalf("expected item quantity 5, got %d", got)
	}
}

func TestRemoveItemIsScopedToCurrentCart(t *testing.T) {
	repository := newFakeRepository()
	repository.cartsByHash[string(testHash(1))] = Cart{ID: "cart-a", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.cartsByHash[string(testHash(2))] = Cart{ID: "cart-b", ExpiresAt: fixedNow().Add(time.Hour)}
	repository.itemCartIDs["item-a"] = "cart-a"
	repository.itemQuantities["item-a"] = 1
	service := NewService(repository, WithClock(fixedClock()))

	removedCart, err := service.RemoveItem(context.Background(), testHash(2), "item-a")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if removedCart != nil {
		t.Fatal("expected item from another cart not to renew cart")
	}
	if _, exists := repository.itemQuantities["item-a"]; !exists {
		t.Fatal("expected item to stay in original cart")
	}

	removedCart, err = service.RemoveItem(context.Background(), testHash(1), "item-a")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if removedCart == nil || removedCart.ID != "cart-a" {
		t.Fatalf("expected cart-a to be renewed, got %#v", removedCart)
	}
	if _, exists := repository.itemQuantities["item-a"]; exists {
		t.Fatal("expected item to be removed by owning cart")
	}
}

type fakeRepository struct {
	cartsByHash     map[string]Cart
	products        map[string]ProductForAdd
	itemsByCart     map[string][]StoredItem
	quantitiesByKey map[string]int
	itemCartIDs     map[string]string
	itemQuantities  map[string]int

	createCalls        int
	addCalls           int
	listItemsCalls     int
	lastAddedVariantID *string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		cartsByHash:     map[string]Cart{},
		products:        map[string]ProductForAdd{},
		itemsByCart:     map[string][]StoredItem{},
		quantitiesByKey: map[string]int{},
		itemCartIDs:     map[string]string{},
		itemQuantities:  map[string]int{},
	}
}

func (r *fakeRepository) FindActiveCart(_ context.Context, tokenHash []byte, now time.Time) (Cart, error) {
	activeCart, ok := r.cartsByHash[string(tokenHash)]
	if !ok || !activeCart.ExpiresAt.After(now) {
		return Cart{}, ErrNotFound
	}

	return activeCart, nil
}

func (r *fakeRepository) CreateCart(_ context.Context, tokenHash []byte, expiresAt time.Time) (Cart, error) {
	r.createCalls++
	activeCart := Cart{ID: "cart-created", ExpiresAt: expiresAt}
	r.cartsByHash[string(tokenHash)] = activeCart
	r.itemsByCart[activeCart.ID] = nil

	return activeCart, nil
}

func (r *fakeRepository) RenewCart(_ context.Context, cartID string, expiresAt time.Time) (Cart, error) {
	for hash, activeCart := range r.cartsByHash {
		if activeCart.ID == cartID {
			activeCart.ExpiresAt = expiresAt
			r.cartsByHash[hash] = activeCart
			return activeCart, nil
		}
	}

	return Cart{}, ErrNotFound
}

func (r *fakeRepository) ProductForAddBySlug(_ context.Context, slug string) (ProductForAdd, error) {
	product, ok := r.products[slug]
	if !ok {
		return ProductForAdd{}, ErrNotFound
	}

	return product, nil
}

func (r *fakeRepository) ListItems(_ context.Context, cartID string) ([]StoredItem, error) {
	r.listItemsCalls++
	return r.itemsByCart[cartID], nil
}

func (r *fakeRepository) AddItem(_ context.Context, cartID string, productID string, variantID *string, quantity int) error {
	r.addCalls++
	r.lastAddedVariantID = variantID
	key := itemKey(cartID, productID, variantID)
	current := r.quantitiesByKey[key]
	if current+quantity > MaxQuantity {
		return ErrQuantityLimit
	}

	r.quantitiesByKey[key] = current + quantity
	return nil
}

func (r *fakeRepository) UpdateItemQuantity(_ context.Context, cartID string, itemID string, quantity int) (bool, error) {
	if r.itemCartIDs[itemID] != cartID {
		return false, nil
	}

	r.itemQuantities[itemID] = quantity
	return true, nil
}

func (r *fakeRepository) RemoveItem(_ context.Context, cartID string, itemID string) (bool, error) {
	if r.itemCartIDs[itemID] != cartID {
		return false, nil
	}

	delete(r.itemQuantities, itemID)
	delete(r.itemCartIDs, itemID)
	return true, nil
}

func itemKey(cartID string, productID string, variantID *string) string {
	if variantID == nil {
		return cartID + ":" + productID + ":"
	}

	return cartID + ":" + productID + ":" + *variantID
}

func testHash(value byte) []byte {
	hash := make([]byte, HashByteLength)
	for i := range hash {
		hash[i] = value
	}

	return hash
}

func fixedClock() func() time.Time {
	return fixedNow
}

func fixedNow() time.Time {
	return time.Date(2026, 9, 9, 19, 0, 0, 0, time.UTC)
}
