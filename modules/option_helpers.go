package modules

import (
	"context"
	"log/slog"
	"time"
)

// DefaultAttrFromContext 规范化上下文属性提取函数列表。
func DefaultAttrFromContext(fns []func(ctx context.Context) []slog.Attr) []func(ctx context.Context) []slog.Attr {
	if fns == nil {
		return []func(ctx context.Context) []slog.Attr{}
	}
	return fns
}

// DefaultTimeout 返回默认超时时间。
func DefaultTimeout(timeout, fallback time.Duration) time.Duration {
	if timeout == 0 {
		return fallback
	}
	return timeout
}
