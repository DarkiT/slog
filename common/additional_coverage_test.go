package common

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestLRUCache_AdditionalEdgeCases 测试LRU缓存的边界情况
func TestLRUCache_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "零容量缓存",
			test: func(t *testing.T) {
				cache := NewLRUCache(0)
				// 零容量会被设置为默认容量100，所以应该能存储内容
				stats := cache.GetStats()
				if stats.Capacity != 100 {
					t.Errorf("零容量应该被设置为默认值100，实际: %d", stats.Capacity)
				}
			},
		},
		{
			name: "负容量缓存",
			test: func(t *testing.T) {
				cache := NewLRUCache(-1)
				// 负容量会被设置为默认容量100
				stats := cache.GetStats()
				if stats.Capacity != 100 {
					t.Errorf("负容量应该被设置为默认值100，实际: %d", stats.Capacity)
				}
			},
		},
		{
			name: "大容量缓存",
			test: func(t *testing.T) {
				cache := NewLRUCache(10000)

				// 添加大量数据
				for i := 0; i < 1000; i++ {
					cache.Put(i, i*2)
				}

				// 验证数据存在
				for i := 0; i < 1000; i++ {
					if value, exists := cache.Get(i); !exists || value != i*2 {
						t.Errorf("大容量缓存数据丢失: key=%d", i)
					}
				}
			},
		},
		{
			name: "nil值存储",
			test: func(t *testing.T) {
				cache := NewLRUCache(5)
				cache.Put("nil_key", nil)

				if value, exists := cache.Get("nil_key"); !exists || value != nil {
					t.Error("应该能够存储nil值")
				}
			},
		},
		{
			name: "相同键多次Put",
			test: func(t *testing.T) {
				cache := NewLRUCache(3)

				cache.Put("key", "value1")
				cache.Put("key", "value2")
				cache.Put("key", "value3")

				if value, exists := cache.Get("key"); !exists || value != "value3" {
					t.Error("相同键的最后值应该被保留")
				}

				if cache.Size() != 1 {
					t.Errorf("缓存大小应该为1，实际: %d", cache.Size())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestLRUCache_Statistics 测试缓存统计功能
func TestLRUCache_Statistics(t *testing.T) {
	cache := NewLRUCache(3)

	// 初始统计
	stats := cache.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("初始统计应该全为零")
	}

	// 添加数据并访问
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	// 命中测试
	cache.Get("a")
	cache.Get("b")
	cache.Get("a") // 再次命中

	// 未命中测试
	cache.Get("d")
	cache.Get("e")

	// 导致淘汰
	cache.Put("d", 4) // 应该淘汰c

	stats = cache.GetStats()
	if stats.Hits != 3 {
		t.Errorf("期望3次命中，实际: %d", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("期望2次未命中，实际: %d", stats.Misses)
	}
	// 注意：LRUCacheStats没有Evictions字段，所以我们检查其他指标
}

// TestLRUCache_ClearAndSize 测试清除和大小功能
func TestLRUCache_ClearAndSize(t *testing.T) {
	cache := NewLRUCache(5)

	// 添加数据
	for i := 0; i < 3; i++ {
		cache.Put(i, i*2)
	}

	if cache.Size() != 3 {
		t.Errorf("期望大小为3，实际: %d", cache.Size())
	}

	// 清除缓存
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("清除后大小应该为0，实际: %d", cache.Size())
	}

	// 验证数据确实被清除
	for i := 0; i < 3; i++ {
		if _, exists := cache.Get(i); exists {
			t.Errorf("清除后数据不应该存在: key=%d", i)
		}
	}

	// 验证统计被重置
	stats := cache.GetStats()
	if stats.Hits != 0 || stats.Misses != 3 { // 刚才的Get调用产生了3次miss
		t.Error("清除后统计应该正确")
	}
}

// TestLRUCache_ThreadSafety 测试线程安全
func TestLRUCache_ThreadSafety(t *testing.T) {
	cache := NewLRUCache(100)
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				cache.Put(key, key*2)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				cache.Get(key)
			}
		}(i)
	}

	// 并发清除和统计
	wg.Add(2)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 10)
		cache.Clear()
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			cache.GetStats()
			time.Sleep(time.Microsecond * 100)
		}
	}()

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 测试完成
	case <-time.After(5 * time.Second):
		t.Fatal("并发测试超时")
	}
}

// TestTieredPools_Coverage 测试分级池的更多覆盖
func TestTieredPools_Coverage(t *testing.T) {
	pools := NewTieredPools()

	tests := []struct {
		name         string
		expectedSize int
		bufferType   BufferSize
	}{
		{
			name:         "小buffer",
			expectedSize: 100,
			bufferType:   SmallBuffer,
		},
		{
			name:         "中buffer",
			expectedSize: 3000,
			bufferType:   MediumBuffer,
		},
		{
			name:         "大buffer",
			expectedSize: 10000,
			bufferType:   LargeBuffer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := pools.GetBuffer(tt.expectedSize)
			if buffer == nil {
				t.Error("获取的buffer不应该为nil")
			}

			if buffer.Size() != tt.bufferType {
				t.Errorf("期望buffer类型 %v，实际: %v", tt.bufferType, buffer.Size())
			}

			// 测试写入功能
			testData := "test data for buffer"
			buffer.WriteString(testData)

			if buffer.String() != testData {
				t.Error("buffer写入失败")
			}

			// 放回池中
			pools.PutBuffer(buffer)
		})
	}
}

