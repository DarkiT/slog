package slog

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger 结构体定义了两个不同格式的日志记录器：文本记录器和 JSON 记录器。
type Logger struct {
	prefix string
	ctx    context.Context
	text   *slog.Logger         // 用于文本格式的日志记录器
	json   *slog.Logger         // 用于 JSON 格式的日志记录器
	opts   *slog.HandlerOptions // 处理程序选项
}

// GetLevel 方法用于获取当前的日志等级
func (l *Logger) GetLevel() Level {
	if l.text != nil {
		if l.text.Enabled(l.ctx, LevelDebug) {
			return LevelDebug
		}
		if l.text.Enabled(l.ctx, LevelInfo) {
			return LevelInfo
		}
		if l.text.Enabled(l.ctx, LevelWarn) {
			return LevelWarn
		}
		if l.text.Enabled(l.ctx, LevelError) {
			return LevelError
		}
		if l.text.Enabled(l.ctx, LevelFatal) {
			return LevelFatal
		}
		if l.text.Enabled(l.ctx, LevelTrace) {
			return LevelTrace
		}
	} else if l.json != nil {
		if l.json.Enabled(l.ctx, LevelDebug) {
			return LevelDebug
		}
		if l.json.Enabled(l.ctx, LevelInfo) {
			return LevelInfo
		}
		if l.json.Enabled(l.ctx, LevelWarn) {
			return LevelWarn
		}
		if l.json.Enabled(l.ctx, LevelError) {
			return LevelError
		}
		if l.json.Enabled(l.ctx, LevelFatal) {
			return LevelFatal
		}
		if l.json.Enabled(l.ctx, LevelTrace) {
			return LevelTrace
		}
	}
	return LevelInfo
}

// logWithLevel 处理日志记录的通用逻辑
func (l *Logger) logWithLevel(level Level, msg string, args ...any) {
	var r slog.Record
	if formatLog(msg, args...) {
		r = newRecord(level, msg, args...)
	} else {
		r = newRecord(level, msg)
		r.Add(args...)
	}

	handle(l, r, level) // 处理记录
}

// Debug 方法用于记录调试级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.logWithLevel(LevelDebug, msg, args...)
}

// Info 方法用于记录信息级别的日志。
func (l *Logger) Info(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// Warn 方法用于记录警告级别的日志。
func (l *Logger) Warn(msg string, args ...any) {
	l.logWithLevel(LevelWarn, msg, args...)
}

// Error 方法用于记录错误级别的日志。
func (l *Logger) Error(msg string, args ...any) {
	l.logWithLevel(LevelError, msg, args...)
}

// Trace 方法用于记录跟踪级别的日志。
func (l *Logger) Trace(msg string, args ...any) {
	l.logWithLevel(LevelTrace, msg, args...)
}

// Fatal 方法用于记录严重错误级别的日志，并终止程序。
func (l *Logger) Fatal(msg string, args ...any) {
	l.logWithLevel(LevelFatal, msg, args...)
	os.Exit(1) // 终止程序
}

// logfWithLevel 处理格式化日志记录的通用逻辑
func (l *Logger) logfWithLevel(level Level, format string, args ...any) {
	r := newRecord(level, format, args...)
	handle(l, r, level)
}

// Debugf 方法用于格式化并记录调试级别的日志。
func (l *Logger) Debugf(format string, args ...any) {
	l.logfWithLevel(LevelDebug, format, args...)
}

// Infof 方法用于格式化并记录信息级别的日志。
func (l *Logger) Infof(format string, args ...any) {
	l.logfWithLevel(LevelInfo, format, args...)
}

// Warnf 方法用于格式化并记录警告级别的日志。
func (l *Logger) Warnf(format string, args ...any) {
	l.logfWithLevel(LevelWarn, format, args...)
}

// Errorf 方法用于格式化并记录错误级别的日志。
func (l *Logger) Errorf(format string, args ...any) {
	l.logfWithLevel(LevelError, format, args...)
}

// Tracef 方法用于记录跟踪级别的日志。
func (l *Logger) Tracef(format string, args ...any) {
	l.logfWithLevel(LevelTrace, format, args...)
}

// Fatalf 方法用于格式化并记录严重错误级别的日志，并终止程序。
func (l *Logger) Fatalf(format string, args ...any) {
	l.logfWithLevel(LevelFatal, format, args...)
	os.Exit(1) // 终止程序
}

// Printf 为了兼容fmt.Printf风格输出
func (l *Logger) Printf(format string, args ...any) {
	l.logWithLevel(LevelInfo, format, args...)
}

// Println 为了兼容fmt.Println风格输出
func (l *Logger) Println(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// With 方法返回一个新的 Logger，其中包含给定的参数。
func (l *Logger) With(args ...any) *Logger {
	return l.withLogger(nil, args...)
}

// WithGroup 方法返回一个新的 Logger，其中包含给定的组名。
func (l *Logger) WithGroup(name string) *Logger {
	return l.withLogger(&name, nil)
}

// withLogger 创建新的日志记录器
func (l *Logger) withLogger(group *string, args ...any) *Logger {
	if l.text == nil && l.json == nil {
		return l
	}

	var text, json *slog.Logger
	if l.text != nil {
		l.text.Enabled(l.ctx, l.GetLevel())
		if group != nil {
			text = l.text.WithGroup(*group)
		} else {
			text = l.text.With(args...)
		}
	}
	if l.json != nil {
		l.json.Enabled(l.ctx, l.GetLevel())
		if group != nil {
			json = l.json.WithGroup(*group)
		} else {
			json = l.json.With(args...)
		}
	}

	return &Logger{text: text, ctx: l.ctx, json: json}
}

// Log 方法用于记录指定级别的日志。
func (l *Logger) Log(parent context.Context, level Level, msg string, args ...any) {
	l.logWithContext(parent, level, msg, args...)
}

// LogAttrs 方法用于记录具有指定属性的日志。
func (l *Logger) LogAttrs(parent context.Context, level Level, msg string, attrs ...slog.Attr) {
	if parent != nil {
		l.ctx = parent
	}

	r := newRecord(level, msg)
	r.AddAttrs(attrs...)
	if textEnabled && l.text.Enabled(l.ctx, level) {
		l.text.Handler().Handle(l.ctx, r)
	}

	if jsonEnabled && l.json.Enabled(l.ctx, level) {
		l.json.Handler().Handle(l.ctx, r)
	}
}

// logWithContext 处理带有上下文的日志记录
func (l *Logger) logWithContext(ctx context.Context, level Level, msg string, args ...any) {
	if ctx != nil {
		l.ctx = ctx
	}

	r := newRecord(level, msg)
	r.Add(args...)
	if textEnabled && l.text.Enabled(l.ctx, level) {
		l.text.Handler().Handle(l.ctx, r)
	}

	if jsonEnabled && l.json.Enabled(l.ctx, level) {
		l.json.Handler().Handle(l.ctx, r)
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
