package templates

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/orders"
	"github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/a-h/templ"
)

func renderTemplate(t *testing.T, component templ.Component) string {
	t.Helper()
	var html strings.Builder
	if err := component.Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	return html.String()
}

func TestCheckoutCSSRemovesDuplicateShippingIndicatorAndScopesStatusBadge(t *testing.T) {
	source, err := os.ReadFile("../assets/css/app.css")
	if err != nil {
		t.Fatalf("expected CSS source to be readable, got %v", err)
	}
	css := string(source)
	if strings.Contains(css, ".shipping-option:has(input:checked)::after") {
		t.Fatal("expected duplicate teal shipping indicator pseudo-element to be removed")
	}
	if !strings.Contains(css, ".order-status-panel .order-status-badge") || !strings.Contains(css, "width: fit-content") {
		t.Fatal("expected status badge styles to be scoped under order status panel")
	}
}

func TestCheckoutStepperStructureAndAccessibleState(t *testing.T) {
	html := renderTemplate(t, CheckoutStepper(2, false))
	for _, expected := range []string{`class="checkout-step-marker"`, `class="checkout-step-label"`, `class="sr-only"`, `aria-current="step"`} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected stepper to contain %q", expected)
		}
	}
	if strings.Contains(html, `checkout-step-state`) || strings.Contains(html, `checkout-step-copy`) {
		t.Fatal("expected checkout state text to remain sr-only without visual state wrappers")
	}
}

func TestCheckoutDetailsCPFHelpIsNeutral(t *testing.T) {
	html := renderTemplate(t, CheckoutDetails(customers.CheckoutPage{Cart: cart.CartView{SubtotalBRL: "R$ 0,00"}}))
	if strings.Contains(html, "documentação do envio") {
		t.Fatal("expected CPF help not to mention shipping documentation")
	}
	if !strings.Contains(html, "Usado para identificação do pedido.") {
		t.Fatal("expected neutral CPF help copy")
	}
}

func TestCheckoutDecorativeSVGsHaveExplicitSize(t *testing.T) {
	html := renderTemplate(t, CheckoutShipping(shipping.CheckoutShippingPage{SelectedMethod: shipping.DeliveryMethodShipping, AddressForm: customers.CheckoutForm{Values: customers.CheckoutInput{CountryCode: customers.CountryCodeBR}}}))
	if !strings.Contains(html, `class="checkout-ui-icon" width="20" height="20"`) {
		t.Fatal("expected checkout SVG icons to keep explicit dimensions")
	}
}

func TestCheckoutReviewPickupDoesNotRenderAddress(t *testing.T) {
	html := renderTemplate(t, CheckoutReview(orders.ReviewPage{
		Items:      []orders.ReviewItem{{ProductName: "Produto", Quantity: 1}},
		Address:    orders.ReviewAddress{LineOne: "Avenida Paulista, 1000", LineTwo: "Centro - Sao Paulo/SP"},
		HasAddress: true,
		Shipping:   orders.ReviewShipping{DeliveryMethod: shipping.DeliveryMethodPickup, PriceBRL: "R$ 0,00"},
	}))
	if strings.Contains(html, "Endereço de entrega") || strings.Contains(html, "Avenida Paulista") || strings.Contains(html, "Editar endereço") {
		t.Fatal("expected pickup review to suppress stale address data")
	}
}

func TestCheckoutReviewShippingRendersAddressEditLink(t *testing.T) {
	html := renderTemplate(t, CheckoutReview(orders.ReviewPage{
		Items:      []orders.ReviewItem{{ProductName: "Produto", Quantity: 1}},
		Address:    orders.ReviewAddress{LineOne: "Rua Um, 12", LineTwo: "Centro - Vila Velha/ES", CountryCode: "BR"},
		HasAddress: true,
		Shipping:   orders.ReviewShipping{DeliveryMethod: shipping.DeliveryMethodShipping, ServiceName: "PAC", PriceBRL: "R$ 18,90"},
	}))
	for _, expected := range []string{"Endereço de entrega", "Editar endereço", `href="/checkout/frete?delivery_method=shipping#shipping-address"`} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected shipping review to contain %q", expected)
		}
	}
}

func TestPaymentCardKeepsSmallIconAndStatusBadge(t *testing.T) {
	html := renderTemplate(t, OrderConfirmation(orders.OrderPage{ID: "order-1", OrderNumberLabel: "#1001", StatusLabel: "Aguardando pagamento"}, payments.OrderPaymentView{CanPay: true}))
	for _, expected := range []string{`class="payment-card-icon"`, `class="checkout-ui-icon" width="20" height="20"`, `order-status-badge`, "Aguardando pagamento"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected payment page to contain %q", expected)
		}
	}
}

func TestAdminPickupDetailDoesNotRenderEmptyAddress(t *testing.T) {
	html := renderTemplate(t, AdminOrderDetail(admin.OrderDetail{Shipping: admin.OrderShipping{DeliveryMethod: shipping.DeliveryMethodPickup, PriceBRL: "R$ 0,00"}}, "", ""))
	for _, forbidden := range []string{"Endereço</span>", "Cidade</span>", "País</span>"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("expected pickup admin detail not to render empty address field %q", forbidden)
		}
	}
	if !strings.Contains(html, "Retirada no local") {
		t.Fatal("expected pickup admin detail to render pickup delivery")
	}
}
