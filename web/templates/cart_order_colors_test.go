package templates

import (
	"context"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/orders"
	"github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/a-h/templ"
)

func TestCommercialColorOnCartAndOrderPages(t *testing.T) {
	for _, color := range []string{"", "Azul comercial"} {
		pages := map[string]templ.Component{
			"cart":   CartPage(cart.CartView{Lines: []cart.CartLine{{ProductName: "Dragao", HasVariant: true, VariantName: "Grande", ColorName: color, Quantity: 1, Available: true}}}),
			"review": CheckoutReview(orders.ReviewPage{Items: []orders.ReviewItem{{ProductName: "Dragao", HasVariant: true, VariantName: "Grande", ColorName: color, Quantity: 1}}}),
			"order":  OrderConfirmation(orders.OrderPage{Items: []orders.OrderItem{{ProductName: "Dragao", HasVariant: true, VariantName: "Grande", ColorName: color, Quantity: 1}}}, payments.OrderPaymentView{}),
			"admin":  AdminOrderDetail(admin.OrderDetail{Items: []admin.OrderDetailItem{{ProductName: "Dragao", HasVariant: true, VariantName: "Grande", ColorName: color, Quantity: 1}}}, "", ""),
		}
		for name, page := range pages {
			t.Run(name+"/"+color, func(t *testing.T) {
				var html strings.Builder
				if err := page.Render(context.Background(), &html); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(html.String(), "Dragao") || !strings.Contains(html.String(), "Grande") {
					t.Fatal("product or variant disappeared")
				}
				if color != "" && !strings.Contains(html.String(), "Cor: Azul comercial") {
					t.Fatal("commercial color missing")
				}
				if color == "" && strings.Contains(html.String(), "Cor:") {
					t.Fatal("empty color label on legacy item")
				}
			})
		}
	}
}
