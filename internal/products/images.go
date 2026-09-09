package products

import (
	"net/url"
	"strings"
)

const ProductImagesBucket = "product-images"

func PublicProductImageURL(supabaseURL string, storagePath string) string {
	if strings.TrimSpace(supabaseURL) == "" || !validStoragePath(storagePath) {
		return ""
	}

	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(supabaseURL), "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return ""
	}

	if base.Scheme != "https" && base.Scheme != "http" {
		return ""
	}

	escapedPath := escapeStoragePath(storagePath)
	root := base.Scheme + "://" + base.Host + strings.TrimRight(base.EscapedPath(), "/")

	return root + "/storage/v1/object/public/" + ProductImagesBucket + "/" + escapedPath
}

func validStoragePath(storagePath string) bool {
	path := strings.TrimSpace(storagePath)
	if path == "" || path != storagePath {
		return false
	}

	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.HasPrefix(lowerPath, "http:") || strings.HasPrefix(lowerPath, "https:") {
		return false
	}

	if strings.Contains(strings.Split(path, "/")[0], ":") {
		return false
	}

	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}

	return true
}

func escapeStoragePath(storagePath string) string {
	segments := strings.Split(storagePath, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}
