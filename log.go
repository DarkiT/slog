package slog

import (
	"context"
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
	once                     sync.Once
	modules                  map[string]*Logger
	levelVar                 slog.LevelVar
	textEnabled, jsonEnabled bool
	ch                       chan slog.Record
	defaultLogger            Logger
	levelJsonNames           = map[Level]string{
		LevelInfo:  "Info",
		LevelDebug: "Debug",
		LevelWarn:  "Warn",
		LevelError: "Error",
		LevelTrace: "Trace",
		LevelFatal: "Fatal",
	}
)

func init() {
	if isTerminal() {
		SetTextLogger(os.Stdout, false, false)
	} else {
		SetJsonLogger(os.Stdout, true)
	}

	defaultLogger = *Default()
	defaultLogger.ctx = context.Background()
}

// NewOptions 创建新的处理程序选项。
func NewOptions(options *slog.HandlerOptions) *slog.HandlerOptions {
	if options == nil {
		options = &slog.HandlerOptions{
			Level: &levelVar,
		}
	}

	if levelVar.Level() <= LevelDebug {
		options.AddSource = true
	}

	AddSource(options)

	return options
}

func New(handler slog.Handler) *slog.Logger {
	return slog.New(handler)
}

func Default(module ...string) *Logger {
	once.Do(func() {
		modules = make(map[string]*Logger)
	})

	mod := strings.Join(module, ":")
	if log, ok := modules[mod]; ok {
		return log
	}

	newLogger := defaultLogger
	newLogger.prefix = mod
	newLogger.ctx = context.Background()

	modules[mod] = &newLogger

	return &newLogger
}

func setDefaultSlogHandlerOptions(l *slog.HandlerOptions) {
	l.AddSource = false
	l.Level = &levelVar
}

func AddSource(options *slog.HandlerOptions) {
	options.Level = &levelVar
	options.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.SourceKey:
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
		case LevelKey:
			level := a.Value.Any().(Level)
			a.Value = slog.StringValue(levelJsonNames[level])
		}
		return a
	}
}

func SetTextLogger(writer io.Writer, noColor, addSource bool) {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() <= LevelDebug

	logger := New(NewConsoleHandler(writer, options))
	defaultLogger.setTextLogger(logger)
}

func SetJsonLogger(writer io.Writer, addSource bool) {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() <= LevelDebug

	logger := New(slog.NewJSONHandler(writer, options))
	defaultLogger.setJsonLogger(logger)
}

func (l *Logger) setTextLogger(logger *slog.Logger) {
	l.text = logger
	textEnabled = true
}

func (l *Logger) setJsonLogger(logger *slog.Logger) {
	l.json = logger
	jsonEnabled = true
}

func (l *Logger) log(level Level, msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(level, msg, args...)
	} else {
		r = newRecord(level, msg)
		r.Add(args...)
	}
	handle(l, r, level)
}

func (l *Logger) logf(level Level, format string, args ...any) {
	r := newRecord(level, format, args...)
	handle(l, r, level)
}

func handle(l *Logger, r slog.Record, level slog.Level) {
	r.AddAttrs(slog.String("$service", l.prefix))
	//l.With("$service", l.prefix)
	if v, ok := l.ctx.Value(fields).(*sync.Map); ok {
		v.Range(func(key, val any) bool {
			r.AddAttrs(slog.Any(key.(string), val))
			return true
		})
	}

	if textEnabled && l.text != nil && l.text.Enabled(l.ctx, level) {
		_ = l.text.Handler().Handle(l.ctx, r)
	}
	if jsonEnabled && l.json != nil && l.json.Enabled(l.ctx, level) {
		_ = l.json.Handler().Handle(l.ctx, r)
	}

	if ch != nil {
		select {
		case ch <- r:
		default:
		}
	}
}

func newRecord(level Level, format string, args ...any) slog.Record {
	t := time.Now()
	var pcs [1]uintptr
	runtime.Callers(4, pcs[:]) // skip [runtime.Callers, this function, this function's caller]

	if args == nil {
		return slog.NewRecord(t, level, format, pcs[0])
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pcs[0])
}

func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

// 对外公开的接口
func Debug(msg string, args ...any)     { defaultLogger.log(LevelDebug, msg, args...) }
func Info(msg string, args ...any)      { defaultLogger.log(LevelInfo, msg, args...) }
func Warn(msg string, args ...any)      { defaultLogger.log(LevelWarn, msg, args...) }
func Error(msg string, args ...any)     { defaultLogger.log(LevelError, msg, args...) }
func Trace(msg string, args ...any)     { defaultLogger.log(LevelTrace, msg, args...) }
func Fatal(msg string, args ...any)     { defaultLogger.log(LevelFatal, msg, args...); os.Exit(1) }
func Debugf(format string, args ...any) { defaultLogger.logf(LevelDebug, format, args...) }
func Infof(format string, args ...any)  { defaultLogger.logf(LevelInfo, format, args...) }
func Warnf(format string, args ...any)  { defaultLogger.logf(LevelWarn, format, args...) }
func Errorf(format string, args ...any) { defaultLogger.logf(LevelError, format, args...) }
func Tracef(format string, args ...any) { defaultLogger.logf(LevelTrace, format, args...) }
func Fatalf(format string, args ...any) {
	defaultLogger.logf(LevelFatal, format, args...)
	os.Exit(1)
}
func Println(msg string, args ...any)   { defaultLogger.log(LevelInfo, msg, args...) }
func Printf(format string, args ...any) { defaultLogger.log(LevelInfo, format, args...) }
func SetLevelTrace()                    { levelVar.Set(LevelTrace) }
func SetLevelDebug()                    { levelVar.Set(LevelDebug) }
func SetLevelInfo()                     { levelVar.Set(LevelInfo) }
func SetLevelWarn()                     { levelVar.Set(LevelWarn) }
func SetLevelError()                    { levelVar.Set(LevelError) }
func SetLevelFatal()                    { levelVar.Set(LevelFatal) }
func EnableTextLogger()                 { textEnabled = true }
func EnableJsonLogger()                 { jsonEnabled = true }
func DisableTextLogger() {
	if jsonEnabled {
		textEnabled = false
	}
}

func DisableJsonLogger() {
	if textEnabled {
		jsonEnabled = false
	}
}

func GetChannel() chan slog.Record {
	if ch == nil {
		ch = make(chan slog.Record, 200)
	}
	return ch
}
