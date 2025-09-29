package slog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ProgressBuilder 进度条建造者，使用链式调用构建进度条配置
// 这个设计比原来的8个不同方法更加优雅和易用
type ProgressBuilder struct {
	logger      *Logger
	message     string
	progress    *float64      // nil表示自动进度，非nil表示手动进度
	duration    time.Duration // 仅在自动进度时使用
	width       int
	writer      io.Writer // nil使用默认writer
	level       Level
	style       ProgressStyle
	showPercent bool
	brackets    [2]string // [左括号, 右括号]
	fillChars   [3]string // [已完成, 进行中, 未完成]
	timeFormat  string
}

// ProgressStyle 进度条样式枚举
type ProgressStyle int

const (
	StyleDefault ProgressStyle = iota
	StyleArrows
	StyleDots
	StyleBlocks
	StyleCustom
)

// String 返回样式的字符串表示
func (ps ProgressStyle) String() string {
	switch ps {
	case StyleDefault:
		return "default"
	case StyleArrows:
		return "arrows"
	case StyleDots:
		return "dots"
	case StyleBlocks:
		return "blocks"
	case StyleCustom:
		return "custom"
	default:
		return "default"
	}
}

// getStyleChars 根据样式获取填充字符
func (ps ProgressStyle) getStyleChars() [3]string {
	switch ps {
	case StyleDefault:
		return [3]string{"=", ">", " "}
	case StyleArrows:
		return [3]string{"->", ">>", "  "}
	case StyleDots:
		return [3]string{"●", "◐", "○"}
	case StyleBlocks:
		return [3]string{"█", "▌", " "}
	default:
		return [3]string{"=", ">", " "}
	}
}

// NewProgressBar 创建进度条建造者
// 这是新API的入口点，替换了原来的8个方法
func (l *Logger) NewProgressBar(message string) *ProgressBuilder {
	return &ProgressBuilder{
		logger:      l,
		message:     message,
		width:       30,
		level:       l.level,
		style:       StyleDefault,
		showPercent: true,
		brackets:    [2]string{"[", "]"},
		fillChars:   StyleDefault.getStyleChars(),
		timeFormat:  TimeFormat,
	}
}

// Value 设置手动进度值 (0.0 - 1.0)
// 设置后将使用手动进度模式，不会自动递增
func (pb *ProgressBuilder) Value(progress float64) *ProgressBuilder {
	// 确保进度值在有效范围内
	if progress < 0.0 {
		progress = 0.0
	} else if progress > 1.0 {
		progress = 1.0
	}
	pb.progress = &progress
	return pb
}

// Duration 设置自动进度持续时间
// 仅在未设置手动进度值时有效
func (pb *ProgressBuilder) Duration(d time.Duration) *ProgressBuilder {
	pb.duration = d
	return pb
}

// Milliseconds 设置自动进度持续时间（毫秒）
// 向后兼容方法
func (pb *ProgressBuilder) Milliseconds(ms int) *ProgressBuilder {
	pb.duration = time.Duration(ms) * time.Millisecond
	return pb
}

// Width 设置进度条宽度
func (pb *ProgressBuilder) Width(w int) *ProgressBuilder {
	if w < 5 {
		w = 5 // 最小宽度
	} else if w > 100 {
		w = 100 // 最大宽度
	}
	pb.width = w
	return pb
}

// To 设置输出目标
func (pb *ProgressBuilder) To(writer io.Writer) *ProgressBuilder {
	pb.writer = writer
	return pb
}

// Level 设置日志级别
func (pb *ProgressBuilder) Level(level Level) *ProgressBuilder {
	pb.level = level
	return pb
}

// Style 设置进度条样式
func (pb *ProgressBuilder) Style(style ProgressStyle) *ProgressBuilder {
	pb.style = style
	if style != StyleCustom {
		pb.fillChars = style.getStyleChars()
	}
	return pb
}

// CustomChars 设置自定义填充字符
// 自动将样式设置为StyleCustom
func (pb *ProgressBuilder) CustomChars(filled, head, empty string) *ProgressBuilder {
	pb.style = StyleCustom
	pb.fillChars = [3]string{filled, head, empty}
	return pb
}

// Brackets 设置左右括号
func (pb *ProgressBuilder) Brackets(left, right string) *ProgressBuilder {
	pb.brackets = [2]string{left, right}
	return pb
}

// ShowPercent 设置是否显示百分比
func (pb *ProgressBuilder) ShowPercent(show bool) *ProgressBuilder {
	pb.showPercent = show
	return pb
}

// TimeFormat 设置时间格式
func (pb *ProgressBuilder) TimeFormat(format string) *ProgressBuilder {
	pb.timeFormat = format
	return pb
}

