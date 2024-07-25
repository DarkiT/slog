package slog

import (
	"context"
	"log/slog"
	"os"
)

// Logger 结构体定义了两个不同格式的日志记录器：文本记录器和 JSON 记录器。
type Logger struct {
	text *slog.Logger         // 用于文本格式的日志记录器
	json *slog.Logger         // 用于 JSON 格式的日志记录器
	opts *slog.HandlerOptions // 处理程序选项
}

// Debug 方法用于记录调试级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	r := newRecord(slog.LevelDebug, msg) // 创建一个新的记录
	r.Add(args...)                       // 添加参数到记录
	handle(l, r, slog.LevelDebug)        // 处理记录
}

// Info 方法用于记录信息级别的日志。
func (l *Logger) Info(msg string, args ...any) {
	r := newRecord(slog.LevelInfo, msg)
	r.Add(args...)
	handle(l, r, slog.LevelInfo)
}

// Warn 方法用于记录警告级别的日志。
func (l *Logger) Warn(msg string, args ...any) {
	r := newRecord(slog.LevelWarn, msg)
	r.Add(args...)
	handle(l, r, slog.LevelWarn)
}

// Error 方法用于记录错误级别的日志。
func (l *Logger) Error(msg string, args ...any) {
	r := newRecord(slog.LevelError, msg)
	r.Add(args...)
	handle(l, r, slog.LevelError)
}

// Panic 方法用于记录严重错误级别的日志，并终止程序。
func (l *Logger) Panic(msg string, args ...any) {
	r := newRecord(LevelFatal, msg)
	r.Add(args...)
	handle(l, r, LevelFatal)
	os.Exit(1) // 终止程序
}

// Debugf 方法用于格式化并记录调试级别的日志。
func (l *Logger) Debugf(format string, args ...any) {
	r := newRecord(slog.LevelDebug, format, args...)
	handle(l, r, slog.LevelDebug)
}

// Infof 方法用于格式化并记录信息级别的日志。
func (l *Logger) Infof(format string, args ...any) {
	r := newRecord(slog.LevelInfo, format, args...)
	handle(l, r, slog.LevelInfo)
}

// Warnf 方法用于格式化并记录警告级别的日志。
func (l *Logger) Warnf(format string, args ...any) {
	r := newRecord(slog.LevelWarn, format, args...)
	handle(l, r, slog.LevelWarn)
}

// Errorf 方法用于格式化并记录错误级别的日志。
func (l *Logger) Errorf(format string, args ...any) {
	r := newRecord(slog.LevelError, format, args...)
	handle(l, r, slog.LevelError)
}

// Panicf 方法用于格式化并记录严重错误级别的日志，并终止程序。
func (l *Logger) Panicf(format string, args ...any) {
	r := newRecord(LevelFatal, format, args...)
	handle(l, r, LevelFatal)
	os.Exit(1) // 终止程序
}

// Printf 为了兼容fmt.Printf风格输出
func (l *Logger) Printf(msg string, args ...any) {
	r := newRecord(LevelInfo, msg)
	r.Add(args...)
	handle(l, r, LevelInfo)
}

// Println 为了兼容fmt.Println风格输出
func (l *Logger) Println(msg string, args ...any) {
	r := newRecord(LevelInfo, msg)
	r.Add(args...)
	handle(l, r, LevelInfo)
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

// WithValue 在上下文中存储一个键值对，并返回新的上下文
func (l *Logger) WithValue(parent context.Context, key string, val any) context.Context {
	return WithValue(parent, key, val)
}

// Log 方法用于记录指定级别的日志。
func (l *Logger) Log(parent context.Context, level slog.Level, msg string, args ...any) {
	lv := level
	if parent != nil {
		ctx = parent // 使用给定的上下文
	}

	r := newRecord(lv, msg)
	r.Add(args...)
	if textEnabled && l.text.Enabled(ctx, lv) {
		l.text.Handler().Handle(ctx, r) // 处理文本格式的日志
	}

	if jsonEnabled && l.json.Enabled(ctx, lv) {
		l.json.Handler().Handle(ctx, r) // 处理 JSON 格式的日志
	}
}

// LogAttrs 方法用于记录具有指定属性的日志。
func (l *Logger) LogAttrs(parent context.Context, level slog.Level, msg string, attrs ...Attr) {
	lv := level
	if parent != nil {
		ctx = parent // 使用给定的上下文
	}

	r := newRecord(lv, msg)
	r.AddAttrs(attrs...)
	if textEnabled && l.text.Enabled(ctx, lv) {
		l.text.Handler().Handle(ctx, r)
	}

	if jsonEnabled && l.json.Enabled(ctx, lv) {
		l.json.Handler().Handle(ctx, r)
	}

}
