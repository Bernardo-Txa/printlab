package products

import (
	"net/url"
	"regexp"
)

var commercialHex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// SelectCommercialColor only updates page state, independently of variants.
// Unknown or stale choices are ignored so old product links remain usable.
func (d *ProductDetail) SelectCommercialColor(slug string) {
	d.SelectedColor = nil
	for i := range d.AvailableColors {
		if d.AvailableColors[i].Slug == slug {
			d.SelectedColor = &d.AvailableColors[i]
			return
		}
	}
}

func (d ProductDetail) SelectionURL(variantSlug, colorSlug string) string {
	q := url.Values{}
	if variantSlug != "" {
		q.Set("variante", variantSlug)
	}
	if colorSlug != "" {
		q.Set("cor", colorSlug)
	}
	path := "/produtos/" + d.Product.Slug
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return path
}

func (d ProductDetail) ColorURL(slug string) string {
	variant := ""
	if d.SelectedVariant != nil {
		variant = d.SelectedVariant.Slug
	}
	return d.SelectionURL(variant, slug)
}

func (d ProductDetail) VariantURL(slug string) string {
	color := ""
	if d.SelectedColor != nil {
		color = d.SelectedColor.Slug
	}
	return d.SelectionURL(slug, color)
}

func (c ProductAvailableColor) SwatchStyle() string {
	if commercialHex.MatchString(c.HexColor) {
		return "background-color:" + c.HexColor
	}
	return ""
}
