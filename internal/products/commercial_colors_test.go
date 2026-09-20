package products

import "testing"

func TestCommercialColorPageState(t *testing.T) {
	d := ProductDetail{Product: Product{Slug: "dragao"}, AvailableColors: []ProductAvailableColor{{Color: Color{Slug: "azul", HexColor: "#0099FF"}}}, SelectedVariant: &ProductVariant{Slug: "grande"}}
	d.SelectCommercialColor("azul")
	if d.SelectedColor == nil || d.ColorURL("azul") != "/produtos/dragao?cor=azul&variante=grande" || d.VariantURL("pequeno") != "/produtos/dragao?cor=azul&variante=pequeno" {
		t.Fatalf("selection not preserved: %+v", d)
	}
	if d.SelectedColor.SwatchStyle() != "background-color:#0099FF" {
		t.Fatal("hex missing")
	}
	d.SelectCommercialColor("inexistente")
	if d.SelectedColor != nil || d.SelectedVariant.Slug != "grande" {
		t.Fatal("stale color changed variant")
	}
	d.AvailableColors = nil
	d.SelectCommercialColor("azul")
	if d.SelectedColor != nil {
		t.Fatal("unavailable color selected")
	}
	for _, hex := range []string{"", "red;background:url(x)", "#fff"} {
		if (ProductAvailableColor{Color: Color{HexColor: hex}}).SwatchStyle() != "" {
			t.Fatal("unsafe hex accepted")
		}
	}
}
