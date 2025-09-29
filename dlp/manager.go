package dlp

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultDesensitizerManager 默认脱敏器管理器实现
type DefaultDesensitizerManager struct {
	mu            sync.RWMutex
	desensitizers map[string]Desensitizer
	typeMapping   map[string][]string // 类型到脱敏器名称的映射
	enabled       atomic.Bool
	stats         ManagerStats
	logger        Logger
}

// initializeStats 初始化统计信息
func (dm *DefaultDesensitizerManager) initializeStats() {
	dm.stats = ManagerStats{
		TypeCoverage:       make(map[string]int),
		PerformanceMetrics: make(map[string]PerformanceMetrics),
	}
}

// RegisterDesensitizer 注册脱敏器
func (dm *DefaultDesensitizerManager) RegisterDesensitizer(desensitizer Desensitizer) error {
	if desensitizer == nil {
		return fmt.Errorf("desensitizer cannot be nil")
	}

	name := desensitizer.Name()
	if name == "" {
		return fmt.Errorf("desensitizer name cannot be empty")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 检查是否已存在
	if _, exists := dm.desensitizers[name]; exists {
		return fmt.Errorf("desensitizer '%s' already registered", name)
	}

	// 注册脱敏器
	dm.desensitizers[name] = desensitizer

	// 更新类型映射
	dm.updateTypeMapping(name, desensitizer)

	// 更新统计信息
	dm.updateStatsAfterRegistration(name, desensitizer)

	if dm.logger != nil {
		dm.logger.Debug("脱敏器已注册", "name", name)
	}

	return nil
}

// updateTypeMapping 更新类型映射
func (dm *DefaultDesensitizerManager) updateTypeMapping(name string, desensitizer Desensitizer) {
	// 如果是类型专用脱敏器，获取支持的类型
	if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
		for _, dataType := range typeSpecific.GetSupportedTypes() {
			if dm.typeMapping[dataType] == nil {
				dm.typeMapping[dataType] = make([]string, 0)
			}
			dm.typeMapping[dataType] = append(dm.typeMapping[dataType], name)
		}
	}
}

// updateStatsAfterRegistration 注册后更新统计信息
func (dm *DefaultDesensitizerManager) updateStatsAfterRegistration(name string, desensitizer Desensitizer) {
	dm.stats.TotalDesensitizers++
	if desensitizer.Enabled() {
		dm.stats.EnabledDesensitizers++
	}

	// 初始化性能指标
	dm.stats.PerformanceMetrics[name] = PerformanceMetrics{
		SuccessRate: 1.0, // 初始成功率为100%
	}

	// 更新类型覆盖
	if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
		for _, dataType := range typeSpecific.GetSupportedTypes() {
			dm.stats.TypeCoverage[dataType]++
		}
	}
}

// UnregisterDesensitizer 注销脱敏器
func (dm *DefaultDesensitizerManager) UnregisterDesensitizer(name string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	desensitizer, exists := dm.desensitizers[name]
	if !exists {
		return fmt.Errorf("desensitizer '%s' not found", name)
	}

	// 从类型映射中移除
	dm.removeFromTypeMapping(name, desensitizer)

	// 删除脱敏器
	delete(dm.desensitizers, name)

	// 更新统计信息
	dm.updateStatsAfterUnregistration(name, desensitizer)

	if dm.logger != nil {
		dm.logger.Debug("脱敏器已注销", "name", name)
	}

	return nil
}

// removeFromTypeMapping 从类型映射中移除
func (dm *DefaultDesensitizerManager) removeFromTypeMapping(name string, desensitizer Desensitizer) {
	if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
		for _, dataType := range typeSpecific.GetSupportedTypes() {
			if names := dm.typeMapping[dataType]; names != nil {
				// 移除指定名称
				filtered := make([]string, 0, len(names))
				for _, n := range names {
					if n != name {
						filtered = append(filtered, n)
					}
				}
				if len(filtered) == 0 {
					delete(dm.typeMapping, dataType)
				} else {
					dm.typeMapping[dataType] = filtered
				}
			}
		}
	}
}

