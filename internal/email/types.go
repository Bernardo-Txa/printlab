package email

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultSenderName  = "PrintLab"
	DefaultSenderEmail = "acesso@printlab3d.com.br"
	DefaultReplyTo     = "acesso@printlab3d.com.br"
)

type TemplateKind string

const (
	TemplateSignupConfirmation TemplateKind = "signup_confirmation"
	TemplateMagicLink          TemplateKind = "magic_link"
	TemplateAccessRecovery     TemplateKind = "access_recovery"
	TemplateOrderConfirmation  TemplateKind = "order_confirmation"
	TemplateOrderStatusUpdate  TemplateKind = "order_status_update"
	TemplatePaymentConfirmed   TemplateKind = "payment_confirmed"
)

type Message struct {
	Kind         TemplateKind
	Subject      string
	Preheader    string
	HTML         string
	Text         string
	SenderName   string
	SenderEmail  string
	ReplyToEmail string
}

type LinkEmailData struct {
	SiteURL       string
	ActionURL     string
	RecipientName string
}

type OrderEmailData struct {
	SiteURL             string
	OrderURL            string
	TrackingURL         string
	OrderNumberLabel    string
	CustomerName        string
	StatusLabel         string
	Items               []OrderEmailItem
	ProductsSubtotalBRL string
	ShippingPriceBRL    string
	TotalBRL            string
	DeliveryMethod      string
	ShippingService     string
	ShippingCarrier     string
	DeliveryTime        string
}

type OrderEmailItem struct {
	ProductName  string
	VariantName  string
	ColorName    string
	Quantity     int
	UnitPriceBRL string
	LineTotalBRL string
}

func (d LinkEmailData) validate() error {
	if !validHTTPURL(d.SiteURL) {
		return fmt.Errorf("email site url required")
	}
	if !validHTTPURL(d.ActionURL) {
		return fmt.Errorf("email action url required")
	}
	return nil
}

func (d OrderEmailData) validate(requireStatus bool) error {
	if !validHTTPURL(d.SiteURL) {
		return fmt.Errorf("email site url required")
	}
	if d.OrderNumberLabel == "" {
		return fmt.Errorf("email order number required")
	}
	if requireStatus && d.StatusLabel == "" {
		return fmt.Errorf("email status label required")
	}
	if len(d.Items) == 0 {
		return fmt.Errorf("email order items required")
	}
	if d.ProductsSubtotalBRL == "" || d.ShippingPriceBRL == "" || d.TotalBRL == "" {
		return fmt.Errorf("email order totals required")
	}
	if d.OrderURL != "" && !validHTTPURL(d.OrderURL) {
		return fmt.Errorf("email order url invalid")
	}
	if d.TrackingURL != "" && !validHTTPURL(d.TrackingURL) {
		return fmt.Errorf("email tracking url invalid")
	}
	return nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

func defaultMessage(kind TemplateKind, subject string, preheader string, html string, text string) Message {
	return Message{
		Kind:         kind,
		Subject:      subject,
		Preheader:    preheader,
		HTML:         html,
		Text:         text,
		SenderName:   DefaultSenderName,
		SenderEmail:  DefaultSenderEmail,
		ReplyToEmail: DefaultReplyTo,
	}
}
