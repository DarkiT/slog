package slog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestLoggerWithGroup 测试日志分组功能
func TestLoggerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	// 创建一个带有分组的日志记录器
	groupLogger := logger.WithGroup("testGroup")
	groupLogger.Info("group message", "key", "value")

	output := buf.String()
	// 检查输出中是否包含分组信息
	if !strings.Contains(output, "testGroup") {
		t.Errorf("Expected output to contain group name, got: %s", output)
	}
}

// TestLoggerFormat 测试格式化日志功能
func TestLoggerFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	// 测试不同格式的日志输出
	testCases := []struct {
		name     string
		logFunc  func(string, ...any)
		message  string
		args     []any
		expected string
	}{
		{
			name:     "Infof",
			logFunc:  logger.Infof,
			message:  "formatted %s",
			args:     []any{"message"},
			expected: "formatted message",
		},
		{
			name:     "Errorf",
			logFunc:  logger.Errorf,
			message:  "error: %d",
			args:     []any{42},
			expected: "error: 42",
		},
		{
			name:     "Warnf",
			logFunc:  logger.Warnf,
			message:  "warning %s: %d",
			args:     []any{"code", 123},
			expected: "warning code: 123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.logFunc(tc.message, tc.args...)
			output := buf.String()
			if !strings.Contains(output, tc.expected) {
				t.Errorf("Expected output to contain '%s', got: %s", tc.expected, output)
			}
		})
	}
}

// TestLoggerWith 测试With函数添加context
func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	// 创建带有附加字段的日志记录器
	withLogger := logger.With(
		"string", "value",
		"number", 42,
		"bool", true,
	)

	buf.Reset()
	withLogger.Info("test with")

	output := buf.String()
	// 检查所有附加字段是否存在
	if !strings.Contains(output, "string=value") {
		t.Errorf("Expected output to contain 'string=value', got: %s", output)
	}
	if !strings.Contains(output, "number=42") {
		t.Errorf("Expected output to contain 'number=42', got: %s", output)
	}
	if !strings.Contains(output, "bool=true") {
		t.Errorf("Expected output to contain 'bool=true', got: %s", output)
	}
}

// TestSubscribe 测试订阅功能
func TestSubscribe(t *testing.T) {
	logger := NewLogger(nil, false, false)

	// 创建订阅
	records, cancel := Subscribe(10)
	defer cancel()

	// 发送几条日志消息
	logger.Info("test message 1")
	logger.Error("test message 2")

	// 检查是否收到日志记录
	receivedCount := 0
	timeout := time.After(time.Second)

	for receivedCount < 2 {
		select {
		case record := <-records:
			receivedCount++
			if record.Level != LevelInfo && record.Level != LevelError {
				t.Errorf("Unexpected log level: %v", record.Level)
			}
		case <-timeout:
			t.Fatalf("Timed out waiting for log records, received %d", receivedCount)
			return
		}
	}
}

// TestDefaultWithModules 测试带模块前缀的Default函数
func TestDefaultWithModules(t *testing.T) {
	var buf bytes.Buffer
	logger = NewLogger(&buf, false, false)

	// 使用模块名创建新日志记录器
	moduleLogger := Default("test", "module")

	moduleLogger.Info("module message")

	output := buf.String()
	if !strings.Contains(output, "test.module") {
		t.Errorf("Expected output to contain module name 'test.module', got: %s", output)
	}
}

// TestProgressBar 测试进度条功能
func TestProgressBar(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	// 测试进度条值设置
	logger.ProgressBarWithValue("Test Progress", 0.5, 10)

	output := buf.String()
	if !strings.Contains(output, "Test Progress") {
		t.Errorf("Expected output to contain progress message, got: %s", output)
	}

	// 检查进度百分比
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected output to contain \"50%%\", got: %s", output)
	}
}

// TestFormatLog 测试格式检测功能
func TestFormatLog(t *testing.T) {
	// 测试不含格式说明符的情况
	result := formatLog("Simple message")
	if result {
		t.Error("formatLog should return false for message without format specifiers")
	}

	// 测试包含格式说明符的情况
	result = formatLog("Message with %s", "placeholder")
	if !result {
		t.Error("formatLog should return true for message with format specifiers")
	}
}

