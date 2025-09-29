package modules

import (
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
)

// ================================
// 基础实现 - 实现单一职责接口的具体类型
// ================================

// BaseComponent 基础组件实现
type BaseComponent struct {
	name       string
	moduleType ModuleType
	enabled    atomic.Bool
	priority   int32
}

// NewBaseComponent 创建基础组件
func NewBaseComponent(name string, moduleType ModuleType) *BaseComponent {
	bc := &BaseComponent{
		name:       name,
		moduleType: moduleType,
	}
	bc.enabled.Store(true) // 默认启用
	return bc
}

// Name 实现Named接口
func (bc *BaseComponent) Name() string {
	return bc.name
}

// Type 实现Typed接口
func (bc *BaseComponent) Type() ModuleType {
	return bc.moduleType
}

// Enabled 实现Enableable接口
func (bc *BaseComponent) Enabled() bool {
	return bc.enabled.Load()
}

// Enable 实现Enableable接口
func (bc *BaseComponent) Enable() {
	bc.enabled.Store(true)
}

// Disable 实现Enableable接口
func (bc *BaseComponent) Disable() {
	bc.enabled.Store(false)
}

// Priority 实现Prioritized接口
func (bc *BaseComponent) Priority() int {
	return int(atomic.LoadInt32(&bc.priority))
}

// SetPriority 实现Prioritized接口
func (bc *BaseComponent) SetPriority(priority int) {
	atomic.StoreInt32(&bc.priority, int32(priority))
}

// ================================
// 配置管理组件
// ================================

// ConfigurableComponent 可配置组件
type ConfigurableComponent struct {
	*BaseComponent
	config Config
}

// NewConfigurableComponent 创建可配置组件
func NewConfigurableComponent(name string, moduleType ModuleType) *ConfigurableComponent {
	return &ConfigurableComponent{
		BaseComponent: NewBaseComponent(name, moduleType),
		config:        make(Config),
	}
}

// Configure 实现Configurable接口
func (cc *ConfigurableComponent) Configure(config Config) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}
	cc.config = make(Config)
	for k, v := range config {
		cc.config[k] = v
	}
	return nil
}

// GetConfig 实现Configurable接口
func (cc *ConfigurableComponent) GetConfig() Config {
	result := make(Config)
	for k, v := range cc.config {
		result[k] = v
	}
	return result
}

// ================================
// 处理器组件
// ================================

// HandlerComponent 处理器组件
type HandlerComponent struct {
	*ConfigurableComponent
	handler slog.Handler
}

// NewHandlerComponent 创建处理器组件
func NewHandlerComponent(name string, moduleType ModuleType, handler slog.Handler) *HandlerComponent {
	return &HandlerComponent{
		ConfigurableComponent: NewConfigurableComponent(name, moduleType),
		handler:               handler,
	}
}

// Handler 实现HandlerProvider接口
func (hc *HandlerComponent) Handler() slog.Handler {
	return hc.handler
}

// SetHandler 实现HandlerProvider接口
func (hc *HandlerComponent) SetHandler(handler slog.Handler) {
	hc.handler = handler
}

// ================================
// 生命周期管理组件
// ================================

// LifecycleComponent 生命周期组件
type LifecycleComponent struct {
	*HandlerComponent
	initialized atomic.Bool
	running     atomic.Bool
	disposed    atomic.Bool
}

// NewLifecycleComponent 创建生命周期组件
func NewLifecycleComponent(name string, moduleType ModuleType, handler slog.Handler) *LifecycleComponent {
	return &LifecycleComponent{
		HandlerComponent: NewHandlerComponent(name, moduleType, handler),
	}
}

// Initialize 实现Initializable接口
func (lc *LifecycleComponent) Initialize() error {
	if lc.initialized.Load() {
		return errors.New("component already initialized")
	}

	// 执行初始化逻辑
	// 这里可以添加具体的初始化代码

	lc.initialized.Store(true)
	return nil
}

// IsInitialized 实现Initializable接口
func (lc *LifecycleComponent) IsInitialized() bool {
	return lc.initialized.Load()
}

// Start 实现Startable接口
func (lc *LifecycleComponent) Start() error {
	if !lc.initialized.Load() {
		return errors.New("component not initialized")
	}
	if lc.running.Load() {
		return errors.New("component already running")
	}

	// 执行启动逻辑
	// 这里可以添加具体的启动代码

	lc.running.Store(true)
	return nil
}

// Stop 实现Startable接口
func (lc *LifecycleComponent) Stop() error {
	if !lc.running.Load() {
		return errors.New("component not running")
	}

	// 执行停止逻辑
	// 这里可以添加具体的停止代码

	lc.running.Store(false)
	return nil
}

// IsRunning 实现Startable接口
func (lc *LifecycleComponent) IsRunning() bool {
	return lc.running.Load()
}

// Dispose 实现Disposable接口
func (lc *LifecycleComponent) Dispose() error {
	if lc.disposed.Load() {
		return errors.New("component already disposed")
	}

	// 如果正在运行，先停止
	if lc.running.Load() {
		if err := lc.Stop(); err != nil {
			return err
		}
	}

	// 执行资源清理逻辑
	// 这里可以添加具体的清理代码

	lc.disposed.Store(true)
	return nil
}

// IsDisposed 实现Disposable接口
func (lc *LifecycleComponent) IsDisposed() bool {
	return lc.disposed.Load()
}

// ================================
// 监控组件
// ================================

// MonitoredComponent 被监控的组件
type MonitoredComponent struct {
	*LifecycleComponent
	metrics map[string]interface{}
	healthy atomic.Bool
}

