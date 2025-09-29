package modules

import (
	"log/slog"
)

// ================================
// 核心基础接口 - 遵循单一职责原则
// ================================

// Named 命名接口 - 只负责提供名称
type Named interface {
	// Name 返回组件名称
	Name() string
}

// Typed 类型接口 - 只负责提供类型信息
type Typed interface {
	// Type 返回组件类型
	Type() ModuleType
}

// Configurable 可配置接口 - 只负责配置管理
type Configurable interface {
	// Configure 配置组件
	Configure(config Config) error

	// GetConfig 获取当前配置（可选实现）
	GetConfig() Config
}

// Enableable 可启用接口 - 只负责启用状态管理
type Enableable interface {
	// Enabled 返回组件是否启用
	Enabled() bool

	// Enable 启用组件
	Enable()

	// Disable 禁用组件
	Disable()
}

// Prioritized 优先级接口 - 只负责优先级管理
type Prioritized interface {
	// Priority 返回优先级，数字越小优先级越高
	Priority() int

	// SetPriority 设置优先级
	SetPriority(priority int)
}

// ================================
// 功能专用接口
// ================================

// HandlerProvider 处理器提供者接口 - 只负责提供slog处理器
type HandlerProvider interface {
	// Handler 返回slog处理器
	Handler() slog.Handler

	// SetHandler 设置slog处理器
	SetHandler(handler slog.Handler)
}

// Formatter 格式化器接口 - 只负责数据格式化
type Formatter interface {
	// Format 格式化数据
	Format(data interface{}) ([]byte, error)

	// MimeType 返回格式化后的MIME类型
	MimeType() string
}

// Processor 处理器接口 - 只负责数据处理
type Processor interface {
	// Process 处理数据
	Process(data interface{}) (interface{}, error)
}

// Middleware 中间件接口 - 只负责中间件逻辑
type Middleware interface {
	// Handle 中间件处理函数
	Handle(next func()) func()

	// Order 中间件执行顺序
	Order() int
}

// Sink 数据接收器接口 - 只负责数据输出
type Sink interface {
	// Write 写入数据
	Write(data []byte) (int, error)

	// Flush 刷新缓冲区
	Flush() error

	// Close 关闭接收器
	Close() error
}

// ================================
// 生命周期管理接口
// ================================

// Initializable 可初始化接口 - 只负责初始化
type Initializable interface {
	// Initialize 初始化组件
	Initialize() error

	// IsInitialized 检查是否已初始化
	IsInitialized() bool
}

// Startable 可启动接口 - 只负责启动/停止
type Startable interface {
	// Start 启动组件
	Start() error

	// Stop 停止组件
	Stop() error

	// IsRunning 检查是否正在运行
	IsRunning() bool
}

// Disposable 可释放接口 - 只负责资源清理
type Disposable interface {
	// Dispose 释放资源
	Dispose() error

	// IsDisposed 检查是否已释放
	IsDisposed() bool
}

// ================================
// 监控和诊断接口
// ================================

// Healthable 健康检查接口 - 只负责健康状态
type Healthable interface {
	// HealthCheck 执行健康检查
	HealthCheck() error

	// IsHealthy 检查是否健康
	IsHealthy() bool
}

// Measurable 可度量接口 - 只负责性能指标
type Measurable interface {
	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}

	// ResetMetrics 重置性能指标
	ResetMetrics()
}

// ================================
// 组合接口 - 通过接口组合实现复杂功能
// ================================

// ComponentInterface 组件基础接口 - 组合核心基础功能
type ComponentInterface interface {
	Named
	Typed
	Enableable
}

// ConfigurableModule 可配置模块接口
type ConfigurableModule interface {
	ComponentInterface
	Configurable
}

// PrioritizedModule 有优先级的模块接口
type PrioritizedModule interface {
	ComponentInterface
	Prioritized
}

// HandlerModule 处理器模块接口
type HandlerModule interface {
	ComponentInterface
	HandlerProvider
}

// FormatterModule 格式化器模块接口
type FormatterModule interface {
	ComponentInterface
	Formatter
}

// ProcessorModule 处理器模块接口
type ProcessorModule interface {
	ComponentInterface
	Processor
}

// MiddlewareModule 中间件模块接口
type MiddlewareModule interface {
	ComponentInterface
	Middleware
}

// SinkModule 接收器模块接口
type SinkModule interface {
	ComponentInterface
	Sink
}

// FullModule 完整模块接口 - 包含所有基础功能
type FullModule interface {
	ComponentInterface
	Configurable
	Prioritized
	HandlerProvider
}

// ManagedModule 托管模块接口 - 包含生命周期管理
type ManagedModule interface {
	FullModule
	Initializable
	Startable
	Disposable
}

// MonitoredModule 被监控的模块接口 - 包含监控功能
type MonitoredModule interface {
	FullModule
	Healthable
	Measurable
}

// ================================
// 接口适配器
// ================================

// ModuleAdapter 模块适配器，帮助将旧接口转换为新接口
type ModuleAdapter struct {
	module Module
}

// NewModuleAdapter 创建模块适配器
func NewModuleAdapter(module Module) *ModuleAdapter {
	return &ModuleAdapter{module: module}
}

// AsNamed 转换为Named接口
func (ma *ModuleAdapter) AsNamed() Named {
	return ma.module
}

