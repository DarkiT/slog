package common

import (
	"strings"
	"sync"
	"sync/atomic"
)

// BufferSize 定义buffer大小级别
type BufferSize int

const (
	SmallBuffer  BufferSize = iota // 小buffer: 256B - 2KB
	MediumBuffer                   // 中buffer: 2KB - 8KB
	LargeBuffer                    // 大buffer: 8KB - 32KB
)

const (
	smallBufferMaxSize  = 2 * 1024 // 2KB
	smallBufferInitSize = 512      // 512B

	mediumBufferMaxSize  = 8 * 1024 // 8KB
	mediumBufferInitSize = 4 * 1024 // 4KB

	largeBufferMaxSize  = 32 * 1024 // 32KB
	largeBufferInitSize = 16 * 1024 // 16KB

	// 字符串构建器配置
	smallStringCapacity  = 256
	mediumStringCapacity = 1024
	largeStringCapacity  = 4096
)

// BufferPoolStats 对象池统计信息
type BufferPoolStats struct {
	Gets     int64   `json:"gets"`      // 获取次数
	Puts     int64   `json:"puts"`      // 放回次数
	News     int64   `json:"news"`      // 新建次数
	Discards int64   `json:"discards"`  // 丢弃次数
	PoolSize int     `json:"pool_size"` // 当前池大小
	HitRate  float64 `json:"hit_rate"`  // 命中率
}

// TieredBuffer 分级buffer
type TieredBuffer struct {
	data     []byte
	capacity int
	size     BufferSize
	useCount int32
}

// TieredPools 分级对象池管理器
type TieredPools struct {
	// Buffer pools
	smallBufferPool  *BufferPool
	mediumBufferPool *BufferPool
	largeBufferPool  *BufferPool

	// String builder pools
	smallStringPool  *StringBuilderPool
	mediumStringPool *StringBuilderPool
	largeStringPool  *StringBuilderPool

	// 统计信息
	stats map[BufferSize]*PoolStats
	mu    sync.RWMutex
}

// BufferPool 单级buffer池
type BufferPool struct {
	pool     sync.Pool
	maxSize  int
	initSize int
	stats    *PoolStats
}

// StringBuilderPool 字符串构建器池
type StringBuilderPool struct {
	pool     sync.Pool
	capacity int
	stats    *PoolStats
}

// PoolStats 池统计信息
type PoolStats struct {
	gets     int64
	puts     int64
	news     int64
	discards int64
}

// Gets 获取"获取"操作次数
func (ps *PoolStats) Gets() int64 {
	return atomic.LoadInt64(&ps.gets)
}

// Puts 获取"放回"操作次数
func (ps *PoolStats) Puts() int64 {
	return atomic.LoadInt64(&ps.puts)
}

// News 获取"新建"操作次数
func (ps *PoolStats) News() int64 {
	return atomic.LoadInt64(&ps.news)
}

// Discards 获取"丢弃"操作次数
func (ps *PoolStats) Discards() int64 {
	return atomic.LoadInt64(&ps.discards)
}

// NewTieredPools 创建分级对象池管理器
func NewTieredPools() *TieredPools {
	tp := &TieredPools{
		stats: make(map[BufferSize]*PoolStats),
	}

	// 初始化统计
	tp.stats[SmallBuffer] = &PoolStats{}
	tp.stats[MediumBuffer] = &PoolStats{}
	tp.stats[LargeBuffer] = &PoolStats{}

	// 初始化buffer池
	tp.smallBufferPool = NewBufferPool(smallBufferInitSize, smallBufferMaxSize, tp.stats[SmallBuffer])
	tp.mediumBufferPool = NewBufferPool(mediumBufferInitSize, mediumBufferMaxSize, tp.stats[MediumBuffer])
	tp.largeBufferPool = NewBufferPool(largeBufferInitSize, largeBufferMaxSize, tp.stats[LargeBuffer])

	// 初始化字符串构建器池
	tp.smallStringPool = NewStringBuilderPool(smallStringCapacity, tp.stats[SmallBuffer])
	tp.mediumStringPool = NewStringBuilderPool(mediumStringCapacity, tp.stats[MediumBuffer])
	tp.largeStringPool = NewStringBuilderPool(largeStringCapacity, tp.stats[LargeBuffer])

	return tp
}

