package slog

import (
	"context"
	"log/slog"
	"os"
)

// Logger 结构体定义了两个不同格式的日志记录器：文本记录器和 JSON 记录器。
type Logger struct {
	prefix, fields string
	ctx            context.Context
	text           *slog.Logger         // 用于文本格式的日志记录器
	json           *slog.Logger         // 用于 JSON 格式的日志记录器
	opts           *slog.HandlerOptions // 处理程序选项
}

// Debug 方法用于记录调试级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelDebug, msg, args...)
	} else {
		r = newRecord(LevelDebug, msg)
		r.Add(args...)
	}
	handle(l, r, LevelDebug) // 处理记录
}

// Info 方法用于记录信息级别的日志。
func (l *Logger) Info(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelInfo, msg, args...)
	} else {
		r = newRecord(LevelInfo, msg)
		r.Add(args...)
	}
	handle(l, r, LevelInfo)
}

// Warn 方法用于记录警告级别的日志。
func (l *Logger) Warn(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelWarn, msg, args...)
	} else {
		r = newRecord(LevelWarn, msg)
		r.Add(args...)
	}
	handle(l, r, LevelWarn)
}

// Error 方法用于记录错误级别的日志。
func (l *Logger) Error(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelError, msg, args...)
	} else {
		r = newRecord(LevelError, msg)
		r.Add(args...)
	}
	handle(l, r, LevelError)
}

// Trace 方法用于记录跟踪级别的日志。
func (l *Logger) Trace(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelTrace, msg, args...)
	} else {
		r = newRecord(LevelTrace, msg)
		r.Add(args...)
	}
	handle(l, r, LevelTrace)
}

// Fatal 方法用于记录严重错误级别的日志，并终止程序。
func (l *Logger) Fatal(msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(LevelFatal, msg, args...)
	} else {
		r = newRecord(LevelFatal, msg)
		r.Add(args...)
	}
	handle(l, r, LevelFatal)
	os.Exit(1) // 终止程序
}

// Debugf 方法用于格式化并记录调试级别的日志。
func (l *Logger) Debugf(format string, args ...any) {
	r := newRecord(LevelDebug, format, args...)
	handle(l, r, LevelDebug)
}

// Infof 方法用于格式化并记录信息级别的日志。
func (l *Logger) Infof(format string, args ...any) {
	r := newRecord(LevelInfo, format, args...)
	handle(l, r, LevelInfo)
}

// Warnf 方法用于格式化并记录警告级别的日志。
func (l *Logger) Warnf(format string, args ...any) {
	r := newRecord(LevelWarn, format, args...)
	handle(l, r, LevelWarn)
}

// Errorf 方法用于格式化并记录错误级别的日志。
func (l *Logger) Errorf(format string, args ...any) {
	r := newRecord(LevelError, format, args...)
	handle(l, r, LevelError)
}

// Tracef 方法用于记录跟踪级别的日志。
func (l *Logger) Tracef(msg string, args ...any) {
	r := newRecord(LevelTrace, msg, args...)
	handle(l, r, LevelTrace)
}

// Fatalf 方法用于格式化并记录严重错误级别的日志，并终止程序。
func (l *Logger) Fatalf(format string, args ...any) {
	r := newRecord(LevelFatal, format, args...)
	handle(l, r, LevelFatal)
	os.Exit(1) // 终止程序
}

// Printf 为了兼容fmt.Printf风格输出
func (l *Logger) Printf(msg string, args ...any) {
	l.Info(msg, args...)
}

// Println 为了兼容fmt.Println风格输出
func (l *Logger) Println(msg string, args ...any) {
	l.Info(msg, args...)
}

// With 方法返回一个新的 Logger，其中包含给定的参数。
func (l *Logger) With(args ...any) *Logger {
	if l.text == nil && l.json == nil {
		return l // 如果文本和 JSON 记录器都为空，返回原始 Logger
	}

	var text, json *slog.Logger
	if l.text != nil {
		text = l.text.With(args...)
	}
	if l.json != nil {
		json = l.json.With(args...)
	}
	return &Logger{text: text, json: json} // 返回一个新的 Logger
}

// WithGroup 方法返回一个新的 Logger，其中包含给定的组名。
func (l *Logger) WithGroup(name string) *Logger {
	if l.text == nil && l.json == nil {
		return l
	}

	var text, json *slog.Logger
	if l.text != nil {
		text = l.text.WithGroup(name)
	}
	if l.json != nil {
		json = l.json.WithGroup(name)
	}

	return &Logger{text: text, json: json}
}

// Log 方法用于记录指定级别的日志。
func (l *Logger) Log(parent context.Context, level slog.Level, msg string, args ...any) {
	lv := level
	if parent != nil {
		l.ctx = parent // 使用给定的上下文
	}

	r := newRecord(lv, msg)
	r.Add(args...)
	if textEnabled && l.text.Enabled(l.ctx, lv) {
		l.text.Handler().Handle(l.ctx, r) // 处理文本格式的日志
	}

	if jsonEnabled && l.json.Enabled(l.ctx, lv) {
		l.json.Handler().Handle(l.ctx, r) // 处理 JSON 格式的日志
	}
}

// LogAttrs 方法用于记录具有指定属性的日志。
func (l *Logger) LogAttrs(parent context.Context, level slog.Level, msg string, attrs ...Attr) {
	lv := level
	if parent != nil {
		l.ctx = parent // 使用给定的上下文
	}

	r := newRecord(lv, msg)
	r.AddAttrs(attrs...)
	if textEnabled && l.text.Enabled(l.ctx, lv) {
		l.text.Handler().Handle(l.ctx, r)
	}

	if jsonEnabled && l.json.Enabled(l.ctx, lv) {
		l.json.Handler().Handle(l.ctx, r)
	}

}
