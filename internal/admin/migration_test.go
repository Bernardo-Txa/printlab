package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAdminSessionsMigrationDocumentsSessionSchema(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_create_admin_sessions.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one create_admin_sessions migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"create table public.admin_sessions",
		"id uuid primary key default gen_random_uuid()",
		"auth_user_id uuid not null",
		"token_hash bytea not null unique",
		"created_at timestamptz not null default now()",
		"expires_at timestamptz not null",
		"octet_length(token_hash) = 32",
		"expires_at > created_at",
		"admin_sessions_expires_at_idx",
		"alter table public.admin_sessions enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected admin sessions migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"references auth.users",
		"create policy",
		"insert into",
		"password",
		"access_token",
		"refresh_token",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected admin sessions migration not to contain %q", forbidden)
		}
	}
}

func TestCreateAdminOrderEventsMigrationDocumentsAuditSchema(t *testing.T) {
	matches, err := filepath.Glob("../../supabase/migrations/*_create_admin_order_events.sql")
	if err != nil {
		t.Fatalf("expected migration glob to work, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one create_admin_order_events migration, got %v", matches)
	}

	source, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("expected migration to be readable, got %v", err)
	}

	sql := strings.ToLower(string(source))
	for _, expected := range []string{
		"create table public.admin_order_events",
		"id uuid primary key default gen_random_uuid()",
		"order_id uuid not null",
		"actor_auth_user_id uuid not null",
		"event_type text not null",
		"from_status text not null",
		"to_status text not null",
		"created_at timestamptz not null default now()",
		"references public.orders (id)",
		"on delete restrict",
		"event_type in ('production_status_changed', 'shipping_status_changed')",
		"admin_order_events_order_created_at_idx",
		"alter table public.admin_order_events enable row level security",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("expected admin order events migration to contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		"references auth.users",
		"create policy",
		"insert into",
		"password",
		"checkout_url",
		"transaction_nsu",
		"invoice_slug",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected admin order events migration not to contain %q", forbidden)
		}
	}
}
