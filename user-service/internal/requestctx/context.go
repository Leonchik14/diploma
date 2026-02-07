package requestctx

import "context"

// userIDKey — приватный тип ключа, чтобы не конфликтовать с другими пакетами.
type userIDKey struct{}

var key = &userIDKey{}

// WithUserID сохраняет user_id в контексте (вызывается из interceptor).
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, key, userID)
}

// UserID возвращает user_id из контекста. ok == false, если значение отсутствует или неверного типа.
func UserID(ctx context.Context) (userID uint, ok bool) {
	v := ctx.Value(key)
	if v == nil {
		return 0, false
	}
	userID, ok = v.(uint)
	return userID, ok
}
