package admin

import (
	"context"
	"errors"
	"testing"
)

const testCommercialColorID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

type commercialColorTestRepository struct {
	*imageTestRepository
	form    AdminProductForm
	options []AdminProductColorOption
	saved   *AdminProductSaveInput
}

func (r *commercialColorTestRepository) GetAdminProductForm(context.Context, string) (AdminProductFormPage, error) {
	return AdminProductFormPage{Form: r.form}, nil
}
func (r *commercialColorTestRepository) ListAdminProductColors(context.Context, string) ([]AdminProductColorOption, error) {
	return append([]AdminProductColorOption(nil), r.options...), nil
}
func (r *commercialColorTestRepository) CreateAdminProduct(_ context.Context, input AdminProductSaveInput) (string, error) {
	r.saved = &input
	return "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", nil
}
func (r *commercialColorTestRepository) UpdateAdminProduct(_ context.Context, input AdminProductSaveInput) error {
	r.saved = &input
	return nil
}

func TestCommercialColorsProductSave(t *testing.T) {
	for _, test := range []struct {
		name                                                                                                            string
		create, wasActive, active, hadColor, selectColor, inactiveColor, invalidColor, duplicate, invalidOrder, wantErr bool
	}{
		{name: "inactive draft without colors", create: true},
		{name: "active creation requires color", create: true, active: true, wantErr: true},
		{name: "active creation with color", create: true, active: true, selectColor: true},
		{name: "activation requires color", active: true, wantErr: true},
		{name: "activation with color", active: true, selectColor: true},
		{name: "inactive draft cannot add inactive color", selectColor: true, inactiveColor: true, wantErr: true},
		{name: "inactive creation cannot add inactive color", create: true, selectColor: true, inactiveColor: true, wantErr: true},
		{name: "inactive color does not permit activation", active: true, selectColor: true, inactiveColor: true, wantErr: true},
		{name: "existing inactive color can be preserved", wasActive: true, active: false, hadColor: true, selectColor: true, inactiveColor: true},
		{name: "legacy active edit without colors", wasActive: true, active: true},
		{name: "cannot remove final color while active", wasActive: true, active: true, hadColor: true, wantErr: true},
		{name: "deactivation allows removing all colors", wasActive: true, hadColor: true},
		{name: "duplicate color", selectColor: true, duplicate: true, wantErr: true},
		{name: "unknown color", selectColor: true, invalidColor: true, wantErr: true},
		{name: "invalid ordering", selectColor: true, invalidOrder: true, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &commercialColorTestRepository{imageTestRepository: newImageTestRepository(),
				form:    AdminProductForm{Slug: "produto", IsActive: test.wasActive},
				options: []AdminProductColorOption{{AdminSelectOption: AdminSelectOption{ID: testCommercialColorID, Active: !test.inactiveColor, Selected: test.hadColor}, SortOrder: "7"}},
			}
			service := NewService(fakeAuthClient{}, repo, testAdminUserID, CookieOptions{})
			form := AdminProductForm{Name: "Produto", PriceBRL: "10,00", IsActive: test.active,
				ShippingWeightG: "100", ShippingHeightMM: "100", ShippingWidthMM: "200", ShippingLengthMM: "300",
				CommercialColorOrders: map[string]string{testCommercialColorID: "7"},
			}
			if test.selectColor {
				form.CommercialColorIDs = []string{testCommercialColorID}
			}
			if test.duplicate {
				form.CommercialColorIDs = append(form.CommercialColorIDs, testCommercialColorID)
			}
			if test.invalidColor {
				form.CommercialColorIDs = []string{"cccccccc-cccc-cccc-cccc-cccccccccccc"}
			}
			if test.invalidOrder {
				form.CommercialColorOrders[testCommercialColorID] = "-1"
			}
			var page AdminProductFormPage
			var err error
			if test.create {
				_, page, err = service.CreateAdminProduct(context.Background(), form)
			} else {
				page, err = service.UpdateAdminProduct(context.Background(), "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", form)
			}
			if test.wantErr {
				if !errors.Is(err, ErrValidation) || repo.saved != nil || page.Errors.Get("commercial_color_ids") == "" {
					t.Fatalf("expected color validation before save: page=%+v, err=%v, saved=%+v", page, err, repo.saved)
				}
				if !test.selectColor && page.CommercialColors[0].Selected {
					t.Fatal("unchecked color was reselected on validation error")
				}
				return
			}
			if err != nil || repo.saved == nil {
				t.Fatalf("save: %v", err)
			}
			if test.selectColor && (len(repo.saved.CommercialColors) != 1 || repo.saved.CommercialColors[0].SortOrder != 7) {
				t.Fatalf("selection/order lost: %+v", repo.saved)
			}
		})
	}
}
