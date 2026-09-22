package cart

import "context"

type unitCountContextKey struct{}

func WithUnitCount(ctx context.Context, count int) context.Context {
	if count < 0 {
		count = 0
	}
	return context.WithValue(ctx, unitCountContextKey{}, count)
}

func UnitCountFromContext(ctx context.Context) int {
	count, _ := ctx.Value(unitCountContextKey{}).(int)
	if count < 0 {
		return 0
	}
	return count
}
