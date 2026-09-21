package orders

import (
	"os"
	"strings"
	"testing"
)

func TestPickupDeliveryMethodMigrationUsesExplicitConditionalConstraints(t *testing.T) {
	source, err := os.ReadFile("../../supabase/migrations/20260920234746_add_pickup_delivery_method.sql")
	if err != nil {
		t.Fatalf("expected pickup migration to be readable, got %v", err)
	}

	migration := strings.ToLower(string(source))
	for _, required := range []string{
		"alter table public.cart_shipping_selections",
		"add column delivery_method text not null default 'shipping'",
		"cart_shipping_selections_delivery_method_allowed",
		"delivery_method in ('shipping', 'pickup')",
		"delivery_method = 'pickup' and shipping_box_id is null",
		"price_cents = 0",
		"delivery_method = 'shipping' and shipping_box_id is not null",
		"alter column shipping_box_id drop not null",
		"alter table public.order_shipping_details",
		"order_shipping_details_delivery_method_allowed",
		"delivery_method = 'pickup' and provider = ''",
		"shipping_box_name = ''",
		"delivery_method = 'shipping' and btrim(provider) <> ''",
	} {
		if !strings.Contains(migration, required) {
			t.Fatalf("expected pickup migration to contain %q", required)
		}
	}

	for _, forbidden := range []string{
		"update public.products",
		"update public.shipping_boxes",
		"insert into public.shipping_boxes",
		"insert into public.products",
		"delete from public.orders",
		"truncate",
	} {
		if strings.Contains(migration, forbidden) {
			t.Fatalf("expected pickup migration not to contain %q", forbidden)
		}
	}
}
