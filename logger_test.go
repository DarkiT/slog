package slog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLoggerWithGroup 测试日志分组功能
func TestLoggerWithGroup(t *testing.T) {
	// 重置环境
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, false)
	SetLevelInfo() // 确保级别允许Info日志
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, false)
	logger.SetLevel(LevelInfo) // 确保实例级别允许Info日志

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
	ResetGlobalLogger(&buf, false, false)
	SetLevelTrace() // 设置为最低级别以允许所有日志
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, false)
	logger.SetLevel(LevelTrace) // 确保实例级别允许所有日志

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
	ResetGlobalLogger(&buf, false, false)
	SetLevelInfo() // 确保级别允许Info日志
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, false)
	logger.SetLevel(LevelInfo) // 确保实例级别允许Info日志

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

func TestDerivedSlogLoggerRetainsLazyAttrsAndGroups(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, false)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, false).WithGroup("request").With("trace_id", "t-1")

	buf.Reset()
	logger.GetSlogLogger().Info("derived slog logger", "user", "alice")

	output := buf.String()
	if !strings.Contains(output, "request.trace_id=t-1") {
		t.Fatalf("expected derived slog logger to include grouped trace_id, got: %s", output)
	}
	if !strings.Contains(output, "request.user=alice") {
		t.Fatalf("expected derived slog logger to include grouped runtime attr, got: %s", output)
	}
}

func TestDLPMessageDesensitizesRawSensitiveValueWithoutKeyword(t *testing.T) {
	resetForTest()
	EnableDLPLogger()
	defer DisableDLPLogger()

	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	logger.Info("联系我 13812345678")

	output := buf.String()
	if strings.Contains(output, "13812345678") {
		t.Fatalf("expected raw sensitive value in message to be desensitized, got %q", output)
	}
	if !strings.Contains(output, "138****5678") {
		t.Fatalf("expected masked phone value in output, got %q", output)
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
		case event := <-records:
			receivedCount++
			record := event.Record
			if record.Level != LevelInfo && record.Level != LevelError {
				t.Errorf("Unexpected log level: %v", record.Level)
			}
			if event.Rendered == "" || event.Format == "" {
				t.Fatalf("expected subscription event to include active semantic render, got %+v", event)
			}
		case <-timeout:
			t.Fatalf("Timed out waiting for log records, received %d", receivedCount)
			return
		}
	}
}

func TestSubscribeWithOptions_DropOldestStats(t *testing.T) {
	logger := NewLogger(nil, false, false)
	records, cancel := SubscribeWithOptions(SubscribeOptions{
		BufferSize:   1,
		Backpressure: SubscriptionDropOldest,
	})
	defer cancel()

	logger.Info("msg-1")
	logger.Info("msg-2")
	logger.Info("msg-3")
	time.Sleep(20 * time.Millisecond)

	stats := GetSubscriptionStats()
	if stats.Dropped == 0 {
		t.Fatal("expected dropped records with drop_oldest policy")
	}
	if stats.DroppedOldest == 0 {
		t.Fatal("expected dropped_oldest metric to increase")
	}

	// 消费一条，验证订阅通道仍可用。
	select {
	case <-records:
	case <-time.After(time.Second):
		t.Fatal("expected at least one record in subscriber channel")
	}
}

func TestSubscribeWithOptions_DropNewestStats(t *testing.T) {
	logger := NewLogger(nil, false, false)
	_, cancel := SubscribeWithOptions(SubscribeOptions{
		BufferSize:   1,
		Backpressure: SubscriptionDropNewest,
	})
	defer cancel()

	logger.Info("msg-1")
	logger.Info("msg-2")
	logger.Info("msg-3")
	time.Sleep(20 * time.Millisecond)

	stats := GetSubscriptionStats()
	if stats.DroppedNewest == 0 {
		t.Fatal("expected dropped_newest metric to increase")
	}
}