// updateStatsAfterUnregistration 注销后更新统计信息
func (dm *DefaultDesensitizerManager) updateStatsAfterUnregistration(name string, desensitizer Desensitizer) {
	dm.stats.TotalDesensitizers--
	if desensitizer.Enabled() {
		dm.stats.EnabledDesensitizers--
	}

	// 删除性能指标
	delete(dm.stats.PerformanceMetrics, name)

	// 更新类型覆盖
	if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
		for _, dataType := range typeSpecific.GetSupportedTypes() {
			if count := dm.stats.TypeCoverage[dataType]; count > 1 {
				dm.stats.TypeCoverage[dataType]--
			} else {
				delete(dm.stats.TypeCoverage, dataType)
			}
		}
	}
}

// GetDesensitizer 获取指定名称的脱敏器
func (dm *DefaultDesensitizerManager) GetDesensitizer(name string) (Desensitizer, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	desensitizer, exists := dm.desensitizers[name]
	return desensitizer, exists
}

// GetDesensitizersForType 获取支持指定类型的所有脱敏器
func (dm *DefaultDesensitizerManager) GetDesensitizersForType(dataType string) []Desensitizer {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := dm.typeMapping[dataType]
	if len(names) == 0 {
		return nil
	}

	desensitizers := make([]Desensitizer, 0, len(names))
	for _, name := range names {
		if desensitizer, exists := dm.desensitizers[name]; exists && desensitizer.Enabled() {
			desensitizers = append(desensitizers, desensitizer)
		}
	}

	return desensitizers
}

// ListDesensitizers 列出所有已注册的脱敏器
func (dm *DefaultDesensitizerManager) ListDesensitizers() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := make([]string, 0, len(dm.desensitizers))
	for name := range dm.desensitizers {
		names = append(names, name)
	}
	return names
}

// EnableAll 启用所有脱敏器
func (dm *DefaultDesensitizerManager) EnableAll() {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, desensitizer := range dm.desensitizers {
		if !desensitizer.Enabled() {
			desensitizer.Enable()
		}
	}

	// 更新统计信息
	dm.stats.EnabledDesensitizers = dm.stats.TotalDesensitizers

	if dm.logger != nil {
		dm.logger.Debug("所有脱敏器已启用")
	}
}

// DisableAll 禁用所有脱敏器
func (dm *DefaultDesensitizerManager) DisableAll() {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, desensitizer := range dm.desensitizers {
		if desensitizer.Enabled() {
			desensitizer.Disable()
		}
	}

	// 更新统计信息
	dm.stats.EnabledDesensitizers = 0

	if dm.logger != nil {
		dm.logger.Debug("所有脱敏器已禁用")
	}
}

// GetStats 获取管理器统计信息
func (dm *DefaultDesensitizerManager) GetStats() ManagerStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// 创建统计信息的副本
	stats := ManagerStats{
		TotalDesensitizers:   dm.stats.TotalDesensitizers,
		EnabledDesensitizers: dm.stats.EnabledDesensitizers,
		TypeCoverage:         make(map[string]int),
		PerformanceMetrics:   make(map[string]PerformanceMetrics),
	}

	// 复制类型覆盖
	for k, v := range dm.stats.TypeCoverage {
		stats.TypeCoverage[k] = v
	}

	// 复制性能指标
	for k, v := range dm.stats.PerformanceMetrics {
		stats.PerformanceMetrics[k] = v
	}

	return stats
}

