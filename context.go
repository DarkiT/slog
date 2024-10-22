package slog

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// 上下文键优化
type contextKey struct {
	name string
}

var (
	fieldsKey = &contextKey{"slog_fields"}

	// 全局context池
	ctxPool = &sync.Pool{
		New: func() interface{} {
			return newFields()
		},
	}
)

// Fields 优化实现
type Fields struct {
	store      atomic.Value // *sync.Map
	refs       int64        // 引用计数
	cleanOnce  sync.Once    // 确保只清理一次
	lastAccess int64        // 最后访问时间
	mu         sync.RWMutex // 添加互斥锁以保护并发访问
}

func newFields() *Fields {
	f := &Fields{}
	f.store.Store(&sync.Map{})
	f.lastAccess = time.Now().UnixNano()
	return f
}

// cleanup优化实现
func (f *Fields) cleanup() {
	if atomic.AddInt64(&f.refs, -1) == 0 {
		f.cleanOnce.Do(func() {
			f.store.Store(&sync.Map{})
			// 只有在超过一定时间未访问时才放回池中
			if time.Now().UnixNano()-atomic.LoadInt64(&f.lastAccess) > int64(time.Minute) {
				ctxPool.Put(f)
			}
		})
	}
}

// WithContext 使用给定的上下文
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
		newLogger.text = slog.New(newAddonsHandler(newLogger.text.Handler(), slogPfx))
	}
	if newLogger.json != nil {
		newLogger.json = slog.New(newAddonsHandler(newLogger.json.Handler(), slogPfx))
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

	var fields *Fields
	if f := newLogger.ctx.Value(fieldsKey); f != nil {
		// 使用新的 Fields 来避免影响原有的
		fields = newFields()
		m := &sync.Map{}
		// 复制现有值
		existingFields := f.(*Fields)
		if existing := existingFields.store.Load().(*sync.Map); existing != nil {
			existing.Range(func(k, v interface{}) bool {
				m.Store(k, v)
				return true
			})
		}
		// 添加新值
		m.Store(key, val)
		fields.store.Store(m)
	} else {
		fields = newFields()
		m := &sync.Map{}
		m.Store(key, val)
		fields.store.Store(m)
	}

	// 创建新的上下文
	newLogger.ctx = context.WithValue(
		newLogger.ctx,
		fieldsKey,
		fields,
	)

	return newLogger
}

// 定期清理过期的context
func init() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			cleanupExpiredContexts()
		}
	}()
}

func cleanupExpiredContexts() {
	threshold := time.Now().Add(-time.Hour).UnixNano()
	ctxPool.New = func() interface{} {
		f := newFields()
		if atomic.LoadInt64(&f.lastAccess) < threshold {
			f.cleanup()
		}
		return f
	}
}
