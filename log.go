package slog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/darkit/slog/formatter"
)

var (
	// 创建扩展配置
	ext = &extends{
		PrefixKeys: []string{"$module"},
	}
	dlpEnabled               atomic.Bool
	subscribers              sync.Map // 存储所有订阅者
	textEnabled, jsonEnabled = true, false
	levelVar                 = slog.LevelVar{}
)

// Subscriber 订阅者结构
type Subscriber struct {
	ch     chan slog.Record
	cancel context.CancelFunc
}

func init() {
	levelVar.Set(slog.LevelInfo)
	NewLogger(os.Stdout, false, false)
}

func New(handler Handler) *slog.Logger {
	return slog.New(handler)
}

// NewLogger 创建一个包含文本和JSON格式的日志记录器
func NewLogger(w io.Writer, noColor, addSource bool) *Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug

	// 如果需要DLP,则初始化
	if dlpEnabled.Load() {
		ext.EnableDLP()
	}

	if w == nil || (w != os.Stdout && !isWriter(w)) {
		w = NewWriter()
	}

	logger = &Logger{
		w:       w,
		noColor: noColor,
		level:   levelVar.Level(),
		ctx:     context.Background(),
		text:    slog.New(newAddonsHandler(NewConsoleHandler(w, noColor, options), ext)),
		json:    slog.New(newAddonsHandler(NewJSONHandler(w, options), ext)),
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
	if len(modules) == 0 {
		return logger
	}

	// 构建模块标识符
	module := strings.Join(modules, ".")

	// 创建新的带模块前缀的logger
	newLogger := logger.clone()

	// 创建新的上下文
	newLogger.ctx = context.Background() // 确保每个模块有独立的上下文

	// 设置模块前缀
	newHandler := newAddonsHandler(newLogger.text.Handler(), ext)
	newHandler.prefixes[0] = slog.StringValue(module)

	newLogger.text = slog.New(newHandler)
	if newLogger.json != nil {
		jsonHandler := newAddonsHandler(newLogger.json.Handler(), ext)
		jsonHandler.prefixes[0] = slog.StringValue(module)
		newLogger.json = slog.New(jsonHandler)
	}

	return newLogger
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

// 辅助便捷方法

// Dynamic 动态输出带点号动画效果
//
//   - msg: 要显示的消息内容
//   - frames: 动画更新的总帧数
//   - interval: 每次更新的时间间隔(毫秒)
func Dynamic(msg string, frames int, interval int) {
	logger.Dynamic(msg, frames, interval)
}

// Progress 全局进度显示
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
func Progress(msg string, durationMs int) {
	logger.Progress(msg, durationMs)
}

// Countdown 全局倒计时显示
//
//   - msg: 要显示的消息内容
//   - seconds: 倒计时的秒数
func Countdown(msg string, seconds int) {
	logger.Countdown(msg, seconds)
}

// Loading 全局加载动画
//
//   - msg: 要显示的消息内容
//   - seconds: 动画持续的秒数
func Loading(msg string, seconds int) {
	logger.Loading(msg, seconds)
}

// ProgressBar 全局方法：显示带有可视化进度条的日志
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBar(msg string, durationMs int, barWidth int, level ...Level) *Logger {
	return Default().ProgressBar(msg, durationMs, barWidth, level...)
}

// ProgressBarWithValue 全局方法：显示指定进度值的进度条
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithValue(msg string, progress float64, barWidth int, level ...Level) {
	Default().ProgressBarWithValue(msg, progress, barWidth, level...)
}

// ProgressBarWithValueTo 全局方法：显示指定进度值的进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithValueTo(msg string, progress float64, barWidth int, writer io.Writer, level ...Level) {
	Default().ProgressBarWithValueTo(msg, progress, barWidth, writer, level...)
}

// ProgressBarWithOptions 全局方法：显示可高度定制的进度条
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithOptions(msg string, durationMs int, barWidth int, opts progressBarOptions, level ...Level) *Logger {
	return Default().ProgressBarWithOptions(msg, durationMs, barWidth, opts, level...)
}

// ProgressBarWithOptionsTo 全局方法：显示可高度定制的进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithOptionsTo(msg string, durationMs int, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) *Logger {
	return Default().ProgressBarWithOptionsTo(msg, durationMs, barWidth, opts, writer, level...)
}

