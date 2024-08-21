package slog

import (
	"context"
	"sync"
)

var fields = "slog_fields"

// WithContext 使用给定的上下文
func WithContext(ctx context.Context) *Logger {
	logger.ctx = ctx
	return &logger
}

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func WithValue(key string, val any) *Logger {
	ctx := withValue(&logger, key, val)
	logger.WithContext(ctx)
	return &logger
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

type listMap struct {
	m       sync.Map
	maxSize int
}

func NewListMap(num int) *listMap {
	return &listMap{
		maxSize: num,
	}
}

// RPush 将元素从右侧推入列表
func (c *listMap) RPush(key string, value interface{}) {
	c.m.LoadOrStore(key, &sync.Mutex{})
	c.m.Range(func(k, v interface{}) bool {
		if k == key {
			v.(*sync.Mutex).Lock()
			defer v.(*sync.Mutex).Unlock()

			if list, ok := c.m.Load(key); ok {
				l := list.([]interface{})
				l = append(l, value)
				c.m.Store(key, l)
			} else {
				c.m.Store(key, []interface{}{value})
			}
		}
		return true
	})
}

// LPop 从列表左侧弹出元素
func (c *listMap) LPop(key string) (interface{}, bool) {
	c.m.LoadOrStore(key, &sync.Mutex{})
	var value interface{}
	c.m.Range(func(k, v interface{}) bool {
		if k == key {
			v.(*sync.Mutex).Lock()
			defer v.(*sync.Mutex).Unlock()

			if list, ok := c.m.Load(key); ok {
				l := list.([]interface{})
				if len(l) > 0 {
					value = l[0]
					l = l[1:]
					c.m.Store(key, l)
					return false
				}
			}
		}
		return true
	})
	return value, value != nil
}

// LLen 返回列表长度
func (c *listMap) LLen(key string) int {
	if list, ok := c.m.Load(key); ok {
		l := list.([]interface{})
		return len(l)
	}
	return 0
}

// LTrim 裁剪列表
func (c *listMap) LTrim(key string, start, stop int) {
	c.m.LoadOrStore(key, &sync.Mutex{})
	c.m.Range(func(k, v interface{}) bool {
		if k == key {
			v.(*sync.Mutex).Lock()
			defer v.(*sync.Mutex).Unlock()

			if list, ok := c.m.Load(key); ok {
				l := list.([]interface{})
				if start < 0 {
					start = len(l) + start
				}
				if stop < 0 {
					stop = len(l) + stop
				}
				if start < 0 {
					start = 0
				}
				if stop >= len(l) {
					stop = len(l) - 1
				}
				if start <= stop && stop < len(l) {
					c.m.Store(key, l[start:stop+1])
				}
			}
		}
		return true
	})
}

// AddToListWithLimit 推入元素并限制列表大小
func (c *listMap) AddToListWithLimit(listName string, value interface{}) {
	if c.maxSize == 0 {
		return
	}
	// 将新元素推入列表
	c.RPush(listName, value)
	// 获取列表长度是否超限
	if c.LLen(listName) > c.maxSize {
		// 修剪列表到最大大小
		c.LTrim(listName, -c.maxSize, -1)
	}
}
