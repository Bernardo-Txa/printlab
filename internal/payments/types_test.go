package payments

import (
	"errors"
	"testing"
)

func TestValidateCheckoutURLUsesExplicitHostAllowlist(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "current io host", rawURL: "https://checkout.infinitepay.io/algum-path"},
		{name: "historical br host", rawURL: "https://checkout.infinitepay.com.br/algum-path"},
		{name: "uppercase host", rawURL: "https://CHECKOUT.INFINITEPAY.IO/algum-path"},
		{name: "http is rejected", rawURL: "http://checkout.infinitepay.io/algum-path", wantErr: true},
		{name: "evil subdomain is rejected", rawURL: "https://evil.infinitepay.io/algum-path", wantErr: true},
		{name: "suffix spoof is rejected", rawURL: "https://checkout.infinitepay.io.evil.com/algum-path", wantErr: true},
		{name: "registrable domain is rejected", rawURL: "https://infinitepay.io/algum-path", wantErr: true},
		{name: "api checkout host is rejected", rawURL: "https://api.checkout.infinitepay.io/algum-path", wantErr: true},
		{name: "explicit port is rejected", rawURL: "https://checkout.infinitepay.io:443/algum-path", wantErr: true},
		{name: "unrelated host is rejected", rawURL: "https://example.com/algum-path", wantErr: true},
		{name: "root path is rejected", rawURL: "https://checkout.infinitepay.io/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCheckoutURL(tt.rawURL)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidCheckoutURL) {
					t.Fatalf("expected ErrInvalidCheckoutURL, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected valid checkout URL, got %v", err)
			}
		})
	}
}
