package payments

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestAddOrderPaymentsMigrationDocumentsPaymentSchema(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_add_order_payments.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one add_order_payments migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"drop constraint orders_status_allowed",
		"status in ('pending_payment', 'paid')",
		"create table public.order_payments",
		"order_id uuid primary key",
		"provider text not null default 'infinitepay'",
		"status text not null default 'pending'",
		"order_nsu text not null unique",
		"checkout_url text null",
		"invoice_slug text null",
		"transaction_nsu text null",
		"amount_cents bigint null",
		"paid_amount_cents bigint null",
		"installments integer null",
		"capture_method text null",
		"references public.orders (id)",
		"on delete cascade",
		"provider = 'infinitepay'",
		"status in ('pending', 'paid')",
		"amount_cents >= 0",
		"paid_amount_cents >= 0",
		"installments > 0",
		"order_payments_transaction_nsu_unique_idx",
		"where transaction_nsu is not null",
		"alter table public.order_payments enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"create policy",
		"insert into",
		"webhook_url",
		"database_url",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected migration not to contain %q", forbidden)
		}
	}
}

func TestAllowCurrentInfinitePayCheckoutHostMigrationUpdatesConstraint(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_allow_current_infinitepay_checkout_host.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one allow_current_infinitepay_checkout_host migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"alter table public.order_payments",
		"drop constraint order_payments_checkout_url_host",
		"add constraint order_payments_checkout_url_host check",
		"checkout_url is null",
		"^https://checkout[.]infinitepay[.]com[.]br/.+",
		"^https://checkout[.]infinitepay[.]io/.+",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"20260911193009_add_order_payments",
		"http://",
		"evil.infinitepay.io",
		"%.infinitepay.io",
		"infinitepay.io%",
		"insert into",
		"create policy",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected migration not to contain %q", forbidden)
		}
	}

	patterns := checkoutURLConstraintPatterns(t, string(source))
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		{name: "current io host", url: "https://checkout.infinitepay.io/teste", allowed: true},
		{name: "historical br host", url: "https://checkout.infinitepay.com.br/teste", allowed: true},
		{name: "http rejected", url: "http://checkout.infinitepay.io/teste"},
		{name: "evil subdomain rejected", url: "https://evil.infinitepay.io/teste"},
		{name: "suffix spoof rejected", url: "https://checkout.infinitepay.io.evil.com/teste"},
		{name: "api checkout rejected", url: "https://api.checkout.infinitepay.io/teste"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchesAny := false
			for _, pattern := range patterns {
				if pattern.MatchString(tt.url) {
					matchesAny = true
					break
				}
			}
			if matchesAny != tt.allowed {
				t.Fatalf("expected allowed=%v for %q, got %v", tt.allowed, tt.url, matchesAny)
			}
		})
	}
}

func checkoutURLConstraintPatterns(t *testing.T, source string) []*regexp.Regexp {
	t.Helper()

	literalPattern := regexp.MustCompile(`checkout_url ~ '([^']+)'`)
	matches := literalPattern.FindAllStringSubmatch(source, -1)
	if len(matches) != 2 {
		t.Fatalf("expected two checkout_url regex patterns, got %d", len(matches))
	}

	patterns := make([]*regexp.Regexp, 0, len(matches))
	for _, match := range matches {
		pattern, err := regexp.Compile(match[1])
		if err != nil {
			t.Fatalf("expected checkout_url regex %q to compile, got %v", match[1], err)
		}
		patterns = append(patterns, pattern)
	}

	return patterns
}
