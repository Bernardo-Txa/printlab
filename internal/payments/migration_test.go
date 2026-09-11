package payments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddOrderPaymentsMigrationDocumentsPaymentSchema(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_add_order_payments.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one add_order_payments migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"drop constraint orders_status_allowed",
		"status in ('pending_payment', 'paid')",
		"create table public.order_payments",
		"order_id uuid primary key",
		"provider text not null default 'infinitepay'",
		"status text not null default 'pending'",
		"order_nsu text not null unique",
		"checkout_url text null",
		"invoice_slug text null",
		"transaction_nsu text null",
		"amount_cents bigint null",
		"paid_amount_cents bigint null",
		"installments integer null",
		"capture_method text null",
		"references public.orders (id)",
		"on delete cascade",
		"provider = 'infinitepay'",
		"status in ('pending', 'paid')",
		"amount_cents >= 0",
		"paid_amount_cents >= 0",
		"installments > 0",
		"order_payments_transaction_nsu_unique_idx",
		"where transaction_nsu is not null",
		"alter table public.order_payments enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"create policy",
		"insert into",
		"webhook_url",
		"database_url",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected migration not to contain %q", forbidden)
		}
	}
}