// NewBufferPool 创建buffer池
func NewBufferPool(initSize, maxSize int, stats *PoolStats) *BufferPool {
	bp := &BufferPool{
		maxSize:  maxSize,
		initSize: initSize,
		stats:    stats,
	}

	bp.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&stats.news, 1)
			return &TieredBuffer{
				data:     make([]byte, 0, initSize),
				capacity: initSize,
			}
		},
	}

	return bp
}

// NewStringBuilderPool 创建字符串构建器池
func NewStringBuilderPool(capacity int, stats *PoolStats) *StringBuilderPool {
	sp := &StringBuilderPool{
		capacity: capacity,
		stats:    stats,
	}

	sp.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&stats.news, 1)
			builder := &strings.Builder{}
			builder.Grow(capacity)
			return builder
		},
	}

	return sp
}

// GetBuffer 根据期望大小获取最适合的buffer
func (tp *TieredPools) GetBuffer(expectedSize int) *TieredBuffer {
	switch {
	case expectedSize <= smallBufferMaxSize:
		return tp.getBufferFromPool(tp.smallBufferPool, SmallBuffer)
	case expectedSize <= mediumBufferMaxSize:
		return tp.getBufferFromPool(tp.mediumBufferPool, MediumBuffer)
	default:
		return tp.getBufferFromPool(tp.largeBufferPool, LargeBuffer)
	}
}

// getBufferFromPool 从指定池获取buffer
func (tp *TieredPools) getBufferFromPool(pool *BufferPool, size BufferSize) *TieredBuffer {
	atomic.AddInt64(&pool.stats.gets, 1)
	buffer := pool.pool.Get().(*TieredBuffer)
	buffer.size = size
	buffer.Reset()
	return buffer
}

// GetStringBuilder 根据期望容量获取字符串构建器
func (tp *TieredPools) GetStringBuilder(expectedCapacity int) *strings.Builder {
	var builder *strings.Builder
	switch {
	case expectedCapacity <= smallStringCapacity:
		builder = tp.getStringBuilderFromPool(tp.smallStringPool)
	case expectedCapacity <= mediumStringCapacity:
		builder = tp.getStringBuilderFromPool(tp.mediumStringPool)
	default:
		builder = tp.getStringBuilderFromPool(tp.largeStringPool)
	}

	// 确保容量满足要求
	if builder.Cap() < expectedCapacity {
		builder.Grow(expectedCapacity - builder.Cap())
	}

	return builder
}

// getStringBuilderFromPool 从指定池获取字符串构建器
func (tp *TieredPools) getStringBuilderFromPool(pool *StringBuilderPool) *strings.Builder {
	atomic.AddInt64(&pool.stats.gets, 1)
	builder := pool.pool.Get().(*strings.Builder)
	builder.Reset()
	return builder
}

// PutBuffer 将buffer放回对应的池
func (tp *TieredPools) PutBuffer(buffer *TieredBuffer) {
	if buffer == nil {
		return
	}

	var pool *BufferPool
	switch buffer.size {
	case SmallBuffer:
		pool = tp.smallBufferPool
	case MediumBuffer:
		pool = tp.mediumBufferPool
	case LargeBuffer:
		pool = tp.largeBufferPool
	default:
		return // 未知大小，直接丢弃
	}

	// 检查是否适合放回池中
	if cap(buffer.data) <= pool.maxSize && atomic.LoadInt32(&buffer.useCount) < 1000 {
		atomic.AddInt64(&pool.stats.puts, 1)
		pool.pool.Put(buffer)
	} else {
		atomic.AddInt64(&pool.stats.discards, 1)
	}
}

