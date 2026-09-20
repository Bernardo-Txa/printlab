package orders

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run against a disposable database with the versioned migrations applied.
func TestCommercialColorCartToOrder(t *testing.T) {
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
	for _, withVariant := range []bool{false, true} {
		for _, withColors := range []bool{false, true} {
			t.Run(fmt.Sprintf("variant=%t/colors=%t", withVariant, withColors), func(t *testing.T) {
				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				productSlug, blueSlug, redSlug := "color-product-"+suffix, "blue-"+suffix, "red-"+suffix
				var productID, blueID, redID, variantID, materialID, boxID, cartID, orderID string
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
					for _, row := range []struct{ table, id string }{{"orders", orderID}, {"carts", cartID}, {"products", productID}, {"materials", materialID}, {"colors", blueID}, {"colors", redID}, {"shipping_boxes", boxID}} {
						if row.id != "" {
							if _, err := pool.Exec(ctx, "delete from public."+row.table+" where id = $1::uuid", row.id); err != nil {
								t.Errorf("cleanup %s: %v", row.table, err)
							}
						}
					}
				})
				productID = id(`insert into public.products(name,slug,price_cents,is_active,shipping_weight_g,shipping_height_mm,shipping_width_mm,shipping_length_mm) values('Dragao',$1,100,true,100,10,10,10) returning id::text`, productSlug)
				blueID = id(`insert into public.colors(name,slug) values('Azul',$1) returning id::text`, blueSlug)
				redID = id(`insert into public.colors(name,slug) values('Vermelho',$1) returning id::text`, redSlug)
				if withColors {
					exec(`insert into public.product_colors(product_id,color_id) values($1::uuid,$2::uuid),($1::uuid,$3::uuid)`, productID, blueID, redID)
				}
				if withVariant {
					variantID = id(`insert into public.product_variants(product_id,name,slug,is_active,price_cents) values($1::uuid,'Padrao','padrao',true,150) returning id::text`, productID)
					materialID = id(`insert into public.materials(name,slug) values('PLA',$1) returning id::text`, "material-"+suffix)
					exec(`insert into public.variant_filaments(variant_id,material_id,color_id,estimated_weight_mg) values($1::uuid,$2::uuid,$3::uuid,1000)`, variantID, materialID, redID)
				}
				hash := make([]byte, 32)
				if _, err := rand.Read(hash); err != nil {
					t.Fatal(err)
				}
				cartRepo := cart.NewPostgresRepository(pool)
				service := cart.NewService(cartRepo)
				add := func(colorSlug string, quantity int) error {
					input := cart.AddItemInput{ProductSlug: productSlug, ColorSlug: colorSlug, Quantity: quantity}
					if withVariant {
						input.VariantSlug = "padrao"
					}
					active, err := service.Add(ctx, hash, input)
					if err == nil {
						cartID = active.ID
					}
					return err
				}
				if withColors {
					for _, color := range []string{blueSlug, redSlug, blueSlug, ""} {
						if err := add(color, 1); err != nil {
							t.Fatal(err)
						}
					}
					if err := add(blueSlug, 98); !errors.Is(err, cart.ErrQuantityLimit) {
						t.Fatalf("quantity limit: %v", err)
					}
					if err := add("invalid-color", 1); !errors.Is(err, cart.ErrInvalidColor) {
						t.Fatalf("invalid color: %v", err)
					}
				} else {
					if err := add("", 1); err != nil {
						t.Fatal(err)
					}
					if err := add(blueSlug, 1); !errors.Is(err, cart.ErrInvalidColor) {
						t.Fatalf("unlinked color: %v", err)
					}
				}
				stored, err := cartRepo.ListItems(ctx, cartID)
				if err != nil {
					t.Fatal(err)
				}
				wantLines := 1
				if withColors {
					wantLines = 3
				}
				if len(stored) != wantLines {
					t.Fatalf("cart lines=%d want=%d", len(stored), wantLines)
				}
				if withColors {
					foundBlue := false
					for _, line := range stored {
						if line.ColorID == blueID {
							foundBlue = true
							if line.Quantity != 2 || line.ColorName != "Azul" || !line.ColorAvailable {
								t.Fatalf("blue=%+v", line)
							}
						}
					}
					if !foundBlue {
						t.Fatal("missing blue cart item")
					}
					exec(`update public.colors set is_active=false where id=$1::uuid`, blueID)
					view, err := service.View(ctx, hash)
					if err != nil || !view.HasUnavailableItems {
						t.Fatalf("inactive color cart: %+v %v", view, err)
					}
					if _, _, err := NewPostgresRepository(pool).reviewItems(ctx, pool, cartID); !errors.Is(err, ErrUnavailableItems) {
						t.Fatalf("inactive review accepted: %v", err)
					}
					if err := cartRepo.AddItem(ctx, cartID, productID, optionalID(variantID), &blueID, 1); !errors.Is(err, cart.ErrInvalidColor) {
						t.Fatalf("repository accepted inactive color: %v", err)
					}
					exec(`update public.colors set is_active=true where id=$1::uuid`, blueID)
					exec(`delete from public.product_colors where product_id=$1::uuid and color_id=$2::uuid`, productID, blueID)
					if _, _, err := NewPostgresRepository(pool).reviewItems(ctx, pool, cartID); !errors.Is(err, ErrUnavailableItems) {
						t.Fatalf("unlinked review accepted: %v", err)
					}
					exec(`insert into public.product_colors(product_id,color_id) values($1::uuid,$2::uuid)`, productID, blueID)
				}
				exec(`insert into public.cart_customer_details(cart_id,full_name,email,phone,cpf) values($1::uuid,'Cliente Teste','color-test@example.com','+5527999999999','52998224725')`, cartID)
				exec(`insert into public.cart_shipping_addresses(cart_id,postal_code,street,number,district,city,state,country_code) values($1::uuid,'29100000','Rua Teste','1','Centro','Vila Velha','ES','BR')`, cartID)
				boxID = id(`insert into public.shipping_boxes(name,slug,internal_height_mm,internal_width_mm,internal_length_mm,external_height_mm,external_width_mm,external_length_mm,packaging_weight_g) values('Caixa',$1,100,100,100,100,100,100,10) returning id::text`, "box-"+suffix)
				repo := NewPostgresRepository(pool)
				_, shippingItems, err := repo.reviewItems(ctx, pool, cartID)
				if err != nil {
					t.Fatal(err)
				}
				box := shipping.ShippingBox{ID: boxID, Name: "Caixa", Slug: "box-" + suffix, Internal: shipping.DimensionsMM{Height: 100, Width: 100, Length: 100}, External: shipping.DimensionsMM{Height: 100, Width: 100, Length: 100}, PackagingWeightG: 10}
				params := ReviewParams{OriginPostalCode: "29100000", ServiceCodes: []string{"1"}}
				inputHash, err := shipping.BuildCartInputHash(params.OriginPostalCode, "29100000", params.ServiceCodes, shippingItems, box)
				if err != nil {
					t.Fatal(err)
				}
				now := time.Now()
				exec(`insert into public.cart_shipping_selections(cart_id,shipping_box_id,provider,service_code,service_name,price_cents,package_weight_g,package_height_mm,package_width_mm,package_length_mm,input_hash,quoted_at,expires_at) values($1::uuid,$2::uuid,'superfrete','1','PAC',100,410,100,100,100,$3,$4,$5)`, cartID, boxID, inputHash, now, now.Add(time.Hour))
				page, err := repo.Review(ctx, hash, now, params)
				if err != nil {
					t.Fatal(err)
				}
				result, err := repo.Confirm(ctx, hash, page.Fingerprint, now, params)
				if err != nil {
					t.Fatal(err)
				}
				orderID = result.OrderID
				// Names in a placed order must survive catalog edits.
				exec(`update public.colors set name='Renomeada' where id=$1::uuid`, blueID)
				items, err := repo.orderItems(ctx, pool, orderID)
				if err != nil || len(items) != wantLines {
					t.Fatalf("order=%+v err=%v", items, err)
				}
				foundBlue := false
				for _, item := range items {
					if item.HasVariant != withVariant {
						t.Fatalf("variant changed: %+v", item)
					}
					if withVariant && (len(item.Filaments) != 1 || item.Filaments[0].ColorName != "Vermelho") {
						t.Fatalf("recipe changed: %+v", item)
					}
					if item.ColorID == blueID {
						foundBlue = true
						if item.ColorName != "Azul" || item.ColorSlug != blueSlug || item.Quantity != 2 {
							t.Fatalf("snapshot=%+v", item)
						}
					}
					if !withColors && (item.ColorID != "" || item.ColorName != "") {
						t.Fatalf("colorless order changed: %+v", item)
					}
				}
				if withColors && !foundBlue {
					t.Fatal("commercial color lost at confirmation")
				}
				detail, err := admin.NewPostgresRepository(pool).GetOrder(ctx, orderID)
				if err != nil || len(detail.Items) != wantLines {
					t.Fatalf("admin detail=%+v err=%v", detail, err)
				}
				if withColors {
					found := false
					for _, item := range detail.Items {
						if item.ColorID == blueID && item.ColorName == "Azul" {
							found = true
						}
					}
					if !found {
						t.Fatal("commercial snapshot missing from Admin")
					}
					exec(`delete from public.product_colors where product_id=$1::uuid and color_id=$2::uuid`, productID, blueID)
					exec(`delete from public.colors where id=$1::uuid`, blueID)
					historical, err := repo.orderItems(ctx, pool, orderID)
					if err != nil {
						t.Fatal(err)
					}
					found = false
					for _, item := range historical {
						if item.ColorName == "Azul" && item.ColorID == "" {
							found = true
						}
					}
					if !found {
						t.Fatal("color deletion lost historical name")
					}
				}
				again, err := repo.Confirm(ctx, hash, page.Fingerprint, now, params)
				if err != nil || again.OrderID != orderID {
					t.Fatalf("confirmation lost idempotency: %+v %v", again, err)
				}
			})
		}
	}
}

func optionalID(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
