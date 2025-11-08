package common

import (
	"strings"
	"sync"
)

// PoolMigrationManager 对象池迁移管理器
// 帮助逐步从传统sync.Pool迁移到分级对象池
type PoolMigrationManager struct {
	tieredPools *TieredPools

	// 迁移统计
	migrationStats struct {
		totalMigrations   int64
		successMigrations int64
		mu                sync.RWMutex
	}
}

// NewPoolMigrationManager 创建池迁移管理器
func NewPoolMigrationManager() *PoolMigrationManager {
	return &PoolMigrationManager{
		tieredPools: NewTieredPools(),
	}
}

// MigrateStringBuilderPool 将字符串构建器池迁移到分级池
// 这个函数可以替代现有的sync.Pool使用
func (pm *PoolMigrationManager) MigrateStringBuilderPool(oldPool *sync.Pool) *TieredStringBuilderPool {
	return &TieredStringBuilderPool{
		manager: pm,
		oldPool: oldPool, // 保持向后兼容
	}
}

// TieredStringBuilderPool 分级字符串构建器池包装器
// 提供与sync.Pool兼容的接口，但内部使用分级池
type TieredStringBuilderPool struct {
	manager *PoolMigrationManager
	oldPool *sync.Pool // 向后兼容的旧池
}

// Get 获取字符串构建器（兼容sync.Pool接口）
func (tsp *TieredStringBuilderPool) Get() interface{} {
	// 使用中等大小作为默认值
	return tsp.manager.tieredPools.GetStringBuilder(mediumStringCapacity)
}

// GetWithCapacity 根据期望容量获取字符串构建器
func (tsp *TieredStringBuilderPool) GetWithCapacity(expectedCapacity int) *strings.Builder {
	return tsp.manager.tieredPools.GetStringBuilder(expectedCapacity)
}

// Put 放回字符串构建器（兼容sync.Pool接口）
func (tsp *TieredStringBuilderPool) Put(x interface{}) {
	if builder, ok := x.(*strings.Builder); ok {
		// 根据当前容量放回到合适的池
		capacity := builder.Cap()
		tsp.manager.tieredPools.PutStringBuilder(builder, capacity)
	}
}

// PutWithCapacity 根据期望容量放回字符串构建器
func (tsp *TieredStringBuilderPool) PutWithCapacity(builder *strings.Builder, expectedCapacity int) {
	tsp.manager.tieredPools.PutStringBuilder(builder, expectedCapacity)
}

// SmartStringBuilderPool 智能字符串构建器池
// 自动根据使用模式选择最佳的池大小
type SmartStringBuilderPool struct {
	tieredPools *TieredPools

	// 使用模式统计
	usageStats struct {
		smallUsage  int64
		mediumUsage int64
		largeUsage  int64
		mu          sync.RWMutex
	}
}

// NewSmartStringBuilderPool 创建智能字符串构建器池
func NewSmartStringBuilderPool() *SmartStringBuilderPool {
	return &SmartStringBuilderPool{
		tieredPools: NewTieredPools(),
	}
}

// GetOptimal 根据使用历史获取最优大小的字符串构建器
func (ssp *SmartStringBuilderPool) GetOptimal() *strings.Builder {
	ssp.usageStats.mu.RLock()
	small := ssp.usageStats.smallUsage
	medium := ssp.usageStats.mediumUsage
	large := ssp.usageStats.largeUsage
	ssp.usageStats.mu.RUnlock()

	// 根据使用频率选择最佳大小
	total := small + medium + large
	if total == 0 {
		// 初始使用，选择中等大小
		return ssp.tieredPools.GetStringBuilder(mediumStringCapacity)
	}

	smallRatio := float64(small) / float64(total)
	mediumRatio := float64(medium) / float64(total)

	switch {
	case smallRatio > 0.6:
		return ssp.tieredPools.GetStringBuilder(smallStringCapacity)
	case mediumRatio > 0.4:
		return ssp.tieredPools.GetStringBuilder(mediumStringCapacity)
	default:
		return ssp.tieredPools.GetStringBuilder(largeStringCapacity)
	}
}