// TestNewOptions 测试选项创建
func TestNewOptions(t *testing.T) {
	options := NewOptions(nil)

	// 验证选项设置
	if options.Level != &levelVar {
		t.Error("Level should be set to levelVar")
	}

	// 测试ReplaceAttr函数
	source := &slog.Source{
		Function: "TestFunc",
		File:     "/path/to/file.go",
		Line:     42,
	}

	attr := slog.Attr{
		Key:   slog.SourceKey,
		Value: slog.AnyValue(source),
	}

	newAttr := options.ReplaceAttr(nil, attr)
	newSource := newAttr.Value.Any().(*slog.Source)

	if newSource.File != "file.go" {
		t.Errorf("Expected file to be 'file.go', got: %s", newSource.File)
	}
}

// TestLoggerWithValue 测试带值的日志记录器
func TestLoggerWithValue(t *testing.T) {
	// 重置环境
	resetForTest()

	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)
	logger.SetLevel(LevelTrace) // 确保最低日志级别
	EnableTextLogger()          // 确保文本输出启用

	// 直接检查全局设置
	t.Logf("Text enabled: %v, JSON enabled: %v, Level: %v",
		textEnabled, jsonEnabled, levelVar.Level())

	logger.WithValue("test", "value").Info("test message")

	output := buf.String()
	t.Logf("Buffer content: %q", output) // 打印实际输出内容

	if !strings.Contains(output, "test=value") {
		t.Errorf("Expected output to contain context value, got: %s", output)
	}
}

// TestLoggerLevel 测试日志级别设置和过滤
func TestLoggerLevel(t *testing.T) {
	// 重置全局配置
	resetForTest()

	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)
	// 启用文本输出
	EnableTextLogger()

	// 设置日志级别为Info
	logger.SetLevel(LevelInfo)

	// Debug级别的日志不应该输出
	buf.Reset()
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Errorf("Debug message should not be logged at Info level, got: %s", buf.String())
	}

	// Info级别的日志应该输出
	buf.Reset()
	logger.Info("info message")
	t.Logf("Info output: %q", buf.String()) // 添加实际输出日志
	if buf.Len() == 0 {
		t.Error("Info message should be logged at Info level")
	}
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("Expected output to contain 'info message', got: %s", buf.String())
	}

	// 设置日志级别为Debug
	logger.SetLevel(LevelDebug)

	// Debug级别的日志现在应该输出
	buf.Reset()
	logger.Debug("debug message")
	if buf.Len() == 0 {
		t.Error("Debug message should be logged at Debug level")
	}
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("Expected output to contain 'debug message', got: %s", buf.String())
	}
}

// resetForTest 用于重置测试环境
func resetForTest() {
	// 重置全局配置
	levelVar.Set(LevelTrace) // 使用最低级别确保所有日志都能输出
	// 启用文本输出，禁用JSON输出
	textEnabled, jsonEnabled = true, false
}

// TestJSONLogger 测试JSON格式输出
func TestJSONLogger(t *testing.T) {
	// 重置全局配置
	resetForTest()

	var buf bytes.Buffer

	// 创建新的logger并确保启用JSON格式
	logger := NewLogger(&buf, false, false)
	EnableJsonLogger()
	DisableTextLogger()

	// 输出一条日志
	logger.Info("json test", "key", "value")

	jsonOutput := buf.String()
	t.Logf("JSON output: %q", jsonOutput) // 输出实际JSON内容以便调试

	// 检查是否为空
	if len(jsonOutput) == 0 {
		t.Fatal("Expected JSON output but got empty string")
	}

	// 解析JSON输出
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput was: %q", err, jsonOutput)
	}

	// 验证JSON字段
	if msg, ok := logEntry["msg"]; !ok || msg != "json test" {
		t.Errorf("Expected msg field to be 'json test', got: %v", msg)
	}

	if val, ok := logEntry["key"]; !ok || val != "value" {
		t.Errorf("Expected key field to be 'value', got: %v", val)
	}

	// 恢复默认设置
	EnableTextLogger()
	DisableJsonLogger()
}
