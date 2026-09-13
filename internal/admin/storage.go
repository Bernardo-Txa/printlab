package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const AdminImageBucket = products.ProductImagesBucket

type StorageClient interface {
	CreateSignedUpload(ctx context.Context, bucket string, objectPath string) (SignedUpload, error)
	StatObject(ctx context.Context, bucket string, objectPath string) (StorageObject, error)
	DeleteObject(ctx context.Context, bucket string, objectPath string) error
}

type SignedUpload struct {
	UploadURL string
	Token     string
	Path      string
}

type StorageObject struct {
	ContentType string
	Size        int64
}

type SupabaseStorageClientConfig struct {
	SupabaseURL string
	SecretKey   string
	HTTPClient  *http.Client
}

type SupabaseStorageClient struct {
	baseURL    *url.URL
	storageURL string
	secretKey  string
	httpClient *http.Client
}

func NewSupabaseStorageClient(cfg SupabaseStorageClientConfig) (*SupabaseStorageClient, error) {
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.SupabaseURL), "/"))
	if err != nil || base.Scheme == "" || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, ErrStorageUnavailable
	}
	secretKey := strings.TrimSpace(cfg.SecretKey)
	if secretKey == "" {
		return nil, ErrStorageUnavailable
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	storageURL := base.Scheme + "://" + base.Host + strings.TrimRight(base.EscapedPath(), "/") + "/storage/v1"
	return &SupabaseStorageClient{
		baseURL:    base,
		storageURL: storageURL,
		secretKey:  secretKey,
		httpClient: client,
	}, nil
}

func (c *SupabaseStorageClient) CreateSignedUpload(ctx context.Context, bucket string, objectPath string) (SignedUpload, error) {
	if c == nil || c.httpClient == nil {
		return SignedUpload{}, ErrStorageUnavailable
	}

	endpoint := c.storageEndpoint("/object/upload/sign/" + escapeStoragePath(bucket) + "/" + escapeStoragePath(objectPath))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return SignedUpload{}, ErrStorageUnavailable
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-upsert", "false")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SignedUpload{}, ErrStorageUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		drainResponse(resp.Body)
		return SignedUpload{}, ErrStorageUnavailable
	}

	var payload struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&payload); err != nil {
		return SignedUpload{}, ErrStorageUnavailable
	}
	uploadURL := c.resolveStorageURL(payload.URL)
	if uploadURL == "" {
		return SignedUpload{}, ErrStorageUnavailable
	}

	return SignedUpload{UploadURL: uploadURL, Token: payload.Token, Path: objectPath}, nil
}

func (c *SupabaseStorageClient) StatObject(ctx context.Context, bucket string, objectPath string) (StorageObject, error) {
	if c == nil || c.httpClient == nil {
		return StorageObject{}, ErrStorageUnavailable
	}

	endpoint := c.storageEndpoint("/object/info/" + escapeStoragePath(bucket) + "/" + escapeStoragePath(objectPath))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return StorageObject{}, ErrStorageUnavailable
	}
	c.authorize(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return StorageObject{}, ErrStorageUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		drainResponse(resp.Body)
		return StorageObject{}, ErrInvalidImagePath
	}

	var payload struct {
		Size        json.Number `json:"size"`
		ContentType string      `json:"content_type"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 16<<10))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return StorageObject{}, ErrInvalidImagePath
	}
	size, err := payload.Size.Int64()
	if err != nil || size <= 0 {
		return StorageObject{}, ErrInvalidImagePath
	}
	contentType := strings.TrimSpace(payload.ContentType)
	if contentType == "" {
		return StorageObject{}, ErrInvalidImagePath
	}

	return StorageObject{
		ContentType: contentType,
		Size:        size,
	}, nil
}

func (c *SupabaseStorageClient) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	if c == nil || c.httpClient == nil {
		return ErrStorageUnavailable
	}

	body, err := json.Marshal(struct {
		Prefixes []string `json:"prefixes"`
	}{Prefixes: []string{objectPath}})
	if err != nil {
		return ErrStorageUnavailable
	}

	endpoint := c.storageEndpoint("/object/" + escapeStoragePath(bucket))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return ErrStorageUnavailable
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrStorageUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		drainResponse(resp.Body)
		return ErrStorageUnavailable
	}
	drainResponse(resp.Body)

	return nil
}

func (c *SupabaseStorageClient) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("apikey", c.secretKey)
}

func (c *SupabaseStorageClient) storageEndpoint(path string) string {
	return c.storageURL + path
}

func (c *SupabaseStorageClient) resolveStorageURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	if strings.HasPrefix(value, "/") {
		return c.storageURL + value
	}

	return c.storageURL + "/" + strings.TrimLeft(value, "/")
}

func escapeStoragePath(value string) string {
	segments := strings.Split(value, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}

func drainResponse(body io.Reader) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4<<10))
}

func storageError(err error) error {
	if errors.Is(err, ErrInvalidImagePath) {
		return ErrInvalidImagePath
	}
	return ErrStorageUnavailable
}
