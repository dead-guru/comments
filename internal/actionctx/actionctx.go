package actionctx

import "context"

type adminIDKey struct{}

func WithAdminID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, adminIDKey{}, id)
}

func AdminID(ctx context.Context) *int64 {
	id, ok := ctx.Value(adminIDKey{}).(int64)
	if !ok {
		return nil
	}
	return &id
}
