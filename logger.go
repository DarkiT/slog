package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	LevelTrace Level = -8 // 跟踪级别，最详细的日志记录
	LevelDebug Level = -4 // 调试级别，用于开发调试
	LevelInfo  Level = 0  // 信息级别，普通日志信息
	LevelWarn  Level = 4  // 警告级别，潜在的问题
	LevelError Level = 8  // 错误级别，需要注意的错误
	LevelFatal Level = 12 // 致命级别，会导致程序退出的错误
)

var (
	logger     *Logger                     // 全局日志记录器实例
	TimeFormat = "2006/01/02 15:04.05.000" // 默认时间格式

	// 日志级别对应的TXT名称映射
	levelTextNames = map[slog.Leveler]string{
		LevelInfo:  "I",
		LevelDebug: "D",
		LevelWarn:  "W",
		LevelError: "E",
		LevelTrace: "T",
		LevelFatal: "F",
	}
	// 日志级别对应的JSON名称映射
	levelJsonNames = map[Level]string{
		LevelInfo:  "Info",
		LevelDebug: "Debug",
		LevelWarn:  "Warn",
		LevelError: "Error",
		LevelTrace: "Trace",
		LevelFatal: "Fatal",
	}
)

// Logger 结构体定义，实现日志记录功能
type Logger struct {
	w       io.Writer
	text    *slog.Logger    // 文本格式日志记录器
	json    *slog.Logger    // JSON格式日志记录器
	ctx     context.Context // 上下文信息
	noColor bool            // 是否禁用颜色输出
	level   Level           // 日志级别
	mu      sync.Mutex      // 添加互斥锁，用于处理并发
}

// GetLevel 获取当前日志级别
// 优先返回原子存储的级别，否则返回有效级别
func (l *Logger) GetLevel() Level {
	return levelVar.Level()
}

// SetLevel 设置日志级别
// 同时更新普通存储和原子存储
func (l *Logger) SetLevel(level any) *Logger {
	if err := SetLevel(level); err != nil {
		l.Error("SetLogLevel", "error", err.Error())
	}
	l.level = l.GetLevel()
	return l
}

// GetSlogLogger 方法
func (l *Logger) GetSlogLogger() *slog.Logger {
	if jsonEnabled && !textEnabled {
		return l.json
	}
	return l.text
}

// logWithLevel 使用指定级别记录日志
// 非格式化日志的内部实现
func (l *Logger) logWithLevel(level Level, msg string, args ...any) {
	l.logRecord(level, msg, false, args...)
}

// logfWithLevel 使用指定级别记录格式化日志
// 格式化日志的内部实现
func (l *Logger) logfWithLevel(level Level, format string, args ...any) {
	l.logRecord(level, fmt.Sprintf(format, args...), true, args...)
}

// logRecord 日志记录的核心实现
// 处理所有类型的日志记录请求
func (l *Logger) logRecord(level Level, msg string, sprintf bool, args ...any) {
	// 使用 logger 的 context 而不是空 context
	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var r slog.Record
	if !sprintf && formatLog(msg, args...) {
		r = newRecord(level, msg, args...)
	} else if !sprintf {
		r = newRecord(level, msg)
		r.Add(args...)
	} else {
		r = newRecord(level, msg)
	}

	// 处理现有的日志输出
	if textEnabled && l.text != nil && l.text.Enabled(ctx, level) {
		_ = l.text.Handler().Handle(ctx, r)
	}
	if jsonEnabled && l.json != nil && l.json.Enabled(ctx, level) {
		_ = l.json.Handler().Handle(ctx, r)
	}

	// 向所有订阅者发送日志记录
	subscribers.Range(func(key, value interface{}) bool {
		sub := value.(*Subscriber)
		select {
		case sub.ch <- r:

		default:
			// channel已满，使用滑动窗口策略
			select {
			case <-sub.ch: // 移除最旧的消息
				sub.ch <- r
			default:
				// 完全阻塞时跳过
			}
		}
		return true
	})
}

// With 创建一个带有额外字段的新日志记录器
func (l *Logger) With(args ...any) *Logger {
	if len(args) == 0 {
		return l
	}

	newLogger := l.clone()

	// 更新text logger
	if l.text != nil {
		newLogger.text = l.text.With(args...)
	}

	// 更新json logger
	if l.json != nil {
		newLogger.json = l.json.With(args...)
	}

	return newLogger
}

// WithGroup 在当前日志记录器基础上创建一个新的日志组
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func (l *Logger) WithGroup(name string) *Logger {
	// 如果组名为空则返回当前logger
	if name == "" {
		return l
	}

	// 创建新的logger
	newLogger := l.clone()

	// 处理text logger
	if l.text != nil {
		newLogger.text = l.text.WithGroup(name)
	}

	// 处理json logger
	if l.json != nil {
		newLogger.json = l.json.WithGroup(name)
	}

	return newLogger
}

// Debug 记录Debug级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.logWithLevel(LevelDebug, msg, args...)
}

