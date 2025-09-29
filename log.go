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
)

var (
	// 创建扩展配置
	ext = &extensions{
		prefixKeys: []string{"$module"},
	}
	dlpEnabled               atomic.Bool
	subscribers              sync.Map // 存储所有订阅者 - 实际类型为 map[int64]*subscriber
	textEnabled, jsonEnabled = true, false
	levelVar                 = slog.LevelVar{}
)

// subscriberState 订阅者状态
type subscriberState int32

const (
	stateActive subscriberState = iota
	stateClosing
	stateClosed
)

// subscriber 订阅者结构（升级为原子状态管理）
type subscriber struct {
	ch     chan slog.Record
	cancel context.CancelFunc
	state  atomic.Int32 // 原子状态管理
	once   sync.Once    // 确保只关闭一次
}

// isActive 检查订阅者是否活跃
func (s *subscriber) isActive() bool {
	return subscriberState(s.state.Load()) == stateActive
}

// close 安全地关闭订阅者
func (s *subscriber) close() {
	// 原子地设置状态为正在关闭
	if s.state.CompareAndSwap(int32(stateActive), int32(stateClosing)) {
		s.once.Do(func() {
			s.cancel()
			close(s.ch)
			s.state.Store(int32(stateClosed))
		})
	}
}

// trySend 尝试发送日志记录，如果失败则返回false
func (s *subscriber) trySend(record slog.Record) bool {
	// 双重检查：先检查状态，再尝试发送
	if !s.isActive() {
		return false
	}

	select {
	case s.ch <- record:
		return true
	default:
		// Channel满了，使用滑动窗口策略
		select {
		case <-s.ch: // 移除最旧的消息
			select {
			case s.ch <- record:
				return true
			default:
				return false
			}
		default:
			return false
		}
	}
}

func init() {
	levelVar.Set(slog.LevelInfo)
	// 使用LoggerManager而不是直接创建全局logger
	// 这确保了更好的状态管理和实例隔离
	config := &GlobalConfig{
		DefaultWriter:  os.Stdout,
		DefaultLevel:   LevelInfo,
		DefaultNoColor: false,
		DefaultSource:  false,
		EnableText:     true,
		EnableJSON:     false,
	}
	globalManager.Configure(config)
}

func New(handler Handler) *slog.Logger {
	return slog.New(handler)
}

// NewLogger 创建一个包含文本和JSON格式的日志记录器
// 现在使用LoggerManager来管理实例，确保更好的状态隔离
func NewLogger(w io.Writer, noColor, addSource bool) *Logger {
	// 如果使用默认参数，直接返回管理器的默认实例
	if w == os.Stdout && !noColor && !addSource {
		return globalManager.GetDefault()
	}

	// 为自定义参数创建新实例
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug

	// 如果需要DLP,则初始化
	if dlpEnabled.Load() {
		ext.enableDLP()
	}

	if w == nil {
		w = NewWriter()
	}

	newLogger := &Logger{
		w:       w,
		noColor: noColor,
		level:   levelVar.Level(),
		ctx:     context.Background(),
		config:  DefaultConfig(), // 使用实例级别的配置
		text:    slog.New(newAddonsHandler(NewConsoleHandler(w, noColor, options), ext)),
		json:    slog.New(newAddonsHandler(NewJSONHandler(w, options), ext)),
	}

	// 保持向后兼容性：如果全局logger未设置，则设置为此实例
	// 但现在优先使用LoggerManager管理的实例
	if logger == nil {
		logger = newLogger
	}

	return newLogger
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

// NewLoggerWithJSON 创建一个JSON格式的日志记录器
func NewLoggerWithJSON(writer io.Writer, addSource bool) Logger {
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
			a.Value = slog.StringValue(levelJSONNames[level])
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
		// 使用LoggerManager获取默认logger而不是全局变量
		return globalManager.GetDefault()
	}

	// 构建模块标识符
	module := strings.Join(modules, ".")

	// 创建新的带模块前缀的logger
	newLogger := globalManager.GetDefault().clone()

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
	return globalManager.GetDefault().GetSlogLogger()
}

// GetLevel 获取全局日志级别。
func GetLevel() Level { return levelVar.Level() }

// Debug 记录全局Debug级别的日志。
func Debug(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelDebug, msg, args...)
}

// Info 记录全局Info级别的日志。
func Info(msg string, args ...any) { globalManager.GetDefault().logWithLevel(LevelInfo, msg, args...) }

// Warn 记录全局Warn级别的日志。
func Warn(msg string, args ...any) { globalManager.GetDefault().logWithLevel(LevelWarn, msg, args...) }

// Error 记录全局Error级别的日志。
func Error(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelError, msg, args...)
}

// Trace 记录全局Trace级别的日志。
func Trace(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelTrace, msg, args...)
}

// Fatal 记录全局Fatal级别的日志，并退出程序。
func Fatal(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelFatal, msg, args...)
	os.Exit(1)
}

// Debugf 记录格式化的全局Debug级别的日志。
func Debugf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelDebug, format, args...)
}

// Infof 记录格式化的全局Info级别的日志。
func Infof(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelInfo, format, args...)
}

