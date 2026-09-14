package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransientCleanupMigrationSchedulesOnlyExpiredTransientData(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_schedule_transient_data_cleanup.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one schedule_transient_data_cleanup migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected transient cleanup migration to be readable, got %v", err)
	}

	migration := strings.ToLower(string(source))
	for _, required := range []string{
		"create extension if not exists pg_cron",
		"cron.schedule(",
		"'printlab_transient_data_cleanup'",
		"'17 3 * * *'",
		"delete from public.admin_sessions",
		"delete from public.carts",
		"where expires_at <= now()",
	} {
		if !strings.Contains(migration, required) {
			t.Fatalf("expected transient cleanup migration to contain %q", required)
		}
	}

	for _, forbidden := range []string{
		"delete from public.orders",
		"delete from public.order_customer_details",
		"delete from public.order_shipping_addresses",
		"delete from public.order_items",
		"delete from public.order_payments",
		"delete from public.order_fulfillment",
		"delete from public.admin_order_events",
		"truncate",
		"http",
		"secret",
	} {
		if strings.Contains(migration, forbidden) {
			t.Fatalf("expected transient cleanup migration not to contain %q", forbidden)
		}
	}
}

func TestTransientCleanupForeignKeysPreserveOrders(t *testing.T) {
	for _, check := range []struct {
		path     string
		required []string
	}{
		{
			path: "../../supabase/migrations/20260909194855_create_carts.sql",
			required: []string{
				"constraint cart_items_cart_id_fkey foreign key (cart_id)",
				"on delete cascade",
			},
		},
		{
			path: "../../supabase/migrations/20260909203845_create_cart_customer_details.sql",
			required: []string{
				"constraint cart_customer_details_cart_id_fkey foreign key (cart_id)",
				"constraint cart_shipping_addresses_cart_id_fkey foreign key (cart_id)",
				"on delete cascade",
			},
		},
		{
			path: "../../supabase/migrations/20260909220454_add_shipping_profiles_and_selections.sql",
			required: []string{
				"constraint cart_shipping_selections_cart_id_fkey foreign key (cart_id)",
				"on delete cascade",
			},
		},
		{
			path: "../../supabase/migrations/20260909233711_create_orders.sql",
			required: []string{
				"constraint orders_source_cart_id_fkey foreign key (source_cart_id)",
				"on delete set null",
			},
		},
	} {
		source, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("expected migration %s to be readable, got %v", check.path, err)
		}
		migration := strings.ToLower(string(source))
		for _, required := range check.required {
			if !strings.Contains(migration, required) {
				t.Fatalf("expected migration %s to contain %q", check.path, required)
			}
		}
	}
}
