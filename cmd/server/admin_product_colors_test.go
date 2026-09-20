package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
	"regexp"
)

// Read actual rendered controls so changes to field names or form placement
// cannot silently break the existing POST parser.
func TestProductCommercialColorFormRoundTrip(t *testing.T) {
	for _, isNew := range []bool{true, false} {
		page := admindomain.AdminProductFormPage{IsNew: isNew, Action: "/admin/produtos", Form: admindomain.AdminProductForm{Name: "Dragao"}}
		for i, name := range []string{"Azul", "Preto", "Vermelho", "Branco"} {
			id := []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "cccccccc-cccc-cccc-cccc-cccccccccccc", "dddddddd-dddd-dddd-dddd-dddddddddddd"}[i]
			page.CommercialColors = append(page.CommercialColors, admindomain.AdminProductColorOption{AdminSelectOption: admindomain.AdminSelectOption{ID: id, Label: name, Active: true, Selected: !isNew && i < 3}, SortOrder: "5"})
		}
		var out strings.Builder
		if err := templates.AdminProductForm(page).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Cores disponíveis para venda") {
			t.Fatal("missing commercial colors section")
		}
		formHTML := regexp.MustCompile(`(?s)<form[^>]*action="/admin/produtos"[^>]*>(.*?)</form>`).FindStringSubmatch(out.String())
		if len(formHTML) != 2 {
			t.Fatal("missing product form")
		}
		values := url.Values{}
		colorCount, checkedCount := 0, 0
		for _, tag := range regexp.MustCompile(`<input[^>]*>`).FindAllString(formHTML[1], -1) {
			attrs := map[string]string{}
			for _, pair := range regexp.MustCompile(`([a-z_]+)="([^"]*)"`).FindAllStringSubmatch(tag, -1) {
				attrs[pair[1]] = pair[2]
			}
			name := attrs["name"]
			if name == "commercial_color_ids" {
				if strings.Contains(tag, " checked") {
					checkedCount++
				}
				if colorCount < 2 || (isNew && colorCount == 2) {
					values.Add(name, attrs["value"])
				}
				colorCount++
			} else if strings.HasPrefix(name, "commercial_color_order_") {
				if attrs["value"] != "5" || attrs["min"] != "0" {
					t.Fatal("missing saved order or numeric bounds")
				}
				values.Set(name, "9")
			}
		}

		if colorCount != 4 || (isNew && checkedCount != 0) || (!isNew && checkedCount != 3) {
			t.Fatalf("new=%v colors=%d checked=%d", isNew, colorCount, checkedCount)
		}
		req := httptest.NewRequest(http.MethodPost, page.Action, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := req.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form := adminProductFormFromRequest(req)
		want := 2
		if isNew {
			want = 3
		}
		if len(form.CommercialColorIDs) != want {
			t.Fatalf("new=%v parsed=%+v", isNew, form)
		}
		for _, id := range form.CommercialColorIDs {
			if form.CommercialColorOrders[id] != "9" {
				t.Fatal("order lost in POST")
			}
		}
	}
}
