package slog

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestLoggerManager_GetDefault(t *testing.T) {
	// 创建新的管理器实例用于测试
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 测试延迟初始化
	logger1 := manager.GetDefault()
	if logger1 == nil {
		t.Fatal("GetDefault() 应该返回非nil的logger")
	}

	// 测试单例行为
	logger2 := manager.GetDefault()
	if logger1 != logger2 {
		t.Error("GetDefault() 应该返回同一个实例")
	}
}

func TestLoggerManager_GetNamed(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 测试创建命名logger
	apiLogger := manager.GetNamed("api")
	if apiLogger == nil {
		t.Fatal("GetNamed() 应该返回非nil的logger")
	}

	// 测试单例行为
	apiLogger2 := manager.GetNamed("api")
	if apiLogger != apiLogger2 {
		t.Error("GetNamed() 对同一名称应该返回同一个实例")
	}

	// 测试不同名称返回不同实例
	dbLogger := manager.GetNamed("db")
	if apiLogger == dbLogger {
		t.Error("不同名称应该返回不同的logger实例")
	}

	// 测试空名称返回默认logger
	defaultLogger := manager.GetNamed("")
	expectedDefault := manager.GetDefault()
	if defaultLogger != expectedDefault {
		t.Error("空名称应该返回默认logger")
	}
}

func TestLoggerManager_Configure(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 测试nil配置
	err := manager.Configure(nil)
	if err == nil {
		t.Error("Configure(nil) 应该返回错误")
	}

	// 测试有效配置
	newConfig := &GlobalConfig{
		DefaultWriter:  os.Stderr,
		DefaultLevel:   LevelDebug,
		DefaultNoColor: true,
		DefaultSource:  true,
		EnableText:     false,
		EnableJSON:     true,
	}

	err = manager.Configure(newConfig)
	if err != nil {
		t.Fatalf("Configure() 失败: %v", err)
	}

	if manager.config != newConfig {
		t.Error("配置没有正确设置")
	}
}

func TestLoggerManager_Reset(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 创建一些logger实例
	manager.GetDefault()
	manager.GetNamed("test")

	// 验证实例已创建
	if manager.defaultLogger == nil {
		t.Fatal("默认logger应该已创建")
	}
	if len(manager.instances) != 1 {
		t.Fatalf("期望1个命名实例，实际: %d", len(manager.instances))
	}

	// 重置
	manager.Reset()

	// 验证重置后状态
	if manager.defaultLogger != nil {
		t.Error("重置后默认logger应该为nil")
	}
	if len(manager.instances) != 0 {
		t.Errorf("重置后应该没有实例，实际: %d", len(manager.instances))
	}
}

func TestLoggerManager_ListInstances(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 初始状态
	names := manager.ListInstances()
	if len(names) != 0 {
		t.Errorf("初始状态应该没有实例，实际: %v", names)
	}

	// 创建默认logger
	manager.GetDefault()
	names = manager.ListInstances()
	if len(names) != 1 || names[0] != "default" {
		t.Errorf("期望[default]，实际: %v", names)
	}

	// 创建命名logger
	manager.GetNamed("api")
	manager.GetNamed("db")
	names = manager.ListInstances()
	if len(names) != 3 {
		t.Errorf("期望3个实例，实际: %d", len(names))
	}

	// 验证包含所有期望的名称
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	expected := []string{"default", "api", "db"}
	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("缺少期望的实例名称: %s", name)
		}
	}
}

func TestLoggerManager_GetStats(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	// 初始状态
	stats := manager.GetStats()
	if stats.DefaultLoggerExists {
		t.Error("初始状态默认logger不应该存在")
	}
	if stats.InstanceCount != 0 {
		t.Errorf("初始状态实例数量应该为0，实际: %d", stats.InstanceCount)
	}

	// 创建一些实例
	manager.GetDefault()
	manager.GetNamed("test1")
	manager.GetNamed("test2")

	stats = manager.GetStats()
	if !stats.DefaultLoggerExists {
		t.Error("默认logger应该存在")
	}
	if stats.InstanceCount != 2 {
		t.Errorf("期望2个命名实例，实际: %d", stats.InstanceCount)
	}
	if len(stats.InstanceNames) != 2 {
		t.Errorf("期望2个实例名称，实际: %d", len(stats.InstanceNames))
	}
}

func TestLoggerManager_ConcurrentAccess(t *testing.T) {
	manager := &LoggerManager{
		instances: make(map[string]*Logger),
		config:    defaultGlobalConfig,
	}

	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 并发测试
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// 并发访问默认logger
				logger := manager.GetDefault()
				if logger == nil {
					t.Errorf("Goroutine %d: GetDefault返回nil", id)
				}

				// 并发访问命名logger
				name := "test"
				namedLogger := manager.GetNamed(name)
				if namedLogger == nil {
					t.Errorf("Goroutine %d: GetNamed返回nil", id)
				}

				// 并发获取统计信息
				stats := manager.GetStats()
				if !stats.DefaultLoggerExists {
					t.Errorf("Goroutine %d: 默认logger应该存在", id)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证最终状态
	stats := manager.GetStats()
	if !stats.DefaultLoggerExists {
		t.Error("并发测试后默认logger应该存在")
	}
	if stats.InstanceCount != 1 {
		t.Errorf("期望1个命名实例，实际: %d", stats.InstanceCount)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// 测试向后兼容的全局函数

	// 重置全局状态
	globalManager.Reset()

	// 测试Default()
	logger1 := Default()
	if logger1 == nil {
		t.Fatal("Default() 应该返回非nil的logger")
	}

	logger2 := Default()
	if logger1 != logger2 {
		t.Error("Default() 应该返回同一个实例")
	}

	// 测试Named()
	apiLogger := Named("api")
	if apiLogger == nil {
		t.Fatal("Named() 应该返回非nil的logger")
	}

	if apiLogger == logger1 {
		t.Error("命名logger应该与默认logger不同")
	}

	// 测试ConfigureGlobal()
	var buf bytes.Buffer
	config := &GlobalConfig{
		DefaultWriter:  &buf,
		DefaultLevel:   LevelDebug,
		DefaultNoColor: true,
		EnableText:     true,
		EnableJSON:     false,
	}

	err := ConfigureGlobal(config)
	if err != nil {
		t.Fatalf("ConfigureGlobal() 失败: %v", err)
	}
}

func TestResetGlobalLogger(t *testing.T) {
	var buf bytes.Buffer

	// 测试ResetGlobalLogger
	ResetGlobalLogger(&buf, true, true)

	// 验证重置后可以正常工作
	logger := Default()
	if logger == nil {
		t.Fatal("重置后Default()应该返回有效的logger")
	}

	// 简单的日志测试
	logger.Info("测试消息")
	output := buf.String()
	if !strings.Contains(output, "测试消息") {
		t.Error("日志输出应该包含测试消息")
	}
}
