package customers

import (
	"os"
	"strings"
	"testing"
)

func TestCreateCartCustomerDetailsMigration(t *testing.T) {
	source, err := os.ReadFile("../../supabase/migrations/20260909203845_create_cart_customer_details.sql")
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	migration := string(source)
	for _, expected := range []string{
		"create table public.cart_customer_details",
		"create table public.cart_shipping_addresses",
		"cart_id uuid primary key",
		"references public.carts (id)",
		"on delete cascade",
		"cpf ~ '^[0-9]{11}$'",
		"postal_code ~ '^[0-9]{8}$'",
		"country_code = 'BR'",
		"cart_shipping_addresses_state_allowed",
		"alter table public.cart_customer_details enable row level security",
		"alter table public.cart_shipping_addresses enable row level security",
	} {
		if !strings.Contains(migration, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}

	if strings.Contains(strings.ToLower(migration), "insert into") {
		t.Fatal("expected migration not to insert PII or fake data")
	}
}
