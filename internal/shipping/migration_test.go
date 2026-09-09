package shipping

import (
	"os"
	"strings"
	"testing"
)

func TestShippingMigrationContainsProfilesBoxesSelectionsAndRLS(t *testing.T) {
	source, err := os.ReadFile("../../supabase/migrations/20260909220454_add_shipping_profiles_and_selections.sql")
	if err != nil {
		t.Fatalf("expected shipping migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"alter table public.products",
		"shipping_weight_g bigint null",
		"shipping_height_mm integer null",
		"products_shipping_profile_all_or_none",
		"products_shipping_profile_positive",
		"alter table public.product_variants",
		"product_variants_shipping_profile_all_or_none",
		"product_variants_shipping_profile_positive",
		"create table public.shipping_boxes",
		"internal_height_mm integer not null",
		"external_height_mm integer not null",
		"packaging_weight_g integer not null",
		"shipping_boxes_external_dimensions_fit_internal",
		"create table public.cart_shipping_selections",
		"price_cents bigint not null",
		"input_hash bytea not null",
		"octet_length(input_hash) = 32",
		"alter table public.shipping_boxes enable row level security",
		"alter table public.cart_shipping_selections enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected shipping migration to contain %q", expected)
		}
	}

	if strings.Contains(sql, "insert into public.shipping_boxes") {
		t.Fatal("expected shipping migration not to seed physical boxes")
	}
}
