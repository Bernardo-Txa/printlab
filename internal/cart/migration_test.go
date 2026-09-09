package cart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCartsMigrationDocumentsCoreConstraints(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_create_carts.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one create_carts migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	migration := string(source)
	for _, required := range []string{
		"create table public.carts",
		"token_hash bytea not null unique",
		"constraint carts_token_hash_length check (octet_length(token_hash) = 32)",
		"constraint carts_expires_after_created check (expires_at > created_at)",
		"create table public.cart_items",
		"constraint cart_items_quantity_range check (quantity between 1 and 99)",
		"constraint cart_items_variant_product_fkey foreign key (variant_id, product_id)",
		"create unique index cart_items_cart_product_no_variant_unique_idx",
		"where variant_id is null",
		"create unique index cart_items_cart_product_variant_unique_idx",
		"where variant_id is not null",
		"alter table public.carts enable row level security",
		"alter table public.cart_items enable row level security",
	} {
		if !strings.Contains(migration, required) {
			t.Fatalf("expected migration to contain %q", required)
		}
	}
}
