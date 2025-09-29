package modules

import (
	"log/slog"
	"os"
	"testing"
)

// TestInterfaceSegregation 测试接口分离功能
func TestInterfaceSegregation(t *testing.T) {
	// 创建一个基础组件来测试接口
	component := NewBaseComponent("test-component", TypeFormatter)

	// 测试Named接口
	t.Run("Named Interface", func(t *testing.T) {
		var named Named = component
		if named.Name() != "test-component" {
			t.Errorf("Expected name 'test-component', got %s", named.Name())
		}
	})

	// 测试Typed接口
	t.Run("Typed Interface", func(t *testing.T) {
		var typed Typed = component
		if typed.Type() != TypeFormatter {
			t.Errorf("Expected type TypeFormatter, got %v", typed.Type())
		}
	})

	// 测试Enableable接口
	t.Run("Enableable Interface", func(t *testing.T) {
		var enableable Enableable = component

		// 默认应该是启用的
		if !enableable.Enabled() {
			t.Error("Component should be enabled by default")
		}

		// 测试禁用
		enableable.Disable()
		if enableable.Enabled() {
			t.Error("Component should be disabled after calling Disable()")
		}

		// 测试启用
		enableable.Enable()
		if !enableable.Enabled() {
			t.Error("Component should be enabled after calling Enable()")
		}
	})

	// 测试Prioritized接口
	t.Run("Prioritized Interface", func(t *testing.T) {
		var prioritized Prioritized = component

		// 默认优先级应该是0
		if prioritized.Priority() != 0 {
			t.Errorf("Expected default priority 0, got %d", prioritized.Priority())
		}

		// 设置优先级
		prioritized.SetPriority(10)
		if prioritized.Priority() != 10 {
			t.Errorf("Expected priority 10, got %d", prioritized.Priority())
		}
	})
}

// TestConfigurableComponent 测试可配置组件
func TestConfigurableComponent(t *testing.T) {
	component := NewConfigurableComponent("test-configurable", TypeHandler)

	// 测试Configurable接口
	t.Run("Configurable Interface", func(t *testing.T) {
		var configurable Configurable = component

		// 测试配置
		config := Config{
			"key1": "value1",
			"key2": 42,
		}

		err := configurable.Configure(config)
		if err != nil {
			t.Errorf("Configure failed: %v", err)
		}

		// 获取配置
		retrieved := configurable.GetConfig()
		if retrieved["key1"] != "value1" {
			t.Errorf("Expected key1='value1', got %v", retrieved["key1"])
		}
		if retrieved["key2"] != 42 {
			t.Errorf("Expected key2=42, got %v", retrieved["key2"])
		}
	})

	// 测试nil配置
	t.Run("Nil Config", func(t *testing.T) {
		err := component.Configure(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})
}

// TestHandlerComponent 测试处理器组件
func TestHandlerComponent(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	component := NewHandlerComponent("test-handler", TypeHandler, handler)

	// 测试HandlerProvider接口
	t.Run("HandlerProvider Interface", func(t *testing.T) {
		var provider HandlerProvider = component

		// 获取处理器
		retrieved := provider.Handler()
		if retrieved != handler {
			t.Error("Handler mismatch")
		}

		// 设置新处理器
		newHandler := slog.NewJSONHandler(os.Stdout, nil)
		provider.SetHandler(newHandler)

		if provider.Handler() != newHandler {
			t.Error("New handler not set correctly")
		}
	})
}

// TestLifecycleComponent 测试生命周期组件
func TestLifecycleComponent(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	component := NewLifecycleComponent("test-lifecycle", TypeMiddleware, handler)

	// 测试Initializable接口
	t.Run("Initializable Interface", func(t *testing.T) {
		var initializable Initializable = component

		// 初始状态应该是未初始化
		if initializable.IsInitialized() {
			t.Error("Component should not be initialized initially")
		}

		// 初始化
		err := initializable.Initialize()
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}

		if !initializable.IsInitialized() {
			t.Error("Component should be initialized after Initialize()")
		}

		// 重复初始化应该失败
		err = initializable.Initialize()
		if err == nil {
			t.Error("Expected error for duplicate initialization")
		}
	})

	// 测试Startable接口
	t.Run("Startable Interface", func(t *testing.T) {
		var startable Startable = component

		// 确保已初始化
		component.Initialize()

		// 初始状态应该是未运行
		if startable.IsRunning() {
			t.Error("Component should not be running initially")
		}

		// 启动
		err := startable.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}

		if !startable.IsRunning() {
			t.Error("Component should be running after Start()")
		}

		// 停止
		err = startable.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}

		if startable.IsRunning() {
			t.Error("Component should not be running after Stop()")
		}
	})

	// 测试Disposable接口
	t.Run("Disposable Interface", func(t *testing.T) {
		var disposable Disposable = component

		// 初始状态应该是未释放
		if disposable.IsDisposed() {
			t.Error("Component should not be disposed initially")
		}

		// 释放
		err := disposable.Dispose()
		if err != nil {
			t.Errorf("Dispose failed: %v", err)
		}

		if !disposable.IsDisposed() {
			t.Error("Component should be disposed after Dispose()")
		}

		// 重复释放应该失败
		err = disposable.Dispose()
		if err == nil {
			t.Error("Expected error for duplicate disposal")
		}
	})
}

