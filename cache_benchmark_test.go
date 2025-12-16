package slog

import (
	"fmt"
	"sync"
	"testing"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/internal/common"
)

// TestLRUCacheVsSyncMapPerformance 对比LRU缓存和sync.Map的性能
func TestLRUCacheVsSyncMapPerformance(t *testing.T) {
	const iterations = 10000
	const cacheSize = 1000

	// 测试数据
	testCases := make([]string, iterations)
	for i := 0; i < iterations; i++ {
		testCases[i] = fmt.Sprintf("test_format_string_%d_%%s", i%cacheSize)
	}

	t.Log("Performance comparison: LRU Cache vs sync.Map")

	// 测试LRU缓存
	lruCache := common.NewLRUStringCache(cacheSize)

	t.Run("LRU_Cache", func(t *testing.T) {
		start := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key := testCases[i%len(testCases)]
				if val, ok := lruCache.GetString(key); !ok {
					lruCache.PutString(key, "true")
				} else {
					_ = val
				}
			}
		})

		stats := lruCache.GetStats()
		t.Logf("LRU Cache - Operations: %d, Avg per op: %v", iterations, start.NsPerOp())
		t.Logf("LRU Cache - Cache size: %d, Hits: %d, Hit rate: %.2f%%",
			stats.Size, stats.Hits, stats.HitRate*100)
	})

	// 测试sync.Map
	syncMap := &sync.Map{}
	counter := int64(0)

	t.Run("Sync_Map", func(t *testing.T) {
		start := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key := testCases[i%len(testCases)]
				if val, ok := syncMap.Load(key); !ok {
					syncMap.Store(key, "true")
					counter++
				} else {
					_ = val
				}
			}
		})

		t.Logf("sync.Map - Operations: %d, Avg per op: %v", iterations, start.NsPerOp())
		t.Logf("sync.Map - Stored items: %d", counter)
	})
}

// BenchmarkFormatCacheLRU 基准测试LRU格式缓存
func BenchmarkFormatCacheLRU(b *testing.B) {
	// 清理缓存
	formatCache.Clear()

	testMessages := []string{
		"hello world",
		"format %s test",
		"multiple %s %d formats",
		"no format specifiers",
		"single %v format",
		"complex %s format with %d numbers and %v values",
		"plain text message",
		"another %s message",
		"testing %d format",
		"final %v test",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msg := testMessages[i%len(testMessages)]
		formatLog(msg, "test", 123, "value")
	}

	b.StopTimer()

	stats := formatCache.GetStats()
	b.Logf("Final cache stats - Size: %d, Hits: %d, Hit rate: %.2f%%",
		stats.Size, stats.Hits, stats.HitRate*100)
}

// BenchmarkDLPCacheLRU 基准测试DLP引擎的LRU缓存
func BenchmarkDLPCacheLRU(b *testing.B) {
	// 创建DLP引擎实例
	engine := dlp.NewDlpEngine()
	engine.Enable()

	testTexts := []string{
		"联系电话：13812345678",
		"邮箱地址：test@example.com",
		"身份证号：110101199001011234",
		"银行卡号：6222020000000000000",
		"普通文本没有敏感信息",
		"多种信息：手机13987654321，邮箱user@domain.com",
		"IP地址：192.168.1.100",
		"姓名：张三李四",
		"网址：https://www.example.com",
		"简单文本测试",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		text := testTexts[i%len(testTexts)]
		_ = engine.DesensitizeText(text)
	}

	b.StopTimer()

	hits, misses := engine.GetCacheStats()
	totalRequests := hits + misses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(hits) / float64(totalRequests) * 100
	}

	b.Logf("DLP cache stats - Hits: %d, Misses: %d, Hit rate: %.2f%%",
		hits, misses, hitRate)
}

// BenchmarkLRUCacheContention 测试LRU缓存的并发性能
func BenchmarkLRUCacheContention(b *testing.B) {
	cache := common.NewLRUCache(1000)

	// 预填充一些数据
	for i := 0; i < 500; i++ {
		cache.Put(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%1000)

			// 混合读写操作 (80% 读, 20% 写)
			if i%5 == 0 {
				cache.Put(key, fmt.Sprintf("value_%d", i))
			} else {
				cache.Get(key)
			}
			i++
		}
	})

	b.StopTimer()

	stats := cache.GetStats()
	b.Logf("Concurrent cache stats - Size: %d, Hit rate: %.2f%%",
		stats.Size, stats.HitRate*100)
}

// TestLRUCacheMemoryEfficiency 测试LRU缓存的内存效率
func TestLRUCacheMemoryEfficiency(t *testing.T) {
	const maxSize = 100
	cache := common.NewLRUCache(maxSize)

	// 添加超过容量的数据
	for i := 0; i < maxSize*2; i++ {
		cache.Put(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))

		// 验证缓存大小不超过限制
		if cache.Size() > maxSize {
			t.Errorf("Cache size %d exceeds maximum %d", cache.Size(), maxSize)
		}
	}

	// 验证最终大小等于容量
	if cache.Size() != maxSize {
		t.Errorf("Final cache size %d != max size %d", cache.Size(), maxSize)
	}

	// 验证LRU淘汰策略 - 早期的键应该被淘汰
	earlyKeys := 0
	lateKeys := 0

	for i := 0; i < maxSize; i++ {
		if cache.Contains(fmt.Sprintf("key_%d", i)) {
			earlyKeys++
		}
	}

	for i := maxSize; i < maxSize*2; i++ {
		if cache.Contains(fmt.Sprintf("key_%d", i)) {
			lateKeys++
		}
	}

	t.Logf("Early keys remaining: %d, Late keys remaining: %d", earlyKeys, lateKeys)

	// 大部分早期键应该被淘汰，大部分晚期键应该保留
	if earlyKeys > maxSize/4 {
		t.Errorf("Too many early keys remain: %d (expected < %d)", earlyKeys, maxSize/4)
	}

	if lateKeys < maxSize*3/4 {
		t.Errorf("Too few late keys remain: %d (expected > %d)", lateKeys, maxSize*3/4)
	}

	stats := cache.GetStats()
	t.Logf("Memory efficiency test - Final size: %d, Capacity: %d, Utilization: %.1f%%",
		stats.Size, stats.Capacity, float64(stats.Size)/float64(stats.Capacity)*100)
}
