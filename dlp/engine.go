package dlp

import (
	"errors"
	"fmt"
	"hash/fnv"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/darkit/slog/internal/common"
	"github.com/darkit/slog/internal/dlp/cachekey"
)

var (
	ErrInvalidMatcher = errors.New("invalid matcher configuration")
	ErrNotStruct      = errors.New("input must be a struct")
)

// putTieredStringBuilder 将字符串构建器放回分级池
func putTieredStringBuilder(builder *strings.Builder, expectedCapacity int) {
	common.GlobalTieredPools.PutStringBuilder(builder, expectedCapacity)
}

// cacheEntry 缓存条目
type cacheEntry struct {
	result string
	hits   int64 // 命中次数
}

// DlpEngine 定义脱敏引擎结构体
type DlpEngine struct {
	config                *DlpConfig
	searcher              *RegexSearcher           // 保留向后兼容
	structProcessor       *StructDesensitizer      // 新增：结构体脱敏器
	manager               *SecurityEnhancedManager // 新增：安全增强脱敏器管理器
	enabled               atomic.Bool
	cache                 *common.LRUCache // 结果缓存
	typesCache            []string         // 缓存支持的类型列表
	typesCacheMu          sync.RWMutex
	typesCacheKey         int64       // 缓存版本号
	usePluginArchitecture atomic.Bool // 是否使用插件架构
	cacheStats            struct {
		hits   int64
		misses int64
	}
}

// 计算文本哈希（用于短文本缓存键）
func hashText(text string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(text))
	return h.Sum64()
}

// NewDlpEngine 创建新的DLP引擎实例
func NewDlpEngine() *DlpEngine {
	engine := &DlpEngine{
		config:   GetConfig(),
		searcher: NewRegexSearcher(),
		cache:    common.NewLRUCache(1000), // 初始化LRU缓存，容量1000
	}
	engine.structProcessor = NewStructDesensitizer(engine) // 初始化结构体脱敏器
	engine.manager = NewSecurityEnhancedManager()          // 初始化安全增强脱敏器管理器
	engine.enabled.Store(false)
	engine.usePluginArchitecture.Store(false) // 默认不使用插件架构，保持向后兼容

	// 初始化默认脱敏器
	engine.initializeDefaultDesensitizers()

	return engine
}

// initializeDefaultDesensitizers 初始化默认脱敏器
func (e *DlpEngine) initializeDefaultDesensitizers() {
	// 注册所有增强版脱敏器（安全版本）
	// 注意：不默认注册中文姓名脱敏器，因为它容易误判普通文本
	desensitizers := []Desensitizer{
		NewEnhancedPhoneDesensitizer(),
		NewEnhancedEmailDesensitizer(),
		NewEnhancedBankCardDesensitizer(),
		// NewChineseNameDesensitizer(), // 中文姓名脱敏器容易误判，需要用户显式注册
	}

	for _, desensitizer := range desensitizers {
		if err := e.manager.RegisterDesensitizer(desensitizer); err != nil {
			// 日志记录错误，但不中断初始化
			continue
		}
	}
}

// Enable 启用DLP引擎
func (e *DlpEngine) Enable() {
	e.enabled.Store(true)
}

// Disable 禁用DLP引擎
func (e *DlpEngine) Disable() {
	e.enabled.Store(false)
}

// IsEnabled 检查DLP引擎是否启用
func (e *DlpEngine) IsEnabled() bool {
	return e.enabled.Load()
}

// EnablePluginArchitecture 启用插件架构
func (e *DlpEngine) EnablePluginArchitecture() {
	e.usePluginArchitecture.Store(true)
	e.manager.Enable()
}

// DisablePluginArchitecture 禁用插件架构，回退到传统模式
func (e *DlpEngine) DisablePluginArchitecture() {
	e.usePluginArchitecture.Store(false)
	e.manager.Disable()
}

// Version 返回当前规则版本（热更新计数）。
func (e *DlpEngine) Version() int64 {
	if e == nil || e.manager == nil {
		return 0
	}
	return e.manager.CurrentVersion()
}

// IsPluginArchitectureEnabled 检查是否启用插件架构
func (e *DlpEngine) IsPluginArchitectureEnabled() bool {
	return e.usePluginArchitecture.Load()
}

// GetDesensitizerManager 获取脱敏器管理器
func (e *DlpEngine) GetDesensitizerManager() *SecurityEnhancedManager {
	return e.manager
}

