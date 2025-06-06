package web

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ==============================================================================
// TraceID
type ctxKey int

const traceIDKey ctxKey = 1
const responseStatusKey ctxKey = 2

func setTraceID(ctx context.Context, traceID uuid.UUID) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func GetTraceID(ctx context.Context) uuid.UUID {
	trace, ok := ctx.Value(traceIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}
	}
	return trace
}

func injectResponseStatus(ctx context.Context, statusCode int) context.Context {
	return context.WithValue(ctx, responseStatusKey, &statusCode)
}

func setResponseStatus(ctx context.Context, statusCode int) error {
	p, ok := ctx.Value(responseStatusKey).(*int)
	if !ok {
		return errors.New("response status code not found in ctx")
	}
	*p = statusCode
	return nil
}

func GetResponseStatus(ctx context.Context) int {
	p, ok := ctx.Value(responseStatusKey).(*int)
	if !ok {
		return 0
	}
	return *p
}
