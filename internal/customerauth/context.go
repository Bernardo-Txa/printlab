package customerauth

import "context"

type contextKey struct{}

func WithProfile(ctx context.Context, profile Profile) context.Context {
	return context.WithValue(ctx, contextKey{}, profile)
}

func ProfileFromContext(ctx context.Context) (Profile, bool) {
	profile, ok := ctx.Value(contextKey{}).(Profile)
	return profile, ok && profile.ID != ""
}