// getSupportedTypes 获取支持的类型（带缓存）
func (e *DlpEngine) getSupportedTypes() []string {
	// 获取当前searcher的版本号
	currentKey := e.searcher.getTypesVersion()

	e.typesCacheMu.RLock()
	if e.typesCacheKey == currentKey && e.typesCache != nil {
		defer e.typesCacheMu.RUnlock()
		return e.typesCache
	}
	e.typesCacheMu.RUnlock()

	// 需要更新缓存
	e.typesCacheMu.Lock()
	defer e.typesCacheMu.Unlock()

	// 双重检查
	if e.typesCacheKey == currentKey && e.typesCache != nil {
		return e.typesCache
	}

	e.typesCache = e.searcher.GetAllSupportedTypes()
	e.typesCacheKey = currentKey
	return e.typesCache
}

// DesensitizeText 对文本进行脱敏处理
func (e *DlpEngine) DesensitizeText(text string) string {
	if !e.IsEnabled() || text == "" {
		return text
	}

	// 对于超长文本，不使用缓存
	if len(text) > 5000 {
		return e.desensitizeTextWithoutCache(text)
	}

	// 使用 xxhash 的缓存键生成逻辑
	var cacheKey string
	if e.IsPluginArchitectureEnabled() {
		cacheKey = cachekey.Key("plugin", text)
	} else {
		cacheKey = cachekey.FastKey(text)
	}

	// 检查缓存
	if cached, found := e.cache.Get(cacheKey); found {
		atomic.AddInt64(&e.cacheStats.hits, 1)
		return cached.(*cacheEntry).result
	}

	atomic.AddInt64(&e.cacheStats.misses, 1)

	// 处理文本
	result := e.desensitizeTextWithoutCache(text)

	// 只缓存有变化的结果，避免占用太多内存
	if result != text {
		e.cache.Put(cacheKey, &cacheEntry{
			result: result,
			hits:   1,
		})
	}

	return result
}

// desensitizeTextWithoutCache 不使用缓存的文本脱敏处理（优化版本）
func (e *DlpEngine) desensitizeTextWithoutCache(text string) string {
	if e.IsPluginArchitectureEnabled() {
		// 使用插件架构进行自动检测和脱敏
		result, err := e.manager.AutoDetectAndProcess(text)
		if err != nil || result == nil {
			// 降级到传统模式
			return e.searcher.ReplaceAllTypes(text)
		}
		return result.Desensitized
	}

	// 使用传统的批量替换策略，一次性处理所有类型
	return e.searcher.ReplaceAllTypes(text)
}

// DesensitizeSpecificType 对指定类型的敏感信息进行脱敏
func (e *DlpEngine) DesensitizeSpecificType(text string, sensitiveType string) string {
	if !e.IsEnabled() || text == "" {
		return text
	}

	// 对于长文本，不使用缓存
	if len(text) > 5000 {
		if e.IsPluginArchitectureEnabled() {
			result, err := e.manager.ProcessWithType(sensitiveType, text)
			if err != nil || result == nil {
				// 降级到传统模式
				return e.searcher.ReplaceParallel(text, sensitiveType)
			}
			return result.Desensitized
		}
		return e.searcher.ReplaceParallel(text, sensitiveType)
	}

	// 检查缓存（使用优化的缓存键）
	var cacheKey string
	if e.IsPluginArchitectureEnabled() {
		cacheKey = cachekey.KeyWithContext("plugin", sensitiveType, text)
	} else {
		cacheKey = cachekey.KeyWithContext("legacy", sensitiveType, text)
	}

	if cached, found := e.cache.Get(cacheKey); found {
		return cached.(*cacheEntry).result
	}

	// 处理文本
	var result string
	if e.IsPluginArchitectureEnabled() {
		desensitizationResult, err := e.manager.ProcessWithType(sensitiveType, text)
		if err != nil || desensitizationResult == nil {
			// 降级到传统模式
			result = e.searcher.ReplaceParallel(text, sensitiveType)
		} else {
			result = desensitizationResult.Desensitized
		}
	} else {
		result = e.searcher.ReplaceParallel(text, sensitiveType)
	}

	// 只缓存有变化的结果
	if result != text {
		e.cache.Put(cacheKey, &cacheEntry{
			result: result,
			hits:   1,
		})
	}

	return result
}

