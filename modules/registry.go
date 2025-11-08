package modules

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// ModuleType 定义模块类型
type ModuleType int

const (
	TypeFormatter  ModuleType = iota // 格式化器
	TypeHandler                      // 处理器
	TypeMiddleware                   // 中间件
	TypeSink                         // 日志接收器
)

func (mt ModuleType) String() string {
	switch mt {
	case TypeFormatter:
		return "formatter"
	case TypeHandler:
		return "handler"
	case TypeMiddleware:
		return "middleware"
	case TypeSink:
		return "sink"
	default:
		return "unknown"
	}
}

// Config 通用配置接口
type Config map[string]any

// ModuleConfig 模块配置
type ModuleConfig struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Config   Config `json:"config"`
}

// Module 定义模块接口
type Module interface {
	// Name 返回模块名称
	Name() string
	// Type 返回模块类型
	Type() ModuleType
	// Configure 配置模块
	Configure(config Config) error
	// Handler 返回slog处理器
	Handler() slog.Handler
	// Priority 返回优先级，数字越小优先级越高
	Priority() int
	// Enabled 返回模块是否启用
	Enabled() bool
}

// ModuleFactory 模块工厂函数
type ModuleFactory func(config Config) (Module, error)

// Registry 模块注册中心
type Registry struct {
	mu        sync.RWMutex
	modules   map[string]Module
	factories map[string]ModuleFactory
	chains    map[ModuleType][]Module
}

// NewRegistry 创建新的注册中心
func NewRegistry() *Registry {
	return &Registry{
		modules:   make(map[string]Module),
		factories: make(map[string]ModuleFactory),
		chains:    make(map[ModuleType][]Module),
	}
}

// RegisterFactory 注册模块工厂
func (r *Registry) RegisterFactory(name string, factory ModuleFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("factory %s already registered", name)
	}

	r.factories[name] = factory
	return nil
}

// Register 注册模块实例
func (r *Registry) Register(module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := module.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}

	r.modules[name] = module

	// 添加到类型链中
	moduleType := module.Type()
	r.chains[moduleType] = append(r.chains[moduleType], module)

	// 按优先级排序
	sort.Slice(r.chains[moduleType], func(i, j int) bool {
		return r.chains[moduleType][i].Priority() < r.chains[moduleType][j].Priority()
	})

	return nil
}

// Create 通过工厂创建模块
func (r *Registry) Create(name string, config Config) (Module, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("factory %s not found", name)
	}

	return factory(config)
}

// Get 获取模块
func (r *Registry) Get(name string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[name]
	return module, exists
}

// GetByType 按类型获取模块列表
func (r *Registry) GetByType(moduleType ModuleType) []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]Module, len(r.chains[moduleType]))
	copy(modules, r.chains[moduleType])
	return modules
}

// List 列出所有模块
func (r *Registry) List() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
}

// ListFactories 列出所有已注册的工厂名称
func (r *Registry) ListFactories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factories := make([]string, 0, len(r.factories))
	for name := range r.factories {
		factories = append(factories, name)
	}
	return factories
}

// Remove 移除模块
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	delete(r.modules, name)

	// 从类型链中移除
	moduleType := module.Type()
	chain := r.chains[moduleType]
	for i, m := range chain {
		if m.Name() == name {
			r.chains[moduleType] = append(chain[:i], chain[i+1:]...)
			break
		}
	}

	return nil
}

// Update 重新配置已注册模块
func (r *Registry) Update(name string, config Config) error {
	r.mu.RLock()
	module, exists := r.modules[name]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}
	return module.Configure(config)
}

// BaseModule 基础模块实现
type BaseModule struct {
	name     string
	typ      ModuleType
	priority int
	enabled  bool
	handler  slog.Handler
	config   Config
}

// NewBaseModule 创建基础模块
func NewBaseModule(name string, typ ModuleType, priority int) *BaseModule {
	return &BaseModule{
		name:     name,
		typ:      typ,
		priority: priority,
		enabled:  true,
		config:   make(Config),
	}
}

func (m *BaseModule) Name() string          { return m.name }
func (m *BaseModule) Type() ModuleType      { return m.typ }
func (m *BaseModule) Priority() int         { return m.priority }
func (m *BaseModule) Enabled() bool         { return m.enabled }
func (m *BaseModule) Handler() slog.Handler { return m.handler }

func (m *BaseModule) Configure(config Config) error {
	m.config = config
	return nil
}

func (m *BaseModule) SetHandler(handler slog.Handler) {
	m.handler = handler
}

func (m *BaseModule) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// 全局注册中心
var globalRegistry = NewRegistry()

// RegisterModule 全局注册模块
func RegisterModule(module Module) error {
	return globalRegistry.Register(module)
}

// RegisterFactory 全局注册工厂
func RegisterFactory(name string, factory ModuleFactory) error {
	return globalRegistry.RegisterFactory(name, factory)
}

// GetModule 全局获取模块
func GetModule(name string) (Module, bool) {
	return globalRegistry.Get(name)
}

// CreateModule 全局创建模块
func CreateModule(name string, config Config) (Module, error) {
	return globalRegistry.Create(name, config)
}

// GetRegistry 获取全局注册中心
func GetRegistry() *Registry {
	return globalRegistry
}

// UpdateModuleConfig 重新配置现有模块
func UpdateModuleConfig(name string, config Config) error {
	return globalRegistry.Update(name, config)
}
