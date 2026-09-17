package seo

import "context"

type contextKey struct{}

type Context struct {
	Origin  string
	NoIndex bool
}

func WithContext(ctx context.Context, origin string, noIndex bool) context.Context {
	return context.WithValue(ctx, contextKey{}, Context{Origin: origin, NoIndex: noIndex})
}
func CanonicalURL(ctx context.Context, path string) string {
	v, _ := ctx.Value(contextKey{}).(Context)
	if v.Origin == "" || path == "" {
		return ""
	}
	return v.Origin + path
}
func NoIndex(ctx context.Context) bool { v, _ := ctx.Value(contextKey{}).(Context); return v.NoIndex }
