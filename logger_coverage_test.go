package slog

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestLoggerConfiguration 测试日志器配置相关功能
func TestLoggerConfiguration(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "默认配置",
			test: func(t *testing.T) {
				logger := Default()
				if logger == nil {
					t.Error("默认日志器不应该为nil")
				}

				// 测试获取级别
				level := logger.GetLevel()
				if level < LevelTrace || level > LevelFatal {
					t.Errorf("默认级别不合法: %v", level)
				}
			},
		},
		{
			name: "设置级别",
			test: func(t *testing.T) {
				logger := Default()

				// 测试设置不同级别
				levels := []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
				for _, level := range levels {
					logger.SetLevel(level)
					if logger.GetLevel() != level {
						t.Errorf("设置级别失败: 期望 %v, 得到 %v", level, logger.GetLevel())
					}
				}
			},
		},
		{
			name: "获取底层logger",
			test: func(t *testing.T) {
				logger := Default()
				slogLogger := logger.GetSlogLogger()
				if slogLogger == nil {
					t.Error("底层slog.Logger不应该为nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestLoggerOutput 测试日志输出功能
func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false) // 无色，无源码

	tests := []struct {
		name   string
		logFn  func()
		expect string
	}{
		{
			name: "Trace日志",
			logFn: func() {
				logger.SetLevel(LevelTrace)
				logger.Trace("trace message")
			},
			expect: "trace message",
		},
		{
			name: "Debug日志",
			logFn: func() {
				logger.SetLevel(LevelDebug)
				logger.Debug("debug message")
			},
			expect: "debug message",
		},
		{
			name: "Info日志",
			logFn: func() {
				logger.SetLevel(LevelInfo)
				logger.Info("info message")
			},
			expect: "info message",
		},
		{
			name: "Warn日志",
			logFn: func() {
				logger.SetLevel(LevelWarn)
				logger.Warn("warn message")
			},
			expect: "warn message",
		},
		{
			name: "Error日志",
			logFn: func() {
				logger.SetLevel(LevelError)
				logger.Error("error message")
			},
			expect: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn()
			output := buf.String()
			if !strings.Contains(output, tt.expect) {
				t.Errorf("输出应该包含 %q, 得到: %q", tt.expect, output)
			}
		})
	}
}

// TestLoggerFormatting 测试格式化日志
func TestLoggerFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	tests := []struct {
		name   string
		logFn  func()
		expect []string // 期望的包含内容
	}{
		{
			name: "Debugf格式化",
			logFn: func() {
				logger.Debugf("用户 %s 登录成功", "张三")
			},
			expect: []string{"用户", "张三", "登录成功"},
		},
		{
			name: "Infof格式化",
			logFn: func() {
				logger.Infof("处理时间: %d ms", 150)
			},
			expect: []string{"处理时间", "150", "ms"},
		},
		{
			name: "Warnf格式化",
			logFn: func() {
				logger.Warnf("CPU使用率: %.2f%%", 85.67)
			},
			expect: []string{"CPU使用率", "85.67"},
		},
		{
			name: "Errorf格式化",
			logFn: func() {
				logger.Errorf("连接失败: %v", "网络超时")
			},
			expect: []string{"连接失败", "网络超时"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn()
			output := buf.String()
			for _, expect := range tt.expect {
				if !strings.Contains(output, expect) {
					t.Errorf("输出应该包含 %q, 得到: %q", expect, output)
				}
			}
		})
	}
}

// TestLoggerWithFields 测试带字段的日志
func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	// 测试With方法
	enrichedLogger := logger.With("module", "test", "version", "1.0")
	enrichedLogger.Info("测试消息")

	output := buf.String()
	expected := []string{"测试消息", "module", "test", "version", "1.0"}
	for _, expect := range expected {
		if !strings.Contains(output, expect) {
			t.Errorf("输出应该包含 %q, 得到: %q", expect, output)
		}
	}
}

// TestLoggerWithGroupCoverage 测试分组功能覆盖
func TestLoggerWithGroupCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	// 测试WithGroup方法
	groupLogger := logger.WithGroup("api")
	groupLogger.Info("请求处理", "method", "GET", "path", "/users")

	output := buf.String()
	if !strings.Contains(output, "请求处理") {
		t.Errorf("输出应该包含消息")
	}
}

