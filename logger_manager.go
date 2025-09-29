package slog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
)

// LoggerManager 全局日志管理器，负责管理所有logger实例
// 解决全局状态混乱问题，实现实例隔离
type LoggerManager struct {
	mu            sync.RWMutex
	defaultLogger *Logger
	instances     map[string]*Logger
	config        *GlobalConfig
	initialized   atomic.Bool
}

// GlobalConfig 全局配置，与实例配置分离
type GlobalConfig struct {
	DefaultWriter  io.Writer
	DefaultLevel   Level
	DefaultNoColor bool
	DefaultSource  bool
	EnableText     bool
	EnableJSON     bool
}

// defaultGlobalConfig 默认全局配置
var defaultGlobalConfig = &GlobalConfig{
	DefaultWriter:  os.Stdout,
	DefaultLevel:   LevelInfo,
	DefaultNoColor: false,
	DefaultSource:  false,
	EnableText:     true,
	EnableJSON:     false,
}

// globalManager 全局管理器实例
var globalManager = &LoggerManager{
	instances: make(map[string]*Logger),
	config:    defaultGlobalConfig,
}

// GetManager 获取全局管理器实例
func GetManager() *LoggerManager {
	return globalManager
}

// GetDefault 获取默认logger实例
// 线程安全，支持延迟初始化
func (lm *LoggerManager) GetDefault() *Logger {
	lm.mu.RLock()
	if lm.defaultLogger != nil {
		defer lm.mu.RUnlock()
		return lm.defaultLogger
	}
	lm.mu.RUnlock()

	// 需要创建默认实例
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 双重检查
	if lm.defaultLogger != nil {
		return lm.defaultLogger
	}

	// 创建默认logger
	lm.defaultLogger = lm.createLoggerWithConfig("default", lm.config)
	return lm.defaultLogger
}

// GetNamed 获取或创建命名logger实例
// 支持实例隔离，每个名称对应独立的logger
func (lm *LoggerManager) GetNamed(name string) *Logger {
	if name == "" || name == "default" {
		return lm.GetDefault()
	}

	lm.mu.RLock()
	if logger, exists := lm.instances[name]; exists {
		lm.mu.RUnlock()
		return logger
	}
	lm.mu.RUnlock()

	// 需要创建新实例
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 双重检查
	if logger, exists := lm.instances[name]; exists {
		return logger
	}

	// 创建新的logger实例
	logger := lm.createLoggerWithConfig(name, lm.config)
	lm.instances[name] = logger
	return logger
}

// Configure 配置全局设置
// 注意：此操作会影响后续创建的logger，但不会影响已存在的实例
func (lm *LoggerManager) Configure(config *GlobalConfig) error {
	if config == nil {
		return NewInvalidInputError("config", "non-nil GlobalConfig", "nil")
	}

	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.config = config
	return nil
}

// Reset 重置管理器状态
// 清除所有实例，在测试中很有用
func (lm *LoggerManager) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.defaultLogger = nil
	lm.instances = make(map[string]*Logger)
	lm.initialized.Store(false)
}

// ListInstances 列出所有已创建的logger实例名称
func (lm *LoggerManager) ListInstances() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	names := make([]string, 0, len(lm.instances)+1)
	if lm.defaultLogger != nil {
		names = append(names, "default")
	}
	for name := range lm.instances {
		names = append(names, name)
	}
	return names
}

// createLoggerWithConfig 使用全局配置创建logger实例
// 这是创建logger的统一入口，确保配置一致性
func (lm *LoggerManager) createLoggerWithConfig(name string, config *GlobalConfig) *Logger {
	options := NewOptions(nil)
	options.AddSource = config.DefaultSource

	// 如果需要DLP,则初始化
	if dlpEnabled.Load() {
		ext.enableDLP()
	}

	writer := config.DefaultWriter
	if writer == nil {
		writer = os.Stdout
	}

	logger := &Logger{
		w:       writer,
		noColor: config.DefaultNoColor,
		level:   config.DefaultLevel,
		ctx:     context.Background(),
		config:  DefaultConfig(), // 使用实例级别的默认配置
	}

	// 根据全局配置决定启用哪些handler
	if config.EnableText {
		logger.text = slog.New(newAddonsHandler(NewConsoleHandler(writer, config.DefaultNoColor, options), ext))
	}
	if config.EnableJSON {
		logger.json = slog.New(newAddonsHandler(NewJSONHandler(writer, options), ext))
	}

	return logger
}

// Shutdown 关闭管理器
// 清理所有资源，程序退出时调用
func (lm *LoggerManager) Shutdown() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 这里可以添加清理逻辑，比如刷新缓冲区、关闭文件等
	lm.defaultLogger = nil
	lm.instances = make(map[string]*Logger)
}

// Stats 返回管理器统计信息
type ManagerStats struct {
	DefaultLoggerExists bool
	InstanceCount       int
	InstanceNames       []string
}

// GetStats 获取管理器统计信息
func (lm *LoggerManager) GetStats() ManagerStats {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	stats := ManagerStats{
		DefaultLoggerExists: lm.defaultLogger != nil,
		InstanceCount:       len(lm.instances),
		InstanceNames:       make([]string, 0, len(lm.instances)),
	}

	for name := range lm.instances {
		stats.InstanceNames = append(stats.InstanceNames, name)
	}

	return stats
}

// 向后兼容的全局函数

// Named 获取命名logger实例（向后兼容）
func Named(name string) *Logger {
	return globalManager.GetNamed(name)
}

// ConfigureGlobal 配置全局设置（向后兼容）
func ConfigureGlobal(config *GlobalConfig) error {
	return globalManager.Configure(config)
}
