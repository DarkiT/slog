package modules

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
)

// TestPlugin 测试插件实现
type TestPlugin struct {
	*BasePlugin
	priority int
	handler  slog.Handler
}

// NewTestPlugin 创建测试插件
func NewTestPlugin(name string, priority int) *TestPlugin {
	return &TestPlugin{
		BasePlugin: NewBasePlugin(name),
		priority:   priority,
		handler:    slog.NewTextHandler(os.Stdout, nil),
	}
}

// Priority 返回优先级
func (tp *TestPlugin) Priority() int {
	return tp.priority
}

// Handler 返回处理器
func (tp *TestPlugin) Handler() slog.Handler {
	return tp.handler
}

func TestPluginManager_Registration(t *testing.T) {
	pm := NewPluginManager()

	// 测试注册插件
	plugin := NewTestPlugin("test_formatter", 10)
	err := pm.RegisterPlugin(plugin, PluginFormatter)
	if err != nil {
		t.Errorf("RegisterPlugin() error = %v", err)
	}

	// 验证注册成功
	retrieved, exists := pm.GetPlugin("test_formatter")
	if !exists {
		t.Error("Plugin should be registered")
	}
	if retrieved.Name() != "test_formatter" {
		t.Errorf("Retrieved plugin name = %s, want test_formatter", retrieved.Name())
	}

	// 测试重复注册
	err = pm.RegisterPlugin(plugin, PluginFormatter)
	if err == nil {
		t.Error("Should not allow duplicate registration")
	}

	// 测试注销插件
	err = pm.UnregisterPlugin("test_formatter")
	if err != nil {
		t.Errorf("UnregisterPlugin() error = %v", err)
	}

	// 验证注销成功
	_, exists = pm.GetPlugin("test_formatter")
	if exists {
		t.Error("Plugin should be unregistered")
	}
}

func TestPluginManager_TypeFiltering(t *testing.T) {
	pm := NewPluginManager()

	// 注册不同类型的插件
	formatter1 := NewTestPlugin("formatter1", 10)
	formatter2 := NewTestPlugin("formatter2", 5)
	handler1 := NewTestPlugin("handler1", 20)

	pm.RegisterPlugin(formatter1, PluginFormatter)
	pm.RegisterPlugin(formatter2, PluginFormatter)
	pm.RegisterPlugin(handler1, PluginHandler)

	// 测试按类型获取插件
	formatters := pm.GetPluginsByType(PluginFormatter)
	if len(formatters) != 2 {
		t.Errorf("GetPluginsByType(PluginFormatter) length = %d, want 2", len(formatters))
	}

	// 验证优先级排序（formatter2的优先级更高，应该排在前面）
	if formatters[0].Name() != "formatter2" {
		t.Errorf("First formatter should be 'formatter2', got %s", formatters[0].Name())
	}

	handlers := pm.GetPluginsByType(PluginHandler)
	if len(handlers) != 1 || handlers[0].Name() != "handler1" {
		t.Error("Should have exactly one handler")
	}

	// 测试不存在的类型
	sinks := pm.GetPluginsByType(PluginSink)
	if sinks != nil {
		t.Error("Should return nil for empty type")
	}
}

func TestPluginManager_EnableDisable(t *testing.T) {
	pm := NewPluginManager()

	// 注册插件
	plugin := NewTestPlugin("test_plugin", 10)
	pm.RegisterPlugin(plugin, PluginFormatter)

	// 测试默认启用状态
	if !plugin.Enabled() {
		t.Error("Plugin should be enabled by default")
	}

	// 测试禁用插件
	err := pm.DisablePlugin("test_plugin")
	if err != nil {
		t.Errorf("DisablePlugin() error = %v", err)
	}
	if plugin.Enabled() {
		t.Error("Plugin should be disabled")
	}

	// 禁用状态下按类型获取不应包含该插件
	formatters := pm.GetPluginsByType(PluginFormatter)
	if len(formatters) != 0 {
		t.Error("Disabled plugins should not be returned by GetPluginsByType")
	}

	// 测试重新启用
	err = pm.EnablePlugin("test_plugin")
	if err != nil {
		t.Errorf("EnablePlugin() error = %v", err)
	}
	if !plugin.Enabled() {
		t.Error("Plugin should be enabled")
	}

	// 启用后应该重新出现在类型列表中
	formatters = pm.GetPluginsByType(PluginFormatter)
	if len(formatters) != 1 {
		t.Error("Enabled plugin should be returned by GetPluginsByType")
	}
}