// Warnf 记录格式化的全局Warn级别的日志。
func Warnf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelWarn, format, args...)
}

// Errorf 记录格式化的全局Error级别的日志。
func Errorf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelError, format, args...)
}

// Tracef 记录格式化的全局Trace级别的日志。
func Tracef(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelTrace, format, args...)
}

// Fatalf 记录格式化的全局Fatal级别的日志，并退出程序。
func Fatalf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Println 记录信息级别的日志。
func Println(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelInfo, msg, args...)
}

// Printf 记录信息级别的格式化日志。
func Printf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelInfo, format, args...)
}

// 辅助便捷方法

// Dynamic 动态输出带点号动画效果
//
//   - msg: 要显示的消息内容
//   - frames: 动画更新的总帧数
//   - interval: 每次更新的时间间隔(毫秒)
func Dynamic(msg string, frames int, interval int) {
	globalManager.GetDefault().Dynamic(msg, frames, interval)
}

// Progress 全局进度显示
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
func Progress(msg string, durationMs int) {
	globalManager.GetDefault().Progress(msg, durationMs)
}

// Countdown 全局倒计时显示
//
//   - msg: 要显示的消息内容
//   - seconds: 倒计时的秒数
func Countdown(msg string, seconds int) {
	globalManager.GetDefault().Countdown(msg, seconds)
}

// Loading 全局加载动画
//
//   - msg: 要显示的消息内容
//   - seconds: 动画持续的秒数
func Loading(msg string, seconds int) {
	globalManager.GetDefault().Loading(msg, seconds)
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
	return globalManager.GetDefault().With(args...)
}

// WithGroup 创建一个带有指定组名的全局日志记录器
// 这是一个包级别的便捷方法
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func WithGroup(name string) *Logger { return globalManager.GetDefault().WithGroup(name) }

// WithValue 在全局上下文中添加键值对并返回新的 Logger
func WithValue(key string, val any) *Logger {
	// 获取现有的全局Logger并调用其WithValue方法
	return globalManager.GetDefault().WithValue(key, val)
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

// ResetGlobalLogger 重置全局logger实例
// 这在某些情况下很有用，比如需要更改全局logger的输出目标
func ResetGlobalLogger(w io.Writer, noColor, addSource bool) *Logger {
	// 使用LoggerManager重置全局状态
	config := &GlobalConfig{
		DefaultWriter:  w,
		DefaultLevel:   LevelInfo,
		DefaultNoColor: noColor,
		DefaultSource:  addSource,
		EnableText:     true,
		EnableJSON:     false,
	}
	globalManager.Configure(config)
	globalManager.Reset()

	// 同时保持向后兼容性，清空旧的全局变量
	logger = nil

	return globalManager.GetDefault()
}

// GetGlobalLogger 返回全局logger实例
// 如果全局logger未初始化，则创建一个默认的
func GetGlobalLogger() *Logger {
	if logger == nil {
		NewLogger(os.Stdout, false, false)
	}
	return logger
}

// Subscribe 订阅日志记录
// 创建一个新的日志订阅，返回接收日志记录的通道和取消订阅的函数
//
// 参数:
//   - size: 通道缓冲区大小，决定可以在不阻塞的情况下缓存多少日志记录
//
// 返回值:
//   - <-chan slog.Record: 只读的日志记录通道，用于接收日志
//   - context.CancelFunc: 取消订阅的函数，调用后会停止接收日志并清理资源
func Subscribe(size uint16) (<-chan slog.Record, context.CancelFunc) {
	ch := make(chan slog.Record, size)
	ctx, cancel := context.WithCancel(context.Background())

	sub := &subscriber{
		ch:     ch,
		cancel: cancel,
	}
	sub.state.Store(int32(stateActive)) // 设置为活跃状态

	// 生成唯一的订阅者ID
	subID := time.Now().UnixNano()
	subscribers.Store(subID, sub)

	// 创建安全的取消函数
	safeCancel := func() {
		sub.close()
		subscribers.Delete(subID)
	}

	// 监听context取消
	go func() {
		<-ctx.Done()
		safeCancel()
	}()

	return ch, safeCancel
}

// EnableTextLogger 启用文本日志记录器。
func EnableTextLogger() {
	textEnabled = true
}

// EnableJSONLogger 启用 JSON 日志记录器。
func EnableJSONLogger() {
	jsonEnabled = true
}

// DisableTextLogger 禁用文本日志记录器。
func DisableTextLogger() {
	textEnabled = false
}

// DisableJSONLogger 禁用 JSON 日志记录器。
func DisableJSONLogger() {
	jsonEnabled = false
}

// EnableDLPLogger 启用日志脱敏功能
func EnableDLPLogger() {
	dlpEnabled.Store(true)
	if ext != nil {
		ext.enableDLP()
	}
}

// DisableDLPLogger 禁用日志脱敏功能
func DisableDLPLogger() {
	dlpEnabled.Store(false)
	if ext != nil {
		ext.disableDLP()
	}
}

// IsDLPEnabled 检查DLP是否启用
func IsDLPEnabled() bool {
	return dlpEnabled.Load()
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
