package dlp

// Desensitizer 脱敏器接口 - 插拔式架构的核心
// 所有脱敏器都必须实现这个接口
type Desensitizer interface {
	// Name 返回脱敏器名称，用于注册和识别
	Name() string

	// Supports 检查是否支持指定的数据类型
	Supports(dataType string) bool

	// Desensitize 执行脱敏操作
	Desensitize(data string) (string, error)

	// Configure 配置脱敏器参数
	Configure(config map[string]any) error

	// Enabled 检查脱敏器是否启用
	Enabled() bool

	// Enable 启用脱敏器
	Enable()

	// Disable 禁用脱敏器
	Disable()
}

// TypeSpecificDesensitizer 类型专用脱敏器接口
// 用于特定类型的专业脱敏
type TypeSpecificDesensitizer interface {
	Desensitizer

	// GetSupportedTypes 获取支持的所有类型
	GetSupportedTypes() []string

	// GetTypePattern 获取类型的正则表达式模式
	GetTypePattern(dataType string) string

	// ValidateType 验证数据是否符合指定类型
	ValidateType(data string, dataType string) bool
}

// CacheableDesensitizer 可缓存脱敏器接口
// 支持结果缓存以提升性能
type CacheableDesensitizer interface {
	Desensitizer

	// CacheEnabled 检查是否启用缓存
	CacheEnabled() bool

	// SetCacheEnabled 设置缓存状态
	SetCacheEnabled(enabled bool)

	// ClearCache 清空缓存
	ClearCache()

	// GetCacheStats 获取缓存统计
	GetCacheStats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits     int64   // 缓存命中次数
	Misses   int64   // 缓存未命中次数
	Size     int64   // 缓存大小
	HitRatio float64 // 命中率
}

// Logger 简化的日志接口，避免循环依赖
type Logger interface {
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

// DesensitizerManager 脱敏器管理器接口
// 管理所有注册的脱敏器
type DesensitizerManager interface {
	// RegisterDesensitizer 注册脱敏器
	RegisterDesensitizer(desensitizer Desensitizer) error

	// UpsertDesensitizer 注册或热替换脱敏器，返回版本号
	UpsertDesensitizer(desensitizer Desensitizer) (int64, error)

	// UnregisterDesensitizer 注销脱敏器
	UnregisterDesensitizer(name string) error

	// GetDesensitizer 获取指定名称的脱敏器
	GetDesensitizer(name string) (Desensitizer, bool)

	// GetDesensitizersForType 获取支持指定类型的所有脱敏器
	GetDesensitizersForType(dataType string) []Desensitizer

	// ListDesensitizers 列出所有已注册的脱敏器
	ListDesensitizers() []string

	// EnableAll 启用所有脱敏器
	EnableAll()

	// DisableAll 禁用所有脱敏器
	DisableAll()

	// GetStats 获取管理器统计信息
	GetStats() ManagerStats

	// CurrentVersion 获取当前版本号
	CurrentVersion() int64
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalDesensitizers   int                           // 总脱敏器数量
	EnabledDesensitizers int                           // 启用的脱敏器数量
	TypeCoverage         map[string]int                // 类型覆盖情况
	PerformanceMetrics   map[string]PerformanceMetrics // 性能指标
	Version              int64                         // 版本号（热更新标记）
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalCalls      int64   // 总调用次数
	TotalDuration   int64   // 总耗时（纳秒）
	AverageDuration float64 // 平均耗时（纳秒）
	ErrorCount      int64   // 错误次数
	SuccessRate     float64 // 成功率
}

// DesensitizationResult 脱敏结果
type DesensitizationResult struct {
	Original     string         // 原始数据
	Desensitized string         // 脱敏后数据
	DataType     string         // 数据类型
	Desensitizer string         // 使用的脱敏器
	Duration     int64          // 处理耗时（纳秒）
	Cached       bool           // 是否来自缓存
	Error        error          // 错误信息
	Metadata     map[string]any // 额外元数据
}
