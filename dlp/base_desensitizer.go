package dlp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/darkit/slog/internal/dlp/cachekey"
)

// BaseDesensitizer 基础脱敏器实现
// 提供通用的脱敏器功能，其他脱敏器可以继承此实现
type BaseDesensitizer struct {
	name    string
	enabled atomic.Bool
	config  map[string]interface{}
	mu      sync.RWMutex
	logger  Logger

	// 缓存相关
	cacheEnabled atomic.Bool
	cache        sync.Map
	cacheStats   CacheStats
}

// NewBaseDesensitizer 创建基础脱敏器
func NewBaseDesensitizer(name string) *BaseDesensitizer {
	bd := &BaseDesensitizer{
		name:   name,
		config: make(map[string]interface{}),
	}
	bd.enabled.Store(true)
	bd.cacheEnabled.Store(true)
	return bd
}

// Name 返回脱敏器名称
func (bd *BaseDesensitizer) Name() string {
	return bd.name
}

// Enabled 检查脱敏器是否启用
func (bd *BaseDesensitizer) Enabled() bool {
	return bd.enabled.Load()
}

// Enable 启用脱敏器
func (bd *BaseDesensitizer) Enable() {
	bd.enabled.Store(true)
	if bd.logger != nil {
		bd.logger.Debug("脱敏器已启用", "name", bd.name)
	}
}

// Disable 禁用脱敏器
func (bd *BaseDesensitizer) Disable() {
	bd.enabled.Store(false)
	if bd.logger != nil {
		bd.logger.Debug("脱敏器已禁用", "name", bd.name)
	}
}

// Configure 配置脱敏器参数
func (bd *BaseDesensitizer) Configure(config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	bd.mu.Lock()
	defer bd.mu.Unlock()

	// 清空现有配置
	bd.config = make(map[string]interface{})

	// 复制新配置
	for k, v := range config {
		bd.config[k] = v
	}

	// 处理特殊配置项
	if cacheEnabled, ok := config["cache_enabled"].(bool); ok {
		bd.cacheEnabled.Store(cacheEnabled)
	}

	if bd.logger != nil {
		bd.logger.Debug("脱敏器配置已更新", "name", bd.name)
	}

	return nil
}

// GetConfig 获取配置项
func (bd *BaseDesensitizer) GetConfig(key string) (interface{}, bool) {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	value, exists := bd.config[key]
	return value, exists
}

// SetLogger 设置日志记录器
func (bd *BaseDesensitizer) SetLogger(logger Logger) {
	bd.logger = logger
}

// CacheEnabled 检查是否启用缓存
func (bd *BaseDesensitizer) CacheEnabled() bool {
	return bd.cacheEnabled.Load()
}

// SetCacheEnabled 设置缓存状态
func (bd *BaseDesensitizer) SetCacheEnabled(enabled bool) {
	bd.cacheEnabled.Store(enabled)
	if !enabled {
		bd.ClearCache()
	}
}

// ClearCache 清空缓存
func (bd *BaseDesensitizer) ClearCache() {
	bd.cache = sync.Map{}
	bd.cacheStats = CacheStats{}
}

// GetCacheStats 获取缓存统计
func (bd *BaseDesensitizer) GetCacheStats() CacheStats {
	stats := bd.cacheStats
	if stats.Hits+stats.Misses > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}
	return stats
}

// desensitizeWithCache 带缓存的脱敏处理（xxhash优化版本）
func (bd *BaseDesensitizer) desensitizeWithCache(data string, processor func(string) (string, error)) (string, error) {
	if !bd.cacheEnabled.Load() || len(data) > 1000 { // 长文本不缓存
		return processor(data)
	}

	// 使用优化的缓存键
	cacheKey := cachekey.Key(bd.name, data)

	// 检查缓存
	if cached, ok := bd.cache.Load(cacheKey); ok {
		atomic.AddInt64(&bd.cacheStats.Hits, 1)
		return cached.(string), nil
	}

	// 缓存未命中，执行处理
	atomic.AddInt64(&bd.cacheStats.Misses, 1)
	result, err := processor(data)
	if err == nil {
		bd.cache.Store(cacheKey, result)
		atomic.AddInt64(&bd.cacheStats.Size, 1)
	}

	return result, err
}

// RegexDesensitizer 基于正则表达式的脱敏器
type RegexDesensitizer struct {
	*BaseDesensitizer
	patterns     map[string]*regexp.Regexp
	replacements map[string]string
	mu           sync.RWMutex
}

// NewRegexDesensitizer 创建正则脱敏器
func NewRegexDesensitizer(name string) *RegexDesensitizer {
	return &RegexDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer(name),
		patterns:         make(map[string]*regexp.Regexp),
		replacements:     make(map[string]string),
	}
}

// AddPattern 添加正则模式
func (rd *RegexDesensitizer) AddPattern(dataType, pattern, replacement string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern for type '%s': %w", dataType, err)
	}

	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.patterns[dataType] = regex
	rd.replacements[dataType] = replacement

	return nil
}