// AsTyped 转换为Typed接口
func (ma *ModuleAdapter) AsTyped() Typed {
	return ma.module
}

// AsConfigurable 转换为Configurable接口
func (ma *ModuleAdapter) AsConfigurable() Configurable {
	if configurable, ok := ma.module.(Configurable); ok {
		return configurable
	}
	// 返回一个基于旧Module接口的适配器
	return &configurableAdapter{module: ma.module}
}

// AsEnableable 转换为Enableable接口
func (ma *ModuleAdapter) AsEnableable() Enableable {
	if enableable, ok := ma.module.(Enableable); ok {
		return enableable
	}
	// 返回一个基于旧Module接口的适配器
	return &enableableAdapter{module: ma.module}
}

// AsPrioritized 转换为Prioritized接口
func (ma *ModuleAdapter) AsPrioritized() Prioritized {
	if prioritized, ok := ma.module.(Prioritized); ok {
		return prioritized
	}
	// 返回一个基于旧Module接口的适配器
	return &prioritizedAdapter{module: ma.module}
}

// AsHandlerProvider 转换为HandlerProvider接口
func (ma *ModuleAdapter) AsHandlerProvider() HandlerProvider {
	if provider, ok := ma.module.(HandlerProvider); ok {
		return provider
	}
	// 返回一个基于旧Module接口的适配器
	return &handlerProviderAdapter{module: ma.module}
}

// ================================
// 适配器实现
// ================================

// configurableAdapter 配置接口适配器
type configurableAdapter struct {
	module Module
}

func (ca *configurableAdapter) Configure(config Config) error {
	return ca.module.Configure(config)
}

func (ca *configurableAdapter) GetConfig() Config {
	// 旧接口没有GetConfig方法，返回空配置
	return make(Config)
}

// enableableAdapter 启用接口适配器
type enableableAdapter struct {
	module Module
}

func (ea *enableableAdapter) Enabled() bool {
	return ea.module.Enabled()
}

func (ea *enableableAdapter) Enable() {
	// 旧接口没有Enable方法，暂时留空
}

func (ea *enableableAdapter) Disable() {
	// 旧接口没有Disable方法，暂时留空
}

// prioritizedAdapter 优先级接口适配器
type prioritizedAdapter struct {
	module Module
}

func (pa *prioritizedAdapter) Priority() int {
	return pa.module.Priority()
}

func (pa *prioritizedAdapter) SetPriority(priority int) {
	// 旧接口没有SetPriority方法，暂时留空
}

// handlerProviderAdapter 处理器提供者接口适配器
type handlerProviderAdapter struct {
	module Module
}

func (hpa *handlerProviderAdapter) Handler() slog.Handler {
	return hpa.module.Handler()
}

func (hpa *handlerProviderAdapter) SetHandler(handler slog.Handler) {
	// 旧接口没有SetHandler方法，暂时留空
}

// ================================
// 接口检查工具
// ================================

// InterfaceChecker 接口检查工具
type InterfaceChecker struct{}

// CheckNamed 检查是否实现Named接口
func (ic *InterfaceChecker) CheckNamed(obj interface{}) (Named, bool) {
	named, ok := obj.(Named)
	return named, ok
}

// CheckTyped 检查是否实现Typed接口
func (ic *InterfaceChecker) CheckTyped(obj interface{}) (Typed, bool) {
	typed, ok := obj.(Typed)
	return typed, ok
}

// CheckConfigurable 检查是否实现Configurable接口
func (ic *InterfaceChecker) CheckConfigurable(obj interface{}) (Configurable, bool) {
	configurable, ok := obj.(Configurable)
	return configurable, ok
}

// CheckEnableable 检查是否实现Enableable接口
func (ic *InterfaceChecker) CheckEnableable(obj interface{}) (Enableable, bool) {
	enableable, ok := obj.(Enableable)
	return enableable, ok
}

// CheckHandlerProvider 检查是否实现HandlerProvider接口
func (ic *InterfaceChecker) CheckHandlerProvider(obj interface{}) (HandlerProvider, bool) {
	provider, ok := obj.(HandlerProvider)
	return provider, ok
}

// CheckFullModule 检查是否实现FullModule接口
func (ic *InterfaceChecker) CheckFullModule(obj interface{}) (FullModule, bool) {
	module, ok := obj.(FullModule)
	return module, ok
}

// GetImplementedInterfaces 获取对象实现的所有接口列表
func (ic *InterfaceChecker) GetImplementedInterfaces(obj interface{}) []string {
	var interfaces []string

	if _, ok := ic.CheckNamed(obj); ok {
		interfaces = append(interfaces, "Named")
	}
	if _, ok := ic.CheckTyped(obj); ok {
		interfaces = append(interfaces, "Typed")
	}
	if _, ok := ic.CheckConfigurable(obj); ok {
		interfaces = append(interfaces, "Configurable")
	}
	if _, ok := ic.CheckEnableable(obj); ok {
		interfaces = append(interfaces, "Enableable")
	}
	if _, ok := ic.CheckHandlerProvider(obj); ok {
		interfaces = append(interfaces, "HandlerProvider")
	}

	return interfaces
}

// 全局接口检查器实例
var GlobalInterfaceChecker = &InterfaceChecker{}