// Info 记录信息级别的日志
func (l *Logger) Info(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// Warn 记录警告级别的日志
func (l *Logger) Warn(msg string, args ...any) {
	l.logWithLevel(LevelWarn, msg, args...)
}

// Error 记录错误级别的日志
func (l *Logger) Error(msg string, args ...any) {
	l.logWithLevel(LevelError, msg, args...)
}

// Fatal 记录致命错误并终止程序
func (l *Logger) Fatal(msg string, args ...any) {
	l.logWithLevel(LevelFatal, msg, args...)
	os.Exit(1)
}

// Trace 记录跟踪级别的日志
func (l *Logger) Trace(msg string, args ...any) {
	l.logWithLevel(LevelTrace, msg, args...)
}

// 以下是格式化日志方法的实现

// Debugf 记录格式化的调试级别日志
func (l *Logger) Debugf(format string, args ...any) {
	l.logfWithLevel(LevelDebug, format, args...)
}

// Infof 记录格式化的信息级别日志
func (l *Logger) Infof(format string, args ...any) {
	l.logfWithLevel(LevelInfo, format, args...)
}

// Warnf 记录格式化的警告级别日志
func (l *Logger) Warnf(format string, args ...any) {
	l.logfWithLevel(LevelWarn, format, args...)
}

// Errorf 记录格式化的错误级别日志
func (l *Logger) Errorf(format string, args ...any) {
	l.logfWithLevel(LevelError, format, args...)
}

// Fatalf 记录格式化的致命错误并终止程序
func (l *Logger) Fatalf(format string, args ...any) {
	l.logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Tracef 记录格式化的跟踪级别日志
func (l *Logger) Tracef(format string, args ...any) {
	l.logfWithLevel(LevelTrace, format, args...)
}

// Printf 兼容标准库的格式化日志方法
func (l *Logger) Printf(format string, args ...any) {
	l.logWithLevel(LevelInfo, format, args...)
}

// Println 兼容标准库的普通日志方法
func (l *Logger) Println(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// appendColorLevel 添加带颜色的日志级别
func (l *Logger) appendColorLevel(level Level) string {
	if !textEnabled {
		return fmt.Sprintf("[%s]", levelJsonNames[level])
	}

	// 获取对应的颜色代码
	color, ok := levelColorMap[level]
	if !ok {
		color = ansiBrightRed
	}

	// 如果没有禁用颜色，则添加颜色代码
	if !l.noColor {
		return fmt.Sprintf("%s[%s]%s", color, levelTextNames[level], ansiReset)
	}

	return fmt.Sprintf("[%s]", levelTextNames[level])
}

// logWithDynamic 处理动态输出的日志
//
// msg: 要显示的消息
// render: 动态渲染函数，用于生成每一帧的内容
// interval: 每次更新的时间间隔(毫秒)
func (l *Logger) logWithDynamic(level Level, render func(i int) string, interval int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用 handler 的 writer
	w := l.w

	// 动态输出内容
	for i := 0; ; i++ {
		content := render(i)
		if content == "" {
			break
		}

		if textEnabled {
			// 文本格式输出
			fmt.Fprintf(w, "\r%s %s %s",
				time.Now().Format(TimeFormat),
				l.appendColorLevel(level),
				content)
		} else {
			// JSON格式输出
			fmt.Fprintf(w, "\r{\"time\":\"%s\",\"level\":\"%s\",\"msg\":\"%s\"}",
				time.Now().Format(TimeFormat),
				levelJsonNames[level],
				content)
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	fmt.Fprintln(w)
}

// Dynamic 动态输出带点号动画效果
//
//   - msg: 要显示的消息内容
//   - frames: 动画更新的总帧数
//   - interval: 每次更新的时间间隔(毫秒)
func (l *Logger) Dynamic(msg string, frames int, interval int) {
	l.logWithDynamic(l.level, func(i int) string {
		if i >= frames {
			return ""
		}
		return fmt.Sprintf("%s%s", msg, strings.Repeat(".", i%4))
	}, interval)
}

// Progress 显示进度百分比
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
func (l *Logger) Progress(msg string, durationMs int) {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	l.logWithDynamic(l.level, func(i int) string {
		if i > steps {
			return ""
		}
		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}
		return fmt.Sprintf("%s %.1f%%", msg, progress)
	}, interval)
}

// Countdown 显示倒计时
//
//   - msg: 要显示的消息内容
//   - seconds: 倒计时的秒数
func (l *Logger) Countdown(msg string, seconds int) {
	l.logWithDynamic(l.level, func(i int) string {
		remaining := seconds - i
		if remaining < 0 {
			return ""
		}
		return fmt.Sprintf("%s %ds", msg, remaining)
	}, 1000)
}

// Loading 显示加载动画
//
//   - msg: 要显示的消息内容
//   - seconds: 持续时间(秒)
func (l *Logger) Loading(msg string, seconds int) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	steps := seconds * 10 // 每秒10帧

	l.logWithDynamic(l.level, func(i int) string {
		if i >= steps {
			return ""
		}
		return fmt.Sprintf("%s %s", spinner[i%len(spinner)], msg)
	}, 100) // 固定100ms间隔，即10帧/秒
}

// clone 创建Logger的深度复制
func (l *Logger) clone() *Logger {
	newLogger := &Logger{
		w:       l.w,
		text:    l.text,
		json:    l.json,
		noColor: l.noColor,
		level:   l.level,
		ctx:     l.ctx, // 确保正确复制context
	}
	if newLogger.ctx == nil {
		newLogger.ctx = context.Background()
	}
	return newLogger
}

// newRecord 创建新的日志记录
// 设置时间戳、级别、消息和调用栈信息
func newRecord(level Level, format string, args ...any) slog.Record {
	t := time.Now()
	var pcs [1]uintptr
	// 跳过runtime.Callers和当前函数调用
	runtime.Callers(5, pcs[:])

	if args == nil {
		return slog.NewRecord(t, level, format, pcs[0])
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pcs[0])
}

// formatLog 检查是否需要格式化日志消息
// 返回true表示需要格式化，false表示不需要
func formatLog(msg string, args ...any) bool {
	if len(args) == 0 {
		return false
	}
	return len(args) > 0 && strings.Contains(msg, "%") && !strings.Contains(msg, "%%")
}
