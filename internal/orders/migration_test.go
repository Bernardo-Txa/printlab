package orders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateOrdersMigrationDocumentsSnapshotSchema(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_create_orders.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one create_orders migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"alter table public.carts",
		"add column converted_at timestamptz null",
		"carts_converted_at_idx",
		"create table public.orders",
		"order_number bigint generated always as identity",
		"source_cart_id uuid null",
		"references public.carts (id)",
		"on delete set null",
		"orders_source_cart_id_unique_idx",
		"status text not null default 'pending_payment'",
		"currency text not null default 'brl'",
		"orders_total_matches_components",
		"create table public.order_customer_details",
		"create table public.order_shipping_addresses",
		"create table public.order_shipping_details",
		"shipping_box_name text not null",
		"create table public.order_items",
		"product_id uuid null",
		"variant_id uuid null",
		"unit_print_time_minutes integer null",
		"unit_estimated_filament_weight_mg bigint null",
		"create table public.order_item_filaments",
		"estimated_weight_mg_per_unit bigint not null",
		"alter table public.orders enable row level security",
		"alter table public.order_customer_details enable row level security",
		"alter table public.order_shipping_addresses enable row level security",
		"alter table public.order_shipping_details enable row level security",
		"alter table public.order_items enable row level security",
		"alter table public.order_item_filaments enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected orders migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"references public.materials",
		"references public.colors",
		"references public.variant_filaments",
		"create policy",
		"insert into public.orders",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected orders migration not to contain %q", forbidden)
		}
	}
}
