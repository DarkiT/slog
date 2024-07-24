package slog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	logger                            Logger
	levelVar                          slog.LevelVar
	slogHandler                       slog.Handler
	textEnabled, jsonEnabled, noColor bool
)

var (
	ch             chan slog.Record
	LevelJsonNames = map[slog.Leveler]string{
		LevelInfo:  "Info",
		LevelDebug: "Debug",
		LevelWarn:  "Warn",
		LevelError: "Error",
		LevelTrace: "Trace",
		LevelFatal: "Fatal",
	}
)

func init() {
	SetTextLogger(os.Stdout, true)
}

// DisableColor 禁用颜色
func DisableColor(b bool) {
	noColor = b
}

// EnableTextLogger 启用文本记录器。
func EnableTextLogger() {
	textEnabled = true
}

// EnableJsonLogger 启用 JSON 记录器。
func EnableJsonLogger() {
	jsonEnabled = true
}

// DisableTextLogger 禁用文本记录器。
func DisableTextLogger() {
	if !jsonEnabled {
		return
	}
	textEnabled = false
}

// DisableJsonLogger 禁用 JSON 记录器。
func DisableJsonLogger() {
	if !textEnabled {
		return
	}
	jsonEnabled = false
}

// Default 返回默认记录器。
func Default() *Logger {
	return &logger
}

func setDefaultSlogHandlerOptions(l *slog.HandlerOptions) {
	l.AddSource = false
	l.Level = &levelVar
}

// AddSource 将源添加到 slog 处理程序选项。
func AddSource(options *slog.HandlerOptions) {
	options.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
		}
		if a.Key == slog.LevelKey {
			level := a.Value.Any().(slog.Level)
			var (
				levelLabel string
				exists     bool
			)
			if jsonEnabled {
				levelLabel, exists = LevelJsonNames[level]
			} else {
				levelLabel, exists = LevelTextNames[level]
			}
			if exists {
				a.Value = slog.StringValue(levelLabel)
			} else {
				levelLabel = level.String()
			}

		}
		return a
	}
}

// SetTextLogger 设置并启用文本记录器。
func SetTextLogger(writer io.Writer, addSource bool) {
	options := &slog.HandlerOptions{}

	setDefaultSlogHandlerOptions(options)

	AddSource(options)
	if levelVar.Level() <= LevelDebug || addSource {
		options.AddSource = true
	}

	slogHandler = NewHandler(writer, options)
	logger.text = slog.New(slogHandler)
	textEnabled = true
}

// SetJsonLogger 设置并启用 JSON 记录器。
func SetJsonLogger(writer io.Writer, addSource bool) {
	options := &slog.HandlerOptions{}
	setDefaultSlogHandlerOptions(options)

	AddSource(options)
	if levelVar.Level() <= LevelDebug || addSource {
		options.AddSource = true
	}
	slogHandler = slog.NewJSONHandler(writer, options)
	logger.json = slog.New(slogHandler)
	jsonEnabled = true
}

// LevelTrace 将默认记录器的级别设置为跟踪。
func SetLevelTrace() {
	levelVar.Set(LevelTrace)
}

// SetLevelDebug 将默认记录器的级别设置为调试。
func SetLevelDebug() {
	levelVar.Set(LevelDebug)
}

// SetLevelInfo 将默认记录器的级别设置为信息。
func SetLevelInfo() {
	levelVar.Set(LevelInfo)
}

// SetLevelWarn 将默认记录器的级别设置为警告。
func SetLevelWarn() {
	levelVar.Set(LevelWarn)
}

// SetLevelError 将默认记录器的级别设置为错误。
func SetLevelError() {
	levelVar.Set(LevelError)
}

// SetLevelFatal 将默认记录器的级别设置为错误。
func SetLevelFatal() {
	levelVar.Set(LevelFatal)
}

// GetHandler 返回当前的 slog.Handler 实例。
func GetHandler() slog.Handler {
	return slogHandler
}

// Println 记录调试消息。
//
//	log.Println("hello world")
//	log.Println("hello world", "age", 18, "name", "foo")
func Println(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelInfo, msg, args...)
	} else {
		r = newRecord(slog.LevelInfo, msg)
		r.Add(args...)
	}
	handle(nil, r, slog.LevelInfo)
}

// Printf 记录调试消息。
//
//	log.Printf("hello world")
//	log.Printf("hello world", "age", 18, "name", "foo")
func Printf(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelInfo, msg, args...)
	} else {
		r = newRecord(slog.LevelInfo, msg)
		r.Add(args...)
	}
	handle(nil, r, slog.LevelInfo)
}

// Debug 记录调试消息。
//
//	log.Debug("hello world")
//	log.Debug("hello world", "age", 18, "name", "foo")
func Debug(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelDebug, msg, args...)
	} else {
		r = newRecord(slog.LevelDebug, msg)
		r.Add(args...)
	}
	handle(nil, r, slog.LevelDebug)
}