func TestSubscribeWithOptions_BlockWithTimeoutStats(t *testing.T) {
	logger := NewLogger(nil, false, false)
	_, cancel := SubscribeWithOptions(SubscribeOptions{
		BufferSize:   1,
		Backpressure: SubscriptionBlockWithTimeout,
		BlockTimeout: 2 * time.Millisecond,
	})
	defer cancel()

	logger.Info("msg-1")
	logger.Info("msg-2")
	logger.Info("msg-3")
	time.Sleep(30 * time.Millisecond)

	stats := GetSubscriptionStats()
	if stats.DroppedTimed == 0 {
		t.Fatal("expected dropped_timed_out metric to increase")
	}
}

func TestSubscribeCancelConcurrentPublish(t *testing.T) {
	logger := NewLogger(nil, false, false)
	records, cancel := SubscribeWithOptions(SubscribeOptions{
		BufferSize:   1,
		Backpressure: SubscriptionDropOldest,
	})
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case _, ok := <-records:
				if !ok {
					return
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			logger.Info("concurrent publish", "idx", i)
			if i == 32 {
				cancel()
			}
		}
	}()

	wg.Wait()
	close(done)
	cancel()
}

// TestDefaultWithModules 测试带模块前缀的Default函数
func TestDefaultWithModules(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, false)
	SetLevelInfo() // 确保级别允许Info日志
	EnableTextLogger()
	DisableJSONLogger()

	// 使用模块名创建新日志记录器
	moduleLogger := Default("test", "module")

	moduleLogger.Info("module message")

	output := buf.String()
	if !strings.Contains(output, "test.module") {
		t.Errorf("Expected output to contain module name 'test.module', got: %s", output)
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
		isGlobalTextEnabled(), isGlobalJSONEnabled(), levelVar.Level())

	logger.WithValue("test", "value").Info("test message")

	output := buf.String()
	t.Logf("Buffer content: %q", output) // 打印实际输出内容

	if !strings.Contains(output, "test=value") {
		t.Errorf("Expected output to contain context value, got: %s", output)
	}
}

func TestLoggerWithValueSurvivesDerivedContextCancel(t *testing.T) {
	resetForTest()

	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false).WithValue("trace_id", "abc-123")
	logger.SetLevel(LevelInfo)
	EnableTextLogger()

	timeoutLogger, cancel := logger.WithTimeout(time.Hour)
	cancel()

	timeoutLogger.Info("after cancel")
	output := buf.String()
	if !strings.Contains(output, "trace_id=abc-123") {
		t.Fatalf("context cancellation should not clear logger fields, got %q", output)
	}
}

func TestDefaultModuleWorksWhenTextDisabled(t *testing.T) {
	resetForTest()
	t.Cleanup(func() {
		_ = GetManager().Configure(defaultGlobalConfig)
		GetManager().Reset()
		resetForTest()
	})

	var buf bytes.Buffer
	config := &GlobalConfig{
		DefaultWriter:  &buf,
		DefaultLevel:   LevelInfo,
		DefaultNoColor: true,
		DefaultSource:  false,
		EnableText:     false,
		EnableJSON:     true,
	}
	if err := GetManager().Configure(config); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	GetManager().Reset()

	logger := Default("api")
	logger.Info("module json")

	output := buf.String()
	if !strings.Contains(output, "[api] module json") {
		t.Fatalf("module prefix should work without text handler, got %q", output)
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
	setGlobalTextEnabled(true)
	setGlobalJSONEnabled(false)
}

func TestLoggerSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelInfo)

	buf.Reset()
	emitSourceInfoLog(logger)

	output := buf.String()
	if !strings.Contains(output, "logger_test.go") {
		t.Fatalf("expected source to point at test file, got: %s", output)
	}
	if strings.Contains(output, "logger.go") {
		t.Fatalf("expected source to skip wrapper file, got: %s", output)
	}
}

func TestGlobalSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	emitGlobalSourceInfoLog()

	output := buf.String()
	if !strings.Contains(output, "logger_test.go") {
		t.Fatalf("expected global source to point at test file, got: %s", output)
	}
	if strings.Contains(output, "log.go") || strings.Contains(output, "logger.go") {
		t.Fatalf("expected global source to skip slog wrappers, got: %s", output)
	}
}

func emitSourceInfoLog(logger *Logger) {
	logger.Info("source check")
}

func emitGlobalSourceInfoLog() {
	Info("global source check")
}

func TestWrappedSourcePointsToBusinessCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitWrappedSourceInfoLog(logger)
	})
}

func TestGlobalFormattedSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitGlobalFormattedSourceInfoLog()
	})
}

func TestLoggerFormattedSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitLoggerFormattedSourceInfoLog(logger)
	})
}

func TestLoggerContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitLoggerContextSourceInfoLog(logger)
	})
}

func TestDerivedLoggerSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitDerivedLoggerSourceInfoLog(logger)
	})
}

func testWithSlogSourceLogger(t *testing.T, emit func(logger *Logger)) {
	t.Helper()
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelInfo)

	buf.Reset()
	emit(logger)

	assertSlogSourceOutput(t, buf.String())
}

func assertSlogSourceOutput(t *testing.T, output string) {
	t.Helper()
	if !strings.Contains(output, "logger_test.go") {
		t.Fatalf("expected source to point at test file, got: %s", output)
	}
	if strings.Contains(output, "logger.go") {
		t.Fatalf("expected source to skip slog wrapper file, got: %s", output)
	}
	if strings.Contains(output, "log.go") {
		t.Fatalf("expected source to skip slog global wrapper file, got: %s", output)
	}
	if strings.Contains(output, "zap.go") {
		t.Fatalf("expected source to skip zap wrapper file, got: %s", output)
	}
}

func emitWrappedSourceInfoLog(logger *Logger) {
	wrappedSourceBridge(logger)
}

func wrappedSourceBridge(logger *Logger) {
	logger.Info("wrapped source check")
}

func emitGlobalFormattedSourceInfoLog() {
	globalFormattedSourceBridge()
}

func globalFormattedSourceBridge() {
	Infof("global formatted source %s", "check")
}

func emitLoggerFormattedSourceInfoLog(logger *Logger) {
	loggerFormattedSourceBridge(logger)
}

func loggerFormattedSourceBridge(logger *Logger) {
	logger.Infof("logger formatted source %s", "check")
}

func emitLoggerContextSourceInfoLog(logger *Logger) {
	loggerContextSourceBridge(logger)
}

func loggerContextSourceBridge(logger *Logger) {
	logger.InfoContext(context.Background(), "logger context source check", "key", "value")
}

func emitDerivedLoggerSourceInfoLog(logger *Logger) {
	derivedLoggerSourceBridge(logger.With("scope", "derived"))
}

func derivedLoggerSourceBridge(logger *Logger) {
	logger.Info("derived logger source check")
}

func benchmarkWrappedSourceBridge(logger *Logger) {
	logger.Info("benchmark helper should not appear")
}

func emitBenchmarkWrappedSourceLog(logger *Logger) {
	benchmarkWrappedSourceBridge(logger)
}

func TestBenchmarkLikeWrappedSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		emitBenchmarkWrappedSourceLog(logger)
	})
}

func testWithSlogSourceLoggerGlobalOnly(t *testing.T, emit func()) {
	t.Helper()
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	emit()

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		InfoContext(context.Background(), "global context source check", "key", "value")
	})
}

func TestGlobalPrintLikeSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Printf("global printf source %s", "check")
	})
}

func TestGlobalDebugLikeSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Debug("global debug source check")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerWithGroupSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		groupBridge(logger.WithGroup("source-group"))
	})
}

func groupBridge(logger *Logger) {
	logger.Info("group source check")
}

func TestLoggerWarnSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		warnBridge(logger)
	})
}