// ProcessWithDesensitizer 使用指定脱敏器处理数据
func (dm *DefaultDesensitizerManager) ProcessWithDesensitizer(name string, data string) (*DesensitizationResult, error) {
	desensitizer, exists := dm.GetDesensitizer(name)
	if !exists {
		return nil, fmt.Errorf("desensitizer '%s' not found", name)
	}

	if !desensitizer.Enabled() {
		return &DesensitizationResult{
			Original:     data,
			Desensitized: data, // 禁用时返回原数据
			Desensitizer: name,
			Error:        fmt.Errorf("desensitizer '%s' is disabled", name),
		}, nil
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行脱敏
	result, err := desensitizer.Desensitize(data)
	duration := time.Since(startTime).Nanoseconds()

	// 检查是否来自缓存
	cached := false
	if cacheable, ok := desensitizer.(CacheableDesensitizer); ok && cacheable.CacheEnabled() {
		// 这里可以添加更精确的缓存检测逻辑
		cached = duration < 1000 // 如果处理时间非常短，可能来自缓存
	}

	// 更新性能指标
	dm.updatePerformanceMetrics(name, duration, err == nil)

	return &DesensitizationResult{
		Original:     data,
		Desensitized: result,
		Desensitizer: name,
		Duration:     duration,
		Cached:       cached,
		Error:        err,
		Metadata:     make(map[string]interface{}),
	}, nil
}

// ProcessWithType 使用类型自动选择脱敏器处理数据
func (dm *DefaultDesensitizerManager) ProcessWithType(dataType string, data string) (*DesensitizationResult, error) {
	desensitizers := dm.GetDesensitizersForType(dataType)
	if len(desensitizers) == 0 {
		return &DesensitizationResult{
			Original:     data,
			Desensitized: data,
			DataType:     dataType,
			Error:        fmt.Errorf("no desensitizer found for type '%s'", dataType),
		}, fmt.Errorf("no desensitizer found for type '%s'", dataType)
	}

	// 使用第一个可用的脱敏器
	desensitizer := desensitizers[0]
	return dm.ProcessWithDesensitizer(desensitizer.Name(), data)
}

// updatePerformanceMetrics 更新性能指标
func (dm *DefaultDesensitizerManager) updatePerformanceMetrics(name string, duration int64, success bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	metrics := dm.stats.PerformanceMetrics[name]
	metrics.TotalCalls++
	metrics.TotalDuration += duration

	if success {
		metrics.SuccessRate = float64(metrics.TotalCalls-metrics.ErrorCount) / float64(metrics.TotalCalls)
	} else {
		metrics.ErrorCount++
		metrics.SuccessRate = float64(metrics.TotalCalls-metrics.ErrorCount) / float64(metrics.TotalCalls)
	}

	metrics.AverageDuration = float64(metrics.TotalDuration) / float64(metrics.TotalCalls)
	dm.stats.PerformanceMetrics[name] = metrics
}

// AutoDetectAndProcess 自动检测数据类型并处理
func (dm *DefaultDesensitizerManager) AutoDetectAndProcess(data string) (*DesensitizationResult, error) {
	if !dm.enabled.Load() {
		return &DesensitizationResult{
			Original:     data,
			Desensitized: data,
			Error:        fmt.Errorf("manager is disabled"),
		}, fmt.Errorf("manager is disabled")
	}

	// 检查是否是单一类型的纯净数据（如纯手机号、纯邮箱）
	var singleTypeDesensitizer Desensitizer
	var matchCount int

	func() {
		dm.mu.RLock()
		defer dm.mu.RUnlock()

		for _, desensitizer := range dm.desensitizers {
			if !desensitizer.Enabled() {
				continue
			}

			// 检查是否支持数据类型
			if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
				for _, dataType := range typeSpecific.GetSupportedTypes() {
					if typeSpecific.ValidateType(data, dataType) {
						matchCount++
						if singleTypeDesensitizer == nil {
							singleTypeDesensitizer = desensitizer
						}
						break // 避免同一个脱敏器重复计算
					}
				}
			}
		}
	}()

	// 如果只有一个脱敏器匹配，说明是单一类型数据，直接使用该脱敏器
	if matchCount == 1 && singleTypeDesensitizer != nil {
		if result, err := singleTypeDesensitizer.Desensitize(data); err == nil {
			return &DesensitizationResult{
				Original:     data,
				Desensitized: result,
				Error:        nil,
			}, nil
		}
	}

	// 如果多个脱敏器匹配，说明是混合文本，使用正则替换方式处理所有类型
	if matchCount > 1 {
		result := data

		func() {
			dm.mu.RLock()
			defer dm.mu.RUnlock()

			// 按照优先级顺序处理：手机号 -> 邮箱 -> 身份证 -> 银行卡 -> 中文姓名
			typeOrder := []string{"phone", "email", "id_card", "bank_card", "chinese_name"}

			for _, dataType := range typeOrder {
				for _, desensitizer := range dm.desensitizers {
					if !desensitizer.Enabled() {
						continue
					}

					if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
						for _, supportedType := range typeSpecific.GetSupportedTypes() {
							if supportedType == dataType {
								// 使用脱敏器的内部正则替换逻辑
								if processed, err := desensitizer.Desensitize(result); err == nil {
									result = processed
								}
								break
							}
						}
					}
				}
			}
		}()

		return &DesensitizationResult{
			Original:     data,
			Desensitized: result,
			Error:        nil,
		}, nil
	}

	// 没有找到匹配的脱敏器时，仍然尝试处理可能包含的敏感信息
	// 这对于长文本或混合文本非常重要
	result := data

	func() {
		dm.mu.RLock()
		defer dm.mu.RUnlock()

		// 按照优先级顺序处理所有类型
		typeOrder := []string{"phone", "email", "id_card", "bank_card"}

		for _, dataType := range typeOrder {
			for _, desensitizer := range dm.desensitizers {
				if !desensitizer.Enabled() {
					continue
				}

				if typeSpecific, ok := desensitizer.(TypeSpecificDesensitizer); ok {
					for _, supportedType := range typeSpecific.GetSupportedTypes() {
						if supportedType == dataType {
							// 使用脱敏器的内部正则替换逻辑
							if processed, err := desensitizer.Desensitize(result); err == nil {
								result = processed
							}
							break
						}
					}
				}
			}
		}
	}()

	return &DesensitizationResult{
		Original:     data,
		Desensitized: result,
		Error:        nil,
	}, nil
}

