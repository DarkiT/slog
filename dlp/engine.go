package dlp

import (
	"errors"
	"fmt"
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

// cacheEntry 缓存条目
type cacheEntry struct {
	result string
	hits   int64 // 命中次数
}

const negativeCacheTextMaxLen = 128

// DlpEngine 定义脱敏引擎结构体
type DlpEngine struct {
	config          *DlpConfig
	searcher        *RegexSearcher
	structProcessor *StructDesensitizer // 新增：结构体脱敏器
	manager         *DefaultDesensitizerManager
	enabled         atomic.Bool
	cache           *common.LRUCache // 结果缓存
	typesCache      []string         // 缓存支持的类型列表
	typesCacheMu    sync.RWMutex
	typesCacheKey   int64 // 缓存版本号
	cacheStats      struct {
		hits   int64
		misses int64
	}
}

// NewDlpEngine 创建新的DLP引擎实例
func NewDlpEngine() *DlpEngine {
	engine := &DlpEngine{
		config:   GetConfig(),
		searcher: NewRegexSearcher(),
		cache:    common.NewLRUCache(1000), // 初始化LRU缓存，容量1000
	}
	engine.structProcessor = NewStructDesensitizer(engine) // 初始化结构体脱敏器
	engine.manager = NewDefaultDesensitizerManager()
	engine.enabled.Store(false)

	// 初始化默认脱敏器
	engine.initializeDefaultDesensitizers()

	return engine
}

// initializeDefaultDesensitizers 初始化默认脱敏器
func (e *DlpEngine) initializeDefaultDesensitizers() {
	// 注意：不默认注册中文姓名脱敏器，因为它容易误判普通文本
	desensitizers := []Desensitizer{
		NewEnhancedPhoneDesensitizer(),
		NewEnhancedEmailDesensitizer(),
		NewEnhancedBankCardDesensitizer(),
		NewEnhancedIDCardDesensitizer(),
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

// IsPluginArchitectureEnabled 兼容接口，固定返回 false。
func (e *DlpEngine) IsPluginArchitectureEnabled() bool {
	return false
}

// EnablePluginArchitecture 兼容接口，当前无操作。
func (e *DlpEngine) EnablePluginArchitecture() {}

// DisablePluginArchitecture 兼容接口，当前无操作。
func (e *DlpEngine) DisablePluginArchitecture() {}

// GetSupportedTypesWithPlugin 兼容接口，返回当前类型映射。
func (e *DlpEngine) GetSupportedTypesWithPlugin() map[string][]string {
	if e == nil || e.manager == nil {
		return nil
	}
	return e.manager.GetTypeMapping()
}

// Version 返回当前规则版本（热更新计数）。
func (e *DlpEngine) Version() int64 {
	if e == nil || e.manager == nil {
		return 0
	}
	return e.manager.CurrentVersion()
}

// GetDesensitizerManager 获取脱敏器管理器
func (e *DlpEngine) GetDesensitizerManager() *DefaultDesensitizerManager {
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

	if !mayContainSensitiveData(text) {
		return text
	}

	// 对于超长文本，不使用缓存
	if len(text) > 5000 {
		return e.desensitizeTextWithoutCache(text)
	}

	cacheKey := cachekey.FastKey(text)

	// 检查缓存
	if cached, found := e.cache.Get(cacheKey); found {
		atomic.AddInt64(&e.cacheStats.hits, 1)
		return cached.(*cacheEntry).result
	}

	atomic.AddInt64(&e.cacheStats.misses, 1)

	// 处理文本
	result := e.desensitizeTextWithoutCache(text)

	// 对短文本允许负缓存，避免安全消息被反复全量扫描。
	if result != text || len(text) <= negativeCacheTextMaxLen {
		e.cache.Put(cacheKey, &cacheEntry{
			result: result,
			hits:   1,
		})
	}

	return result
}

func mayContainSensitiveData(text string) bool {
	if text == "" {
		return false
	}

	hasAt := false
	hasDigit := false
	longAlphaNumRun := 0

	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch {
		case ch >= '0' && ch <= '9':
			hasDigit = true
			longAlphaNumRun++
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			longAlphaNumRun++
		default:
			longAlphaNumRun = 0
		}

		switch ch {
		case '@':
			hasAt = true
		case ':', '/', '.', '-', '_':
			if hasDigit || hasAt {
				return true
			}
		}

		if hasDigit && longAlphaNumRun >= 8 {
			return true
		}
	}

	if hasAt || hasDigit {
		return true
	}

	return strings.ContainsAny(text, "张李王赵钱孙周吴郑冯陈褚卫蒋沈韩杨朱秦尤许何吕施孔曹严华金魏陶姜戚谢邹喻柏水窦章云苏潘葛奚范彭郎鲁韦昌马苗凤花方俞任袁柳酆鲍史唐费廉岑薛雷贺倪汤滕殷罗毕郝邬安常乐于时傅皮卞齐康伍余元卜顾孟平黄和穆萧尹")
}

// desensitizeTextWithoutCache 不使用缓存的文本脱敏处理（优化版本）
func (e *DlpEngine) desensitizeTextWithoutCache(text string) string {
	// 先走管理器主路径，再用 regex 做补充覆盖，确保混合文本中的遗漏类型也能被脱敏。
	result, err := e.manager.AutoDetectAndProcess(text)
	if err != nil || result == nil {
		return e.searcher.ReplaceAllTypes(text)
	}
	managerResult := result.Desensitized
	if managerResult == text {
		return e.searcher.ReplaceAllTypes(text)
	}
	// manager 已处理部分内容时，再执行一次 regex 兜底补全。
	return e.searcher.ReplaceAllTypes(managerResult)
}

// DesensitizeSpecificType 对指定类型的敏感信息进行脱敏
func (e *DlpEngine) DesensitizeSpecificType(text string, sensitiveType string) string {
	if !e.IsEnabled() || text == "" {
		return text
	}

	// 对于长文本，不使用缓存
	if len(text) > 5000 {
		result, err := e.manager.ProcessWithType(sensitiveType, text)
		if err != nil || result == nil {
			return e.searcher.ReplaceParallel(text, sensitiveType)
		}
		if result.Desensitized == text {
			return e.searcher.ReplaceParallel(text, sensitiveType)
		}
		return result.Desensitized
	}

	cacheKey := cachekey.KeyWithContext("default", sensitiveType, text)

	if cached, found := e.cache.Get(cacheKey); found {
		return cached.(*cacheEntry).result
	}

	desensitizationResult, err := e.manager.ProcessWithType(sensitiveType, text)
	var result string
	if err != nil || desensitizationResult == nil {
		result = e.searcher.ReplaceParallel(text, sensitiveType)
	} else {
		result = desensitizationResult.Desensitized
		if result == text {
			result = e.searcher.ReplaceParallel(text, sensitiveType)
		}
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
func (e *DlpEngine) DesensitizeStruct(data any) error {
	if !e.IsEnabled() {
		return nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
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
	if err := e.searcher.AddMatcher(matcher); err != nil {
		return err
	}

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
func (e *DlpEngine) DesensitizeStructAdvanced(data any) error {
	if !e.IsEnabled() {
		return nil
	}
	return e.structProcessor.DesensitizeStructAdvanced(data)
}

// BatchDesensitizeStruct 批量结构体脱敏处理（新方法）
func (e *DlpEngine) BatchDesensitizeStruct(data any) error {
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

// ClearDesensitizerCaches 清除所有脱敏器缓存
func (e *DlpEngine) ClearDesensitizerCaches() {
	e.manager.ClearAllCaches()
	e.ClearCache() // 也清除引擎自身的缓存
}

// DisableMatchers 禁用指定的匹配器。
func (e *DlpEngine) DisableMatchers(matcherNames ...string) {
	e.searcher.DisableMatchers(matcherNames...)
}

// EnableMatchers 重新启用之前被禁用的匹配器。
func (e *DlpEngine) EnableMatchers(matcherNames ...string) {
	e.searcher.EnableMatchers(matcherNames...)
}

// SetMatcherEnabled 启用或禁用单个匹配器。
func (e *DlpEngine) SetMatcherEnabled(name string, enabled bool) {
	e.searcher.SetMatcherEnabled(name, enabled)
}

// IsMatcherDisabled 检查指定名称的匹配器是否被禁用
func (e *DlpEngine) IsMatcherDisabled(name string) bool {
	return e.searcher.IsMatcherDisabled(name)
}

// DisabledMatchers 返回所有被禁用的匹配器名称列表。
func (e *DlpEngine) DisabledMatchers() []string {
	return e.searcher.DisabledMatchers()
}

// EnabledMatchers 返回所有处于启用状态的匹配器名称列表。
func (e *DlpEngine) EnabledMatchers() []string {
	return e.searcher.EnabledMatchers()
}

// DesensitizeAttrsOnly 仅对属性值进行脱敏，不对消息文本进行脱敏。
// msg 为原始消息文本，原样返回；attrs 中的每个值会经过 DLP 脱敏处理后返回。
func (e *DlpEngine) DesensitizeAttrsOnly(msg string, attrs map[string]string) (string, map[string]string) {
	if !e.IsEnabled() || len(attrs) == 0 {
		return msg, attrs
	}

	desensitizedAttrs := make(map[string]string, len(attrs))
	for k, v := range attrs {
		desensitizedAttrs[k] = e.DesensitizeText(v)
	}
	return msg, desensitizedAttrs
}
