package payments

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryUsesTransactionsLocksAndOrderSnapshots(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}
	sql := strings.ToLower(string(source))

	for _, expected := range []string{
		"r.pool.begin(ctx)",
		"tx.rollback(ctx)",
		"for update of o",
		"for update of o, p",
		"public.order_customer_details",
		"public.order_shipping_addresses",
		"public.order_shipping_details",
		"public.order_items",
		"public.order_payments",
		"status = 'paid'",
		"where p.order_nsu = $1",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected repository source to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"public.cart_items",
		"public.cart_customer_details",
		"public.cart_shipping_addresses",
		"public.cart_shipping_selections",
		"checkout_url=%",
		"checkout_url = %",
		"transaction_nsu=%",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected repository source not to contain %q", forbidden)
		}
	}
}
