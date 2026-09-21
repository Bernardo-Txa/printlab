package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	texttemplate "text/template"
)

type pageData struct {
	SiteURL       string
	Preheader     string
	Title         string
	Intro         string
	CTAURL        string
	CTALabel      string
	FallbackLabel string
	FooterNote    string
	Order         *OrderEmailData
}

const baseHTMLTemplate = `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
</head>
<body style="margin:0;background:#f5f8fc;color:#071c36;font-family:Arial,Helvetica,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">{{.Preheader}}</div>
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f8fc;margin:0;padding:24px 0;">
    <tr>
      <td align="center" style="padding:0 16px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:640px;background:#ffffff;border:1px solid #d6e4ef;border-radius:16px;overflow:hidden;">
          <tr>
            <td style="background:#071c36;padding:28px 28px 24px;">
              <p style="margin:0;color:#10c5b7;font-size:12px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;">PrintLab</p>
              <h1 style="margin:12px 0 0;color:#ffffff;font-size:30px;line-height:1.15;font-weight:700;">{{.Title}}</h1>
            </td>
          </tr>
          <tr>
            <td style="padding:28px;">
              <p style="margin:0 0 20px;color:#33465d;font-size:16px;line-height:1.6;">{{.Intro}}</p>
              {{if .Order}}{{template "order" .Order}}{{end}}
              {{if .CTAURL}}
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 12px;">
                <tr>
                  <td style="border-radius:8px;background:#006de0;">
                    <a href="{{.CTAURL}}" style="display:inline-block;padding:13px 18px;color:#ffffff;font-size:15px;font-weight:700;text-decoration:none;">{{.CTALabel}}</a>
                  </td>
                </tr>
              </table>
              <p style="margin:12px 0 0;color:#5b6c7e;font-size:13px;line-height:1.6;">{{.FallbackLabel}}:<br><a href="{{.CTAURL}}" style="color:#006de0;word-break:break-all;">{{.CTAURL}}</a></p>
              {{end}}
            </td>
          </tr>
          <tr>
            <td style="border-top:1px solid #d6e4ef;background:#eef6fb;padding:20px 28px;">
              <p style="margin:0;color:#5b6c7e;font-size:13px;line-height:1.6;">{{.FooterNote}}</p>
              <p style="margin:10px 0 0;color:#5b6c7e;font-size:12px;line-height:1.5;">PrintLab · E-mail transacional, sem conteúdo promocional.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
{{define "order"}}
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #d6e4ef;border-radius:12px;margin:18px 0 0;">
  <tr><td style="padding:16px;background:#eef6fb;border-bottom:1px solid #d6e4ef;"><strong style="font-size:16px;color:#071c36;">Pedido {{.OrderNumberLabel}}</strong>{{if .StatusLabel}}<br><span style="font-size:13px;color:#5b6c7e;">Status: {{.StatusLabel}}</span>{{end}}</td></tr>
  {{range .Items}}
  <tr>
    <td style="padding:14px 16px;border-bottom:1px solid #d6e4ef;">
      <p style="margin:0;color:#071c36;font-size:15px;font-weight:700;">{{.ProductName}}</p>
      {{if .VariantName}}<p style="margin:4px 0 0;color:#5b6c7e;font-size:13px;">Variante: {{.VariantName}}</p>{{end}}
      {{if .ColorName}}<p style="margin:4px 0 0;color:#5b6c7e;font-size:13px;">Cor: {{.ColorName}}</p>{{end}}
      <p style="margin:8px 0 0;color:#33465d;font-size:14px;">Quantidade: {{.Quantity}} · Unitário: {{.UnitPriceBRL}} · Subtotal: {{.LineTotalBRL}}</p>
    </td>
  </tr>
  {{end}}
  <tr><td style="padding:14px 16px;border-bottom:1px solid #d6e4ef;color:#33465d;font-size:14px;line-height:1.6;">Entrega: {{deliveryLabel .}}{{if .ShippingCarrier}}<br>Transportadora: {{.ShippingCarrier}}{{end}}{{if .ShippingService}}<br>Serviço: {{.ShippingService}}{{end}}{{if .DeliveryTime}}<br>Prazo: {{.DeliveryTime}}{{end}}</td></tr>
  <tr><td style="padding:16px;">
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
      <tr><td style="padding:4px 0;color:#5b6c7e;font-size:14px;">Produtos</td><td align="right" style="padding:4px 0;color:#071c36;font-size:14px;font-weight:700;">{{.ProductsSubtotalBRL}}</td></tr>
      <tr><td style="padding:4px 0;color:#5b6c7e;font-size:14px;">Frete</td><td align="right" style="padding:4px 0;color:#071c36;font-size:14px;font-weight:700;">{{.ShippingPriceBRL}}</td></tr>
      <tr><td style="padding:10px 0 0;color:#071c36;font-size:16px;font-weight:700;">Total</td><td align="right" style="padding:10px 0 0;color:#006de0;font-size:18px;font-weight:700;">{{.TotalBRL}}</td></tr>
    </table>
  </td></tr>
</table>
{{end}}`

const baseTextTemplate = `{{.Title}}

{{.Intro}}
{{if .Order}}
Pedido {{.Order.OrderNumberLabel}}{{if .Order.StatusLabel}}
Status: {{.Order.StatusLabel}}{{end}}

Itens:
{{range .Order.Items}}- {{.ProductName}}{{if .VariantName}} | Variante: {{.VariantName}}{{end}}{{if .ColorName}} | Cor: {{.ColorName}}{{end}} | Quantidade: {{.Quantity}} | Unitário: {{.UnitPriceBRL}} | Subtotal: {{.LineTotalBRL}}
{{end}}
Entrega: {{deliveryLabel .Order}}{{if .Order.ShippingCarrier}}
Transportadora: {{.Order.ShippingCarrier}}{{end}}{{if .Order.ShippingService}}
Serviço: {{.Order.ShippingService}}{{end}}{{if .Order.DeliveryTime}}
Prazo: {{.Order.DeliveryTime}}{{end}}
Produtos: {{.Order.ProductsSubtotalBRL}}
Frete: {{.Order.ShippingPriceBRL}}
Total: {{.Order.TotalBRL}}
{{end}}{{if .CTAURL}}
{{.CTALabel}}: {{.CTAURL}}
{{.FallbackLabel}}: {{.CTAURL}}
{{end}}
{{.FooterNote}}

PrintLab - E-mail transacional, sem conteúdo promocional.
`

var htmlTemplate = template.Must(template.New("email-html").Funcs(template.FuncMap{"deliveryLabel": deliveryLabel}).Parse(baseHTMLTemplate))
var textTemplate = texttemplate.Must(texttemplate.New("email-text").Funcs(texttemplate.FuncMap{"deliveryLabel": deliveryLabel}).Parse(baseTextTemplate))

func render(kind TemplateKind, subject string, data pageData) (Message, error) {
	var html bytes.Buffer
	if err := htmlTemplate.Execute(&html, data); err != nil {
		return Message{}, fmt.Errorf("render email html: %w", err)
	}
	var text bytes.Buffer
	if err := textTemplate.Execute(&text, data); err != nil {
		return Message{}, fmt.Errorf("render email text: %w", err)
	}
	return defaultMessage(kind, subject, data.Preheader, html.String(), strings.TrimSpace(text.String())+"\n"), nil
}

func deliveryLabel(order OrderEmailData) string {
	if order.DeliveryMethod == "pickup" {
		return "Retirada no local"
	}
	if order.ShippingService != "" {
		return order.ShippingService
	}
	return "Receber em casa"
}
