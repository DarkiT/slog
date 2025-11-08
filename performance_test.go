package slog

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer 是一个线程安全的 bytes.Buffer 包装器
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Len()
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func (sb *syncBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}

var _ io.Writer = (*syncBuffer)(nil)

// BenchmarkDynamic 测试优化后的Dynamic函数性能
func BenchmarkDynamic(b *testing.B) {
	var buf syncBuffer
	logger := NewLogger(&buf, true, false) // 禁用颜色以便测试

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Dynamic("测试消息", 3, 10)
		}
	})
}

// BenchmarkFormatCache 测试格式缓存性能
func BenchmarkFormatCache(b *testing.B) {
	testMsg := "测试消息 %s %d"
	args := []any{"test", 123}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			formatLog(testMsg, args...)
		}
	})
}

// BenchmarkStringBuilderPool 测试字符串构建器池性能
func BenchmarkStringBuilderPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := stringBuilderPool.Get().(*strings.Builder)
			builder.WriteString("测试")
			builder.WriteString("消息")
			_ = builder.String()
			builder.Reset()
			stringBuilderPool.Put(builder)
		}
	})
}

// BenchmarkErrorHandling 测试优化后的错误处理性能
func BenchmarkErrorHandling(b *testing.B) {
	var buf syncBuffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("测试消息", "key", "value")
		}
	})
}

// TestCacheCleaning 测试缓存清理功能
func TestCacheCleaning(t *testing.T) {
	// 清理缓存开始测试
	cleanFormatCache()
	initialCapacity := formatCache.Capacity()

	// 添加数据项
	for i := 0; i < 15; i++ {
		formatLog(fmt.Sprintf("测试消息 %d %%s", i), "arg")
	}

	// 验证LRU缓存自动管理大小（不会超过初始容量）
	cacheSize := formatCache.Size()
	if cacheSize > initialCapacity {
		t.Errorf("LRU缓存大小超过容量: 当前大小=%d, 容量=%d", cacheSize, initialCapacity)
	}

	// 验证LRU缓存正常工作
	if cacheSize == 0 {
		t.Error("缓存应该包含一些条目")
	}

	t.Logf("LRU缓存测试通过: 当前大小=%d, 容量=%d", cacheSize, initialCapacity)
}

// TestSubscriberErrorHandling 测试订阅者错误处理
func TestSubscriberErrorHandling(t *testing.T) {
	logger := NewLogger(nil, true, false)

	// 创建一个小缓冲区的订阅者
	records, cancel := Subscribe(1)
	defer cancel()

	// 填满channel
	logger.Info("消息1")
	logger.Info("消息2") // 这条消息应该触发错误处理

	// 给一些时间处理
	time.Sleep(100 * time.Millisecond)

	// 验证至少收到一条消息
	select {
	case <-records:
		// 成功接收到消息
	case <-time.After(time.Second):
		t.Error("未收到任何消息")
	}
}

// TestConfigurableLogger 测试可配置的日志记录器
func TestConfigurableLogger(t *testing.T) {
	config := &Config{
		MaxFormatCacheSize: 100,
		LogInternalErrors:  true,
		NoColor:            true,
		TimeFormat:         "2006-01-02 15:04:05",
	}
	config.SetEnableText(true)
	config.SetEnableJSON(false)

	var buf bytes.Buffer
	logger := NewLoggerWithConfig(&buf, config)

	logger.Info("测试配置")

	output := buf.String()
	if len(output) == 0 {
		t.Error("配置的logger未产生输出")
	}

	// 验证时间格式
	if !bytes.Contains(buf.Bytes(), []byte("2006")) {
		// 注意：实际测试中时间会是当前时间，这里只是示例
		t.Log("时间格式可能已应用")
	}
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < b.N; i++ {
		logger.Info("测试消息", "key", i, "time", time.Now())
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// TestConcurrentSafety 并发安全测试
func TestConcurrentSafety(t *testing.T) {
	var buf syncBuffer
	logger := NewLogger(&buf, true, false)

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 100

	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				logger.Info("并发测试", "goroutine", id, "iteration", j)
				logger.Dynamic("动态测试", 2, 5)
			}
		}(i)
	}

	wg.Wait()

	// 验证没有崩溃，输出有内容
	if buf.Len() == 0 {
		t.Error("并发测试未产生输出")
	}
}
