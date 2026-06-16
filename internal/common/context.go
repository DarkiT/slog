package common

import (
	"context"
	"fmt"
	"log/slog"
)

func ContextExtractor(ctx context.Context, fns []func(ctx context.Context) []slog.Attr) []slog.Attr {
	attrs := []slog.Attr{}
	for _, fn := range fns {
		attrs = append(attrs, fn(ctx)...)
	}
	return attrs
}

func ExtractFromContext(keys ...any) func(ctx context.Context) []slog.Attr {
	return func(ctx context.Context) []slog.Attr {
		attrs := []slog.Attr{}
		for _, key := range keys {
			name, ok := key.(string)
			if !ok {
				name = fmt.Sprint(key)
			}
			var value any
			if ctx != nil {
				value = ctx.Value(key)
			}
			attrs = append(attrs, slog.Any(name, value))
		}
		return attrs
	}
}