// DesensitizeStruct 对结构体进行脱敏处理
func (e *DlpEngine) DesensitizeStruct(data interface{}) error {
	if !e.IsEnabled() {
		return nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("dlp")
		if tag == "" {
			continue
		}

		if field.Kind() == reflect.String {
			desensitized := e.DesensitizeSpecificType(field.String(), tag)
			field.SetString(desensitized)
		}
	}

	return nil
}

// DetectSensitiveInfo 检测文本中的所有敏感信息（优化版本）
func (e *DlpEngine) DetectSensitiveInfo(text string) map[string][]MatchResult {
	if !e.IsEnabled() || text == "" {
		return nil
	}

	// 使用批量检测，一次性检测所有类型
	return e.searcher.DetectAllTypes(text)
}

// RegisterCustomMatcher 注册自定义匹配器
func (e *DlpEngine) RegisterCustomMatcher(matcher *Matcher) error {
	if matcher.Pattern == "" || matcher.Name == "" {
		return ErrInvalidMatcher
	}

	regex, err := regexp.Compile(matcher.Pattern)
	if err != nil {
		return err
	}

	matcher.Regex = regex
	e.searcher.AddMatcher(matcher)

	// 清除类型缓存
	e.typesCacheMu.Lock()
	e.typesCache = nil
	e.typesCacheMu.Unlock()

	return nil
}

// GetSupportedTypes 获取所有支持的敏感信息类型
func (e *DlpEngine) GetSupportedTypes() []string {
	return e.getSupportedTypes()
}

// ClearCache 清除缓存
func (e *DlpEngine) ClearCache() {
	e.cache.Clear()
	atomic.StoreInt64(&e.cacheStats.hits, 0)
	atomic.StoreInt64(&e.cacheStats.misses, 0)
}

// GetCacheStats 获取缓存统计信息
func (e *DlpEngine) GetCacheStats() (hits, misses int64) {
	return atomic.LoadInt64(&e.cacheStats.hits), atomic.LoadInt64(&e.cacheStats.misses)
}

// DesensitizeStructAdvanced 高级结构体脱敏处理（新方法）
// 支持：嵌套结构体、slice/array、map、多种数据类型、递归处理
func (e *DlpEngine) DesensitizeStructAdvanced(data interface{}) error {
	if !e.IsEnabled() {
		return nil
	}
	return e.structProcessor.DesensitizeStructAdvanced(data)
}

// BatchDesensitizeStruct 批量结构体脱敏处理（新方法）
func (e *DlpEngine) BatchDesensitizeStruct(data interface{}) error {
	if !e.IsEnabled() {
		return nil
	}
	return e.structProcessor.BatchDesensitizeStruct(data)
}

// RegisterCustomDesensitizer 注册自定义脱敏器
func (e *DlpEngine) RegisterCustomDesensitizer(desensitizer Desensitizer) error {
	return e.manager.RegisterDesensitizer(desensitizer)
}

// UnregisterDesensitizer 注销脱敏器
func (e *DlpEngine) UnregisterDesensitizer(name string) error {
	return e.manager.UnregisterDesensitizer(name)
}

// ListRegisteredDesensitizers 列出所有已注册的脱敏器
func (e *DlpEngine) ListRegisteredDesensitizers() []string {
	return e.manager.ListDesensitizers()
}

// GetDesensitizerStats 获取脱敏器统计信息
func (e *DlpEngine) GetDesensitizerStats() ManagerStats {
	return e.manager.GetStats()
}

// EnableDesensitizer 启用特定脱敏器
func (e *DlpEngine) EnableDesensitizer(name string) error {
	if desensitizer, exists := e.manager.GetDesensitizer(name); exists {
		desensitizer.Enable()
		return nil
	}
	return fmt.Errorf("desensitizer '%s' not found", name)
}

// DisableDesensitizer 禁用特定脱敏器
func (e *DlpEngine) DisableDesensitizer(name string) error {
	if desensitizer, exists := e.manager.GetDesensitizer(name); exists {
		desensitizer.Disable()
		return nil
	}
	return fmt.Errorf("desensitizer '%s' not found", name)
}

// GetSupportedTypesWithPlugin 获取插件架构支持的所有类型
func (e *DlpEngine) GetSupportedTypesWithPlugin() map[string][]string {
	if e.IsPluginArchitectureEnabled() {
		return e.manager.GetTypeMapping()
	}
	return nil
}

// ClearDesensitizerCaches 清除所有脱敏器缓存
func (e *DlpEngine) ClearDesensitizerCaches() {
	e.manager.ClearAllCaches()
	e.ClearCache() // 也清除引擎自身的缓存
}
