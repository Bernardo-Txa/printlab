package admin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdminProductColorsAssociation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repo := NewPostgresRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	input := AdminProductSaveInput{Name: "Commercial color test", Slug: "commercial-colors-" + suffix, PriceCents: 100}
	productID, err := repo.CreateAdminProduct(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ID = productID
	var blueID, redID, productionColorID, materialID, variantID, recipeID string
	// Remove the product before its referenced production data.
	t.Cleanup(func() {
		for _, query := range []struct{ sql, id string }{
			{"delete from public.products where id = $1::uuid", productID},
			{"delete from public.materials where id = $1::uuid", materialID},
			{"delete from public.colors where id = $1::uuid", blueID},
			{"delete from public.colors where id = $1::uuid", redID},
			{"delete from public.colors where id = $1::uuid", productionColorID},
		} {
			if query.id != "" {
				if _, err := pool.Exec(ctx, query.sql, query.id); err != nil {
					t.Errorf("fixture cleanup: %v", err)
				}
			}
		}
	})
	assertColors := func(want ...string) {
		t.Helper()
		colors, err := repo.ListProductColors(ctx, productID)
		if err != nil {
			t.Fatal(err)
		}
		if len(colors) != len(want) {
			t.Fatalf("colors=%+v, want %v", colors, want)
		}
		for i, id := range want {
			if colors[i].ColorID != id {
				t.Fatalf("ordered colors=%+v, want %v", colors, want)
			}
		}
	}
	assertColors() // A: drafts exist without commercial associations.
	for _, color := range []struct {
		name, slug string
		dst        *string
	}{
		{"Azul", "blue-", &blueID}, {"Vermelho", "red-", &redID}, {"Azul Ceu Cliever", "production-", &productionColorID},
	} {
		if err := pool.QueryRow(ctx, `insert into public.colors (name, slug) values ($1, $2) returning id::text`, color.name, color.slug+suffix).Scan(color.dst); err != nil {
			t.Fatal(err)
		}
	}
	input.CommercialColors = []ProductColorSelection{{ColorID: blueID, SortOrder: 20}}
	if err := repo.UpdateAdminProduct(ctx, input); err != nil {
		t.Fatal(err)
	}
	assertColors(blueID) // B: explicit commercial association.
	// C: the DB and repository reject duplicates and preserve the prior selection.
	_, err = pool.Exec(ctx, `insert into public.product_colors (product_id, color_id) values ($1::uuid, $2::uuid)`, productID, blueID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "product_colors_product_color_unique" {
		t.Fatalf("expected unique constraint: %v", err)
	}
	input.CommercialColors = append(input.CommercialColors, input.CommercialColors[0])
	if err := repo.UpdateAdminProduct(ctx, input); !errors.Is(err, ErrValidation) {
		t.Fatalf("duplicate: %v", err)
	}
	assertColors(blueID)
	input.CommercialColors = []ProductColorSelection{{ColorID: blueID, SortOrder: 20}, {ColorID: redID, SortOrder: 5}}
	if err := repo.UpdateAdminProduct(ctx, input); err != nil {
		t.Fatal(err)
	}
	assertColors(redID, blueID) // D: configured order, not name or insertion order.
	options, err := repo.ListAdminProductColors(ctx, productID)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) < 2 || options[0].ID != redID || options[1].ID != blueID {
		t.Fatalf("admin order=%+v", options)
	}
	// E/F: a production recipe has its own color and must not follow commercial choices.
	if err := pool.QueryRow(ctx, `insert into public.materials (name, slug) values ('PLA', $1) returning id::text`, "material-"+suffix).Scan(&materialID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into public.product_variants (product_id, name, slug) values ($1::uuid, 'Original variant', 'original') returning id::text`, productID).Scan(&variantID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into public.variant_filaments (variant_id, material_id, color_id, estimated_weight_mg, label) values ($1::uuid, $2::uuid, $3::uuid, 350000, 'PLA Azul Ceu Cliever') returning id::text`, variantID, materialID, productionColorID).Scan(&recipeID); err != nil {
		t.Fatal(err)
	}
	input.CommercialColors = []ProductColorSelection{{ColorID: blueID, SortOrder: 0}}
	if err := repo.UpdateAdminProduct(ctx, input); err != nil {
		t.Fatal(err)
	}
	assertColors(blueID)
	input.CommercialColors = nil
	if err := repo.UpdateAdminProduct(ctx, input); err != nil {
		t.Fatal(err)
	}
	assertColors()
	var gotMaterial, gotColor, gotVariant, label string
	var weight int64
	if err := pool.QueryRow(ctx, `select material_id::text, color_id::text, variant_id::text, estimated_weight_mg, label from public.variant_filaments where id = $1::uuid`, recipeID).Scan(&gotMaterial, &gotColor, &gotVariant, &weight, &label); err != nil {
		t.Fatal(err)
	}
	if gotMaterial != materialID || gotColor != productionColorID || gotVariant != variantID || weight != 350000 || label != "PLA Azul Ceu Cliever" {
		t.Fatal("commercial edits changed the recipe")
	}
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from public.colors where id = any($1::uuid[])`, []string{blueID, redID, productionColorID}).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatal("commercial removal deleted a color")
	}
	// A failed save must not leave partial product edits or orphan creations.
	input.Name = "Should roll back"
	input.CommercialColors = []ProductColorSelection{{ColorID: "ffffffff-ffff-ffff-ffff-ffffffffffff"}}
	if err := repo.UpdateAdminProduct(ctx, input); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown color: %v", err)
	}
	var name string
	if err := pool.QueryRow(ctx, `select name from public.products where id = $1::uuid`, productID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Commercial color test" {
		t.Fatal("failed update partially persisted")
	}
	input.Slug = "rollback-" + suffix
	if _, err := repo.CreateAdminProduct(ctx, input); !errors.Is(err, ErrValidation) {
		t.Fatalf("failed create: %v", err)
	}
	if err := pool.QueryRow(ctx, `select count(*) from public.products where slug = $1`, input.Slug).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed create left a product")
	}
}
