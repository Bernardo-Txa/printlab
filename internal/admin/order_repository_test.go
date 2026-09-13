package admin

import (
	"os"
	"strings"
	"testing"
)

func TestAdminOrderMutationsUseTransactionLockAndAuditEvent(t *testing.T) {
	source, err := os.ReadFile("order_repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}
	sql := strings.ToLower(string(source))

	for _, expected := range []string{
		"r.pool.begin(ctx)",
		"tx.rollback(ctx)",
		"for update of o, fulfillment",
		"update public.order_fulfillment",
		"production_status = $2",
		"shipping_status = $2",
		"insert into public.admin_order_events",
		"tx.commit(ctx)",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected admin order repository to contain %q", expected)
		}
	}

	productionSource, ok := adminFunctionSource(string(source), "func (r *PostgresRepository) ChangeProductionStatus")
	if !ok {
		t.Fatal("expected ChangeProductionStatus to exist")
	}
	if !strings.Contains(productionSource, "ValidateProductionTransition") || !strings.Contains(productionSource, "EventTypeProductionStatusChanged") {
		t.Fatal("expected production mutation to validate transition and audit production event")
	}

	shippingSource, ok := adminFunctionSource(string(source), "func (r *PostgresRepository) ChangeShippingStatus")
	if !ok {
		t.Fatal("expected ChangeShippingStatus to exist")
	}
	if !strings.Contains(shippingSource, "ValidateShippingTransition") || !strings.Contains(shippingSource, "EventTypeShippingStatusChanged") {
		t.Fatal("expected shipping mutation to validate transition and audit shipping event")
	}
}

func TestAdminOrderListAvoidsPIIAndPaymentIdentifiers(t *testing.T) {
	source, err := os.ReadFile("order_repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	listSource, ok := adminFunctionSource(string(source), "func (r *PostgresRepository) ListOrders")
	if !ok {
		t.Fatal("expected ListOrders to exist")
	}
	sql := strings.ToLower(listSource)

	for _, expected := range []string{
		"public.orders",
		"public.order_fulfillment",
		"public.order_items",
		"o.order_number",
		"o.total_cents",
		"fulfillment.production_status",
		"fulfillment.shipping_status",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected ListOrders source to contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"order_customer_details",
		"order_shipping_addresses",
		"order_payments",
		"checkout_url",
		"transaction_nsu",
		"invoice_slug",
		"cpf",
		"email",
		"phone",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected ListOrders source not to contain %q", forbidden)
		}
	}
}

func adminFunctionSource(source string, marker string) (string, bool) {
	start := strings.Index(source, marker)
	if start == -1 {
		return "", false
	}

	functionSource := source[start:]
	nextFunction := strings.Index(functionSource[len(marker):], "\nfunc ")
	if nextFunction == -1 {
		return functionSource, true
	}

	return functionSource[:len(marker)+nextFunction], true
}
