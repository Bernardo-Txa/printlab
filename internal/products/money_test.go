package products

import "testing"

func TestFormatBRL(t *testing.T) {
	tests := []struct {
		name  string
		cents int64
		want  string
	}{
		{name: "zero", cents: 0, want: "R$ 0,00"},
		{name: "one cent", cents: 1, want: "R$ 0,01"},
		{name: "thirty nine ninety", cents: 3990, want: "R$ 39,90"},
		{name: "one hundred", cents: 10000, want: "R$ 100,00"},
		{name: "thousand separator", cents: 123456, want: "R$ 1.234,56"},
		{name: "ten thousand", cents: 1000000, want: "R$ 10.000,00"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatBRL(test.cents); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