// NewMonitoredComponent 创建被监控的组件
func NewMonitoredComponent(name string, moduleType ModuleType, handler slog.Handler) *MonitoredComponent {
	mc := &MonitoredComponent{
		LifecycleComponent: NewLifecycleComponent(name, moduleType, handler),
		metrics:            make(map[string]interface{}),
	}
	mc.healthy.Store(true) // 默认健康
	return mc
}

// HealthCheck 实现Healthable接口
func (mc *MonitoredComponent) HealthCheck() error {
	// 执行健康检查逻辑
	if mc.disposed.Load() {
		mc.healthy.Store(false)
		return errors.New("component is disposed")
	}

	if !mc.initialized.Load() {
		mc.healthy.Store(false)
		return errors.New("component not initialized")
	}

	// 这里可以添加更多健康检查逻辑

	mc.healthy.Store(true)
	return nil
}

// IsHealthy 实现Healthable接口
func (mc *MonitoredComponent) IsHealthy() bool {
	return mc.healthy.Load()
}

// GetMetrics 实现Measurable接口
func (mc *MonitoredComponent) GetMetrics() map[string]interface{} {
	result := make(map[string]interface{})

	// 基础状态指标
	result["initialized"] = mc.IsInitialized()
	result["running"] = mc.IsRunning()
	result["healthy"] = mc.IsHealthy()
	result["enabled"] = mc.Enabled()
	result["disposed"] = mc.IsDisposed()

	// 自定义指标
	for k, v := range mc.metrics {
		result[k] = v
	}

	return result
}

// ResetMetrics 实现Measurable接口
func (mc *MonitoredComponent) ResetMetrics() {
	mc.metrics = make(map[string]interface{})
}

// AddMetric 添加自定义指标
func (mc *MonitoredComponent) AddMetric(key string, value interface{}) {
	mc.metrics[key] = value
}

// ================================
// 具体功能组件示例
// ================================

// SimpleFormatter 简单格式化器实现
type SimpleFormatter struct {
	*BaseComponent
}

// NewSimpleFormatter 创建简单格式化器
func NewSimpleFormatter(name string) *SimpleFormatter {
	return &SimpleFormatter{
		BaseComponent: NewBaseComponent(name, TypeFormatter),
	}
}

// Format 实现Formatter接口
func (sf *SimpleFormatter) Format(data interface{}) ([]byte, error) {
	// 简单的格式化逻辑
	return []byte(fmt.Sprintf("%v", data)), nil
}

// MimeType 实现Formatter接口
func (sf *SimpleFormatter) MimeType() string {
	return "text/plain"
}

// SimpleProcessor 简单处理器实现
type SimpleProcessor struct {
	*BaseComponent
}

// NewSimpleProcessor 创建简单处理器
func NewSimpleProcessor(name string) *SimpleProcessor {
	return &SimpleProcessor{
		BaseComponent: NewBaseComponent(name, TypeHandler),
	}
}

// Process 实现Processor接口
func (sp *SimpleProcessor) Process(data interface{}) (interface{}, error) {
	// 简单的处理逻辑 - 原样返回
	return data, nil
}

// ================================
// 兼容性适配器
// ================================

// LegacyModuleWrapper 旧模块包装器
// 将旧的Module接口适配到新的接口系统
type LegacyModuleWrapper struct {
	legacyModule Module // 旧的模块实例
}

// NewLegacyModuleWrapper 创建旧模块包装器
func NewLegacyModuleWrapper(module Module) *LegacyModuleWrapper {
	return &LegacyModuleWrapper{
		legacyModule: module,
	}
}

// Name 实现Named接口
func (lmw *LegacyModuleWrapper) Name() string {
	return lmw.legacyModule.Name()
}

// Type 实现Typed接口
func (lmw *LegacyModuleWrapper) Type() ModuleType {
	return lmw.legacyModule.Type()
}

// Enabled 实现Enableable接口
func (lmw *LegacyModuleWrapper) Enabled() bool {
	return lmw.legacyModule.Enabled()
}

// Enable 实现Enableable接口
func (lmw *LegacyModuleWrapper) Enable() {
	// 旧接口没有Enable方法，这里可以使用反射或其他方式
	// 暂时留空，实际使用时需要根据具体情况实现
}

// Disable 实现Enableable接口
func (lmw *LegacyModuleWrapper) Disable() {
	// 旧接口没有Disable方法，这里可以使用反射或其他方式
	// 暂时留空，实际使用时需要根据具体情况实现
}

// Priority 实现Prioritized接口
func (lmw *LegacyModuleWrapper) Priority() int {
	return lmw.legacyModule.Priority()
}

// SetPriority 实现Prioritized接口
func (lmw *LegacyModuleWrapper) SetPriority(priority int) {
	// 旧接口没有SetPriority方法，这里可以使用反射或其他方式
	// 暂时留空，实际使用时需要根据具体情况实现
}

// Configure 实现Configurable接口
func (lmw *LegacyModuleWrapper) Configure(config Config) error {
	return lmw.legacyModule.Configure(config)
}

// GetConfig 实现Configurable接口
func (lmw *LegacyModuleWrapper) GetConfig() Config {
	// 旧接口没有GetConfig方法，返回空配置
	return make(Config)
}

// Handler 实现HandlerProvider接口
func (lmw *LegacyModuleWrapper) Handler() slog.Handler {
	return lmw.legacyModule.Handler()
}

// SetHandler 实现HandlerProvider接口
func (lmw *LegacyModuleWrapper) SetHandler(handler slog.Handler) {
	// 旧接口没有SetHandler方法，这里可以使用反射或其他方式
	// 暂时留空，实际使用时需要根据具体情况实现
}