// Enable 启用管理器
func (dm *DefaultDesensitizerManager) Enable() {
	dm.enabled.Store(true)
	if dm.logger != nil {
		dm.logger.Debug("脱敏器管理器已启用")
	}
}

// Disable 禁用管理器
func (dm *DefaultDesensitizerManager) Disable() {
	dm.enabled.Store(false)
	if dm.logger != nil {
		dm.logger.Debug("脱敏器管理器已禁用")
	}
}

// IsEnabled 检查管理器是否启用
func (dm *DefaultDesensitizerManager) IsEnabled() bool {
	return dm.enabled.Load()
}

// GetTypeMapping 获取类型映射（只读）
func (dm *DefaultDesensitizerManager) GetTypeMapping() map[string][]string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	mapping := make(map[string][]string)
	for k, v := range dm.typeMapping {
		mapping[k] = make([]string, len(v))
		copy(mapping[k], v)
	}
	return mapping
}

// ClearAllCaches 清除所有脱敏器的缓存
func (dm *DefaultDesensitizerManager) ClearAllCaches() {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for name, desensitizer := range dm.desensitizers {
		if cacheable, ok := desensitizer.(CacheableDesensitizer); ok {
			cacheable.ClearCache()
			if dm.logger != nil {
				dm.logger.Debug("已清除脱敏器缓存", "name", name)
			}
		}
	}
}

// GetDetailedStats 获取详细统计信息
func (dm *DefaultDesensitizerManager) GetDetailedStats() map[string]interface{} {
	stats := dm.GetStats()

	detailed := map[string]interface{}{
		"total_desensitizers":   stats.TotalDesensitizers,
		"enabled_desensitizers": stats.EnabledDesensitizers,
		"type_coverage":         stats.TypeCoverage,
		"performance_metrics":   stats.PerformanceMetrics,
		"type_mapping":          dm.GetTypeMapping(),
		"manager_enabled":       dm.IsEnabled(),
	}

	// 添加缓存统计
	cacheStats := make(map[string]CacheStats)
	dm.mu.RLock()
	for name, desensitizer := range dm.desensitizers {
		if cacheable, ok := desensitizer.(CacheableDesensitizer); ok && cacheable.CacheEnabled() {
			cacheStats[name] = cacheable.GetCacheStats()
		}
	}
	dm.mu.RUnlock()
	detailed["cache_stats"] = cacheStats

	return detailed
}

// 全局脱敏器管理器实例
var (
	globalManager     *SecurityEnhancedManager
	globalManagerOnce sync.Once
)

// GetGlobalManager 获取全局脱敏器管理器
func GetGlobalManager() *SecurityEnhancedManager {
	globalManagerOnce.Do(func() {
		globalManager = NewSecurityEnhancedManager() // 使用安全增强版本
	})
	return globalManager
}

// SetGlobalLogger 设置全局管理器的日志记录器
func SetGlobalLogger(logger Logger) {
	if globalManager != nil {
		globalManager.logger = logger
	}
}
