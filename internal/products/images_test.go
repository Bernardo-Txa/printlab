package products

import "testing"

func TestPublicProductImageURL(t *testing.T) {
	got := PublicProductImageURL("https://example.supabase.co/", "produtos/chaveiro azul.webp")
	want := "https://example.supabase.co/storage/v1/object/public/product-images/produtos/chaveiro%20azul.webp"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPublicProductImageURLRejectsUnsafeInput(t *testing.T) {
	tests := []struct {
		name        string
		supabaseURL string
		storagePath string
	}{
		{name: "missing supabase url", supabaseURL: "", storagePath: "produtos/imagem.webp"},
		{name: "invalid supabase url", supabaseURL: "not a url", storagePath: "produtos/imagem.webp"},
		{name: "external storage path", supabaseURL: "https://example.supabase.co", storagePath: "https://evil.example/imagem.webp"},
		{name: "absolute storage path", supabaseURL: "https://example.supabase.co", storagePath: "/produtos/imagem.webp"},
		{name: "traversal storage path", supabaseURL: "https://example.supabase.co", storagePath: "produtos/../secret.webp"},
		{name: "blank segment", supabaseURL: "https://example.supabase.co", storagePath: "produtos//imagem.webp"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PublicProductImageURL(test.supabaseURL, test.storagePath); got != "" {
				t.Fatalf("expected empty URL, got %q", got)
			}
		})
	}
}
