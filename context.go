package slog

import (
	"context"
	"sync"
)

var (
	fields = "slog_fields"
)

// WithContext 使用给定的上下文
func WithContext(parent context.Context) context.Context {
	if parent != nil {
		logger.ctx = parent
	}
	return logger.ctx
}

// WithContext 使用给定的上下文
func (l *Logger) WithContext(parent context.Context) *Logger {
	if parent != nil {
		l.ctx = parent
	}
	return l
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func WithValue(key string, val any) context.Context {
	ctx := withValue(&logger, key, val)
	logger.WithContext(ctx)
	return ctx
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(key string, val any) context.Context {
	ctx := withValue(l, key, val)
	//l.WithContext(ctx)
	return ctx
}

func withValue(l *Logger, key string, val any) context.Context {
	if v, ok := l.ctx.Value(fields).(*sync.Map); ok { // 检查当前上下文中是否已经有 sync.Map
		mapCopy := copySyncMap(v)                         // 复制现有的 sync.Map
		mapCopy.Store(key, val)                           // 在复制的 sync.Map 中存储新的键值对
		l.ctx = context.WithValue(l.ctx, fields, mapCopy) // 将更新后的 sync.Map 存储在新的上下文中
		return l.ctx
	}
	v := &sync.Map{}                            // 如果没有现有的 sync.Map，创建一个新的
	v.Store(key, val)                           // 在新创建的 sync.Map 中存储键值对
	l.ctx = context.WithValue(l.ctx, fields, v) // 将新的 sync.Map 存储在上下文中
	return l.ctx
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