// Start 开始显示进度条
// 这是终端方法，执行实际的进度条显示
func (pb *ProgressBuilder) Start() {
	// 确定输出目标
	writer := pb.writer
	if writer == nil {
		writer = pb.logger.w
		if writer == nil {
			writer = os.Stdout
		}
	}

	// 检查日志级别
	if !pb.logger.isLevelEnabled(pb.level) {
		return
	}

	if pb.progress != nil {
		// 手动进度模式
		pb.renderStaticProgress(writer)
	} else {
		// 自动进度模式
		pb.renderDynamicProgress(writer)
	}
}

// renderStaticProgress 渲染静态进度条（手动进度）
func (pb *ProgressBuilder) renderStaticProgress(writer io.Writer) {
	progressText := pb.formatProgress(*pb.progress)

	// 直接输出一次
	if _, err := writer.Write([]byte(progressText + "\n")); err != nil {
		// 记录内部错误但不阻塞
		if pb.logger.config == nil || pb.logger.config.LogInternalErrors {
			pb.logger.Error("进度条输出失败", "error", err.Error())
		}
	}
}

// renderDynamicProgress 渲染动态进度条（自动进度）
func (pb *ProgressBuilder) renderDynamicProgress(writer io.Writer) {
	if pb.duration <= 0 {
		pb.duration = 3 * time.Second // 默认持续时间
	}

	steps := 100
	interval := pb.duration / time.Duration(steps)
	startTime := time.Now()

	// 使用互斥锁避免并发问题
	pb.logger.mu.Lock()
	defer pb.logger.mu.Unlock()

	for i := 0; i <= steps; i++ {
		// 根据时间计算实际进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(pb.duration)

		if progress > 1.0 {
			progress = 1.0
		}

		progressText := pb.formatProgress(progress)

		// 输出进度条（使用\r回到行首）
		if i == steps {
			// 最后一次输出，添加换行
			progressText += "\n"
		} else {
			progressText = "\r" + progressText
		}

		if _, err := writer.Write([]byte(progressText)); err != nil {
			if pb.logger.config == nil || pb.logger.config.LogInternalErrors {
				pb.logger.Error("进度条输出失败", "error", err.Error())
			}
			break
		}

		if progress >= 1.0 {
			break
		}

		time.Sleep(interval)
	}
}

// formatProgress 格式化进度条字符串
func (pb *ProgressBuilder) formatProgress(progress float64) string {
	// 构建进度条内容
	filledWidth := int(float64(pb.width) * progress)
	emptyWidth := pb.width - filledWidth

	var bar strings.Builder
	bar.Grow(len(pb.message) + pb.width + 20) // 预分配容量

	// 添加消息
	bar.WriteString(pb.message)
	bar.WriteString(" ")

	// 添加左括号
	bar.WriteString(pb.brackets[0])

	// 添加已完成部分
	for i := 0; i < filledWidth-1; i++ {
		bar.WriteString(pb.fillChars[0])
	}

	// 添加进度头部（如果有进度的话）
	if filledWidth > 0 {
		bar.WriteString(pb.fillChars[1])
	}

	// 添加空白部分
	for i := 0; i < emptyWidth; i++ {
		bar.WriteString(pb.fillChars[2])
	}

	// 添加右括号
	bar.WriteString(pb.brackets[1])

	// 添加百分比（如果启用）
	if pb.showPercent {
		bar.WriteString(" ")
		bar.WriteString(fmt.Sprintf("%.1f%%", progress*100))
	}

	return bar.String()
}

// isLevelEnabled 检查日志级别是否启用
// 这是一个辅助方法
func (l *Logger) isLevelEnabled(level Level) bool {
	return level >= l.GetLevel()
}

// 向后兼容方法 - 逐步废弃旧API

// ProgressBarCompat 向后兼容的进度条方法
// @Deprecated: 使用 NewProgressBar().Duration().Width().Start() 替代
func (l *Logger) ProgressBarCompat(msg string, durationMs int, barWidth int, level ...Level) *Logger {
	builder := l.NewProgressBar(msg).Milliseconds(durationMs).Width(barWidth)
	if len(level) > 0 {
		builder = builder.Level(level[0])
	}
	builder.Start()
	return l
}

// ProgressBarWithValueCompat 向后兼容的带值进度条方法
// @Deprecated: 使用 NewProgressBar().Value().Width().Start() 替代
func (l *Logger) ProgressBarWithValueCompat(msg string, progress float64, barWidth int, level ...Level) {
	builder := l.NewProgressBar(msg).Value(progress).Width(barWidth)
	if len(level) > 0 {
		builder = builder.Level(level[0])
	}
	builder.Start()
}

// ProgressBarToCompat 向后兼容的带输出目标进度条方法
// @Deprecated: 使用 NewProgressBar().Duration().Width().To().Start() 替代
func (l *Logger) ProgressBarToCompat(msg string, durationMs int, barWidth int, writer io.Writer, level ...Level) *Logger {
	builder := l.NewProgressBar(msg).Milliseconds(durationMs).Width(barWidth).To(writer)
	if len(level) > 0 {
		builder = builder.Level(level[0])
	}
	builder.Start()
	return l
}