// TestGlobalLoggerFunctions 测试全局日志函数
func TestGlobalLoggerFunctions(t *testing.T) {
	// 保存原始设置
	originalTextEnabled := textEnabled
	originalJSONEnabled := jsonEnabled

	defer func() {
		// 恢复原始设置
		if originalTextEnabled {
			EnableTextLogger()
		} else {
			DisableTextLogger()
		}
		if originalJSONEnabled {
			EnableJSONLogger()
		} else {
			DisableJSONLogger()
		}
	}()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "启用文本日志",
			test: func(t *testing.T) {
				EnableTextLogger()
				if !textEnabled {
					t.Error("文本日志应该被启用")
				}
			},
		},
		{
			name: "禁用文本日志",
			test: func(t *testing.T) {
				DisableTextLogger()
				if textEnabled {
					t.Error("文本日志应该被禁用")
				}
			},
		},
		{
			name: "启用JSON日志",
			test: func(t *testing.T) {
				EnableJSONLogger()
				if !jsonEnabled {
					t.Error("JSON日志应该被启用")
				}
			},
		},
		{
			name: "禁用JSON日志",
			test: func(t *testing.T) {
				DisableJSONLogger()
				if jsonEnabled {
					t.Error("JSON日志应该被禁用")
				}
			},
		},
		{
			name: "DLP日志启用测试",
			test: func(t *testing.T) {
				EnableDLPLogger()
				// 这里只测试函数调用不出错
				DisableDLPLogger()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestLoggerLevelControl 测试日志级别控制
func TestLoggerLevelControl(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 设置为WARN级别，INFO和DEBUG应该被过滤
	logger.SetLevel(LevelWarn)

	tests := []struct {
		name      string
		logFn     func()
		shouldLog bool
	}{
		{
			name: "Trace不应该记录",
			logFn: func() {
				logger.Trace("trace message")
			},
			shouldLog: false,
		},
		{
			name: "Debug不应该记录",
			logFn: func() {
				logger.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name: "Info不应该记录",
			logFn: func() {
				logger.Info("info message")
			},
			shouldLog: false,
		},
		{
			name: "Warn应该记录",
			logFn: func() {
				logger.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name: "Error应该记录",
			logFn: func() {
				logger.Error("error message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn()
			output := buf.String()

			if tt.shouldLog {
				if output == "" {
					t.Error("应该有日志输出")
				}
			} else {
				if output != "" {
					t.Errorf("不应该有日志输出，得到: %q", output)
				}
			}
		})
	}
}

// TestLoggerErrorConditions 测试错误条件
func TestLoggerErrorConditions(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "nil输出器",
			test: func(t *testing.T) {
				// 这应该不会panic
				logger := NewLogger(nil, false, false)
				if logger == nil {
					t.Error("即使输出器为nil，也应该返回有效的logger")
				}
			},
		},
		{
			name: "无效级别",
			test: func(t *testing.T) {
				logger := Default()

				// 设置无效级别
				logger.SetLevel(Level(999))

				// 验证级别是否被正确处理
				currentLevel := logger.GetLevel()
				if currentLevel == Level(999) {
					t.Error("不应该接受无效的日志级别")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestLoggerConcurrency 测试并发安全
func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	// 并发写入测试
	const numGoroutines = 10
	const numMessages = 100

	done := make(chan struct{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()

			for j := 0; j < numMessages; j++ {
				logger.Infof("Goroutine %d - Message %d", id, j)
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	output := buf.String()
	if output == "" {
		t.Error("应该有并发日志输出")
	}
}

// TestUtilityFunctions 测试工具函数
func TestUtilityFunctions(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "级别字符串转换",
			test: func(t *testing.T) {
				levels := []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
				for _, level := range levels {
					str := level.String()
					if str == "" {
						t.Errorf("级别 %d 应该有字符串表示", level)
					}
				}
			},
		},
		{
			name: "模块化日志器",
			test: func(t *testing.T) {
				modules := []string{"auth", "api", "database", "cache"}
				for _, module := range modules {
					logger := Default(module)
					if logger == nil {
						t.Errorf("模块 %s 的日志器不应该为nil", module)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
