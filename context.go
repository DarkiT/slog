package slog

import (
	"context"
	"sync"
)

// 定义 contextKey 类型，用于存储上下文中的键
type contextKey string

var (
	ctx               = context.Background() // 全局上下文变量，初始化为背景上下文
	fields contextKey = "slog_fields"        // 定义一个用于存储字段的上下文键
)

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func WithValue(key string, val any) context.Context {
	if v, ok := ctx.Value(fields).(*sync.Map); ok { // 检查当前上下文中是否已经有 sync.Map
		mapCopy := copySyncMap(v) // 复制现有的 sync.Map
		mapCopy.Store(key, val)   // 在复制的 sync.Map 中存储新的键值对

		ctx = context.WithValue(ctx, fields, mapCopy) // 将更新后的 sync.Map 存储在新的上下文中
		return ctx                                    // 返回新的上下文
	}
	v := &sync.Map{}  // 如果没有现有的 sync.Map，创建一个新的
	v.Store(key, val) // 在新创建的 sync.Map 中存储键值对

	ctx = context.WithValue(ctx, fields, v) // 将新的 sync.Map 存储在上下文中
	return ctx                              // 返回新的上下文
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