// TestTieredPools_StatisticsCoverage 测试分级池统计覆盖
func TestTieredPools_StatisticsCoverage(t *testing.T) {
	pools := NewTieredPools()

	// 获取多个buffer并放回
	buffers := make([]*TieredBuffer, 5)
	for i := 0; i < 5; i++ {
		buffers[i] = pools.GetBuffer(3000) // 3KB -> medium buffer (> 2KB but <= 8KB)
	}

	for _, buffer := range buffers {
		pools.PutBuffer(buffer)
	}

	// 检查统计
	stats := pools.GetStats()
	if len(stats) == 0 {
		t.Error("应该有统计数据")
	}

	// 验证中等buffer池的统计
	if mediumStats, exists := stats[MediumBuffer]; exists {
		if mediumStats.Gets != 5 {
			t.Errorf("期望5次获取，实际: %d", mediumStats.Gets)
		}
		if mediumStats.Puts != 5 {
			t.Errorf("期望5次放回，实际: %d", mediumStats.Puts)
		}
	} else {
		t.Error("应该有中等buffer的统计")
	}
}

// TestTieredPools_ConcurrencyCoverage 测试分级池并发覆盖
func TestTieredPools_ConcurrencyCoverage(t *testing.T) {
	pools := NewTieredPools()
	const numGoroutines = 20
	const numOperations = 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 随机大小的buffer
				size := (id*numOperations + j) % 15000
				buffer := pools.GetBuffer(size)

				// 写入一些数据
				buffer.WriteString("test data")

				// 放回池中
				pools.PutBuffer(buffer)
			}
		}(i)
	}

	// 并发获取统计
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			pools.GetStats()
			time.Sleep(time.Microsecond * 100)
		}
	}()

	// 等待完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 测试完成
	case <-time.After(10 * time.Second):
		t.Fatal("并发测试超时")
	}

	// 验证最终统计
	finalStats := pools.GetStats()
	totalOps := numGoroutines * numOperations

	totalGets := int64(0)
	totalPuts := int64(0)
	for _, stat := range finalStats {
		totalGets += stat.Gets
		totalPuts += stat.Puts
	}

	if totalGets != int64(totalOps) {
		t.Errorf("期望总获取次数 %d，实际: %d", totalOps, totalGets)
	}
	if totalPuts != int64(totalOps) {
		t.Errorf("期望总放回次数 %d，实际: %d", totalOps, totalPuts)
	}
}

// TestTieredBuffer_Methods 测试TieredBuffer的方法
func TestTieredBuffer_Methods(t *testing.T) {
	pools := NewTieredPools()
	buffer := pools.GetBuffer(1000)

	// 测试基本方法
	if buffer.Len() != 0 {
		t.Error("新buffer长度应该为0")
	}

	if buffer.Cap() <= 0 {
		t.Error("buffer容量应该大于0")
	}

	// 测试写入方法
	testData := "Hello, World!"
	n, err := buffer.WriteString(testData)
	if err != nil {
		t.Errorf("写入失败: %v", err)
	}
	if n != len(testData) {
		t.Errorf("期望写入 %d 字节，实际: %d", len(testData), n)
	}

	if buffer.String() != testData {
		t.Error("字符串内容不匹配")
	}

	if buffer.Len() != len(testData) {
		t.Errorf("期望长度 %d，实际: %d", len(testData), buffer.Len())
	}

	// 测试Reset
	buffer.Reset()
	if buffer.Len() != 0 {
		t.Error("Reset后长度应该为0")
	}
	if buffer.String() != "" {
		t.Error("Reset后内容应该为空")
	}

	pools.PutBuffer(buffer)
}

// TestStringCache_Coverage 测试字符串缓存覆盖
func TestStringCache_Coverage(t *testing.T) {
	cache := NewLRUCache(5)

	// 测试基本操作
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	if value, exists := cache.Get("key1"); !exists || value != "value1" {
		t.Error("字符串缓存获取失败")
	}

	if _, exists := cache.Get("nonexistent"); exists {
		t.Error("不存在的键不应该返回true")
	}

	// 测试容量限制
	for i := 3; i <= 10; i++ {
		cache.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	if cache.Size() > 5 {
		t.Errorf("缓存大小不应该超过容量限制，实际: %d", cache.Size())
	}

	// 测试LRU行为
	cache.Get("key6")                 // 访问key6使其成为最近使用
	cache.Put("new_key", "new_value") // 这应该淘汰除key6之外的某个键

	if _, exists := cache.Get("key6"); !exists {
		t.Error("最近访问的key6应该仍然存在")
	}
}
