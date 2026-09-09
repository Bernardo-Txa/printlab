package shipping

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestShippingRepositoryQueriesAreScopedAndExplicit(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}
	sql := strings.ToLower(string(source))

	for _, expected := range []string{
		"from public.cart_items ci",
		"where ci.cart_id = $1::uuid",
		"from public.shipping_boxes",
		"where is_active = true",
		"from public.cart_shipping_selections selection",
		"join public.shipping_boxes box",
		"box.is_active = true",
		"insert into public.cart_shipping_selections",
		"on conflict (cart_id) do update set",
		"package_weight_g",
		"package_height_mm",
		"input_hash",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected repository source to contain %q", expected)
		}
	}

	if strings.Contains(sql, "select *") {
		t.Fatal("expected repository not to use SELECT *")
	}
}

func TestPostgresRepositoryListsProfilesBoxesAndSelections(t *testing.T) {
	ctx := context.Background()
	pool := testShippingPool(t, ctx)
	repository := NewPostgresRepository(pool)

	productID := createShippingProduct(t, ctx, pool, "product-profile", profileFixture(280, 90, 105, 210))
	variantID := createShippingVariant(t, ctx, pool, productID, "variant-profile", profileFixture(300, 100, 120, 220))
	cartID := createShippingCart(t, ctx, pool, "repository")
	createShippingCartItem(t, ctx, pool, cartID, productID, &variantID, 2)

	boxID := createShippingBox(t, ctx, pool, "repository-box", true)
	inactiveBoxID := createShippingBox(t, ctx, pool, "inactive-box", false)

	items, err := repository.ListCartItems(ctx, cartID)
	if err != nil {
		t.Fatalf("expected cart items, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one cart item, got %d", len(items))
	}
	if items[0].VariantProfile == nil || items[0].VariantProfile.WeightG != 300 {
		t.Fatalf("expected variant shipping profile, got %#v", items[0])
	}

	boxes, err := repository.ListActiveBoxes(ctx)
	if err != nil {
		t.Fatalf("expected active boxes, got %v", err)
	}
	if !containsBox(boxes, boxID) || containsBox(boxes, inactiveBoxID) {
		t.Fatalf("expected only active shipping boxes, got %#v", boxes)
	}

	delivery := 3
	quotedAt := time.Now().UTC().Truncate(time.Microsecond)
	selection := ShippingSelection{
		ShippingBoxID:    boxID,
		Provider:         ProviderSuperFrete,
		ServiceCode:      "1",
		ServiceName:      "PAC",
		CarrierName:      "Correios",
		PriceCents:       1890,
		DeliveryTimeDays: &delivery,
		PackageWeightG:   720,
		PackageHeightMM:  120,
		PackageWidthMM:   160,
		PackageLengthMM:  240,
		InputHash:        bytes32(t.Name()),
		QuotedAt:         quotedAt,
		ExpiresAt:        quotedAt.Add(QuoteTTL),
	}
	if err := repository.SaveSelection(ctx, cartID, selection); err != nil {
		t.Fatalf("expected selection to be saved, got %v", err)
	}

	got, found, err := repository.GetSelection(ctx, cartID)
	if err != nil {
		t.Fatalf("expected selection to be read, got %v", err)
	}
	if !found {
		t.Fatal("expected selection to be found")
	}
	if got.PriceCents != 1890 || got.PackageWeightG != 720 || got.ShippingBoxID != boxID || string(got.InputHash) != string(selection.InputHash) {
		t.Fatalf("expected persisted selection snapshot, got %#v", got)
	}

	if _, err := pool.Exec(ctx, `update public.shipping_boxes set is_active = false where id = $1::uuid`, boxID); err != nil {
		t.Fatalf("expected box to be inactivated, got %v", err)
	}
	_, found, err = repository.GetSelection(ctx, cartID)
	if err != nil {
		t.Fatalf("expected inactive box selection read to be safe, got %v", err)
	}
	if found {
		t.Fatal("expected selection for inactive box to be ignored")
	}
}

func testShippingPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected test database pool, got %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

type testProfile struct {
	weightG int64
	height  int
	width   int
	length  int
}

func profileFixture(weightG int64, height int, width int, length int) testProfile {
	return testProfile{weightG: weightG, height: height, width: width, length: length}
}

func createShippingProduct(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug string, profile testProfile) string {
	t.Helper()

	var productID string
	if err := pool.QueryRow(ctx, `
		insert into public.products (
			name,
			slug,
			price_cents,
			is_active,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm
		) values (
			$1,
			$2,
			$3,
			true,
			$4,
			$5,
			$6,
			$7
		)
		returning id::text
	`, "Produto "+slug, slug+"-"+shortHash(t.Name()), 3990, profile.weightG, profile.height, profile.width, profile.length).Scan(&productID); err != nil {
		t.Fatalf("expected product fixture, got %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from public.products where id = $1::uuid`, productID)
	})

	return productID
}

func createShippingVariant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, productID string, slug string, profile testProfile) string {
	t.Helper()

	var variantID string
	if err := pool.QueryRow(ctx, `
		insert into public.product_variants (
			product_id,
			name,
			slug,
			is_active,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm
		) values (
			$1::uuid,
			$2,
			$3,
			true,
			$4,
			$5,
			$6,
			$7
		)
		returning id::text
	`, productID, "Variante "+slug, slug+"-"+shortHash(t.Name()), profile.weightG, profile.height, profile.width, profile.length).Scan(&variantID); err != nil {
		t.Fatalf("expected variant fixture, got %v", err)
	}

	return variantID
}

func createShippingCart(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix string) string {
	t.Helper()

	tokenHash := bytes32(t.Name() + suffix + time.Now().String())
	var cartID string
	if err := pool.QueryRow(ctx, `
		insert into public.carts (
			token_hash,
			expires_at
		) values (
			$1,
			$2
		)
		returning id::text
	`, tokenHash, time.Now().Add(time.Hour)).Scan(&cartID); err != nil {
		t.Fatalf("expected cart fixture, got %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from public.carts where id = $1::uuid`, cartID)
	})

	return cartID
}

func createShippingCartItem(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cartID string, productID string, variantID *string, quantity int) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		insert into public.cart_items (
			cart_id,
			product_id,
			variant_id,
			quantity
		) values (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4
		)
	`, cartID, productID, variantID, quantity)
	if err != nil {
		t.Fatalf("expected cart item fixture, got %v", err)
	}
}

func createShippingBox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug string, active bool) string {
	t.Helper()

	var boxID string
	if err := pool.QueryRow(ctx, `
		insert into public.shipping_boxes (
			name,
			slug,
			internal_height_mm,
			internal_width_mm,
			internal_length_mm,
			external_height_mm,
			external_width_mm,
			external_length_mm,
			packaging_weight_g,
			is_active
		) values (
			$1,
			$2,
			120,
			160,
			240,
			130,
			170,
			250,
			120,
			$3
		)
		returning id::text
	`, "Caixa "+slug, slug+"-"+shortHash(t.Name()), active).Scan(&boxID); err != nil {
		t.Fatalf("expected shipping box fixture, got %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from public.shipping_boxes where id = $1::uuid`, boxID)
	})

	return boxID
}

func containsBox(boxes []ShippingBox, id string) bool {
	for _, box := range boxes {
		if box.ID == id {
			return true
		}
	}

	return false
}

func bytes32(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}

func shortHash(value string) string {
	return InputHashHex(bytes32(value))[:8]
}
