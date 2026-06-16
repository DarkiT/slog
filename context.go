package slog

import (
	"context"
	"log/slog"
	"maps"
	"sync"
	"time"
)

// contextKey 定义上下文键
type contextKey string

const (
	fieldsKey contextKey = "slog_fields"
)

// Fields 存储上下文字段
type Fields struct {
	values map[string]any
	mu     sync.RWMutex
}

// fieldsPool 对象池
var fieldsPool = sync.Pool{
	New: func() any {
		return &Fields{
			values: make(map[string]any),
		}
	},
}

// newFields 创建新的Fields实例
func newFields() *Fields {
	return fieldsPool.Get().(*Fields)
}

// clone 克隆Fields实例
func (f *Fields) clone() *Fields {
	newF := newFields()
	if f != nil {
		f.mu.RLock()
		maps.Copy(newF.values, f.values)
		f.mu.RUnlock()
	}
	return newF
}

// getFields 从上下文中获取Fields
func getFields(ctx context.Context) *Fields {
	if ctx == nil {
		return nil
	}
	if f, ok := ctx.Value(fieldsKey).(*Fields); ok {
		return f
	}
	return nil
}

// WithContext 创建带有上下文的新Logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	newLogger := l.clone()
	newLogger.ctx = ctx

	// 更新 handlers 的 context，避免重复包装
	if newLogger.text != nil {
		newLogger.text = slog.New(cloneHandlerWithContext(newLogger.text.Handler(), ctx))
	}
	if newLogger.json != nil {
		newLogger.json = slog.New(cloneHandlerWithContext(newLogger.json.Handler(), ctx))
	}

	return newLogger
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(key string, val any) *Logger {
	if key == "" || val == nil {
		return l
	}

	newLogger := l.clone()
	if newLogger.ctx == nil {
		newLogger.ctx = context.Background()
	}

	// 创建新的Fields或克隆现有的
	var newFields *Fields
	if oldFields := getFields(newLogger.ctx); oldFields != nil {
		newFields = oldFields.clone()
	} else {
		newFields = fieldsPool.Get().(*Fields)
	}

	// 添加新值
	newFields.mu.Lock()
	newFields.values[key] = val
	newFields.mu.Unlock()

	// 创建新的context
	newLogger.ctx = context.WithValue(newLogger.ctx, fieldsKey, newFields)

	return newLogger
}

// WithTimeout 创建带超时的Logger
func (l *Logger) WithTimeout(timeout time.Duration) (*Logger, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(l.ctx, timeout)
	return l.WithContext(ctx), cancel
}

// WithDeadline 创建带截止时间的Logger
func (l *Logger) WithDeadline(d time.Time) (*Logger, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(l.ctx, d)
	return l.WithContext(ctx), cancel
}
