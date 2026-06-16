package slog

import (
	"context"
	"sync/atomic"
)

// ContextPropagatorFunc 将自定义上下文信息转换成 slog.Attr。
type ContextPropagatorFunc func(ctx context.Context) []Attr

var contextPropagator atomic.Value

// SetContextPropagator 设置全局上下文传播方法。
func SetContextPropagator(fn ContextPropagatorFunc) {
	contextPropagator.Store(fn)
}

func currentContextPropagator() ContextPropagatorFunc {
	if v := contextPropagator.Load(); v != nil {
		if fn, ok := v.(ContextPropagatorFunc); ok {
			return fn
		}
	}
	return nil
}
