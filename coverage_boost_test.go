package slog

import (
	"bytes"
	"strings"
	"testing"
)

// TestLoggerBasicCoverage 基础覆盖率测试
func TestLoggerBasicCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试所有级别的日志记录
	levels := []struct {
		level Level
		logFn func(string, ...any)
		fmtFn func(string, ...any)
		name  string
	}{
		{LevelTrace, logger.Trace, logger.Tracef, "Trace"},
		{LevelDebug, logger.Debug, logger.Debugf, "Debug"},
		{LevelInfo, logger.Info, logger.Infof, "Info"},
		{LevelWarn, logger.Warn, logger.Warnf, "Warn"},
		{LevelError, logger.Error, logger.Errorf, "Error"},
	}

	for _, test := range levels {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			logger.SetLevel(test.level)

			// 测试普通日志
			test.logFn("test message")
			if !strings.Contains(buf.String(), "test message") {
				t.Errorf("%s日志记录失败", test.name)
			}

			// 测试格式化日志
			buf.Reset()
			test.fmtFn("formatted %s %d", "message", 123)
			output := buf.String()
			if !strings.Contains(output, "formatted") || !strings.Contains(output, "message") {
				t.Errorf("%sf格式化日志记录失败", test.name)
			}
		})
	}
}

// TestGlobalFunctionsCoverage 全局函数覆盖率测试
func TestGlobalFunctionsCoverage(t *testing.T) {
	// 测试级别设置函数
	SetLevelTrace()
	SetLevelDebug()
	SetLevelInfo()
	SetLevelWarn()
	SetLevelError()
	SetLevelFatal()

	// 测试启用/禁用函数
	EnableTextLogger()
	DisableTextLogger()
	EnableJSONLogger()
	DisableJSONLogger()
	EnableDLPLogger()
	DisableDLPLogger()

	// 恢复默认设置
	EnableTextLogger()
	DisableJSONLogger()
	SetLevelInfo()
}

// TestLoggerWithFieldsCoverage 带字段日志覆盖率测试
func TestLoggerWithFieldsCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	// 测试With方法
	enriched := logger.With("key1", "value1", "key2", 42)
	enriched.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("With方法测试失败")
	}

	// 测试WithGroup方法
	buf.Reset()
	grouped := logger.WithGroup("testgroup")
	grouped.Info("grouped message", "attr", "value")

	output = buf.String()
	if !strings.Contains(output, "grouped message") {
		t.Error("WithGroup方法测试失败")
	}
}

// TestDefaultLoggerCoverage 默认日志器覆盖率测试
func TestDefaultLoggerCoverage(t *testing.T) {
	// 测试无参数Default
	logger1 := Default()
	if logger1 == nil {
		t.Error("Default()应该返回有效的logger")
	}

	// 测试带模块的Default
	logger2 := Default("module1", "module2")
	if logger2 == nil {
		t.Error("Default(modules...)应该返回有效的logger")
	}

	// 测试获取底层logger
	slogLogger := logger1.GetSlogLogger()
	if slogLogger == nil {
		t.Error("GetSlogLogger()应该返回有效的slog.Logger")
	}

	// 测试级别获取和设置
	original := logger1.GetLevel()
	logger1.SetLevel(LevelWarn)
	if logger1.GetLevel() != LevelWarn {
		t.Error("SetLevel/GetLevel测试失败")
	}
	logger1.SetLevel(original)
}

// TestLoggerLevelFilteringCoverage 日志级别过滤覆盖率测试
func TestLoggerLevelFilteringCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 设置为ERROR级别，较低级别应该被过滤
	logger.SetLevel(LevelError)

	testCases := []struct {
		logFunc   func()
		shouldLog bool
		name      string
	}{
		{func() { logger.Trace("trace") }, false, "Trace"},
		{func() { logger.Debug("debug") }, false, "Debug"},
		{func() { logger.Info("info") }, false, "Info"},
		{func() { logger.Warn("warn") }, false, "Warn"},
		{func() { logger.Error("error") }, true, "Error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.logFunc()
			output := buf.String()

			if tc.shouldLog && output == "" {
				t.Errorf("%s应该被记录", tc.name)
			} else if !tc.shouldLog && output != "" {
				t.Errorf("%s不应该被记录", tc.name)
			}
		})
	}
}

// TestUtilityCoverage 工具函数覆盖率测试
func TestUtilityCoverage(t *testing.T) {
	// 测试Level的String方法
	levels := []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
	for _, level := range levels {
		str := level.String()
		if str == "" {
			t.Errorf("Level %d应该有字符串表示", level)
		}
	}

	// 测试订阅功能
	ch, cancel := Subscribe(10)
	if ch == nil || cancel == nil {
		t.Error("Subscribe应该返回有效的channel和cancel函数")
	}
	cancel() // 清理资源
}

// TestErrorConditionsCoverage 错误条件覆盖率测试
func TestErrorConditionsCoverage(t *testing.T) {
	// 测试nil writer
	logger := NewLogger(nil, false, false)
	if logger == nil {
		t.Error("即使writer为nil也应该返回有效的logger")
	}

	// 测试各种参数组合
	logger1 := NewLogger(&bytes.Buffer{}, true, true)  // color + source
	logger2 := NewLogger(&bytes.Buffer{}, false, true) // no color + source
	logger3 := NewLogger(&bytes.Buffer{}, true, false) // color + no source

	loggers := []*Logger{logger1, logger2, logger3}
	for i, l := range loggers {
		if l == nil {
			t.Errorf("Logger %d应该不为nil", i)
		}
		// 测试基本功能
		l.Info("test")
	}
}

// TestConcurrencyCoverage 并发覆盖率测试
func TestConcurrencyCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelDebug)

	// 简单的并发测试
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				logger.Infof("Goroutine %d message %d", id, j)
			}
		}(i)
	}

	// 等待完成
	for i := 0; i < 3; i++ {
		<-done
	}

	if buf.Len() == 0 {
		t.Error("并发日志记录应该产生输出")
	}
}
