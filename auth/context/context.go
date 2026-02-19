package authentication

import "context"

type ContextKeySessionIDType struct{}

var ContextKeySessionID = ContextKeySessionIDType{}

func SessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(ContextKeySessionID).(string)
	if !ok {
		return "", false
	}

	return sessionID, true
}

const (
	// Anonymous is the guest user id.
	Anonymous = "system:anonymous"

	Authenticated   = "system:authenticated"
	Unauthenticated = "system:unauthenticated"
)

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ContextKeySessionID, sessionID)
}

type contextKeySubject struct{}

func GetSubject(ctx context.Context) string {
	userID, ok := ctx.Value(contextKeySubject{}).(string)
	if !ok {
		return Anonymous
	}

	return userID
}

func WithSubject(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeySubject{}, userID)
}

func WithServiceSubject(ctx context.Context, serviceName string) context.Context {
	return WithSubject(ctx, "system:service:"+serviceName)
}
