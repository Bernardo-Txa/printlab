package orders

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryConfirmUsesTransactionLockAndCleanup(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	confirmSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) Confirm")
	if !ok {
		t.Fatal("expected Confirm to exist")
	}

	for _, expected := range []string{
		"r.pool.Begin(ctx)",
		"tx.Rollback(ctx)",
		"existingOrderForCart",
		"insertOrder(ctx, tx",
		"insertOrderCustomer(ctx, tx",
		"insertOrderAddress(ctx, tx",
		"insertOrderShipping(ctx, tx",
		"insertOrderFulfillment(ctx, tx",
		"insertOrderItems(ctx, tx",
		"convertCart(ctx, tx",
		"clearTemporaryCartData(ctx, tx",
		"tx.Commit(ctx)",
	} {
		if !strings.Contains(confirmSource, expected) {
			t.Fatalf("expected Confirm source to contain %q", expected)
		}
	}

	lockSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) lockCartByToken")
	if !ok {
		t.Fatal("expected lockCartByToken to exist")
	}
	if !strings.Contains(strings.ToLower(lockSource), "for update") {
		t.Fatal("expected lockCartByToken to lock cart row for update")
	}
}

func TestRepositorySnapshotsAndClearsTemporaryCheckoutData(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}
	sql := strings.ToLower(string(source))

	for _, expected := range []string{
		"insert into public.orders",
		"source_cart_id",
		"insert into public.order_customer_details",
		"insert into public.order_shipping_addresses",
		"insert into public.order_shipping_details",
		"insert into public.order_fulfillment",
		"insert into public.order_items",
		"unit_print_time_minutes",
		"unit_estimated_filament_weight_mg",
		"insert into public.order_item_filaments",
		"material_name",
		"color_name",
		"estimated_weight_mg_per_unit",
		"converted_at = $2",
		"delete from public.cart_shipping_selections",
		"delete from public.cart_shipping_addresses",
		"delete from public.cart_customer_details",
		"delete from public.cart_items",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected repository source to contain %q", expected)
		}
	}
}

func TestRepositoryTrackingQueryUsesPublicIdentifierAndMinimalProjection(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	trackSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) Track")
	if !ok {
		t.Fatal("expected Track to exist")
	}
	sql := strings.ToLower(trackSource)

	for _, expected := range []string{
		"where o.public_tracking_id = $1::uuid",
		"join public.order_fulfillment",
		"join public.order_shipping_details",
		"o.order_number",
		"o.status",
		"o.created_at",
		"fulfillment.production_status",
		"fulfillment.shipping_status",
		"shipping.service_name",
		"shipping.carrier_name",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected Track source to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"select *",
		"order_customer_details",
		"order_shipping_addresses",
		"order_items",
		"order_item_filaments",
		"order_payments",
		"checkout_url",
		"transaction_nsu",
		"invoice_slug",
		"public_tracking_id,",
		"o.id::text",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected Track source not to contain %q", forbidden)
		}
	}
}

func TestRepositoryReviewDoesNotRequoteShipping(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	reviewSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) reviewForCart")
	if !ok {
		t.Fatal("expected reviewForCart to exist")
	}

	if !strings.Contains(reviewSource, "shipping.BuildCartInputHash") {
		t.Fatal("expected review to validate the persisted shipping input hash")
	}
	if strings.Contains(strings.ToLower(reviewSource), ".calculate(") {
		t.Fatal("expected review not to call the shipping calculator")
	}
}

func TestRepositoryPreservesInactiveMaterialAndColorRecipes(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	cartFilamentsSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) cartFilaments")
	if !ok {
		t.Fatal("expected cartFilaments to exist")
	}
	sql := strings.ToLower(cartFilamentsSource)

	for _, expected := range []string{
		"join public.materials m",
		"join public.colors c",
		"where ci.cart_id = $1::uuid",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected cartFilaments source to contain %q", expected)
		}
	}
	if strings.Contains(sql, "m.is_active") || strings.Contains(sql, "c.is_active") {
		t.Fatal("expected production recipe snapshots not to filter inactive materials or colors")
	}
}

func repositoryFunctionSource(source string, marker string) (string, bool) {
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
