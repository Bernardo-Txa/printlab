package orders

import (
	"context"
	"crypto/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run against a disposable database with the versioned migrations applied.
func TestPickupCartToOrder(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	var productID, cartID, orderID string
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	id := func(sql string, args ...any) string {
		t.Helper()
		var value string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	t.Cleanup(func() {
		for _, row := range []struct{ table, id string }{
			{"orders", orderID},
			{"carts", cartID},
			{"products", productID},
		} {
			if row.id != "" {
				if _, err := pool.Exec(context.Background(), "delete from public."+row.table+" where id = $1::uuid", row.id); err != nil {
					t.Errorf("cleanup %s: %v", row.table, err)
				}
			}
		}
	})

	tokenHash := make([]byte, 32)
	if _, err := rand.Read(tokenHash); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	productID = id(`insert into public.products(name,slug,price_cents,is_active) values('Retirada Teste',$1,2590,true) returning id::text`, "pickup-product-"+suffix)
	cartID = id(`insert into public.carts(token_hash,expires_at) values($1,$2) returning id::text`, tokenHash, now.Add(time.Hour))
	exec(`insert into public.cart_items(cart_id,product_id,quantity) values($1::uuid,$2::uuid,2)`, cartID, productID)
	exec(`insert into public.cart_customer_details(cart_id,full_name,email,phone,cpf) values($1::uuid,'Cliente Retirada','pickup-test@example.com','+5527999999999','52998224725')`, cartID)
	exec(`insert into public.cart_shipping_addresses(cart_id,postal_code,street,number,district,city,state,country_code) values($1::uuid,'29100000','Rua Teste','1','Centro','Vila Velha','ES','BR')`, cartID)
	exec(`insert into public.cart_shipping_selections(cart_id,delivery_method,shipping_box_id,provider,service_code,service_name,price_cents,package_weight_g,package_height_mm,package_width_mm,package_length_mm,input_hash,quoted_at,expires_at) values($1::uuid,'pickup',null,'','','',0,0,0,0,0,$2,$3,$4)`, cartID, make([]byte, 32), now, now.Add(shipping.QuoteTTL))

	repo := NewPostgresRepository(pool)
	params := ReviewParams{OriginPostalCode: "29100000", ServiceCodes: []string{"1"}}
	page, err := repo.Review(ctx, tokenHash, now, params)
	if err != nil {
		t.Fatal(err)
	}
	if page.Shipping.DeliveryMethod != shipping.DeliveryMethodPickup || page.ShippingPriceCents != 0 || page.TotalCents != 5180 {
		t.Fatalf("unexpected pickup review: shipping=%+v total=%d", page.Shipping, page.TotalCents)
	}

	result, err := repo.Confirm(ctx, tokenHash, page.Fingerprint, now, params)
	if err != nil {
		t.Fatal(err)
	}
	orderID = result.OrderID

	var method, provider, serviceCode, serviceName, boxName string
	var priceCents int64
	err = pool.QueryRow(ctx, `
		select shipping.delivery_method, shipping.provider, shipping.service_code, shipping.service_name, shipping.shipping_box_name, orders.shipping_price_cents
		from public.order_shipping_details shipping
		join public.orders orders on orders.id = shipping.order_id
		where shipping.order_id = $1::uuid
	`, orderID).Scan(&method, &provider, &serviceCode, &serviceName, &boxName, &priceCents)
	if err != nil {
		t.Fatal(err)
	}
	if method != shipping.DeliveryMethodPickup || priceCents != 0 || provider != "" || serviceCode != "" || serviceName != "" || boxName != "" {
		t.Fatalf("unexpected pickup order snapshot: method=%q price=%d provider=%q service=%q name=%q box=%q", method, priceCents, provider, serviceCode, serviceName, boxName)
	}

	detail, err := admin.NewPostgresRepository(pool).GetOrder(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Shipping.DeliveryMethod != shipping.DeliveryMethodPickup || detail.Shipping.PriceBRL != "R$ 0,00" || detail.Shipping.ServiceName != "" {
		t.Fatalf("unexpected admin pickup shipping: %+v", detail.Shipping)
	}
}

func TestPickupSelectionRejectsTamperedPrice(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tokenHash := make([]byte, 32)
	if _, err := rand.Read(tokenHash); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	var cartID string
	if err := pool.QueryRow(ctx, `insert into public.carts(token_hash,expires_at) values($1,$2) returning id::text`, tokenHash, now.Add(time.Hour)).Scan(&cartID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from public.carts where id = $1::uuid`, cartID)
	})

	_, err = pool.Exec(ctx, `insert into public.cart_shipping_selections(cart_id,delivery_method,provider,service_code,service_name,price_cents,package_weight_g,package_height_mm,package_width_mm,package_length_mm,input_hash,quoted_at,expires_at) values($1::uuid,'pickup','','','',999,0,0,0,0,$2,$3,$4)`, cartID, make([]byte, 32), now, now.Add(shipping.QuoteTTL))
	if err == nil {
		t.Fatal("expected tampered pickup price to be rejected")
	}
}
