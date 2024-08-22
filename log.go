package slog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darkit/slog/formatter"
)

var (
	levelVar                 slog.LevelVar
	recordChan               chan slog.Record
	textEnabled, jsonEnabled = true, false
	slogPfx                  = &addons{
		PrefixKeys: []string{"module"},
	}
)

func init() {
	NewLogger(os.Stdout, false, false)
}

func New(handler Handler) *slog.Logger {
	return slog.New(handler)
}

// NewLogger 创建一个包含文本和JSON格式的日志记录器
func NewLogger(writer io.Writer, noColor, addSource bool) Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug
	logger = Logger{
		noColor: noColor,
		level:   levelVar.Level(),
		ctx:     context.Background(),
		text:    slog.New(newAddonsHandler(NewConsoleHandler(writer, noColor, options), slogPfx)),
		json:    slog.New(newAddonsHandler(NewJSONHandler(writer, options), slogPfx)),
	}

	return logger
}

/*
// NewLoggerWithText 创建一个文本格式的日志记录器
func NewLoggerWithText(writer io.Writer, noColor, addSource bool) Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug
	logger = Logger{
		noColor: noColor,
		ctx:     context.Background(),
		text:    slog.New(newAddonsHandler(NewConsoleHandler(writer, noColor, options), slogPfx)),
	}

	return logger
}

// NewLoggerWithJson 创建一个JSON格式的日志记录器
func NewLoggerWithJson(writer io.Writer, addSource bool) Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug
	logger = Logger{
		ctx:  context.Background(),
		json: slog.New(newAddonsHandler(slog.NewJSONHandler(writer, options), slogPfx)),
	}
	return logger
}
*/

// NewOptions 创建新的处理程序选项。
func NewOptions(options *slog.HandlerOptions) *slog.HandlerOptions {
	if options == nil {
		options = &slog.HandlerOptions{
			Level: &levelVar,
		}
	}

	if levelVar.Level() < LevelDebug {
		options.AddSource = true
	}

	options.Level = &levelVar
	options.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.SourceKey:
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
		case LevelKey:
			level := a.Value.Any().(Level)
			a.Value = slog.StringValue(levelJsonNames[level])
		case TimeKey:
			if t, ok := a.Value.Any().(time.Time); ok {
				a.Value = slog.StringValue(t.Format(TimeFormat))
			}
		}
		return a
	}

	return options
}

// Default 返回一个新的带前缀的日志记录器
func Default(modules ...string) *Logger {
	if modules == nil {
		return &logger
	}

	// 返回一个新的 Logger，其中包含给定的参数
	module := strings.Join(modules, ".")
	return logger.With("module", module)
}

// GetSlogLogger 返回原始log/slog的日志记录器
func GetSlogLogger() *slog.Logger {
	return logger.GetSlogLogger()
}

// GetLevel 获取全局日志级别。
func GetLevel() Level { return logger.GetLevel() }

// Debug 记录全局Debug级别的日志。
func Debug(msg string, args ...any) { logger.logWithLevel(LevelDebug, msg, args...) }

// Info 记录全局Info级别的日志。
func Info(msg string, args ...any) { logger.logWithLevel(LevelInfo, msg, args...) }

// Warn 记录全局Warn级别的日志。
func Warn(msg string, args ...any) { logger.logWithLevel(LevelWarn, msg, args...) }

// Error 记录全局Error级别的日志。
func Error(msg string, args ...any) { logger.logWithLevel(LevelError, msg, args...) }

// Trace 记录全局Trace级别的日志。
func Trace(msg string, args ...any) { logger.logWithLevel(LevelTrace, msg, args...) }

// Fatal 记录全局Fatal级别的日志，并退出程序。
func Fatal(msg string, args ...any) { logger.logWithLevel(LevelFatal, msg, args...); os.Exit(1) }

// Debugf 记录格式化的全局Debug级别的日志。
func Debugf(format string, args ...any) { logger.logWithLevel(LevelDebug, format, args...) }

// Infof 记录格式化的全局Info级别的日志。
func Infof(format string, args ...any) { logger.logWithLevel(LevelInfo, format, args...) }

// Warnf 记录格式化的全局Warn级别的日志。
func Warnf(format string, args ...any) { logger.logWithLevel(LevelWarn, format, args...) }

// Errorf 记录格式化的全局Error级别的日志。
func Errorf(format string, args ...any) { logger.logWithLevel(LevelError, format, args...) }

// Tracef 记录格式化的全局Trace级别的日志。
func Tracef(format string, args ...any) { logger.logWithLevel(LevelTrace, format, args...) }

// Fatalf 记录格式化的全局Fatal级别的日志，并退出程序。
func Fatalf(format string, args ...any) {
	logger.logWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Println 记录信息级别的日志。
func Println(msg string, args ...any) { logger.logWithLevel(LevelInfo, msg, args...) }

// Printf 记录信息级别的格式化日志。
func Printf(format string, args ...any) { logger.logWithLevel(LevelInfo, format, args...) }

// With 创建一个新的日志记录器，带有指定的属性。
func With(args ...any) *Logger { return logger.With(args...) }

// With 创建一个新的日志记录器，带有指定的属性。
func WithGroup(name string) *Logger { return logger.WithGroup(name) }

// SetLevelTrace 设置全局日志级别为Trace。
func SetLevelTrace() { levelVar.Set(LevelTrace) }

// SetLevelDebug 设置全局日志级别为Debug。
func SetLevelDebug() { levelVar.Set(LevelDebug) }

// SetLevelInfo 设置全局日志级别为Info。
func SetLevelInfo() { levelVar.Set(LevelInfo) }

// SetLevelWarn 设置全局日志级别为Warn。
func SetLevelWarn() { levelVar.Set(LevelWarn) }

// SetLevelError 设置全局日志级别为Error。
func SetLevelError() { levelVar.Set(LevelError) }

// SetLevelFatal 设置全局日志级别为Fatal。
func SetLevelFatal() { levelVar.Set(LevelFatal) }

// GetChanRecord 使用通道获取日志信息
func GetChanRecord(num ...uint16) chan slog.Record {
	var n uint16 = 500
	if recordChan == nil {
		if num != nil {
			n = num[0]
		}
		recordChan = make(chan slog.Record, n)
	}
	return recordChan
}

// EnableFormatters 启用日志脱敏处理器。
func EnableFormatters(formatters ...formatter.Formatter) {
	slogPfx.formatters = formatters
}

// EnableTextLogger 启用文本日志记录器。
func EnableTextLogger() {
	textEnabled = true
}

// EnableJsonLogger 启用 JSON 日志记录器。
func EnableJsonLogger() {
	jsonEnabled = true
}

// DisableTextLogger 禁用文本日志记录器。
func DisableTextLogger() {
	textEnabled = false
}

// DisableJsonLogger 禁用 JSON 日志记录器。
func DisableJsonLogger() {
	jsonEnabled = false
}