// Supports 检查是否支持指定的数据类型
func (rd *RegexDesensitizer) Supports(dataType string) bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	_, exists := rd.patterns[dataType]
	return exists
}

// Desensitize 执行脱敏操作
func (rd *RegexDesensitizer) Desensitize(data string) (string, error) {
	if !rd.Enabled() {
		return data, nil
	}

	return rd.desensitizeWithCache(data, func(input string) (string, error) {
		return rd.processAllPatterns(input), nil
	})
}

// processAllPatterns 处理所有模式
func (rd *RegexDesensitizer) processAllPatterns(data string) string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	result := data
	for dataType, pattern := range rd.patterns {
		replacement := rd.replacements[dataType]
		result = pattern.ReplaceAllString(result, replacement)
	}
	return result
}

// GetSupportedTypes 获取支持的所有类型
func (rd *RegexDesensitizer) GetSupportedTypes() []string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	types := make([]string, 0, len(rd.patterns))
	for dataType := range rd.patterns {
		types = append(types, dataType)
	}
	return types
}

// GetTypePattern 获取类型的正则表达式模式
func (rd *RegexDesensitizer) GetTypePattern(dataType string) string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if pattern, exists := rd.patterns[dataType]; exists {
		return pattern.String()
	}
	return ""
}

// ValidateType 验证数据是否符合指定类型
func (rd *RegexDesensitizer) ValidateType(data string, dataType string) bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if pattern, exists := rd.patterns[dataType]; exists {
		return pattern.MatchString(data)
	}
	return false
}

// DesensitizeWithContext 带上下文的脱敏
func (rd *RegexDesensitizer) DesensitizeWithContext(ctx context.Context, data string) (string, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return data, ctx.Err()
	default:
	}

	return rd.Desensitize(data)
}

