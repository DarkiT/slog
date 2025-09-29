package common

import (
	"container/list"
	"sync"
)

// LRUCache LRU (Least Recently Used) 缓存实现
// 提供线程安全的LRU缓存，支持自动淘汰最久未使用的条目
type LRUCache struct {
	mu       sync.RWMutex
	capacity int                           // 缓存容量
	cache    map[interface{}]*list.Element // 键到链表节点的映射
	list     *list.List                    // 双向链表，维护访问顺序
	misses   int64                         // 全局未命中次数
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key   interface{} // 缓存键
	Value interface{} // 缓存值
	Hits  int64       // 命中次数
}

// LRUCacheStats LRU缓存统计信息
type LRUCacheStats struct {
	Size     int     `json:"size"`     // 当前大小
	Capacity int     `json:"capacity"` // 最大容量
	Hits     int64   `json:"hits"`     // 命中次数
	Misses   int64   `json:"misses"`   // 未命中次数
	HitRate  float64 `json:"hit_rate"` // 命中率
}

// NewLRUCache 创建新的LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100 // 默认容量
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[interface{}]*list.Element),
		list:     list.New(),
	}
}

// Get 获取缓存值
func (lru *LRUCache) Get(key interface{}) (interface{}, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, exists := lru.cache[key]; exists {
		// 移动到链表头部（最近使用）
		lru.list.MoveToFront(elem)

		// 更新命中统计
		entry := elem.Value.(*CacheEntry)
		entry.Hits++

		return entry.Value, true
	}

	// 未命中，增加miss计数
	lru.misses++
	return nil, false
}

// Put 设置缓存值
func (lru *LRUCache) Put(key interface{}, value interface{}) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	// 检查是否已存在
	if elem, exists := lru.cache[key]; exists {
		// 更新现有条目
		lru.list.MoveToFront(elem)
		entry := elem.Value.(*CacheEntry)
		entry.Value = value
		return
	}

	// 创建新条目
	entry := &CacheEntry{
		Key:   key,
		Value: value,
		Hits:  0,
	}

	// 添加到链表头部和映射中
	elem := lru.list.PushFront(entry)
	lru.cache[key] = elem

	// 检查是否超过容量
	if lru.list.Len() > lru.capacity {
		lru.evictOldest()
	}
}

// evictOldest 淘汰最久未使用的条目
func (lru *LRUCache) evictOldest() {
	if lru.list.Len() == 0 {
		return
	}

	// 获取链表尾部（最久未使用）的元素
	oldest := lru.list.Back()
	if oldest != nil {
		lru.removeElement(oldest)
	}
}

// removeElement 移除指定元素
func (lru *LRUCache) removeElement(elem *list.Element) {
	lru.list.Remove(elem)
	entry := elem.Value.(*CacheEntry)
	delete(lru.cache, entry.Key)
}

// Remove 移除指定键的缓存
func (lru *LRUCache) Remove(key interface{}) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.removeElement(elem)
		return true
	}

	return false
}

// Clear 清空所有缓存
func (lru *LRUCache) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.cache = make(map[interface{}]*list.Element)
	lru.list = list.New()
	lru.misses = 0 // 重置miss计数器
}

// Size 获取当前缓存大小
func (lru *LRUCache) Size() int {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	return lru.list.Len()
}

// Capacity 获取缓存容量
func (lru *LRUCache) Capacity() int {
	return lru.capacity
}

// SetCapacity 设置缓存容量
func (lru *LRUCache) SetCapacity(capacity int) {
	if capacity <= 0 {
		return
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.capacity = capacity

	// 如果新容量小于当前大小，需要淘汰多余的条目
	for lru.list.Len() > lru.capacity {
		lru.evictOldest()
	}
}

// GetStats 获取缓存统计信息
func (lru *LRUCache) GetStats() LRUCacheStats {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	var totalHits int64

	// 统计所有条目的命中次数
	for elem := lru.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*CacheEntry)
		totalHits += entry.Hits
	}

	// 使用全局miss计数器
	totalMisses := lru.misses

	// 计算命中率
	total := totalHits + totalMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(totalHits) / float64(total)
	}

	return LRUCacheStats{
		Size:     lru.list.Len(),
		Capacity: lru.capacity,
		Hits:     totalHits,
		Misses:   totalMisses,
		HitRate:  hitRate,
	}
}

// GetOldestKey 获取最久未使用的键（用于调试）
func (lru *LRUCache) GetOldestKey() interface{} {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if lru.list.Len() == 0 {
		return nil
	}

	oldest := lru.list.Back()
	if oldest == nil {
		return nil
	}

	entry := oldest.Value.(*CacheEntry)
	return entry.Key
}

// GetMostRecentKey 获取最近使用的键（用于调试）
func (lru *LRUCache) GetMostRecentKey() interface{} {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if lru.list.Len() == 0 {
		return nil
	}

	newest := lru.list.Front()
	if newest == nil {
		return nil
	}

	entry := newest.Value.(*CacheEntry)
	return entry.Key
}

// GetKeys 获取所有键（按访问顺序，从最新到最旧）
func (lru *LRUCache) GetKeys() []interface{} {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	keys := make([]interface{}, 0, lru.list.Len())
	for elem := lru.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*CacheEntry)
		keys = append(keys, entry.Key)
	}

	return keys
}

// Contains 检查是否包含指定键
func (lru *LRUCache) Contains(key interface{}) bool {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	_, exists := lru.cache[key]
	return exists
}

// Peek 查看缓存值但不更新访问顺序
func (lru *LRUCache) Peek(key interface{}) (interface{}, bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if elem, exists := lru.cache[key]; exists {
		entry := elem.Value.(*CacheEntry)
		return entry.Value, true
	}

	return nil, false
}

// LRUStringCache 字符串特化的LRU缓存
// 针对字符串键值进行优化的LRU缓存实现
type LRUStringCache struct {
	*LRUCache
}

// NewLRUStringCache 创建字符串LRU缓存
func NewLRUStringCache(capacity int) *LRUStringCache {
	return &LRUStringCache{
		LRUCache: NewLRUCache(capacity),
	}
}

// GetString 获取字符串值
func (lsc *LRUStringCache) GetString(key string) (string, bool) {
	if value, exists := lsc.Get(key); exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// PutString 设置字符串值
func (lsc *LRUStringCache) PutString(key string, value string) {
	lsc.Put(key, value)
}

// GetStringKeys 获取所有字符串键
func (lsc *LRUStringCache) GetStringKeys() []string {
	keys := lsc.GetKeys()
	stringKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		if str, ok := key.(string); ok {
			stringKeys = append(stringKeys, str)
		}
	}

	return stringKeys
}

// LRUCacheOption LRU缓存配置选项
type LRUCacheOption func(*LRUCache)

// WithCapacity 设置缓存容量
func WithCapacity(capacity int) LRUCacheOption {
	return func(cache *LRUCache) {
		cache.SetCapacity(capacity)
	}
}

// NewLRUCacheWithOptions 使用选项创建LRU缓存
func NewLRUCacheWithOptions(options ...LRUCacheOption) *LRUCache {
	cache := NewLRUCache(100) // 默认容量

	for _, option := range options {
		option(cache)
	}

	return cache
}
