package templates

import (
	"context"
	"github.com/Bernardo-Txa/printlab/internal/products"
	"strings"
	"testing"
)

func TestCommercialColorsRendering(t *testing.T) {
	d := products.ProductDetail{Product: products.Product{Name: "Dragao", Slug: "dragao"}, AvailableColors: []products.ProductAvailableColor{{Color: products.Color{Name: "Azul Ceu", Slug: "azul", HexColor: "#0099FF"}}, {Color: products.Color{Name: "Sem hex", Slug: "sem-hex"}}}}
	d.SelectCommercialColor("azul")
	var b strings.Builder
	if err := ProductDetail(d, "Produto").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Escolha a cor", "Cor selecionada:", "Azul Ceu", "#0099FF", `aria-current="true"`, "Sem hex"} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	form := strings.Split(strings.Split(b.String(), `action="/carrinho/adicionar"`)[1], "</form>")[0]
	swatches := strings.Split(strings.Split(b.String(), `aria-label="Cores disponíveis"`)[1], "</nav>")[0]
	if strings.Contains(swatches, ">Azul Ceu<") || strings.Contains(swatches, ">Sem hex<") {
		t.Fatal("color names must not appear as visible swatch labels")
	}
	if !strings.Contains(swatches, `aria-label="Azul Ceu"`) || !strings.Contains(swatches, "commercial-swatch-check") {
		t.Fatal("missing accessible name or selection check")
	}
	if strings.Contains(form, "color") || strings.Contains(form, `name="cor"`) {
		t.Fatal("color leaked into cart submission")
	}
	d.SelectedColor = nil
	b.Reset()
	if err := ProductDetail(d, "Produto").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Selecione uma cor") || strings.Contains(b.String(), "Cor selecionada:") {
		t.Fatal("unselected colors must not show generic instructions or empty labels")
	}
	if !strings.Contains(b.String(), "commercial-swatch-fill") {
		t.Fatal("unselected colors must retain selectable swatches")
	}
	d.AvailableColors = nil
	b.Reset()
	if err := ProductDetail(d, "Produto").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Escolha a cor") {
		t.Fatal("empty selector shown")
	}
}
