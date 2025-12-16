package common

import (
	"strconv"
	"testing"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(3)

	// 测试添加和获取
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	if cache.Size() != 3 {
		t.Errorf("Size() = %d, want 3", cache.Size())
	}

	// 测试获取
	if value, exists := cache.Get("key1"); !exists || value != "value1" {
		t.Error("Should find key1")
	}

	// 测试容量限制
	cache.Put("key4", "value4") // 应该淘汰key2（最久未使用）

	if cache.Size() != 3 {
		t.Errorf("Size should remain 3 after capacity limit, got %d", cache.Size())
	}

	// key2应该被淘汰
	if _, exists := cache.Get("key2"); exists {
		t.Error("key2 should be evicted")
	}

	// key1应该还在（因为刚被访问过）
	if _, exists := cache.Get("key1"); !exists {
		t.Error("key1 should still exist")
	}
}

func TestLRUCache_LRUOrder(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	// 访问a，使其成为最近使用的
	cache.Get("a")

	// 添加新元素，应该淘汰b（最久未使用）
	cache.Put("d", 4)

	if _, exists := cache.Get("b"); exists {
		t.Error("b should be evicted")
	}

	if _, exists := cache.Get("a"); !exists {
		t.Error("a should still exist")
	}

	if _, exists := cache.Get("c"); !exists {
		t.Error("c should still exist")
	}

	if _, exists := cache.Get("d"); !exists {
		t.Error("d should exist")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	// 更新现有键
	cache.Put("key1", "updated_value1")

	if cache.Size() != 2 {
		t.Errorf("Size should remain 2 after update, got %d", cache.Size())
	}

	if value, exists := cache.Get("key1"); !exists || value != "updated_value1" {
		t.Error("key1 should be updated")
	}
}

func TestLRUCache_Remove(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	// 测试移除
	if !cache.Remove("key1") {
		t.Error("Remove should return true for existing key")
	}

	if cache.Size() != 1 {
		t.Errorf("Size should be 1 after removal, got %d", cache.Size())
	}

	if _, exists := cache.Get("key1"); exists {
		t.Error("key1 should be removed")
	}

	// 测试移除不存在的键
	if cache.Remove("nonexistent") {
		t.Error("Remove should return false for non-existing key")
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size should be 0 after clear, got %d", cache.Size())
	}

	if _, exists := cache.Get("key1"); exists {
		t.Error("Cache should be empty after clear")
	}
}

func TestLRUCache_SetCapacity(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// 减少容量，应该淘汰最久未使用的元素
	cache.SetCapacity(2)

	if cache.Size() != 2 {
		t.Errorf("Size should be 2 after capacity reduction, got %d", cache.Size())
	}

	if cache.Capacity() != 2 {
		t.Errorf("Capacity should be 2, got %d", cache.Capacity())
	}

	// 增加容量
	cache.SetCapacity(5)

	if cache.Capacity() != 5 {
		t.Errorf("Capacity should be 5, got %d", cache.Capacity())
	}
}

func TestLRUCache_Stats(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	// 多次访问以增加命中统计
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")

	stats := cache.GetStats()

	if stats.Size != 2 {
		t.Errorf("Stats size = %d, want 2", stats.Size)
	}

	if stats.Capacity != 3 {
		t.Errorf("Stats capacity = %d, want 3", stats.Capacity)
	}

	if stats.Hits != 3 {
		t.Errorf("Stats hits = %d, want 3", stats.Hits)
	}
}

func TestLRUCache_Contains(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Put("key1", "value1")

	if !cache.Contains("key1") {
		t.Error("Should contain key1")
	}

	if cache.Contains("nonexistent") {
		t.Error("Should not contain nonexistent key")
	}
}

func TestLRUCache_Peek(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Peek不应该改变访问顺序
	if value, exists := cache.Peek("key1"); !exists || value != "value1" {
		t.Error("Peek should return correct value")
	}

	// 添加新元素，key1应该被淘汰（因为Peek没有更新访问顺序）
	cache.Put("key4", "value4")

	if _, exists := cache.Get("key1"); exists {
		t.Error("key1 should be evicted after peek")
	}
}

func TestLRUCache_GetKeysOrder(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("first", 1)
	cache.Put("second", 2)
	cache.Put("third", 3)

	// 访问first，使其成为最近使用的
	cache.Get("first")

	keys := cache.GetKeys()

	if len(keys) != 3 {
		t.Errorf("Should have 3 keys, got %d", len(keys))
	}

	// first应该是最新的（在最前面）
	if keys[0] != "first" {
		t.Errorf("Most recent key should be 'first', got %v", keys[0])
	}
}

func TestLRUCache_OldestAndNewest(t *testing.T) {
	cache := NewLRUCache(3)

	cache.Put("oldest", 1)
	cache.Put("middle", 2)
	cache.Put("newest", 3)

	oldestKey := cache.GetOldestKey()
	if oldestKey != "oldest" {
		t.Errorf("Oldest key should be 'oldest', got %v", oldestKey)
	}

	newestKey := cache.GetMostRecentKey()
	if newestKey != "newest" {
		t.Errorf("Newest key should be 'newest', got %v", newestKey)
	}

	// 访问oldest，使其成为最新的
	cache.Get("oldest")

	newestKey = cache.GetMostRecentKey()
	if newestKey != "oldest" {
		t.Errorf("After access, newest key should be 'oldest', got %v", newestKey)
	}

	oldestKey = cache.GetOldestKey()
	if oldestKey != "middle" {
		t.Errorf("After access, oldest key should be 'middle', got %v", oldestKey)
	}
}

func TestLRUStringCache(t *testing.T) {
	cache := NewLRUStringCache(3)

	cache.PutString("key1", "value1")
	cache.PutString("key2", "value2")

	if value, exists := cache.GetString("key1"); !exists || value != "value1" {
		t.Error("String cache should work correctly")
	}

	keys := cache.GetStringKeys()
	if len(keys) != 2 {
		t.Errorf("Should have 2 string keys, got %d", len(keys))
	}
}

func TestLRUCache_Concurrency(t *testing.T) {
	cache := NewLRUCache(100)
	done := make(chan bool, 100)

	// 并发写入
	for i := 0; i < 50; i++ {
		go func(id int) {
			key := strconv.Itoa(id)
			cache.Put(key, id)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 50; i++ {
		go func(id int) {
			key := strconv.Itoa(id % 25) // 读取部分存在的键
			cache.Get(key)
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证缓存状态
	if cache.Size() > cache.Capacity() {
		t.Error("Cache size should not exceed capacity")
	}
}

func TestLRUCache_EdgeCases(t *testing.T) {
	// 测试容量为0
	cache := NewLRUCache(0)
	if cache.Capacity() != 100 { // 应该使用默认容量
		t.Errorf("Should use default capacity when given 0")
	}

	// 测试负容量
	cache2 := NewLRUCache(-1)
	if cache2.Capacity() != 100 { // 应该使用默认容量
		t.Errorf("Should use default capacity when given negative value")
	}

	// 测试空缓存的操作
	emptyCache := NewLRUCache(3)
	if _, exists := emptyCache.Get("nonexistent"); exists {
		t.Error("Empty cache should not contain any keys")
	}

	if emptyCache.GetOldestKey() != nil {
		t.Error("Empty cache should return nil for oldest key")
	}

	if emptyCache.GetMostRecentKey() != nil {
		t.Error("Empty cache should return nil for newest key")
	}

	keys := emptyCache.GetKeys()
	if len(keys) != 0 {
		t.Error("Empty cache should return empty key slice")
	}
}

func TestLRUCacheWithOptions(t *testing.T) {
	cache := NewLRUCacheWithOptions(
		WithCapacity(50),
	)

	if cache.Capacity() != 50 {
		t.Errorf("Options should set capacity to 50, got %d", cache.Capacity())
	}
}

// 性能基准测试

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRUCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 1000)
		cache.Put(key, i)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache(1000)

	// 预填充缓存
	for i := 0; i < 1000; i++ {
		cache.Put(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 1000)
		cache.Get(key)
	}
}

func BenchmarkLRUCache_Mixed(b *testing.B) {
	cache := NewLRUCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 1000)
		if i%3 == 0 {
			cache.Put(key, i)
		} else {
			cache.Get(key)
		}
	}
}

func BenchmarkLRUStringCache_GetString(b *testing.B) {
	cache := NewLRUStringCache(1000)

	// 预填充缓存
	for i := 0; i < 1000; i++ {
		cache.PutString(strconv.Itoa(i), "value"+strconv.Itoa(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 1000)
		cache.GetString(key)
	}
}

func BenchmarkLRUCache_ConcurrentAccess(b *testing.B) {
	cache := NewLRUCache(1000)

	// 预填充缓存
	for i := 0; i < 1000; i++ {
		cache.Put(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := strconv.Itoa(i % 1000)
			if i%4 == 0 {
				cache.Put(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}
