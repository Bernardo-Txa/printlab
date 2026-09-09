package cart

import (
	"math"
	"testing"
)

func TestLineSubtotalCents(t *testing.T) {
	subtotal, err := lineSubtotalCents(3990, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if subtotal != 7980 {
		t.Fatalf("expected subtotal 7980, got %d", subtotal)
	}
}

func TestLineSubtotalCentsRejectsOverflow(t *testing.T) {
	_, err := lineSubtotalCents(math.MaxInt64, 2)
	if err != ErrAmountOverflow {
		t.Fatalf("expected ErrAmountOverflow, got %v", err)
	}
}

func TestAddCentsRejectsOverflow(t *testing.T) {
	_, err := addCents(math.MaxInt64, 1)
	if err != ErrAmountOverflow {
		t.Fatalf("expected ErrAmountOverflow, got %v", err)
	}
}
