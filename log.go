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
	textEnabled, jsonEnabled bool
	logger                   Logger
	levelVar                 slog.LevelVar
	recordChan               chan slog.Record
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
	logger = Logger{
		ctx: context.Background(),
	}
	if isTerminal() {
		SetTextLogger(os.Stdout, false, false)
	} else {
		SetJsonLogger(os.Stdout, true)
	}
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
	if module == nil {
		return &logger
	}

	//返回一个新的 Logger，其中包含给定的参数
	log := logger.With()
	log.prefix = strings.Join(module, ":")
	log.ctx = context.Background()

	return log
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

	logger.setTextLogger(slog.New(NewConsoleHandler(writer, options)))
}

func SetJsonLogger(writer io.Writer, addSource bool) {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() <= LevelDebug

	logger.setJsonLogger(New(slog.NewJSONHandler(writer, options)))
}

func handle(l *Logger, r slog.Record, level slog.Level) {
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

	if recordChan != nil {
		select {
		case recordChan <- r:
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
func GetLevel() Level                   { return logger.GetLevel() }
func Debug(msg string, args ...any)     { logger.log(LevelDebug, msg, args...) }
func Info(msg string, args ...any)      { logger.log(LevelInfo, msg, args...) }
func Warn(msg string, args ...any)      { logger.log(LevelWarn, msg, args...) }
func Error(msg string, args ...any)     { logger.log(LevelError, msg, args...) }
func Trace(msg string, args ...any)     { logger.log(LevelTrace, msg, args...) }
func Fatal(msg string, args ...any)     { logger.log(LevelFatal, msg, args...); os.Exit(1) }
func Debugf(format string, args ...any) { logger.logf(LevelDebug, format, args...) }
func Infof(format string, args ...any)  { logger.logf(LevelInfo, format, args...) }
func Warnf(format string, args ...any)  { logger.logf(LevelWarn, format, args...) }
func Errorf(format string, args ...any) { logger.logf(LevelError, format, args...) }
func Tracef(format string, args ...any) { logger.logf(LevelTrace, format, args...) }
func Fatalf(format string, args ...any) {
	logger.logf(LevelFatal, format, args...)
	os.Exit(1)
}
func Println(msg string, args ...any)   { logger.log(LevelInfo, msg, args...) }
func Printf(format string, args ...any) { logger.log(LevelInfo, format, args...) }
func SetLevelTrace()                    { levelVar.Set(LevelTrace) }
func SetLevelDebug()                    { levelVar.Set(LevelDebug) }
func SetLevelInfo()                     { levelVar.Set(LevelInfo) }
func SetLevelWarn()                     { levelVar.Set(LevelWarn) }
func SetLevelError()                    { levelVar.Set(LevelError) }
func SetLevelFatal()                    { levelVar.Set(LevelFatal) }
func EnableTextLogger()                 { textEnabled = true }
func EnableJsonLogger()                 { jsonEnabled = true }
func DisableTextLogger()                { textEnabled = false }
func DisableJsonLogger()                { jsonEnabled = false }
func GetChannel(num ...uint16) chan slog.Record {
	var n uint16 = 500
	if recordChan == nil {
		if num != nil {
			n = num[0]
		}
		recordChan = make(chan slog.Record, n)
	}
	return recordChan
}
