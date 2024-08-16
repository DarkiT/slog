package slog

import (
	"context"
	"sync"
)

var fields = "slog_fields"

// WithContext 使用给定的上下文
func WithContext(ctx context.Context) *Logger {
	defaultLogger.ctx = ctx
	return &defaultLogger
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func WithValue(key string, val any) *Logger {
	ctx := withValue(&defaultLogger, key, val)
	defaultLogger.WithContext(ctx)
	return &defaultLogger
}

// WithContext 使用给定的上下文
func (l *Logger) WithContext(ctx context.Context) *Logger {
	l.ctx = ctx
	return l
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(key string, val any) *Logger {
	l.ctx = withValue(l, key, val)
	return l
}

func withValue(l *Logger, key string, val any) context.Context {
	if v, ok := l.ctx.Value(fields).(*sync.Map); ok {
		mapCopy := copySyncMap(v)
		mapCopy.Store(key, val)
		return context.WithValue(l.ctx, fields, mapCopy)
	}
	v := &sync.Map{}
	v.Store(key, val)
	return context.WithValue(l.ctx, fields, v)
}

// copySyncMap 复制一个 sync.Map 并返回
func copySyncMap(m *sync.Map) *sync.Map {
	var cp sync.Map
	m.Range(func(k, v interface{}) bool { // 遍历原始 sync.Map
		cp.Store(k, v) // 将每个键值对存储到新的 sync.Map
		return true
	})
	return &cp // 返回新的 sync.Map
}
