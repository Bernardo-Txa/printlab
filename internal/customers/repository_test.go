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

func TestRepositoryGetUsesSingleConsistentJoin(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	getSource, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) Get")
	if !ok {
		t.Fatal("expected Get to exist")
	}

	for _, expected := range []string{
		"from public.cart_customer_details as customer",
		"join public.cart_shipping_addresses as address",
		"on address.cart_id = customer.cart_id",
	} {
		if !strings.Contains(getSource, expected) {
			t.Fatalf("expected Get source to contain %q", expected)
		}
	}

	if strings.Count(strings.ToLower(getSource), "queryrow") != 1 {
		t.Fatal("expected Get to use a single QueryRow statement")
	}
	if strings.Contains(getSource, "select *") {
		t.Fatal("expected repository not to use SELECT *")
	}
}

func TestPostgresRepositoryGetReadsDetailsAndIgnoresPartialState(t *testing.T) {
	ctx := context.Background()
	pool := testCustomerPool(t, ctx)
	repository := NewPostgresRepository(pool)

	completeCartID := createCustomerTestCart(t, ctx, pool, "complete")
	want := repositoryCheckoutDetailsFixture(t)
	if err := repository.Save(ctx, completeCartID, want); err != nil {
		t.Fatalf("expected complete details to be saved, got %v", err)
	}

	got, found, err := repository.Get(ctx, completeCartID)
	if err != nil {
		t.Fatalf("expected complete details to be read, got %v", err)
	}
	if !found {
		t.Fatal("expected complete details to be found")
	}
	if got.Customer.CPF != want.Customer.CPF || got.Address.PostalCode != want.Address.PostalCode || got.Address.Complement == nil || *got.Address.Complement != *want.Address.Complement {
		t.Fatalf("expected complete checkout details, got %#v", got)
	}

	emptyCartID := createCustomerTestCart(t, ctx, pool, "empty")
	_, found, err = repository.Get(ctx, emptyCartID)
	if err != nil {
		t.Fatalf("expected empty checkout details read to be safe, got %v", err)
	}
	if found {
		t.Fatal("expected absent checkout details not to be found")
	}

	customerOnlyCartID := createCustomerTestCart(t, ctx, pool, "customer-only")
	insertCustomerDetailsFixture(t, ctx, pool, customerOnlyCartID, want)
	_, found, err = repository.Get(ctx, customerOnlyCartID)
	if err != nil {
		t.Fatalf("expected customer-only state to be handled safely, got %v", err)
	}
	if found {
		t.Fatal("expected customer-only partial state not to be returned")
	}

	addressOnlyCartID := createCustomerTestCart(t, ctx, pool, "address-only")
	insertShippingAddressFixture(t, ctx, pool, addressOnlyCartID, want)
	_, found, err = repository.Get(ctx, addressOnlyCartID)
	if err != nil {
		t.Fatalf("expected address-only state to be handled safely, got %v", err)
	}
	if found {
		t.Fatal("expected address-only partial state not to be returned")
	}
}

func TestPostgresRepositorySaveRollsBackWhenAddressUpsertFails(t *testing.T) {
	ctx := context.Background()
	pool := testCustomerPool(t, ctx)

	cartID := createCustomerTestCart(t, ctx, pool, "rollback")
	repository := NewPostgresRepository(pool)
	err := repository.Save(ctx, cartID, CheckoutDetails{
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

func testCustomerPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected test database pool, got %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func createCustomerTestCart(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix string) string {
	t.Helper()

	tokenHash := sha256.Sum256([]byte(t.Name() + suffix + time.Now().String()))
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

	return cartID
}

func repositoryCheckoutDetailsFixture(t *testing.T) CheckoutDetails {
	t.Helper()

	details, _, fieldErrors := NormalizeCheckoutInput(validCheckoutInput(func(input *CheckoutInput) {
		input.Complement = "Apto 302"
	}))
	if fieldErrors.Any() {
		t.Fatalf("expected valid repository checkout fixture, got %#v", fieldErrors)
	}

	return details
}

func insertCustomerDetailsFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cartID string, details CheckoutDetails) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		insert into public.cart_customer_details (
			cart_id,
			full_name,
			email,
			phone,
			cpf
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5
		)
	`, cartID, details.Customer.FullName, details.Customer.Email, details.Customer.Phone, details.Customer.CPF); err != nil {
		t.Fatalf("expected customer details fixture, got %v", err)
	}
}

func insertShippingAddressFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cartID string, details CheckoutDetails) {
	t.Helper()

	complement := any(nil)
	if details.Address.Complement != nil {
		complement = *details.Address.Complement
	}

	if _, err := pool.Exec(ctx, `
		insert into public.cart_shipping_addresses (
			cart_id,
			postal_code,
			street,
			number,
			complement,
			district,
			city,
			state,
			country_code
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
	`, cartID, details.Address.PostalCode, details.Address.Street, details.Address.Number, complement, details.Address.District, details.Address.City, details.Address.State, details.Address.CountryCode); err != nil {
		t.Fatalf("expected shipping address fixture, got %v", err)
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