// ProgressBarWithValueAndOptions 全局方法：显示指定进度值的定制进度条
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithValueAndOptions(msg string, progress float64, barWidth int, opts progressBarOptions, level ...Level) {
	Default().ProgressBarWithValueAndOptions(msg, progress, barWidth, opts, level...)
}

// ProgressBarWithValueAndOptionsTo 全局方法：显示指定进度值的定制进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarWithValueAndOptionsTo(msg string, progress float64, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) {
	Default().ProgressBarWithValueAndOptionsTo(msg, progress, barWidth, opts, writer, level...)
}

// ProgressBarTo 全局方法：显示带有可视化进度条的日志，并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用全局默认级别
func ProgressBarTo(msg string, durationMs int, barWidth int, writer io.Writer, level ...Level) *Logger {
	return Default().ProgressBarTo(msg, durationMs, barWidth, writer, level...)
}

// With 创建一个新的日志记录器，带有指定的属性。
func With(args ...any) *Logger {
	return logger.With(args...)
}

// WithGroup 创建一个带有指定组名的全局日志记录器
// 这是一个包级别的便捷方法
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func WithGroup(name string) *Logger { return logger.WithGroup(name) }

// WithValue 在全局上下文中添加键值对并返回新的 Logger
func WithValue(key string, val any) *Logger {
	// 获取现有的全局Logger并调用其WithValue方法
	return logger.WithValue(key, val)
}

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

// SetLevel 动态更新日志级别
// level 可以是数字(-8, -4, 0, 4, 8, 12)或字符串(trace, debug, info, warn, error, fatal)
func SetLevel(level any) error {
	var newLevel Level

	switch v := level.(type) {
	case Level:
		newLevel = v
	case int:
		newLevel = Level(v)
	case string:
		// 将字符串转换为Level
		switch strings.ToLower(v) {
		case "trace":
			newLevel = LevelTrace
		case "debug":
			newLevel = LevelDebug
		case "info":
			newLevel = LevelInfo
		case "warn":
			newLevel = LevelWarn
		case "error":
			newLevel = LevelError
		case "fatal":
			newLevel = LevelFatal
		default:
			return errors.New("invalid log level string")
		}
	default:
		return errors.New("unsupported level type")
	}

	// 验证级别是否有效
	if !isValidLevel(newLevel) {
		return errors.New("invalid log level value")
	}

	levelVar.Set(newLevel)

	return nil
}

// SetTimeFormat 全局方法：设置日志时间格式
//
//   - format: 时间格式字符串，例如 "2006-01-02 15:04:05.000"
func SetTimeFormat(format string) {
	if format != "" {
		TimeFormat = format
	}
}

// Subscribe 订阅日志记录
// size: channel缓冲区大小
// 返回值: 只读channel和取消订阅函数
func Subscribe(size uint16) (<-chan slog.Record, context.CancelFunc) {
	ch := make(chan slog.Record, size)
	ctx, cancel := context.WithCancel(context.Background())

	sub := &Subscriber{
		ch:     ch,
		cancel: cancel,
	}

	// 生成唯一的订阅者ID
	subID := time.Now().UnixNano()
	subscribers.Store(subID, sub)

	// 监听context取消
	go func() {
		<-ctx.Done()
		subscribers.Delete(subID)
		close(ch)
	}()

	return ch, cancel
}

// EnableFormatters 启用日志格式化器。
func EnableFormatters(formatters ...formatter.Formatter) {
	ext.formatters = formatters
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

// EnableDLPLogger 启用日志脱敏功能
func EnableDLPLogger() {
	dlpEnabled.Store(true)
	if ext != nil {
		ext.EnableDLP()
	}
}

// DisableDLPLogger 禁用日志脱敏功能
func DisableDLPLogger() {
	dlpEnabled.Store(false)
	if ext != nil {
		ext.DisableDLP()
	}
}

// IsDLPEnabled 检查DLP是否启用
func IsDLPEnabled() bool {
	return dlpEnabled.Load()
}

// isWriter 辅助函数检查是否为自定义 writer
func isWriter(w interface{}) bool {
	_, ok := w.(*writer)
	return ok
}

// isValidLevel 检查日志级别是否有效
func isValidLevel(level Level) bool {
	validLevels := []Level{
		LevelTrace, // -8
		LevelDebug, // -4
		LevelInfo,  // 0
		LevelWarn,  // 4
		LevelError, // 8
		LevelFatal, // 12
	}

	for _, l := range validLevels {
		if level == l {
			return true
		}
	}
	return false
}