func warnBridge(logger *Logger) {
	logger.Warn("warn source check")
}

func TestLoggerErrorSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		errorBridge(logger)
	})
}

func errorBridge(logger *Logger) {
	logger.Error("error source check")
}

func TestLoggerTracefSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelTrace)

	buf.Reset()
	tracefBridge(logger)

	assertSlogSourceOutput(t, buf.String())
}

func tracefBridge(logger *Logger) {
	logger.Tracef("trace source %s", "check")
}

func TestGlobalTraceSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Trace("global trace source check")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalWarnSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Warn("global warn source check")
	})
}

func TestGlobalErrorSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Error("global error source check")
	})
}

func TestGlobalInfoWithFieldsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Info("global info source check", "key", "value")
	})
}

func TestLoggerInfoWithFieldsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Info("logger info source check", "key", "value")
	})
}

func TestGlobalWithDerivedLoggerSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		derivedLoggerSourceBridge(Default("module").With("scope", "module-derived"))
	})
}

func TestPrintlnSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Println("global println source check")
	})
}

func TestDebugfSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Debugf("global debugf source %s", "check")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerWarnfSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Warnf("warnf source %s", "check")
	})
}

func TestLoggerErrorfSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Errorf("errorf source %s", "check")
	})
}

func TestGlobalWarnfSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Warnf("global warnf source %s", "check")
	})
}

func TestGlobalErrorfSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Errorf("global errorf source %s", "check")
	})
}

func TestGlobalTracefSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Tracef("global tracef source %s", "check")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerTraceContextSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelTrace)

	buf.Reset()
	logger.TraceContext(context.Background(), "logger trace context source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalDebugContextSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	DebugContext(context.Background(), "global debug context source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalTraceContextSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	TraceContext(context.Background(), "global trace context source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalInfofContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		InfofContext(context.Background(), "global infof context %s", "check")
	})
}

func TestLoggerInfofContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.InfofContext(context.Background(), "logger infof context %s", "check")
	})
}

func TestLoggerWarnContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.WarnContext(context.Background(), "logger warn context source check", "key", "value")
	})
}

func TestLoggerErrorContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.ErrorContext(context.Background(), "logger error context source check", "key", "value")
	})
}

func TestGlobalWarnContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		WarnContext(context.Background(), "global warn context source check", "key", "value")
	})
}

func TestGlobalErrorContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		ErrorContext(context.Background(), "global error context source check", "key", "value")
	})
}

func TestGlobalDebugfContextSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	DebugfContext(context.Background(), "global debugf context %s", "check")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerDebugfContextSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelDebug)

	buf.Reset()
	logger.DebugfContext(context.Background(), "logger debugf context %s", "check")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerWarnfContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.WarnfContext(context.Background(), "logger warnf context %s", "check")
	})
}

func TestLoggerErrorfContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.ErrorfContext(context.Background(), "logger errorf context %s", "check")
	})
}

func TestGlobalWarnfContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		WarnfContext(context.Background(), "global warnf context %s", "check")
	})
}

func TestGlobalErrorfContextSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		ErrorfContext(context.Background(), "global errorf context %s", "check")
	})
}

func TestGlobalDefaultModuleLoggerSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		moduleLoggerBridge(Default("module", "sub"))
	})
}

func moduleLoggerBridge(logger *Logger) {
	logger.Info("module logger source check")
}

func TestLoggerWithValueSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.WithValue("trace_id", "abc").Info("with value source check")
	})
}

func TestGlobalDefaultLoggerWithGroupSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		groupBridge(Default("module").WithGroup("group"))
	})
}

func TestLoggerWithMultipleWrappersSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		multiWrapperOne(logger)
	})
}

func multiWrapperOne(logger *Logger) {
	multiWrapperTwo(logger)
}

func multiWrapperTwo(logger *Logger) {
	logger.Info("multi wrapper source check")
}

func TestGlobalMultipleWrappersSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		globalMultiWrapperOne()
	})
}