// TestMonitoredComponent 测试监控组件
func TestMonitoredComponent(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	component := NewMonitoredComponent("test-monitored", TypeSink, handler)

	// 初始化组件
	component.Initialize()

	// 测试Healthable接口
	t.Run("Healthable Interface", func(t *testing.T) {
		var healthable Healthable = component

		// 执行健康检查
		err := healthable.HealthCheck()
		if err != nil {
			t.Errorf("HealthCheck failed: %v", err)
		}

		if !healthable.IsHealthy() {
			t.Error("Component should be healthy after successful health check")
		}
	})

	// 测试Measurable接口
	t.Run("Measurable Interface", func(t *testing.T) {
		var measurable Measurable = component

		// 添加自定义指标
		component.AddMetric("test_counter", 42)
		component.AddMetric("test_gauge", 3.14)

		// 获取指标
		metrics := measurable.GetMetrics()

		// 检查基础指标
		if metrics["initialized"] != true {
			t.Error("Expected initialized=true in metrics")
		}

		if metrics["healthy"] != true {
			t.Error("Expected healthy=true in metrics")
		}

		// 检查自定义指标
		if metrics["test_counter"] != 42 {
			t.Errorf("Expected test_counter=42, got %v", metrics["test_counter"])
		}

		if metrics["test_gauge"] != 3.14 {
			t.Errorf("Expected test_gauge=3.14, got %v", metrics["test_gauge"])
		}

		// 重置指标
		measurable.ResetMetrics()
		newMetrics := measurable.GetMetrics()

		// 基础指标应该仍然存在
		if newMetrics["initialized"] != true {
			t.Error("Basic metrics should not be reset")
		}

		// 自定义指标应该被清除
		if _, exists := newMetrics["test_counter"]; exists {
			t.Error("Custom metrics should be reset")
		}
	})
}

// TestInterfaceComposition 测试接口组合
func TestInterfaceComposition(t *testing.T) {
	handler := slog.NewTextHandler(os.Stdout, nil)
	component := NewMonitoredComponent("test-composition", TypeFormatter, handler)

	// 测试ComponentInterface组合
	t.Run("ComponentInterface Composition", func(t *testing.T) {
		var ci ComponentInterface = component

		// 应该包含Named, Typed, Enableable的所有功能
		if ci.Name() != "test-composition" {
			t.Error("ComponentInterface should include Named functionality")
		}

		if ci.Type() != TypeFormatter {
			t.Error("ComponentInterface should include Typed functionality")
		}

		if !ci.Enabled() {
			t.Error("ComponentInterface should include Enableable functionality")
		}
	})

	// 测试FullModule组合
	t.Run("FullModule Composition", func(t *testing.T) {
		var fm FullModule = component

		// 应该包含所有基础功能
		if fm.Name() != "test-composition" {
			t.Error("FullModule should include Named functionality")
		}

		// 配置功能
		config := Config{"test": "value"}
		err := fm.Configure(config)
		if err != nil {
			t.Error("FullModule should include Configurable functionality")
		}

		// 优先级功能
		fm.SetPriority(5)
		if fm.Priority() != 5 {
			t.Error("FullModule should include Prioritized functionality")
		}

		// 处理器功能
		if fm.Handler() != handler {
			t.Error("FullModule should include HandlerProvider functionality")
		}
	})
}

