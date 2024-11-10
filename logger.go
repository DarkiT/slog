package slog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
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
	logger     Logger                      // 全局日志记录器实例
	TimeFormat = "2006/01/02 15:04.05.000" // 默认时间格式

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
	text    *slog.Logger    // 文本格式日志记录器
	json    *slog.Logger    // JSON格式日志记录器
	ctx     context.Context // 上下文信息
	noColor bool            // 是否禁用颜色输出
	level   Level           // 日志级别

	// 内部优化字段
	levelVar atomic.Value // 用于原子操作的level存储，确保并发安全
}

// getEffectiveLevel 获取有效的日志级别
// 首先尝试从原子存储中获取，如果不存在则遍历所有logger获取最适合的级别
func (l *Logger) getEffectiveLevel() Level {
	// 优先从原子存储中获取
	if val := l.levelVar.Load(); val != nil {
		return val.(Level)
	}

	// 确定需要检查的logger列表
	loggers := []*slog.Logger{l.text, l.json}
	if textEnabled && !jsonEnabled {
		loggers = []*slog.Logger{l.text}
	} else if jsonEnabled && !textEnabled {
		loggers = []*slog.Logger{l.json}
	}

	// 按优先级检查每个级别
	for _, lg := range loggers {
		if lg != nil {
			for _, level := range []Level{
				LevelDebug, LevelInfo, LevelWarn,
				LevelError, LevelFatal, LevelTrace,
			} {
				if lg.Enabled(l.ctx, level) {
					return level
				}
			}
		}
	}

	// 返回全局默认级别
	return levelVar.Level()
}

// GetLevel 获取当前日志级别
// 优先返回原子存储的级别，否则返回有效级别
func (l *Logger) GetLevel() Level {
	if val := l.levelVar.Load(); val != nil {
		return val.(Level)
	}
	return l.getEffectiveLevel()
}

// SetLevel 设置日志级别
// 同时更新普通存储和原子存储
func (l *Logger) SetLevel(level Level) *Logger {
	l.level = level
	l.levelVar.Store(level)
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

	// 修改处理方法，传入正确的 context
	if textEnabled && l.text != nil && l.text.Enabled(ctx, level) {
		_ = l.text.Handler().Handle(ctx, r)
	}
	if jsonEnabled && l.json != nil && l.json.Enabled(ctx, level) {
		_ = l.json.Handler().Handle(ctx, r)
	}
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

// clone 创建Logger的深度复制
func (l *Logger) clone() *Logger {
	newLogger := &Logger{
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
