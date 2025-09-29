package slog

import (
	"bytes"
	"strings"
	"testing"
)

// TestGlobalLevelControl 测试全局日志级别控制
func TestGlobalLevelControl(t *testing.T) {
	// 重置环境
	var buf bytes.Buffer

	// 创建新的全局logger，确保使用buffer输出
	ResetGlobalLogger(&buf, false, false)
	EnableTextLogger()
	DisableJSONLogger()

	// 1. 设置为Info级别，Debug不应该输出
	SetLevelInfo()

	buf.Reset()
	Debug("这是Debug日志，不应该显示")
	debugOutput := buf.String()

	buf.Reset()
	Info("这是Info日志，应该显示")
	infoOutput := buf.String()

	t.Logf("设置Info级别后:")
	t.Logf("Debug输出: %q", debugOutput)
	t.Logf("Info输出: %q", infoOutput)

	// 验证Debug日志不应该输出
	if strings.Contains(debugOutput, "这是Debug日志") {
		t.Errorf("设置Info级别后，Debug日志不应该输出，但实际输出了: %s", debugOutput)
	}

	// 验证Info日志应该输出
	if !strings.Contains(infoOutput, "这是Info日志") {
		t.Errorf("设置Info级别后，Info日志应该输出，但没有输出: %s", infoOutput)
	}

	// 2. 设置为Debug级别，Debug应该输出
	SetLevelDebug()

	buf.Reset()
	Debug("这是Debug日志，现在应该显示")
	debugOutput2 := buf.String()

	t.Logf("设置Debug级别后:")
	t.Logf("Debug输出: %q", debugOutput2)

	// 验证Debug日志现在应该输出
	if !strings.Contains(debugOutput2, "这是Debug日志") {
		t.Errorf("设置Debug级别后，Debug日志应该输出，但没有输出: %s", debugOutput2)
	}

	// 3. 测试不同级别的过滤
	testCases := []struct {
		setLevel   func()
		testLevel  func(string)
		levelName  string
		shouldShow bool
	}{
		{SetLevelError, func(msg string) { Debug(msg) }, "Debug", false},
		{SetLevelError, func(msg string) { Info(msg) }, "Info", false},
		{SetLevelError, func(msg string) { Warn(msg) }, "Warn", false},
		{SetLevelError, func(msg string) { Error(msg) }, "Error", true},

		{SetLevelWarn, func(msg string) { Debug(msg) }, "Debug", false},
		{SetLevelWarn, func(msg string) { Info(msg) }, "Info", false},
		{SetLevelWarn, func(msg string) { Warn(msg) }, "Warn", true},
		{SetLevelWarn, func(msg string) { Error(msg) }, "Error", true},
	}

	for _, tc := range testCases {
		tc.setLevel()
		buf.Reset()

		msg := "测试消息 " + tc.levelName
		tc.testLevel(msg)

		output := buf.String()
		hasOutput := strings.Contains(output, msg)

		if tc.shouldShow && !hasOutput {
			t.Errorf("级别控制测试失败: %s 级别应该显示但没有显示，输出: %q", tc.levelName, output)
		} else if !tc.shouldShow && hasOutput {
			t.Errorf("级别控制测试失败: %s 级别不应该显示但显示了，输出: %q", tc.levelName, output)
		}
	}
}

// TestGlobalVsInstanceLevel 测试全局级别vs实例级别
func TestGlobalVsInstanceLevel(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	// 创建两个不同的logger实例
	logger1 := NewLogger(&buf1, false, false)
	logger2 := NewLogger(&buf2, false, false)

	// 设置全局级别为Error
	SetLevelError()

	// 设置实例级别为Debug
	logger1.SetLevel(LevelDebug)

	// 测试全局方法是否受全局级别影响
	buf1.Reset()
	buf2.Reset()

	Debug("全局Debug消息")

	// 测试实例方法是否受实例级别影响
	logger1.Debug("实例1 Debug消息")
	logger2.Debug("实例2 Debug消息")

	globalOutput := buf1.String() // 全局logger输出到buf1
	instance1Output := buf1.String()
	instance2Output := buf2.String()

	t.Logf("全局Debug输出: %q", globalOutput)
	t.Logf("实例1 Debug输出: %q", instance1Output)
	t.Logf("实例2 Debug输出: %q", instance2Output)

	// 分析结果
	if strings.Contains(globalOutput, "全局Debug消息") {
		t.Logf("全局Debug消息被输出了（可能是问题）")
	} else {
		t.Logf("全局Debug消息被正确过滤")
	}
}
