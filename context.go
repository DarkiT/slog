package slog

import (
	"context"
	"log/slog"
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
	values map[string]interface{}
	mu     sync.RWMutex
}

// fieldsPool 对象池
var fieldsPool = sync.Pool{
	New: func() interface{} {
		return &Fields{
			values: make(map[string]interface{}),
		}
	},
}

// newFields 创建新的Fields实例
func newFields() *Fields {
	return fieldsPool.Get().(*Fields)
}

// release 释放Fields实例
func (f *Fields) release() {
	f.mu.Lock()
	clear(f.values)
	f.mu.Unlock()
	fieldsPool.Put(f)
}

// clone 克隆Fields实例
func (f *Fields) clone() *Fields {
	newF := newFields()
	if f != nil {
		f.mu.RLock()
		for k, v := range f.values {
			newF.values[k] = v
		}
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

	// 监听context取消
	go func() {
		<-ctx.Done()
		if fields := getFields(ctx); fields != nil {
			fields.release()
		}
	}()

	// 更新handlers的context
	if newLogger.text != nil {
		newLogger.text = slog.New(newAddonsHandler(newLogger.text.Handler(), ext))
	}
	if newLogger.json != nil {
		newLogger.json = slog.New(newAddonsHandler(newLogger.json.Handler(), ext))
	}

	return newLogger
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(key string, val interface{}) *Logger {
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
	ctx, cancel := context.WithCancel(newLogger.ctx)
	newLogger.ctx = context.WithValue(ctx, fieldsKey, newFields)

	// 监听context取消
	go func() {
		<-ctx.Done()
		newFields.release()
		cancel()
	}()

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