// TestModuleAdapter 测试模块适配器
func TestModuleAdapter(t *testing.T) {
	// 创建一个旧的Module实例
	oldModule := NewBaseModule("test-old", TypeFormatter, 10)
	oldModule.SetHandler(slog.NewTextHandler(os.Stdout, nil))

	// 创建适配器
	adapter := NewModuleAdapter(oldModule)

	t.Run("Named Adapter", func(t *testing.T) {
		named := adapter.AsNamed()
		if named.Name() != "test-old" {
			t.Errorf("Expected name 'test-old', got %s", named.Name())
		}
	})

	t.Run("Typed Adapter", func(t *testing.T) {
		typed := adapter.AsTyped()
		if typed.Type() != TypeFormatter {
			t.Errorf("Expected type TypeFormatter, got %v", typed.Type())
		}
	})

	t.Run("Configurable Adapter", func(t *testing.T) {
		configurable := adapter.AsConfigurable()

		config := Config{"test": "value"}
		err := configurable.Configure(config)
		if err != nil {
			t.Errorf("Configure failed: %v", err)
		}

		// GetConfig应该返回空配置（适配器实现）
		retrieved := configurable.GetConfig()
		if len(retrieved) != 0 {
			t.Error("Adapter should return empty config for GetConfig()")
		}
	})

	t.Run("Prioritized Adapter", func(t *testing.T) {
		prioritized := adapter.AsPrioritized()

		if prioritized.Priority() != 10 {
			t.Errorf("Expected priority 10, got %d", prioritized.Priority())
		}

		// SetPriority在适配器中是空实现
		prioritized.SetPriority(20)
		// 由于是适配器，优先级不会实际改变
		if prioritized.Priority() != 10 {
			t.Error("Adapter SetPriority should not change original priority")
		}
	})
}

// TestInterfaceChecker 测试接口检查工具
func TestInterfaceChecker(t *testing.T) {
	checker := &InterfaceChecker{}
	component := NewMonitoredComponent("test-checker", TypeHandler, slog.NewTextHandler(os.Stdout, nil))

	t.Run("Check Individual Interfaces", func(t *testing.T) {
		// 检查Named接口
		if named, ok := checker.CheckNamed(component); !ok || named.Name() != "test-checker" {
			t.Error("Should implement Named interface")
		}

		// 检查Typed接口
		if typed, ok := checker.CheckTyped(component); !ok || typed.Type() != TypeHandler {
			t.Error("Should implement Typed interface")
		}

		// 检查Configurable接口
		if configurable, ok := checker.CheckConfigurable(component); !ok {
			t.Error("Should implement Configurable interface")
		} else {
			// 测试配置功能
			config := Config{"key": "value"}
			if err := configurable.Configure(config); err != nil {
				t.Errorf("Configure failed: %v", err)
			}
		}

		// 检查HandlerProvider接口
		if provider, ok := checker.CheckHandlerProvider(component); !ok {
			t.Error("Should implement HandlerProvider interface")
		} else {
			if provider.Handler() == nil {
				t.Error("Handler should not be nil")
			}
		}
	})

	t.Run("Get Implemented Interfaces", func(t *testing.T) {
		interfaces := checker.GetImplementedInterfaces(component)

		expectedInterfaces := []string{"Named", "Typed", "Configurable", "Enableable", "HandlerProvider"}

		for _, expected := range expectedInterfaces {
			found := false
			for _, actual := range interfaces {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected interface %s not found in %v", expected, interfaces)
			}
		}
	})
}

// BenchmarkInterfaceSegregation 接口分离性能基准测试
func BenchmarkInterfaceSegregation(b *testing.B) {
	component := NewMonitoredComponent("benchmark", TypeFormatter, slog.NewTextHandler(os.Stdout, nil))

	b.Run("Direct Method Call", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = component.Name()
			_ = component.Type()
			_ = component.Enabled()
		}
	})

	b.Run("Interface Method Call", func(b *testing.B) {
		var named Named = component
		var typed Typed = component
		var enableable Enableable = component

		for i := 0; i < b.N; i++ {
			_ = named.Name()
			_ = typed.Type()
			_ = enableable.Enabled()
		}
	})

	b.Run("Composed Interface Call", func(b *testing.B) {
		var ci ComponentInterface = component

		for i := 0; i < b.N; i++ {
			_ = ci.Name()
			_ = ci.Type()
			_ = ci.Enabled()
		}
	})
}