func globalMultiWrapperOne() {
	globalMultiWrapperTwo()
}

func globalMultiWrapperTwo() {
	Info("global multi wrapper source check")
}

func TestLoggerDerivedGroupAndFieldsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		derivedLoggerSourceBridge(logger.With("scope", "derived").WithGroup("group"))
	})
}

func TestGlobalDefaultModuleDerivedSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		moduleLoggerBridge(Default("payments").With("scope", "settlement"))
	})
}

func TestLoggerInfofWithFieldsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Infof("logger infof field source %s", "check")
	})
}

func TestGlobalInfofWithFieldsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Infof("global infof source %s", "check")
	})
}

func TestLoggerWithNestedWrappersSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		nestedWrapperLevelOne(logger)
	})
}

func nestedWrapperLevelOne(logger *Logger) {
	nestedWrapperLevelTwo(logger)
}

func nestedWrapperLevelTwo(logger *Logger) {
	logger.Info("nested wrapper source check")
}

func TestGlobalNestedWrappersSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		globalNestedWrapperLevelOne()
	})
}

func globalNestedWrapperLevelOne() {
	globalNestedWrapperLevelTwo()
}

func globalNestedWrapperLevelTwo() {
	Info("global nested wrapper source check")
}

func TestLoggerErrorWithAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Error("logger error attrs source check", "key", "value")
	})
}

func TestGlobalErrorWithAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Error("global error attrs source check", "key", "value")
	})
}

func TestLoggerWarnWithAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.Warn("logger warn attrs source check", "key", "value")
	})
}

func TestGlobalWarnWithAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		Warn("global warn attrs source check", "key", "value")
	})
}

func TestLoggerDebugWithAttrsSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelDebug)

	buf.Reset()
	logger.Debug("logger debug attrs source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalDebugWithAttrsSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelDebug()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Debug("global debug attrs source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerTraceWithAttrsSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	logger := NewLogger(&buf, false, true)
	logger.SetLevel(LevelTrace)

	buf.Reset()
	logger.Trace("logger trace attrs source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestGlobalTraceWithAttrsSourcePointsToCaller(t *testing.T) {
	var buf bytes.Buffer
	ResetGlobalLogger(&buf, false, true)
	SetLevelTrace()
	EnableTextLogger()
	DisableJSONLogger()

	buf.Reset()
	Trace("global trace attrs source check", "key", "value")

	assertSlogSourceOutput(t, buf.String())
}

func TestLoggerModuleDefaultSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		moduleLoggerBridge(Default("orders"))
	})
}

func TestLoggerWithGroupAndAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		groupBridge(logger.WithGroup("group").With("key", "value"))
	})
}

func TestGlobalModuleNestedDerivedSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		moduleLoggerBridge(Default("orders", "payment").With("trace_id", "t-1"))
	})
}

func TestLoggerMultipleWithCallsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		derivedLoggerSourceBridge(logger.With("k1", "v1").With("k2", "v2"))
	})
}

func TestGlobalMultipleModuleCallsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		moduleLoggerBridge(Default("a", "b", "c"))
	})
}

func TestLoggerWithContextAndAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		logger.InfoContext(context.Background(), "logger context attrs source check", "key", "value")
	})
}

func TestGlobalWithContextAndAttrsSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		InfoContext(context.Background(), "global context attrs source check", "key", "value")
	})
}

func TestLoggerRepeatedWrapperChainSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLogger(t, func(logger *Logger) {
		repeatedWrapperOne(logger)
	})
}

func repeatedWrapperOne(logger *Logger) {
	repeatedWrapperTwo(logger)
}

func repeatedWrapperTwo(logger *Logger) {
	repeatedWrapperThree(logger)
}

func repeatedWrapperThree(logger *Logger) {
	logger.Info("repeated wrapper source check")
}

func TestGlobalRepeatedWrapperChainSourcePointsToCaller(t *testing.T) {
	testWithSlogSourceLoggerGlobalOnly(t, func() {
		globalRepeatedWrapperOne()
	})
}

