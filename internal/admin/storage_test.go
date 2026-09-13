package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseStorageClientCreateSignedUpload(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/storage/v1/object/upload/sign/product-images/products/11111111-1111-1111-1111-111111111111/random.png" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sb_secret_test" {
			t.Fatalf("expected authorization header, got %q", got)
		}
		if got := r.Header.Get("apikey"); got != "sb_secret_test" {
			t.Fatalf("expected apikey header, got %q", got)
		}
		if got := r.Header.Get("x-upsert"); got != "false" {
			t.Fatalf("expected no upsert, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"url":   "/object/upload/sign/product-images/products/11111111-1111-1111-1111-111111111111/random.png?token=signed",
			"token": "signed",
		})
	}))
	defer server.Close()

	client := newTestStorageClient(t, server.URL, server.Client())
	upload, err := client.CreateSignedUpload(t.Context(), AdminImageBucket, "products/11111111-1111-1111-1111-111111111111/random.png")
	if err != nil {
		t.Fatalf("expected signed upload, got %v", err)
	}
	if !sawRequest {
		t.Fatal("expected storage request")
	}
	if !strings.HasPrefix(upload.UploadURL, server.URL+"/storage/v1/object/upload/sign/") || upload.Token != "signed" {
		t.Fatalf("unexpected signed upload: %#v", upload)
	}
}

func TestSupabaseStorageClientStatObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("expected HEAD, got %s", r.Method)
		}
		if r.URL.Path != "/storage/v1/object/info/product-images/products/11111111-1111-1111-1111-111111111111/random.webp" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sb_secret_test" {
			t.Fatal("expected secret authorization header")
		}
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Content-Length", "1234")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestStorageClient(t, server.URL, server.Client())
	object, err := client.StatObject(t.Context(), AdminImageBucket, "products/11111111-1111-1111-1111-111111111111/random.webp")
	if err != nil {
		t.Fatalf("expected object metadata, got %v", err)
	}
	if object.ContentType != "image/webp" || object.Size != 1234 {
		t.Fatalf("unexpected object metadata: %#v", object)
	}
}

func TestSupabaseStorageClientDeleteObjectUsesPrefixes(t *testing.T) {
	var prefixes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/storage/v1/object/product-images" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("apikey") != "sb_secret_test" {
			t.Fatal("expected secret apikey header")
		}
		var body struct {
			Prefixes []string `json:"prefixes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("expected JSON body, got %v", err)
		}
		prefixes = body.Prefixes
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestStorageClient(t, server.URL, server.Client())
	err := client.DeleteObject(t.Context(), AdminImageBucket, "products/11111111-1111-1111-1111-111111111111/random.jpg")
	if err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if len(prefixes) != 1 || prefixes[0] != "products/11111111-1111-1111-1111-111111111111/random.jpg" {
		t.Fatalf("unexpected prefixes: %#v", prefixes)
	}
}

func newTestStorageClient(t *testing.T, baseURL string, httpClient *http.Client) *SupabaseStorageClient {
	t.Helper()
	client, err := NewSupabaseStorageClient(SupabaseStorageClientConfig{
		SupabaseURL: baseURL,
		SecretKey:   "sb_secret_test",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("expected storage client, got %v", err)
	}

	return client
}