// PutStringBuilder 将字符串构建器放回对应的池
func (tp *TieredPools) PutStringBuilder(builder *strings.Builder, expectedCapacity int) {
	if builder == nil {
		return
	}

	var pool *StringBuilderPool
	switch {
	case expectedCapacity <= smallStringCapacity:
		pool = tp.smallStringPool
	case expectedCapacity <= mediumStringCapacity:
		pool = tp.mediumStringPool
	default:
		pool = tp.largeStringPool
	}

	// 检查容量是否合适
	if builder.Cap() <= pool.capacity*2 { // 允许一定的容量膨胀
		atomic.AddInt64(&pool.stats.puts, 1)
		pool.pool.Put(builder)
	} else {
		atomic.AddInt64(&pool.stats.discards, 1)
	}
}

// GetStats 获取所有池的统计信息
func (tp *TieredPools) GetStats() map[BufferSize]BufferPoolStats {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make(map[BufferSize]BufferPoolStats)

	for size, stats := range tp.stats {
		gets := atomic.LoadInt64(&stats.gets)
		puts := atomic.LoadInt64(&stats.puts)
		news := atomic.LoadInt64(&stats.news)
		discards := atomic.LoadInt64(&stats.discards)

		hitRate := 0.0
		if gets > 0 {
			hitRate = float64(gets-news) / float64(gets) * 100
		}

		result[size] = BufferPoolStats{
			Gets:     gets,
			Puts:     puts,
			News:     news,
			Discards: discards,
			HitRate:  hitRate,
		}
	}

	return result
}

// Reset 重置buffer
func (tb *TieredBuffer) Reset() {
	tb.data = tb.data[:0]
	atomic.StoreInt32(&tb.useCount, 0)
}

// Write 实现io.Writer接口
func (tb *TieredBuffer) Write(p []byte) (n int, err error) {
	atomic.AddInt32(&tb.useCount, 1)
	tb.data = append(tb.data, p...)
	return len(p), nil
}

// WriteString 写入字符串
func (tb *TieredBuffer) WriteString(s string) (n int, err error) {
	atomic.AddInt32(&tb.useCount, 1)
	tb.data = append(tb.data, s...)
	return len(s), nil
}

// String 返回字符串内容
func (tb *TieredBuffer) String() string {
	return string(tb.data)
}

// Bytes 返回字节内容
func (tb *TieredBuffer) Bytes() []byte {
	return tb.data
}

// Len 返回当前长度
func (tb *TieredBuffer) Len() int {
	return len(tb.data)
}

// Cap 返回容量
func (tb *TieredBuffer) Cap() int {
	return cap(tb.data)
}

// Size 返回buffer大小级别
func (tb *TieredBuffer) Size() BufferSize {
	return tb.size
}

// 全局分级池实例
var GlobalTieredPools = NewTieredPools()

// 便捷函数

// GetSmallBuffer 获取小buffer
func GetSmallBuffer() *TieredBuffer {
	return GlobalTieredPools.GetBuffer(smallBufferInitSize)
}

// GetMediumBuffer 获取中buffer
func GetMediumBuffer() *TieredBuffer {
	return GlobalTieredPools.GetBuffer(mediumBufferInitSize)
}

// GetLargeBuffer 获取大buffer
func GetLargeBuffer() *TieredBuffer {
	return GlobalTieredPools.GetBuffer(largeBufferInitSize)
}

// GetSmallStringBuilder 获取小容量字符串构建器
func GetSmallStringBuilder() *strings.Builder {
	return GlobalTieredPools.GetStringBuilder(smallStringCapacity)
}

// GetMediumStringBuilder 获取中容量字符串构建器
func GetMediumStringBuilder() *strings.Builder {
	return GlobalTieredPools.GetStringBuilder(mediumStringCapacity)
}

// GetLargeStringBuilder 获取大容量字符串构建器
func GetLargeStringBuilder() *strings.Builder {
	return GlobalTieredPools.GetStringBuilder(largeStringCapacity)
}

// PutBuffer 放回buffer到全局池
func PutBuffer(buffer *TieredBuffer) {
	GlobalTieredPools.PutBuffer(buffer)
}

// PutStringBuilder 放回字符串构建器到全局池
func PutStringBuilder(builder *strings.Builder) {
	// 根据当前容量判断应该放到哪个池
	capacity := builder.Cap()
	GlobalTieredPools.PutStringBuilder(builder, capacity)
}