func TestPluginManager_EnableDisableAll(t *testing.T) {
	pm := NewPluginManager()

	// 注册多个插件
	plugins := []*TestPlugin{
		NewTestPlugin("plugin1", 10),
		NewTestPlugin("plugin2", 20),
		NewTestPlugin("plugin3", 30),
	}

	for _, plugin := range plugins {
		pm.RegisterPlugin(plugin, PluginFormatter)
	}

	// 测试禁用所有
	pm.DisableAll()
	for _, plugin := range plugins {
		if plugin.Enabled() {
			t.Errorf("Plugin %s should be disabled", plugin.Name())
		}
	}

	// 获取统计信息验证
	stats := pm.GetStats()
	if stats.EnabledPlugins != 0 {
		t.Errorf("EnabledPlugins should be 0, got %d", stats.EnabledPlugins)
	}

	// 测试启用所有
	pm.EnableAll()
	for _, plugin := range plugins {
		if !plugin.Enabled() {
			t.Errorf("Plugin %s should be enabled", plugin.Name())
		}
	}

	// 验证统计信息
	stats = pm.GetStats()
	if stats.EnabledPlugins != 3 {
		t.Errorf("EnabledPlugins should be 3, got %d", stats.EnabledPlugins)
	}
}

func TestPluginManager_Stats(t *testing.T) {
	pm := NewPluginManager()

	// 注册不同类型的插件
	formatter := NewTestPlugin("formatter", 10)
	handler := NewTestPlugin("handler", 20)
	middleware := NewTestPlugin("middleware", 30)

	pm.RegisterPlugin(formatter, PluginFormatter)
	pm.RegisterPlugin(handler, PluginHandler)
	pm.RegisterPlugin(middleware, PluginMiddleware)

	// 获取统计信息
	stats := pm.GetStats()

	// 验证总数
	if stats.TotalPlugins != 3 {
		t.Errorf("TotalPlugins = %d, want 3", stats.TotalPlugins)
	}
	if stats.EnabledPlugins != 3 {
		t.Errorf("EnabledPlugins = %d, want 3", stats.EnabledPlugins)
	}

	// 验证类型计数
	expectedTypeCounts := map[PluginType]int{
		PluginFormatter:  1,
		PluginHandler:    1,
		PluginMiddleware: 1,
		PluginSink:       0,
	}

	for pluginType, expectedCount := range expectedTypeCounts {
		if stats.TypeCounts[pluginType] != expectedCount {
			t.Errorf("TypeCounts[%s] = %d, want %d",
				pluginType, stats.TypeCounts[pluginType], expectedCount)
		}
	}

	// 验证插件信息
	if len(stats.PluginInfo) != 3 {
		t.Errorf("PluginInfo length = %d, want 3", len(stats.PluginInfo))
	}

	// 验证具体插件信息
	formatterInfo := stats.PluginInfo["formatter"]
	if formatterInfo.Name != "formatter" || formatterInfo.Type != PluginFormatter {
		t.Error("Formatter plugin info incorrect")
	}
	if formatterInfo.Priority != 10 {
		t.Errorf("Formatter priority = %d, want 10", formatterInfo.Priority)
	}
}

func TestPluginManager_Configuration(t *testing.T) {
	pm := NewPluginManager()

	// 注册可配置插件
	plugin := NewTestPlugin("configurable", 10)
	pm.RegisterPlugin(plugin, PluginFormatter)

	// 测试配置
	config := map[string]interface{}{
		"option1": "value1",
		"option2": 42,
		"option3": true,
	}

	err := plugin.Configure(config)
	if err != nil {
		t.Errorf("Configure() error = %v", err)
	}

	// 验证配置
	value1, exists := plugin.GetConfig("option1")
	if !exists || value1 != "value1" {
		t.Error("Configuration should be set correctly")
	}

	value2, exists := plugin.GetConfig("option2")
	if !exists || value2 != 42 {
		t.Error("Integer configuration should be set correctly")
	}

	value3, exists := plugin.GetConfig("option3")
	if !exists || value3 != true {
		t.Error("Boolean configuration should be set correctly")
	}

	// 测试不存在的配置项
	_, exists = plugin.GetConfig("nonexistent")
	if exists {
		t.Error("Nonexistent config should not exist")
	}
}

func TestPluginManager_ErrorHandling(t *testing.T) {
	pm := NewPluginManager()

	// 测试注册nil插件
	err := pm.RegisterPlugin(nil, PluginFormatter)
	if err == nil {
		t.Error("Should not allow nil plugin registration")
	}

	// 测试获取不存在的插件
	_, exists := pm.GetPlugin("nonexistent")
	if exists {
		t.Error("Should not find nonexistent plugin")
	}

	// 测试启用不存在的插件
	err = pm.EnablePlugin("nonexistent")
	if err == nil {
		t.Error("Should return error for nonexistent plugin")
	}

	// 测试禁用不存在的插件
	err = pm.DisablePlugin("nonexistent")
	if err == nil {
		t.Error("Should return error for nonexistent plugin")
	}

	// 测试注销不存在的插件
	err = pm.UnregisterPlugin("nonexistent")
	if err == nil {
		t.Error("Should return error when unregistering nonexistent plugin")
	}
}

