package components

import (
	"net/url"
	"testing"
)

func TestWhatsAppURLUsesOfficialContactAndEscapedMessage(t *testing.T) {
	rawURL := WhatsAppURL(WhatsAppGeneralMessage)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("expected valid WhatsApp URL, got %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "wa.me" || parsed.Path != "/5527998595125" {
		t.Fatalf("expected official WhatsApp endpoint, got %q", rawURL)
	}
	if got := parsed.Query().Get("text"); got != WhatsAppGeneralMessage {
		t.Fatalf("expected escaped general message round trip, got %q", got)
	}
	if WhatsAppPhoneDisplay != "+55 27 99859-5125" {
		t.Fatalf("expected official phone display, got %q", WhatsAppPhoneDisplay)
	}
}

func TestWhatsAppURLFallsBackToGeneralMessage(t *testing.T) {
	parsed, err := url.Parse(WhatsAppURL("  "))
	if err != nil {
		t.Fatalf("expected valid fallback URL, got %v", err)
	}
	if got := parsed.Query().Get("text"); got != WhatsAppGeneralMessage {
		t.Fatalf("expected general fallback message, got %q", got)
	}
}
