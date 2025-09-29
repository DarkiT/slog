package slog

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgressBuilder_Basic(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false) // noColor=true for testing

	// 测试基本的手动进度条
	logger.NewProgressBar("测试进度").Value(0.5).Width(20).Start()

	output := buf.String()
	if !strings.Contains(output, "测试进度") {
		t.Error("输出应该包含进度消息")
	}
	if !strings.Contains(output, "50.0%") {
		t.Error("输出应该包含50%进度")
	}
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("输出应该包含进度条括号")
	}
}

func TestProgressBuilder_ChainedCalls(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试链式调用
	logger.NewProgressBar("链式调用测试").
		Value(0.75).
		Width(30).
		Brackets("<", ">").
		CustomChars("█", "▌", " ").
		ShowPercent(true).
		Level(LevelInfo).
		Start()

	output := buf.String()
	if !strings.Contains(output, "链式调用测试") {
		t.Error("输出应该包含消息")
	}
	if !strings.Contains(output, "75.0%") {
		t.Error("输出应该包含75%进度")
	}
	if !strings.Contains(output, "<") || !strings.Contains(output, ">") {
		t.Error("输出应该包含自定义括号")
	}
}

func TestProgressBuilder_Styles(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试不同样式
	styles := []ProgressStyle{StyleDefault, StyleArrows, StyleDots, StyleBlocks}

	for _, style := range styles {
		buf.Reset()
		logger.NewProgressBar("样式测试").Value(0.5).Width(10).Style(style).Start()

		output := buf.String()
		if !strings.Contains(output, "样式测试") {
			t.Errorf("样式 %v 应该包含消息", style)
		}
	}
}

func TestProgressBuilder_CustomChars(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试自定义字符
	logger.NewProgressBar("自定义字符").
		Value(0.6).
		Width(15).
		CustomChars("*", "+", "-").
		Start()

	output := buf.String()
	// 输出应该包含自定义字符
	if !strings.Contains(output, "*") {
		t.Error("输出应该包含自定义填充字符")
	}
}

func TestProgressBuilder_NoPercent(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试不显示百分比
	logger.NewProgressBar("无百分比").
		Value(0.8).
		Width(20).
		ShowPercent(false).
		Start()

	output := buf.String()
	if strings.Contains(output, "%") {
		t.Error("设置ShowPercent(false)时不应该显示百分比")
	}
}

func TestProgressBuilder_DynamicProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试动态进度条（时间较短以便测试）
	start := time.Now()
	logger.NewProgressBar("动态进度").
		Duration(100 * time.Millisecond).
		Width(10).
		Start()
	elapsed := time.Since(start)

	// 应该大约用时100ms
	if elapsed < 80*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("动态进度条用时不符合预期，实际: %v", elapsed)
	}

	output := buf.String()
	if !strings.Contains(output, "动态进度") {
		t.Error("输出应该包含动态进度消息")
	}
}

func TestProgressBuilder_WidthLimits(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试宽度限制
	logger.NewProgressBar("最小宽度").Value(0.5).Width(1).Start()   // 应该被调整为5
	logger.NewProgressBar("最大宽度").Value(0.5).Width(200).Start() // 应该被调整为100

	// 这里主要测试不会崩溃，具体的宽度限制在内部处理
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("应该有2行输出，实际: %d", len(lines))
	}
}

func TestProgressBuilder_ProgressLimits(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试进度值限制
	logger.NewProgressBar("负值进度").Value(-0.5).Start() // 应该被调整为0.0
	buf.Reset()
	logger.NewProgressBar("超值进度").Value(1.5).Start() // 应该被调整为1.0

	output := buf.String()
	if !strings.Contains(output, "100.0%") {
		t.Error("超过1.0的进度值应该被调整为100%")
	}
}

func TestProgressBuilder_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)
	logger.SetLevel(LevelWarn) // 设置为Warn级别

	// Debug级别的进度条应该被过滤
	logger.NewProgressBar("被过滤的进度").Value(0.5).Level(LevelDebug).Start()
	output := buf.String()
	if strings.Contains(output, "被过滤的进度") {
		t.Error("低于当前级别的进度条应该被过滤")
	}

	buf.Reset()
	// Error级别的进度条应该显示
	logger.NewProgressBar("显示的进度").Value(0.5).Level(LevelError).Start()
	output = buf.String()
	if !strings.Contains(output, "显示的进度") {
		t.Error("高于当前级别的进度条应该显示")
	}
}

func TestProgressBuilder_ToWriter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger(&buf1, true, false)

	// 测试输出到指定Writer
	logger.NewProgressBar("自定义输出").Value(0.5).To(&buf2).Start()

	// buf1应该为空，buf2应该有内容
	if buf1.String() != "" {
		t.Error("原writer不应该有输出")
	}
	if !strings.Contains(buf2.String(), "自定义输出") {
		t.Error("指定的writer应该有输出")
	}
}

func TestProgressBuilder_MillisecondsMethod(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 测试毫秒方法（向后兼容）
	start := time.Now()
	logger.NewProgressBar("毫秒测试").Milliseconds(50).Width(5).Start()
	elapsed := time.Since(start)

	// 应该大约用时50ms
	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("毫秒方法用时不符合预期，实际: %v", elapsed)
	}
}

func TestProgressBuilder_ErrorHandling(t *testing.T) {
	// 测试错误情况不会导致崩溃
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	// 这些调用应该不会崩溃
	logger.NewProgressBar("").Value(0.5).Start()
	logger.NewProgressBar("空消息测试").Value(0.5).Width(0).Start()
	logger.NewProgressBar("无时间").Duration(0).Start()

	// 主要测试不会panic
	output := buf.String()
	if output == "" {
		t.Error("至少应该有一些输出")
	}
}

// 基准测试

func BenchmarkProgressBuilder_Static(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		logger.NewProgressBar("基准测试").Value(0.5).Width(30).Start()
	}
}

func BenchmarkProgressBuilder_Dynamic(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		logger.NewProgressBar("动态基准").Duration(10 * time.Millisecond).Width(10).Start()
	}
}

func BenchmarkProgressBuilder_ChainedCalls(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		logger.NewProgressBar("链式基准").
			Value(0.75).
			Width(25).
			Style(StyleBlocks).
			ShowPercent(true).
			Brackets("{", "}").
			Start()
	}
}
