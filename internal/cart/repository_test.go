package cart

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryItemMutationsAreScopedByCartID(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	updateQuery, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) UpdateItemQuantity")
	if !ok {
		t.Fatal("expected UpdateItemQuantity to exist")
	}
	if !strings.Contains(updateQuery, "where cart_id = $1::uuid") || !strings.Contains(updateQuery, "and id = $2::uuid") {
		t.Fatal("expected update query to scope item mutation by cart_id and item_id")
	}

	removeQuery, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) RemoveItem")
	if !ok {
		t.Fatal("expected RemoveItem to exist")
	}
	if !strings.Contains(removeQuery, "where cart_id = $1::uuid") || !strings.Contains(removeQuery, "and id = $2::uuid") {
		t.Fatal("expected remove query to scope item mutation by cart_id and item_id")
	}
}

func TestRepositoryAddItemUsesAtomicUpsertWithQuantityLimit(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	for _, marker := range []string{
		"func (r *PostgresRepository) addItemWithoutVariant",
		"func (r *PostgresRepository) addItemWithVariant",
	} {
		query, ok := repositoryFunctionSource(string(source), marker)
		if !ok {
			t.Fatalf("expected %s to exist", marker)
		}
		if !strings.Contains(query, "on conflict") || !strings.Contains(query, "public.cart_items.quantity + excluded.quantity") {
			t.Fatalf("expected %s to increment atomically in SQL", marker)
		}
		if !strings.Contains(query, "where public.cart_items.quantity <= 99 - excluded.quantity") {
			t.Fatalf("expected %s to enforce max quantity in SQL", marker)
		}
	}
}

func TestRepositoryIgnoresConvertedCarts(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	for _, marker := range []string{
		"func (r *PostgresRepository) FindActiveCart",
		"func (r *PostgresRepository) CreateCart",
		"func (r *PostgresRepository) RenewCart",
		"func (r *PostgresRepository) addItemWithoutVariant",
		"func (r *PostgresRepository) addItemWithVariant",
		"func (r *PostgresRepository) UpdateItemQuantity",
		"func (r *PostgresRepository) RemoveItem",
	} {
		query, ok := repositoryFunctionSource(string(source), marker)
		if !ok {
			t.Fatalf("expected %s to exist", marker)
		}
		if !strings.Contains(query, "converted_at is null") {
			t.Fatalf("expected %s to ignore converted carts", marker)
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
