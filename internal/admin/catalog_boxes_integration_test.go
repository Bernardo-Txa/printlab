package admin

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdminBoxOperationalDimensionsPersistence(t *testing.T) {
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
	service := NewService(fakeAuthClient{}, repo, testAdminUserID, CookieOptions{})
	form := AdminBoxForm{
		Name:     fmt.Sprintf("Admin box test %d", time.Now().UnixNano()),
		HeightMM: "100", WidthMM: "200", LengthMM: "300",
		PackagingWeightG: "50", IsActive: true, SortOrder: "0",
	}
	id, _, err := service.CreateAdminBox(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "delete from public.shipping_boxes where id = $1::uuid", id); err != nil {
			t.Errorf("delete test box: %v", err)
		}
	})
	assertPersisted := func(want [6]int, wantWeight int) {
		t.Helper()
		var got [6]int
		var weight int
		err := pool.QueryRow(ctx, `select internal_height_mm, internal_width_mm, internal_length_mm,
			external_height_mm, external_width_mm, external_length_mm, packaging_weight_g
			from public.shipping_boxes where id = $1::uuid`, id).
			Scan(&got[0], &got[1], &got[2], &got[3], &got[4], &got[5], &weight)
		if err != nil {
			t.Fatal(err)
		}
		if got != want || weight != wantWeight {
			t.Fatalf("dimensions/weight = %v/%d, want %v/%d", got, weight, want, wantWeight)
		}
	}
	assertPersisted([6]int{100, 200, 300, 100, 200, 300}, 50)

	// Turn the created box into a legacy record with distinct internal dimensions.
	_, err = pool.Exec(ctx, `update public.shipping_boxes
		set internal_height_mm = 98, internal_width_mm = 198, internal_length_mm = 298
		where id = $1::uuid`, id)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.GetAdminBox(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	form = page.Form
	if form.HeightMM != "100" || form.WidthMM != "200" || form.LengthMM != "300" {
		t.Fatalf("legacy form must display external dimensions: %+v", form)
	}
	list, err := service.ListAdminBoxes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, box := range list.Boxes {
		if box.ID == id {
			found = true
			if box.DimensionsLabel != "100 x 200 x 300 mm" {
				t.Fatalf("legacy list dimensions = %q", box.DimensionsLabel)
			}
		}
	}
	if !found {
		t.Fatal("test box missing from administrative list")
	}

	form.PackagingWeightG = "75"
	if _, err := service.UpdateAdminBox(ctx, id, form); err != nil {
		t.Fatal(err)
	}
	assertPersisted([6]int{98, 198, 298, 100, 200, 300}, 75)

	form.HeightMM, form.WidthMM, form.LengthMM = "110", "210", "310"
	if _, err := service.UpdateAdminBox(ctx, id, form); err != nil {
		t.Fatal(err)
	}
	assertPersisted([6]int{110, 210, 310, 110, 210, 310}, 75)
}