func TestPluginManager_ListPlugins(t *testing.T) {
	pm := NewPluginManager()

	// 注册插件
	plugins := []string{"alpha", "beta", "gamma"}
	for _, name := range plugins {
		plugin := NewTestPlugin(name, 10)
		pm.RegisterPlugin(plugin, PluginFormatter)
	}

	// 获取插件列表
	list := pm.ListPlugins()

	// 验证列表长度
	if len(list) != 3 {
		t.Errorf("ListPlugins() length = %d, want 3", len(list))
	}

	// 验证列表已排序
	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range list {
		if name != expected[i] {
			t.Errorf("ListPlugins()[%d] = %s, want %s", i, name, expected[i])
		}
	}
}

func TestPluginManager_ManagerEnableDisable(t *testing.T) {
	pm := NewPluginManager()

	// 测试默认启用状态
	if !pm.IsEnabled() {
		t.Error("Manager should be enabled by default")
	}

	// 测试禁用管理器
	pm.Disable()
	if pm.IsEnabled() {
		t.Error("Manager should be disabled")
	}

	// 测试启用管理器
	pm.Enable()
	if !pm.IsEnabled() {
		t.Error("Manager should be enabled")
	}
}

func TestGlobalPluginManager(t *testing.T) {
	// 测试全局管理器单例
	manager1 := GetGlobalPluginManager()
	manager2 := GetGlobalPluginManager()

	if manager1 != manager2 {
		t.Error("Global plugin manager should be singleton")
	}

	// 测试全局便捷函数
	plugin := NewTestPlugin("global_test", 10)
	err := RegisterGlobalPlugin(plugin, PluginFormatter)
	if err != nil {
		t.Errorf("RegisterGlobalPlugin() error = %v", err)
	}

	// 验证全局获取
	retrieved, exists := GetGlobalPlugin("global_test")
	if !exists {
		t.Error("Should find globally registered plugin")
	}
	if retrieved.Name() != "global_test" {
		t.Error("Global plugin name mismatch")
	}

	// 验证全局类型获取
	formatters := GetGlobalPluginsByType(PluginFormatter)
	if len(formatters) == 0 {
		t.Error("Should find globally registered formatter")
	}
}

// 并发安全测试
func TestPluginManager_Concurrency(t *testing.T) {
	pm := NewPluginManager()

	// 并发注册插件
	numRoutines := 50
	done := make(chan bool, numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			plugin := NewTestPlugin(fmt.Sprintf("plugin_%d", id), id)
			err := pm.RegisterPlugin(plugin, PluginFormatter)
			if err != nil {
				t.Errorf("Concurrent registration failed: %v", err)
			}
			done <- true
		}(i)
	}

	// 等待所有注册完成
	for i := 0; i < numRoutines; i++ {
		<-done
	}

	// 验证注册成功
	stats := pm.GetStats()
	if stats.TotalPlugins != numRoutines {
		t.Errorf("Expected %d plugins, got %d", numRoutines, stats.TotalPlugins)
	}

	// 并发获取插件
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			name := fmt.Sprintf("plugin_%d", id)
			_, exists := pm.GetPlugin(name)
			if !exists {
				t.Errorf("Plugin %s should exist", name)
			}
			done <- true
		}(i)
	}

	// 等待所有获取完成
	for i := 0; i < numRoutines; i++ {
		<-done
	}
}

// 性能基准测试
func BenchmarkPluginManager_Register(b *testing.B) {
	pm := NewPluginManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("plugin_%d", i), i)
		_ = pm.RegisterPlugin(plugin, PluginFormatter)
	}
}

func BenchmarkPluginManager_GetPlugin(b *testing.B) {
	pm := NewPluginManager()

	// 预注册一些插件
	for i := 0; i < 1000; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("plugin_%d", i), i)
		pm.RegisterPlugin(plugin, PluginFormatter)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("plugin_%d", i%1000)
		_, _ = pm.GetPlugin(name)
	}
}

func BenchmarkPluginManager_GetPluginsByType(b *testing.B) {
	pm := NewPluginManager()

	// 预注册一些插件
	for i := 0; i < 100; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("plugin_%d", i), i)
		pm.RegisterPlugin(plugin, PluginFormatter)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.GetPluginsByType(PluginFormatter)
	}
}
