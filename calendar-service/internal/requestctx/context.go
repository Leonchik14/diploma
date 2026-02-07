package requestctx

import "context"

type userIDKey struct{}

var key = &userIDKey{}

func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, key, userID)
}

func UserID(ctx context.Context) (userID uint, ok bool) {
	v := ctx.Value(key)
	if v == nil {
		return 0, false
	}
	userID, ok = v.(uint)
	return userID, ok
}