// Info 记录信息消息。
//
//	log.Info("hello world")
//	log.Info("hello world", "age", 18, "name", "foo")
func Info(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelInfo, msg, args...)
	} else {
		r = newRecord(slog.LevelInfo, msg)
		r.Add(args...)
	}

	handle(nil, r, slog.LevelInfo)
}

// Warn 记录警告消息。
//
//	log.Warn("hello world")
//	log.Warn("hello world", "age", 18, "name", "foo")
func Warn(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelWarn, msg, args...)
	} else {
		r = newRecord(slog.LevelWarn, msg)
		r.Add(args...)
	}
	handle(nil, r, slog.LevelWarn)
}

// Error 记录错误消息。
//
//	log.Error("hello world")
//	log.Error("hello world", "age", 18, "name", "foo")
func Error(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(slog.LevelError, msg, args...)
	} else {
		r = newRecord(slog.LevelError, msg)
		r.Add(args...)
	}
	handle(nil, r, slog.LevelError)
}

// Trace 记录跟踪消息。
//
//	log.Trace("hello world")
//	log.Trace("hello world", "age", 18, "name", "foo")
func Trace(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelTrace, msg, args...)
	} else {
		r = newRecord(LevelTrace, msg)
		r.Add(args...)
	}
	handle(nil, r, LevelTrace)
}

// Panic 记录错误消息并以 `1` 错误代码退出当前程序。
//
//	log.Panic("hello world")
//	log.Panic("hello world", "age", 18, "name", "foo")
func Panic(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelFatal, msg, args...)
	} else {
		r = newRecord(LevelFatal, msg)
		r.Add(args...)
	}
	handle(nil, r, LevelFatal)
	os.Exit(1)
}

// Debugf 记录并格式化调试消息。不能使用属性。
//
//	log.Debugf("hello world")
//	log.Debugf("hello %s", "world")
func Debugf(format string, args ...any) {
	r := newRecord(slog.LevelDebug, format, args...)
	handle(nil, r, slog.LevelDebug)
}

// Infof 记录并格式化信息消息。不能使用属性。
//
//	log.Infof("hello world")
//	log.Infof("hello %s", "world")
func Infof(format string, args ...any) {
	r := newRecord(slog.LevelInfo, format, args...)
	handle(nil, r, slog.LevelInfo)
}

// Warnf 记录并格式化警告消息。不能使用属性。
//
//	log.Warnf("hello world")
//	log.Warnf("hello %s", "world")
func Warnf(format string, args ...any) {
	r := newRecord(slog.LevelWarn, format, args...)
	handle(nil, r, slog.LevelWarn)
}

// Errorf 记录并格式化错误消息。不能使用属性。
//
//	log.Errorf("hello world")
//	log.Errorf("hello %s", "world")
func Errorf(format string, args ...any) {
	r := newRecord(slog.LevelError, format, args...)
	handle(nil, r, slog.LevelError)
}

// Tracef 记录并格式化跟踪消息。不能使用属性。
//
//	log.Tracef("hello world")
//	log.Tracef("hello %s", "world")
func Tracef(format string, args ...any) {
	r := newRecord(LevelTrace, format, args...)
	handle(nil, r, LevelTrace)
}

// Panicf 记录并格式化错误消息并以 `1` 错误代码退出当前程序。不能使用属性。
//
//	log.Panicf("hello world")
//	log.Panicf("hello %s", "world")
func Panicf(format string, args ...any) {
	r := newRecord(LevelFatal, format, args...)
	handle(nil, r, LevelFatal)
	os.Exit(1)
}

// GetChannel 获取通道数据，若通道不存在则自动初始化。
func GetChannel() chan slog.Record {
	if ch == nil {
		ch = make(chan slog.Record, 200)
	}
	return ch
}

func newRecord(level slog.Level, format string, args ...any) slog.Record {
	t := time.Now()

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [runtime.Callers, this function, this function's caller]

	if args == nil {
		return slog.NewRecord(t, level, format, pcs[0])
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pcs[0])
}

func handle(l *Logger, r slog.Record, level slog.Level) {
	if v, ok := ctx.Value(fields).(*sync.Map); ok {
		v.Range(func(key, val any) bool {
			if keyString, ok := key.(string); ok {
				r.AddAttrs(slog.Any(keyString, val))
			}
			return true
		})
	}

	if l == nil {
		if textEnabled && logger.text.Enabled(ctx, level) {
			_ = logger.text.Handler().Handle(ctx, r)
		}
		if jsonEnabled && logger.json.Enabled(ctx, level) {
			_ = logger.json.Handler().Handle(ctx, r)
		}
	} else {
		if textEnabled && l.text.Enabled(ctx, level) {
			_ = l.text.Handler().Handle(ctx, r)
		}
		if jsonEnabled && l.json.Enabled(ctx, level) {
			_ = l.json.Handler().Handle(ctx, r)
		}
	}

	// 使用通道发送
	if ch != nil {
		select {
		case ch <- r:
		default:
		}
	}

}

func formatLog(msg string, args ...any) bool {
	if len(args) == 0 {
		return false
	}
	if strings.Contains(msg, "%") && !strings.Contains(msg, "%%") {
		return true
	}
	return false
}