// PutWithUsageTracking 放回字符串构建器并记录使用模式
func (ssp *SmartStringBuilderPool) PutWithUsageTracking(builder *strings.Builder) {
	capacity := builder.Cap()

	// 记录使用模式
	ssp.usageStats.mu.Lock()
	switch {
	case capacity <= smallStringCapacity:
		ssp.usageStats.smallUsage++
	case capacity <= mediumStringCapacity:
		ssp.usageStats.mediumUsage++
	default:
		ssp.usageStats.largeUsage++
	}
	ssp.usageStats.mu.Unlock()

	// 放回池中
	ssp.tieredPools.PutStringBuilder(builder, capacity)
}

// GetUsageStats 获取使用统计
func (ssp *SmartStringBuilderPool) GetUsageStats() (small, medium, large int64) {
	ssp.usageStats.mu.RLock()
	defer ssp.usageStats.mu.RUnlock()
	return ssp.usageStats.smallUsage, ssp.usageStats.mediumUsage, ssp.usageStats.largeUsage
}

// BufferPoolAdapter 将现有buffer池适配到分级池
type BufferPoolAdapter struct {
	tieredPools *TieredPools

	// 适配器统计
	stats struct {
		adaptedGets int64
		adaptedPuts int64
		mu          sync.RWMutex
	}
}

// NewBufferPoolAdapter 创建buffer池适配器
func NewBufferPoolAdapter() *BufferPoolAdapter {
	return &BufferPoolAdapter{
		tieredPools: NewTieredPools(),
	}
}

// AdaptExistingBuffer 适配现有的buffer到分级池系统
func (bpa *BufferPoolAdapter) AdaptExistingBuffer(data []byte) *TieredBuffer {
	bpa.stats.mu.Lock()
	bpa.stats.adaptedGets++
	bpa.stats.mu.Unlock()

	// 根据数据大小选择合适的buffer
	expectedSize := len(data)
	buffer := bpa.tieredPools.GetBuffer(expectedSize)

	// 如果有现有数据，复制到新buffer
	if len(data) > 0 {
		buffer.Write(data)
	}

	return buffer
}

// ReleaseAdaptedBuffer 释放适配的buffer
func (bpa *BufferPoolAdapter) ReleaseAdaptedBuffer(buffer *TieredBuffer) {
	bpa.stats.mu.Lock()
	bpa.stats.adaptedPuts++
	bpa.stats.mu.Unlock()

	bpa.tieredPools.PutBuffer(buffer)
}

// GetAdapterStats 获取适配器统计
func (bpa *BufferPoolAdapter) GetAdapterStats() (gets, puts int64) {
	bpa.stats.mu.RLock()
	defer bpa.stats.mu.RUnlock()
	return bpa.stats.adaptedGets, bpa.stats.adaptedPuts
}

// 全局实例，便于系统集成
var (
	GlobalPoolMigrationManager = NewPoolMigrationManager()
	GlobalSmartStringPool      = NewSmartStringBuilderPool()
	GlobalBufferAdapter        = NewBufferPoolAdapter()
)

// 便捷函数，用于逐步迁移现有代码

// GetSmartStringBuilder 获取智能字符串构建器
func GetSmartStringBuilder() *strings.Builder {
	return GlobalSmartStringPool.GetOptimal()
}

// PutSmartStringBuilder 放回智能字符串构建器
func PutSmartStringBuilder(builder *strings.Builder) {
	GlobalSmartStringPool.PutWithUsageTracking(builder)
}

// AdaptBuffer 适配现有buffer数据
func AdaptBuffer(data []byte) *TieredBuffer {
	return GlobalBufferAdapter.AdaptExistingBuffer(data)
}

// ReleaseBuffer 释放适配的buffer
func ReleaseBuffer(buffer *TieredBuffer) {
	GlobalBufferAdapter.ReleaseAdaptedBuffer(buffer)
}

// MigrateStringPool 迁移字符串池的便捷函数
func MigrateStringPool(oldPool *sync.Pool) *TieredStringBuilderPool {
	return GlobalPoolMigrationManager.MigrateStringBuilderPool(oldPool)
}