func globalRepeatedWrapperOne() {
	globalRepeatedWrapperTwo()
}

func globalRepeatedWrapperTwo() {
	globalRepeatedWrapperThree()
}

func globalRepeatedWrapperThree() {
	Info("global repeated wrapper source check")
}

// TestJSONLogger 测试JSON格式输出
func TestJSONLogger(t *testing.T) {
	// 重置全局配置
	resetForTest()

	var buf bytes.Buffer

	// 创建配置为JSON格式的logger
	config := DefaultConfig()
	config.SetEnableJSON(true)
	config.SetEnableText(false)
	logger := NewLoggerWithConfig(&buf, config)

	// 输出一条日志
	logger.Info("json test", "key", "value")

	jsonOutput := buf.String()
	t.Logf("JSON output: %q", jsonOutput) // 输出实际JSON内容以便调试

	// 检查是否为空
	if len(jsonOutput) == 0 {
		t.Fatal("Expected JSON output but got empty string")
	}

	// 解析JSON输出
	var logEntry map[string]any
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
	DisableJSONLogger()
}

// TestGlobalTextToggleAffectsDefaultLogger 验证全局文本开关对默认配置实例生效
func TestGlobalTextToggleAffectsDefaultLogger(t *testing.T) {
	resetForTest()
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	logger.Info("text enabled")
	if buf.Len() == 0 {
		t.Fatalf("expected text log when开关开启")
	}

	buf.Reset()
	DisableTextLogger()
	logger.Info("text disabled")
	if buf.Len() != 0 {
		t.Fatalf("expected no text log after DisableTextLogger, got %q", buf.String())
	}

	EnableTextLogger()
	logger.Info("text re-enabled")
	if buf.Len() == 0 {
		t.Fatalf("expected text log after EnableTextLogger")
	}
}

// TestGlobalJSONToggleEnablesDefaultLogger 验证全局 JSON 开关可为默认实例启用 JSON 输出
func TestGlobalJSONToggleEnablesDefaultLogger(t *testing.T) {
	resetForTest()
	DisableTextLogger()
	defer EnableTextLogger()

	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)
	logger.Info("no json yet", "key", "value")
	if buf.Len() != 0 {
		t.Fatalf("expected no output before EnableJSONLogger, got %q", buf.String())
	}

	EnableJSONLogger()
	defer DisableJSONLogger()

	logger.Info("json enabled", "key", "value")
	if buf.Len() == 0 {
		t.Fatalf("expected JSON output after EnableJSONLogger")
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output, err=%v, raw=%q", err, buf.String())
	}
	if entry["msg"] != "json enabled" {
		t.Fatalf("unexpected msg field: %v", entry["msg"])
	}
}

// TestInstanceConfigInheritsGlobalOutputs 验证配置未显式设置时继承全局输出开关
func TestInstanceConfigInheritsGlobalOutputs(t *testing.T) {
	resetForTest()
	DisableTextLogger()
	defer EnableTextLogger()

	var buf bytes.Buffer
	config := &Config{}
	logger := NewLoggerWithConfig(&buf, config)

	logger.Info("no outputs yet")
	if buf.Len() != 0 {
		t.Fatalf("expected no output when继承关闭状态, got %q", buf.String())
	}

	buf.Reset()
	EnableJSONLogger()
	defer DisableJSONLogger()

	logger.Info("json after toggle", "key", "value")
	if buf.Len() == 0 {
		t.Fatal("expected JSON output after全局开启 JSON")
	}

	records := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	last := records[len(records)-1]
	var entry map[string]any
	if err := json.Unmarshal(last, &entry); err != nil {
		t.Fatalf("expected valid JSON output, err=%v, raw=%q", err, string(last))
	}
	if entry["msg"] != "json after toggle" {
		t.Fatalf("unexpected msg field: %v", entry["msg"])
	}
}
