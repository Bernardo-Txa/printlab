package products

import (
	"os"
	"strings"
	"testing"
)

func TestListVariantFilamentsQueryPreservesRetiredMaterialAndColorReferences(t *testing.T) {
	source, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("expected repository source to be readable, got %v", err)
	}

	query, ok := repositoryFunctionSource(string(source), "func (r *PostgresRepository) listVariantFilaments")
	if !ok {
		t.Fatal("expected listVariantFilaments to exist")
	}

	for _, forbidden := range []string{"m.is_active = true", "c.is_active = true"} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("expected recipe query not to filter retired material/color references with %q", forbidden)
		}
	}

	if !strings.Contains(query, "v.is_active = true") {
		t.Fatal("expected recipe query to keep filtering inactive variants")
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
