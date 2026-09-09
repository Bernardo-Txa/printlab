package customers

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositorySaveUsesTransactionAndUpsert(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	saveSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) Save")
	if !ok {
		t.Fatal("expected Save to exist")
	}

	for _, expected := range []string{
		"r.pool.Begin(ctx)",
		"tx.Rollback(ctx)",
		"tx.Commit(ctx)",
		"insert into public.cart_customer_details",
		"insert into public.cart_shipping_addresses",
		"on conflict (cart_id) do update set",
	} {
		if !strings.Contains(saveSource, expected) {
			t.Fatalf("expected Save source to contain %q", expected)
		}
	}

	if strings.Contains(saveSource, "select *") {
		t.Fatal("expected repository not to use SELECT *")
	}
}

func TestPostgresRepositorySaveRollsBackWhenAddressUpsertFails(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected test database pool, got %v", err)
	}
	defer pool.Close()

	tokenHash := sha256.Sum256([]byte(t.Name() + time.Now().String()))
	var cartID string
	if err := pool.QueryRow(ctx, `
		insert into public.carts (
			token_hash,
			expires_at
		) values (
			$1,
			$2
		)
		returning id::text
	`, tokenHash[:], time.Now().Add(time.Hour)).Scan(&cartID); err != nil {
		t.Fatalf("expected cart fixture, got %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from public.carts where id = $1::uuid`, cartID)
	})

	repository := NewPostgresRepository(pool)
	err = repository.Save(ctx, cartID, CheckoutDetails{
		Customer: CustomerDetails{
			FullName: "Joao Silva",
			Email:    "joao@example.com",
			Phone:    "+5527999999999",
			CPF:      "52998224725",
		},
		Address: ShippingAddress{
			PostalCode:  "29100000",
			Street:      "Rua Um",
			Number:      "12A",
			District:    "Centro",
			City:        "Vila Velha",
			State:       "ZZ",
			CountryCode: CountryCodeBR,
		},
	})
	if err == nil {
		t.Fatal("expected invalid address to fail")
	}

	for table, want := range map[string]int{
		"public.cart_customer_details":   0,
		"public.cart_shipping_addresses": 0,
	} {
		var count int
		if err := pool.QueryRow(ctx, `select count(*) from `+table+` where cart_id = $1::uuid`, cartID).Scan(&count); err != nil {
			t.Fatalf("expected count for %s, got %v", table, err)
		}
		if count != want {
			t.Fatalf("expected %s count %d after rollback, got %d", table, want, count)
		}
	}
}

func repositoryFunctionSource(source string, marker string) (string, bool) {
	start := strings.Index(source, marker)
	if start == -1 {
		return "", false
	}

	functionSource := source[start:]
	nextFunction := strings.Index(functionSource[len(marker):], "\nfunc ")
	if nextFunction == -1 {
		return functionSource, true
	}

	return functionSource[:len(marker)+nextFunction], true
}
