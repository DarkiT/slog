package slog

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// contextKey 定义上下文键
type contextKey string

const (
	fieldsKey contextKey = "slog_fields"
	// 定义清理相关的常量
	cleanupInterval = time.Minute
	maxIdleTime     = time.Hour
)

// Fields 存储上下文字段
type Fields struct {
	values     *sync.Map // 存储实际的键值对
	refCount   int32     // 原子引用计数
	lastAccess int64     // 最后访问时间戳
}

// fieldsPool 对象池
var fieldsPool = sync.Pool{
	New: func() interface{} {
		return &Fields{
			values: new(sync.Map),
		}
	},
}

// newFields 创建新的Fields实例
func newFields() *Fields {
	f := fieldsPool.Get().(*Fields)
	atomic.StoreInt32(&f.refCount, 1)
	atomic.StoreInt64(&f.lastAccess, time.Now().UnixNano())
	return f
}

// release 释放Fields实例
func (f *Fields) release() {
	if atomic.AddInt32(&f.refCount, -1) == 0 {
		f.values = new(sync.Map)
		fieldsPool.Put(f)
	}
}

// clone 克隆Fields实例
func (f *Fields) clone() *Fields {
	newF := newFields()
	if f != nil && f.values != nil {
		f.values.Range(func(key, value interface{}) bool {
			newF.values.Store(key, value)
			return true
		})
	}
	return newF
}

// updateLastAccess 更新最后访问时间
func (f *Fields) updateLastAccess() {
	atomic.StoreInt64(&f.lastAccess, time.Now().UnixNano())
}

// getFields 从上下文中获取Fields
func getFields(ctx context.Context) *Fields {
	if ctx == nil {
		return nil
	}
	if f, ok := ctx.Value(fieldsKey).(*Fields); ok {
		f.updateLastAccess()
		return f
	}
	return nil
}

// WithContext 方法保持不变
func WithContext(ctx context.Context) *Logger {
	return logger.WithContext(ctx)
}

// WithContext 使用给定的上下文
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	newLogger := l.clone()
	newLogger.ctx = ctx

	// 同时更新handlers的context
	if newLogger.text != nil {
		newLogger.text = slog.New(newAddonsHandler(newLogger.text.Handler(), ext))
	}
	if newLogger.json != nil {
		newLogger.json = slog.New(newAddonsHandler(newLogger.json.Handler(), ext))
	}

	return newLogger
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(key string, val any) *Logger {
	if key == "" || val == nil {
		return l
	}

	newLogger := l.clone()

	// 确保存在上下文
	if newLogger.ctx == nil {
		newLogger.ctx = context.Background()
	}

	oldFields := getFields(newLogger.ctx)
	newFields := oldFields.clone()
	newFields.values.Store(key, val)

	newLogger.ctx = context.WithValue(newLogger.ctx, fieldsKey, newFields)
	if oldFields != nil {
		oldFields.release()
	}

	return newLogger
}

// 定期清理过期的context
func init() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredFields()
		}
	}()
}

// cleanupExpiredFields 清理过期的Fields
func cleanupExpiredFields() {
	threshold := time.Now().Add(-maxIdleTime).UnixNano()

	// 创建临时对象用于遍历
	temp := fieldsPool.Get().(*Fields)
	if temp != nil {
		// 如果最后访问时间超过阈值，重置该对象
		if atomic.LoadInt64(&temp.lastAccess) < threshold {
			temp.values = new(sync.Map)
			atomic.StoreInt32(&temp.refCount, 0)
		}
		// 将对象放回池中
		fieldsPool.Put(temp)
	}
}
