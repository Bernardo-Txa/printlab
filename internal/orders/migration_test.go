package orders

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func TestOrderTrackingMigrationDocumentsPublicIDAndFulfillment(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_add_order_tracking_and_fulfillment.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one add_order_tracking_and_fulfillment migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"add column public_tracking_id uuid null default gen_random_uuid()",
		"set public_tracking_id = gen_random_uuid()",
		"alter column public_tracking_id set not null",
		"orders_public_tracking_id_unique unique (public_tracking_id)",
		"create table public.order_fulfillment",
		"order_id uuid primary key",
		"references public.orders (id)",
		"on delete cascade",
		"production_status text not null default 'waiting'",
		"shipping_status text not null default 'waiting'",
		"production_status in ('waiting', 'in_production', 'completed')",
		"shipping_status in ('waiting', 'preparing', 'shipped', 'delivered')",
		"shipping_status = 'waiting'",
		"or production_status = 'completed'",
		"insert into public.order_fulfillment (order_id)",
		"on conflict (order_id) do nothing",
		"alter table public.order_fulfillment enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected tracking migration to contain %q", expected)
		}
	}

	if strings.Contains(sql, "create policy") {
		t.Fatal("expected tracking migration not to create public RLS policies")
	}
}

func TestOrderTrackingSchemaWithTestDatabaseURL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected test database connection, got %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("expected transaction, got %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var missingTrackingIDs int
	if err := tx.QueryRow(ctx, `
		select count(*)
		from public.orders
		where public_tracking_id is null
	`).Scan(&missingTrackingIDs); err != nil {
		t.Fatalf("expected public_tracking_id column, got %v", err)
	}
	if missingTrackingIDs != 0 {
		t.Fatalf("expected all existing orders to have public_tracking_id, got %d missing", missingTrackingIDs)
	}

	firstID, firstTrackingID := insertTrackingSchemaOrder(t, ctx, tx)
	_, secondTrackingID := insertTrackingSchemaOrder(t, ctx, tx)
	if firstTrackingID == "" || secondTrackingID == "" || firstTrackingID == secondTrackingID {
		t.Fatalf("expected different generated tracking IDs, got %q and %q", firstTrackingID, secondTrackingID)
	}

	execExpectError(t, ctx, tx, "tracking_null", `
		insert into public.orders (
			public_tracking_id,
			status,
			currency,
			products_subtotal_cents,
			shipping_price_cents,
			total_cents
		) values (
			null,
			'pending_payment',
			'BRL',
			100,
			0,
			100
		)
	`)

	execExpectError(t, ctx, tx, "tracking_unique", `
		insert into public.orders (
			public_tracking_id,
			status,
			currency,
			products_subtotal_cents,
			shipping_price_cents,
			total_cents
		) values (
			$1::uuid,
			'pending_payment',
			'BRL',
			100,
			0,
			100
		)
	`, firstTrackingID)

	if _, err := tx.Exec(ctx, `
		insert into public.order_fulfillment (
			order_id
		) values (
			$1::uuid
		)
	`, firstID); err != nil {
		t.Fatalf("expected default fulfillment to insert, got %v", err)
	}

	execExpectError(t, ctx, tx, "shipping_requires_completed", `
		with created_order as (
			insert into public.orders (
				status,
				currency,
				products_subtotal_cents,
				shipping_price_cents,
				total_cents
			) values (
				'paid',
				'BRL',
				100,
				0,
				100
			)
			returning id
		)
		insert into public.order_fulfillment (
			order_id,
			production_status,
			shipping_status
		)
		select
			id,
			'waiting',
			'shipped'
		from created_order
	`)

	var rowSecurity bool
	if err := tx.QueryRow(ctx, `
		select relrowsecurity
		from pg_class
		where oid = 'public.order_fulfillment'::regclass
	`).Scan(&rowSecurity); err != nil {
		t.Fatalf("expected order_fulfillment RLS metadata, got %v", err)
	}
	if !rowSecurity {
		t.Fatal("expected order_fulfillment RLS to be enabled")
	}
}

func insertTrackingSchemaOrder(t *testing.T, ctx context.Context, tx pgx.Tx) (string, string) {
	t.Helper()

	var orderID string
	var trackingID string
	if err := tx.QueryRow(ctx, `
		insert into public.orders (
			status,
			currency,
			products_subtotal_cents,
			shipping_price_cents,
			total_cents
		) values (
			'pending_payment',
			'BRL',
			100,
			0,
			100
		)
		returning id::text, public_tracking_id::text
	`).Scan(&orderID, &trackingID); err != nil {
		t.Fatalf("expected order insert with generated public_tracking_id, got %v", err)
	}

	return orderID, trackingID
}

func execExpectError(t *testing.T, ctx context.Context, tx pgx.Tx, savepoint string, sql string, args ...any) {
	t.Helper()

	if _, err := tx.Exec(ctx, "savepoint "+savepoint); err != nil {
		t.Fatalf("expected savepoint %s, got %v", savepoint, err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err == nil {
		t.Fatalf("expected statement to fail under savepoint %s", savepoint)
	}
	if _, err := tx.Exec(ctx, "rollback to savepoint "+savepoint); err != nil {
		t.Fatalf("expected rollback to savepoint %s, got %v", savepoint, err)
	}
}