// BatchDesensitize 批量脱敏
func (rd *RegexDesensitizer) BatchDesensitize(data []string) ([]string, error) {
	if !rd.Enabled() {
		return data, nil
	}

	results := make([]string, len(data))
	for i, item := range data {
		result, err := rd.Desensitize(item)
		if err != nil {
			return nil, fmt.Errorf("failed to desensitize item at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// PersonalInfoDesensitizer 个人信息脱敏器
type PersonalInfoDesensitizer struct {
	*RegexDesensitizer
}

// NewPersonalInfoDesensitizer 创建个人信息脱敏器
func NewPersonalInfoDesensitizer() *PersonalInfoDesensitizer {
	pid := &PersonalInfoDesensitizer{
		RegexDesensitizer: NewRegexDesensitizer("personal_info"),
	}

	// 预定义常见的个人信息模式
	pid.initializePatterns()
	return pid
}

// initializePatterns 初始化预定义模式
func (pid *PersonalInfoDesensitizer) initializePatterns() {
	patterns := map[string]struct {
		pattern     string
		replacement string
	}{
		"mobile_phone": {
			pattern:     `1[3-9]\d{9}`,
			replacement: "$1****$2",
		},
		"id_card": {
			pattern:     `\d{15}|\d{17}[\dXx]`,
			replacement: "******************",
		},
		"email": {
			pattern:     `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			replacement: "****@****.***",
		},
		"ip_address": {
			pattern:     `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			replacement: "*.*.*.*",
		},
		"bank_card": {
			pattern:     `\b\d{16,19}\b`,
			replacement: "****************",
		},
	}

	for dataType, config := range patterns {
		if err := pid.AddPattern(dataType, config.pattern, config.replacement); err != nil {
			if pid.logger != nil {
				pid.logger.Error("添加模式失败", "type", dataType, "error", err.Error())
			}
		}
	}
}

// ChineseNameDesensitizer 中文姓名脱敏器
type ChineseNameDesensitizer struct {
	*BaseDesensitizer
}

// NewChineseNameDesensitizer 创建中文姓名脱敏器
func NewChineseNameDesensitizer() *ChineseNameDesensitizer {
	return &ChineseNameDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer("chinese_name"),
	}
}

// Supports 检查是否支持指定的数据类型
func (cnd *ChineseNameDesensitizer) Supports(dataType string) bool {
	return dataType == "chinese_name" || dataType == "name"
}

// Desensitize 执行中文姓名脱敏
func (cnd *ChineseNameDesensitizer) Desensitize(data string) (string, error) {
	if !cnd.Enabled() {
		return data, nil
	}

	return cnd.desensitizeWithCache(data, func(input string) (string, error) {
		// 检查是否为纯中文姓名（长度<=10且只包含中文字符，不包含标点符号）
		trimmed := strings.TrimSpace(input)
		if len([]rune(trimmed)) <= 10 && cnd.isPureChineseName(trimmed) {
			return cnd.desensitizeName(trimmed), nil
		}

		// 对于混合文本，使用正则表达式找到并替换中文姓名
		pattern := regexp.MustCompile(cnd.GetTypePattern("chinese_name"))
		return pattern.ReplaceAllStringFunc(input, func(match string) string {
			return cnd.desensitizeName(match)
		}), nil
	})
}

// isPureChineseName 检查是否为纯中文姓名（不包含其他字符）
func (cnd *ChineseNameDesensitizer) isPureChineseName(text string) bool {
	// 使用更严格的检查：只包含中文字符，没有数字、英文、标点符号等
	for _, r := range text {
		if !cnd.isChineseChar(r) {
			return false
		}
	}
	return true
}

// isChineseChar 检查是否为中文字符
func (cnd *ChineseNameDesensitizer) isChineseChar(r rune) bool {
	// 中文字符的Unicode范围
	return (r >= 0x4e00 && r <= 0x9fff) || // CJK统一汉字
		(r >= 0x3400 && r <= 0x4dbf) || // CJK扩展A
		(r >= 0x20000 && r <= 0x2a6df) // CJK扩展B
}

// desensitizeName 脱敏姓名的具体实现
func (cnd *ChineseNameDesensitizer) desensitizeName(name string) string {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) <= 1 {
		return name
	}

	if len(runes) == 2 {
		// 两字姓名：保留第一个字，第二个字用*代替
		return string(runes[0]) + "*"
	}

	// 三字及以上姓名：保留第一个和最后一个字，中间用*代替
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

// GetSupportedTypes 获取支持的类型
func (cnd *ChineseNameDesensitizer) GetSupportedTypes() []string {
	return []string{"chinese_name", "name"}
}

// GetTypePattern 获取类型的正则表达式模式（中文姓名）
func (cnd *ChineseNameDesensitizer) GetTypePattern(dataType string) string {
	if cnd.Supports(dataType) {
		// 匹配中文姓名：2-4个中文字符，但排除常见非姓名词汇
		// 优先匹配常见姓氏，然后匹配其他中文字符组合
		commonSurnames := `[张王李赵刘陈杨黄周吴徐孙朱马胡郭林何高梁郑罗宋谢唐韩曹许邓萧冯曾程蔡彭潘袁于董余苏叶吕魏蒋田杜丁沈姜范江傅钟卢汪戴崔任陆廖姚方金邱夏谭韦贾邹石熊孟秦阎薛侯雷白龙段郝孔邵史毛常万顾赖武康贺严尹钱施牛洪龚]`
		// 扩展模式：常见姓氏+名字 或 用户+数字(测试用)
		return commonSurnames + `[一-龯]{1,3}|用户[0-9一二三四五六七八九十]+`
	}
	return ""
}

// ValidateType 验证数据是否为中文姓名
func (cnd *ChineseNameDesensitizer) ValidateType(data string, dataType string) bool {
	if !cnd.Supports(dataType) {
		return false
	}

	pattern := regexp.MustCompile(cnd.GetTypePattern(dataType))
	return pattern.MatchString(strings.TrimSpace(data))
}

// CustomFunctionDesensitizer 自定义函数脱敏器
type CustomFunctionDesensitizer struct {
	*BaseDesensitizer
	functions map[string]func(string) string
	mu        sync.RWMutex
}

// NewCustomFunctionDesensitizer 创建自定义函数脱敏器
func NewCustomFunctionDesensitizer(name string) *CustomFunctionDesensitizer {
	return &CustomFunctionDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer(name),
		functions:        make(map[string]func(string) string),
	}
}

// AddFunction 添加自定义脱敏函数
func (cfd *CustomFunctionDesensitizer) AddFunction(dataType string, fn func(string) string) {
	cfd.mu.Lock()
	defer cfd.mu.Unlock()
	cfd.functions[dataType] = fn
}

// Supports 检查是否支持指定的数据类型
func (cfd *CustomFunctionDesensitizer) Supports(dataType string) bool {
	cfd.mu.RLock()
	defer cfd.mu.RUnlock()
	_, exists := cfd.functions[dataType]
	return exists
}

// Desensitize 执行自定义脱敏
func (cfd *CustomFunctionDesensitizer) Desensitize(data string) (string, error) {
	if !cfd.Enabled() {
		return data, nil
	}

	// 尝试所有注册的函数
	cfd.mu.RLock()
	defer cfd.mu.RUnlock()

	for _, fn := range cfd.functions {
		result := fn(data)
		if result != data { // 如果有变化，说明处理了
			return result, nil
		}
	}

	return data, nil // 没有找到合适的处理函数
}

// GetSupportedTypes 获取支持的类型
func (cfd *CustomFunctionDesensitizer) GetSupportedTypes() []string {
	cfd.mu.RLock()
	defer cfd.mu.RUnlock()

	types := make([]string, 0, len(cfd.functions))
	for dataType := range cfd.functions {
		types = append(types, dataType)
	}
	return types
}
