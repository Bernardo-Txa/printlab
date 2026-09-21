package email

import (
	"strings"
	"testing"
)

func TestAuthTemplatesRenderBrandingAndLinks(t *testing.T) {
	templates := []struct {
		name    string
		kind    TemplateKind
		render  func(LinkEmailData) (Message, error)
		subject string
		cta     string
	}{
		{"signup", TemplateSignupConfirmation, SignupConfirmation, "Confirme seu cadastro na PrintLab", "Confirmar cadastro"},
		{"magic", TemplateMagicLink, MagicLink, "Seu link de acesso à PrintLab", "Acessar PrintLab"},
		{"recovery", TemplateAccessRecovery, AccessRecovery, "Recupere seu acesso à PrintLab", "Recuperar acesso"},
	}

	for _, tc := range templates {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := tc.render(LinkEmailData{
				SiteURL:       "https://printlab3d.com.br",
				ActionURL:     "https://printlab3d.com.br/auth/callback?code=example",
				RecipientName: "Cliente",
			})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if msg.Kind != tc.kind || msg.Subject != tc.subject {
				t.Fatalf("unexpected metadata: %#v", msg)
			}
			assertContains(t, msg.HTML, "PrintLab")
			assertContains(t, msg.HTML, "#071c36")
			assertContains(t, msg.HTML, tc.cta)
			assertContains(t, msg.HTML, "https://printlab3d.com.br/auth/callback?code=example")
			assertContains(t, msg.Text, tc.cta)
			assertContains(t, msg.Preheader, "PrintLab")
			if msg.SenderName != DefaultSenderName || msg.SenderEmail != DefaultSenderEmail || msg.ReplyToEmail != DefaultReplyTo {
				t.Fatalf("unexpected sender fields: %#v", msg)
			}
			assertNoSecretsOrInternalData(t, msg)
		})
	}
}

func TestOrderConfirmationShipping(t *testing.T) {
	msg, err := OrderConfirmation(baseOrderData("shipping"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.Subject != "Recebemos seu pedido #1001" {
		t.Fatalf("unexpected subject %q", msg.Subject)
	}
	assertContains(t, msg.HTML, "Pedido #1001")
	assertContains(t, msg.HTML, "Dragão articulado")
	assertContains(t, msg.HTML, "Variante: Grande")
	assertContains(t, msg.HTML, "Cor: Azul")
	assertContains(t, msg.HTML, "Quantidade: 2")
	assertContains(t, msg.HTML, "Sedex")
	assertContains(t, msg.HTML, "Correios")
	assertContains(t, msg.HTML, "R$ 120,00")
	assertContains(t, msg.Text, "Total: R$ 138,00")
	assertContains(t, msg.HTML, "https://printlab3d.com.br/pedido/00000000-0000-4000-8000-000000000001")
	assertNoSecretsOrInternalData(t, msg)
}

func TestOrderConfirmationPickupUsesPersistedFreeShipping(t *testing.T) {
	data := baseOrderData("pickup")
	data.ShippingService = ""
	data.ShippingCarrier = ""
	data.DeliveryTime = ""
	data.ShippingPriceBRL = "Grátis"
	data.TotalBRL = "R$ 120,00"

	msg, err := OrderConfirmation(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertContains(t, msg.HTML, "Retirada no local")
	assertContains(t, msg.HTML, "Grátis")
	assertContains(t, msg.Text, "Entrega: Retirada no local")
	assertNotContains(t, msg.HTML, "SuperFrete")
	assertNotContains(t, msg.HTML, "Correios")
	assertNoSecretsOrInternalData(t, msg)
}

func TestStatusAndPaymentTemplates(t *testing.T) {
	status, err := OrderStatusUpdate(baseOrderData("shipping"))
	if err != nil {
		t.Fatalf("status render: %v", err)
	}
	assertContains(t, status.Subject, "Atualização do pedido #1001")
	assertContains(t, status.HTML, "Status: Em produção")
	assertContains(t, status.HTML, "Acompanhar pedido")

	data := baseOrderData("shipping")
	data.StatusLabel = ""
	payment, err := PaymentConfirmed(data)
	if err != nil {
		t.Fatalf("payment render: %v", err)
	}
	assertContains(t, payment.Subject, "Pagamento confirmado do pedido #1001")
	assertContains(t, payment.HTML, "Pagamento confirmado")
	assertNoSecretsOrInternalData(t, payment)
}

func TestTemplatesRejectMissingOrInvalidLinks(t *testing.T) {
	if _, err := MagicLink(LinkEmailData{SiteURL: "https://printlab3d.com.br", ActionURL: ""}); err == nil {
		t.Fatal("expected missing action URL error")
	}
	if _, err := SignupConfirmation(LinkEmailData{SiteURL: "javascript:alert(1)", ActionURL: "https://printlab3d.com.br/auth"}); err == nil {
		t.Fatal("expected invalid site URL error")
	}
	data := baseOrderData("shipping")
	data.Items = nil
	if _, err := OrderConfirmation(data); err == nil {
		t.Fatal("expected missing items error")
	}
}

func baseOrderData(method string) OrderEmailData {
	return OrderEmailData{
		SiteURL:             "https://printlab3d.com.br",
		OrderURL:            "https://printlab3d.com.br/pedido/00000000-0000-4000-8000-000000000001",
		TrackingURL:         "https://printlab3d.com.br/acompanhar/00000000-0000-4000-8000-000000000002",
		OrderNumberLabel:    "#1001",
		CustomerName:        "Cliente",
		StatusLabel:         "Em produção",
		DeliveryMethod:      method,
		ShippingService:     "Sedex",
		ShippingCarrier:     "Correios",
		DeliveryTime:        "3 dias úteis",
		ProductsSubtotalBRL: "R$ 120,00",
		ShippingPriceBRL:    "R$ 18,00",
		TotalBRL:            "R$ 138,00",
		Items: []OrderEmailItem{
			{
				ProductName:  "Dragão articulado",
				VariantName:  "Grande",
				ColorName:    "Azul",
				Quantity:     2,
				UnitPriceBRL: "R$ 60,00",
				LineTotalBRL: "R$ 120,00",
			},
		},
	}
}

func assertContains(t *testing.T, value string, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("expected %q to contain %q", value, want)
	}
}

func assertNotContains(t *testing.T, value string, forbidden string) {
	t.Helper()
	if strings.Contains(value, forbidden) {
		t.Fatalf("expected %q not to contain %q", value, forbidden)
	}
}

func assertNoSecretsOrInternalData(t *testing.T, msg Message) {
	t.Helper()
	combined := msg.Subject + "\n" + msg.Preheader + "\n" + msg.HTML + "\n" + msg.Text
	for _, forbidden := range []string{
		"smtp.mail.me.com",
		"SMTP password",
		"SUPABASE_SECRET_KEY",
		"Authorization:",
		"Bearer ",
		"filamento",
		"tempo de impressão",
		"margem",
		"SKU",
		"shipping_box",
		"transaction_nsu",
		"invoice_slug",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("message contains forbidden content %q", forbidden)
		}
	}
}
