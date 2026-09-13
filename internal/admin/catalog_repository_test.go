package admin

import (
	"os"
	"strings"
	"testing"
)

func TestAdminCatalogRepositoryUsesTransactionsForDefaultVariant(t *testing.T) {
	source := readAdminCatalogRepository(t)
	for _, marker := range []string{
		"func (r *PostgresRepository) CreateAdminVariant",
		"func (r *PostgresRepository) UpdateAdminVariant",
	} {
		body, ok := adminFunctionSource(source, marker)
		if !ok {
			t.Fatalf("expected %s", marker)
		}
		for _, expected := range []string{
			"r.pool.Begin(ctx)",
			"update public.product_variants",
			"is_default = false",
			"tx.Commit(ctx)",
		} {
			if !strings.Contains(body, expected) {
				t.Fatalf("expected %s to contain %q", marker, expected)
			}
		}
	}
}

func TestAdminCatalogRepositoryScopesChildMutations(t *testing.T) {
	source := readAdminCatalogRepository(t)
	for _, marker := range []string{
		"func (r *PostgresRepository) UpdateAdminVariant",
		"func (r *PostgresRepository) AddAdminRecipeComponent",
		"func (r *PostgresRepository) UpdateAdminRecipeComponent",
		"func (r *PostgresRepository) RemoveAdminRecipeComponent",
	} {
		body, ok := adminFunctionSource(source, marker)
		if !ok {
			t.Fatalf("expected %s", marker)
		}
		if !strings.Contains(body, "product_id") {
			t.Fatalf("expected %s to scope by product_id", marker)
		}
	}
}

func TestAdminCatalogRepositoryDoesNotUseOrderAuditForCatalog(t *testing.T) {
	source := readAdminCatalogRepository(t)
	for _, forbidden := range []string{"admin_order_events", "admin_catalog_events", "select *"} {
		if strings.Contains(strings.ToLower(source), forbidden) {
			t.Fatalf("catalog repository must not contain %q", forbidden)
		}
	}
}

func TestAdminCatalogRepositoryUpdatesUpdatedAt(t *testing.T) {
	source := readAdminCatalogRepository(t)
	for _, marker := range []string{
		"func (r *PostgresRepository) UpdateAdminProduct",
		"func (r *PostgresRepository) UpdateAdminCategory",
		"func (r *PostgresRepository) UpdateAdminVariant",
		"func (r *PostgresRepository) UpdateAdminMaterial",
		"func (r *PostgresRepository) UpdateAdminColor",
		"func (r *PostgresRepository) UpdateAdminBox",
	} {
		body, ok := adminFunctionSource(source, marker)
		if !ok {
			t.Fatalf("expected %s", marker)
		}
		if !strings.Contains(body, "updated_at = now()") {
			t.Fatalf("expected %s to set updated_at", marker)
		}
	}
}

func readAdminCatalogRepository(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("catalog_repository.go")
	if err != nil {
		t.Fatalf("read catalog repository: %v", err)
	}

	return string(data)
}
