package modules

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
)

// PluginType 插件类型枚举
type PluginType string

const (
	PluginFormatter  PluginType = "formatter"
	PluginHandler    PluginType = "handler"
	PluginMiddleware PluginType = "middleware"
	PluginSink       PluginType = "sink"
)

// Plugin 简化的插件接口 - 单一职责原则
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// Enabled 检查插件是否启用
	Enabled() bool

	// Enable 启用插件
	Enable()

	// Disable 禁用插件
	Disable()
}

// ConfigurablePlugin 可配置插件接口
type ConfigurablePlugin interface {
	Plugin

	// Configure 配置插件
	Configure(config map[string]interface{}) error

	// GetConfig 获取配置项
	GetConfig(key string) (interface{}, bool)
}

// HandlerPlugin 处理器插件接口
type HandlerPlugin interface {
	Plugin

	// Handler 返回slog处理器
	Handler() slog.Handler
}

// PrioritizedPlugin 有优先级的插件接口
type PrioritizedPlugin interface {
	Plugin

	// Priority 返回优先级，数字越小优先级越高
	Priority() int
}

// PluginInfo 插件信息
type PluginInfo struct {
	Name     string                 `json:"name"`
	Type     PluginType             `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// PluginManager 简化的插件管理器
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin                // 所有插件
	types   map[PluginType]map[string]Plugin // 按类型分组的插件
	enabled atomic.Bool                      // 管理器启用状态
	stats   PluginStats                      // 统计信息
}

// PluginStats 插件统计信息
type PluginStats struct {
	TotalPlugins   int                   `json:"total_plugins"`
	EnabledPlugins int                   `json:"enabled_plugins"`
	TypeCounts     map[PluginType]int    `json:"type_counts"`
	PluginInfo     map[string]PluginInfo `json:"plugin_info"`
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager() *PluginManager {
	pm := &PluginManager{
		plugins: make(map[string]Plugin),
		types:   make(map[PluginType]map[string]Plugin),
		stats: PluginStats{
			TypeCounts: make(map[PluginType]int),
			PluginInfo: make(map[string]PluginInfo),
		},
	}
	pm.enabled.Store(true)

	// 初始化类型映射
	for _, pluginType := range []PluginType{PluginFormatter, PluginHandler, PluginMiddleware, PluginSink} {
		pm.types[pluginType] = make(map[string]Plugin)
	}

	return pm
}

// RegisterPlugin 注册插件
func (pm *PluginManager) RegisterPlugin(plugin Plugin, pluginType PluginType) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查是否已存在
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin '%s' already registered", name)
	}

	// 注册插件
	pm.plugins[name] = plugin
	pm.types[pluginType][name] = plugin

	// 更新统计信息
	pm.updateStatsAfterRegistration(name, plugin, pluginType)

	return nil
}

// UnregisterPlugin 注销插件
func (pm *PluginManager) UnregisterPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin '%s' not found", name)
	}

	// 从所有类型映射中移除
	for pluginType, typeMap := range pm.types {
		if _, exists := typeMap[name]; exists {
			delete(typeMap, name)
			pm.updateStatsAfterUnregistration(name, plugin, pluginType)
			break
		}
	}

	// 从主映射中移除
	delete(pm.plugins, name)

	return nil
}

// GetPlugin 获取指定名称的插件
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// GetPluginsByType 获取指定类型的所有插件（按优先级排序）
func (pm *PluginManager) GetPluginsByType(pluginType PluginType) []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	typeMap := pm.types[pluginType]
	if len(typeMap) == 0 {
		return nil
	}

	plugins := make([]Plugin, 0, len(typeMap))
	for _, plugin := range typeMap {
		if plugin.Enabled() {
			plugins = append(plugins, plugin)
		}
	}

	// 按优先级排序（如果插件支持优先级）
	sort.Slice(plugins, func(i, j int) bool {
		pi, ok1 := plugins[i].(PrioritizedPlugin)
		pj, ok2 := plugins[j].(PrioritizedPlugin)

		if ok1 && ok2 {
			return pi.Priority() < pj.Priority()
		}
		if ok1 {
			return true // 有优先级的排前面
		}
		if ok2 {
			return false
		}
		// 都没有优先级则按名称排序
		return plugins[i].Name() < plugins[j].Name()
	})

	return plugins
}

// ListPlugins 列出所有插件名称
func (pm *PluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// EnablePlugin 启用指定插件
func (pm *PluginManager) EnablePlugin(name string) error {
	plugin, exists := pm.GetPlugin(name)
	if !exists {
		return fmt.Errorf("plugin '%s' not found", name)
	}

	plugin.Enable()

	// 更新统计信息
	pm.mu.Lock()
	if info, exists := pm.stats.PluginInfo[name]; exists {
		info.Enabled = true
		pm.stats.PluginInfo[name] = info
		pm.recalculateEnabledCount()
	}
	pm.mu.Unlock()

	return nil
}

// DisablePlugin 禁用指定插件
func (pm *PluginManager) DisablePlugin(name string) error {
	plugin, exists := pm.GetPlugin(name)
	if !exists {
		return fmt.Errorf("plugin '%s' not found", name)
	}

	plugin.Disable()

	// 更新统计信息
	pm.mu.Lock()
	if info, exists := pm.stats.PluginInfo[name]; exists {
		info.Enabled = false
		pm.stats.PluginInfo[name] = info
		pm.recalculateEnabledCount()
	}
	pm.mu.Unlock()

	return nil
}

// EnableAll 启用所有插件
func (pm *PluginManager) EnableAll() {
	pm.mu.RLock()
	plugins := make(map[string]Plugin)
	for name, plugin := range pm.plugins {
		plugins[name] = plugin
	}
	pm.mu.RUnlock()

	for name, plugin := range plugins {
		plugin.Enable()
		// 更新统计信息
		pm.mu.Lock()
		if info, exists := pm.stats.PluginInfo[name]; exists {
			info.Enabled = true
			pm.stats.PluginInfo[name] = info
		}
		pm.mu.Unlock()
	}

	pm.mu.Lock()
	pm.stats.EnabledPlugins = pm.stats.TotalPlugins
	pm.mu.Unlock()
}

// DisableAll 禁用所有插件
func (pm *PluginManager) DisableAll() {
	pm.mu.RLock()
	plugins := make(map[string]Plugin)
	for name, plugin := range pm.plugins {
		plugins[name] = plugin
	}
	pm.mu.RUnlock()

	for name, plugin := range plugins {
		plugin.Disable()
		// 更新统计信息
		pm.mu.Lock()
		if info, exists := pm.stats.PluginInfo[name]; exists {
			info.Enabled = false
			pm.stats.PluginInfo[name] = info
		}
		pm.mu.Unlock()
	}

	pm.mu.Lock()
	pm.stats.EnabledPlugins = 0
	pm.mu.Unlock()
}

// GetStats 获取插件统计信息
func (pm *PluginManager) GetStats() PluginStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 创建副本
	stats := PluginStats{
		TotalPlugins:   pm.stats.TotalPlugins,
		EnabledPlugins: pm.stats.EnabledPlugins,
		TypeCounts:     make(map[PluginType]int),
		PluginInfo:     make(map[string]PluginInfo),
	}

	// 复制类型计数
	for k, v := range pm.stats.TypeCounts {
		stats.TypeCounts[k] = v
	}

	// 复制插件信息
	for k, v := range pm.stats.PluginInfo {
		stats.PluginInfo[k] = v
	}

	return stats
}

// IsEnabled 检查管理器是否启用
func (pm *PluginManager) IsEnabled() bool {
	return pm.enabled.Load()
}

// Enable 启用管理器
func (pm *PluginManager) Enable() {
	pm.enabled.Store(true)
}

// Disable 禁用管理器
func (pm *PluginManager) Disable() {
	pm.enabled.Store(false)
}

// updateStatsAfterRegistration 注册后更新统计信息
func (pm *PluginManager) updateStatsAfterRegistration(name string, plugin Plugin, pluginType PluginType) {
	pm.stats.TotalPlugins++
	pm.stats.TypeCounts[pluginType]++

	if plugin.Enabled() {
		pm.stats.EnabledPlugins++
	}

	// 创建插件信息
	info := PluginInfo{
		Name:    name,
		Type:    pluginType,
		Enabled: plugin.Enabled(),
	}

	// 添加优先级信息（如果支持）
	if prioritized, ok := plugin.(PrioritizedPlugin); ok {
		info.Priority = prioritized.Priority()
	}

	// 添加配置信息（如果支持）
	if _, ok := plugin.(ConfigurablePlugin); ok {
		// 这里可以扩展获取配置的逻辑
		info.Config = make(map[string]interface{})
	}

	pm.stats.PluginInfo[name] = info
}

// updateStatsAfterUnregistration 注销后更新统计信息
func (pm *PluginManager) updateStatsAfterUnregistration(name string, plugin Plugin, pluginType PluginType) {
	pm.stats.TotalPlugins--
	pm.stats.TypeCounts[pluginType]--

	if plugin.Enabled() {
		pm.stats.EnabledPlugins--
	}

	delete(pm.stats.PluginInfo, name)
}

// recalculateEnabledCount 重新计算启用的插件数量
func (pm *PluginManager) recalculateEnabledCount() {
	count := 0
	for _, info := range pm.stats.PluginInfo {
		if info.Enabled {
			count++
		}
	}
	pm.stats.EnabledPlugins = count
}

// BasePlugin 基础插件实现
type BasePlugin struct {
	name    string
	enabled atomic.Bool
	config  map[string]interface{}
	mu      sync.RWMutex
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(name string) *BasePlugin {
	bp := &BasePlugin{
		name:   name,
		config: make(map[string]interface{}),
	}
	bp.enabled.Store(true)
	return bp
}

// Name 返回插件名称
func (bp *BasePlugin) Name() string {
	return bp.name
}

// Enabled 检查插件是否启用
func (bp *BasePlugin) Enabled() bool {
	return bp.enabled.Load()
}

// Enable 启用插件
func (bp *BasePlugin) Enable() {
	bp.enabled.Store(true)
}

// Disable 禁用插件
func (bp *BasePlugin) Disable() {
	bp.enabled.Store(false)
}

// Configure 配置插件
func (bp *BasePlugin) Configure(config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()

	// 清空并复制配置
	bp.config = make(map[string]interface{})
	for k, v := range config {
		bp.config[k] = v
	}

	return nil
}

// GetConfig 获取配置项
func (bp *BasePlugin) GetConfig(key string) (interface{}, bool) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	value, exists := bp.config[key]
	return value, exists
}

// 全局插件管理器实例
var (
	globalPluginManager     *PluginManager
	globalPluginManagerOnce sync.Once
)

// GetGlobalPluginManager 获取全局插件管理器
func GetGlobalPluginManager() *PluginManager {
	globalPluginManagerOnce.Do(func() {
		globalPluginManager = NewPluginManager()
	})
	return globalPluginManager
}

// 全局便捷函数

// RegisterGlobalPlugin 全局注册插件
func RegisterGlobalPlugin(plugin Plugin, pluginType PluginType) error {
	return GetGlobalPluginManager().RegisterPlugin(plugin, pluginType)
}

// GetGlobalPlugin 全局获取插件
func GetGlobalPlugin(name string) (Plugin, bool) {
	return GetGlobalPluginManager().GetPlugin(name)
}

// GetGlobalPluginsByType 全局获取指定类型的插件
func GetGlobalPluginsByType(pluginType PluginType) []Plugin {
	return GetGlobalPluginManager().GetPluginsByType(pluginType)
}
