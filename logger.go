package slog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	LevelTrace Level = -8
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
	LevelFatal Level = 12
)

var (
	logger         Logger
	TimeFormat     = "2006/01/02 15:04.05.000"
	levelJsonNames = map[Level]string{
		LevelInfo:  "Info",
		LevelDebug: "Debug",
		LevelWarn:  "Warn",
		LevelError: "Error",
		LevelTrace: "Trace",
		LevelFatal: "Fatal",
	}
)

type Logger struct {
	text    *slog.Logger
	json    *slog.Logger
	ctx     context.Context
	noColor bool
	level   Level
}

// getEffectiveLevel 获取日志的有效级别。
func (l *Logger) getEffectiveLevel() Level {
	loggers := []*slog.Logger{l.text, l.json}
	if textEnabled && !jsonEnabled {
		loggers = []*slog.Logger{l.text}
	} else if jsonEnabled && !textEnabled {
		loggers = []*slog.Logger{l.json}
	}

	for _, lg := range loggers {
		if lg != nil {
			if lg.Enabled(l.ctx, LevelDebug) {
				return LevelDebug
			}
			if lg.Enabled(l.ctx, LevelInfo) {
				return LevelInfo
			}
			if lg.Enabled(l.ctx, LevelWarn) {
				return LevelWarn
			}
			if lg.Enabled(l.ctx, LevelError) {
				return LevelError
			}
			if lg.Enabled(l.ctx, LevelFatal) {
				return LevelFatal
			}
			if lg.Enabled(l.ctx, LevelTrace) {
				return LevelTrace
			}
		}
	}
	return levelVar.Level()
}

// GetLevel 获取当前日志级别。
func (l *Logger) GetLevel() Level {
	return l.getEffectiveLevel()
}

// SetLevel 动态设置日志打印级别。
func (l *Logger) SetLevel(level Level) *Logger {
	l.level = level
	return l
}

// logWithLevel 记录指定级别的日志。
func (l *Logger) logWithLevel(level Level, msg string, args ...any) {
	l.logRecord(level, msg, false, args...)
}

// logfWithLevel 记录格式化的日志。
func (l *Logger) logfWithLevel(level Level, format string, args ...any) {
	l.logRecord(level, fmt.Sprintf(format, args...), true, args...)
}

// withLogger 创建一个新的日志记录器，包含指定的分组或属性。
func (l *Logger) withLogger(group *string, args ...any) *Logger {
	newLogger := *l
	// 根据是否分组来初始化新的日志记录器
	if l.text != nil {
		if group != nil {
			newLogger.text = l.text.WithGroup(*group)
		} else {
			newLogger.text = l.text.With(args...)
		}
	}
	if l.json != nil {
		if group != nil {
			newLogger.json = l.json.WithGroup(*group)
		} else {
			newLogger.json = l.json.With(args...)
		}
	}
	return &newLogger
}

// With 创建一个新的日志记录器，带有指定的属性。
func (l *Logger) With(args ...any) *Logger {
	return l.withLogger(nil, args...)
}

// WithGroup 创建一个新的日志记录器，带有指定的分组。
func (l *Logger) WithGroup(name string) *Logger {
	return l.withLogger(&name)
}

// Debug 记录Debug级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.logWithLevel(LevelDebug, msg, args...)
}

// Info 记录Info级别的日志。
func (l *Logger) Info(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// Warn 记录Warn级别的日志。
func (l *Logger) Warn(msg string, args ...any) {
	l.logWithLevel(LevelWarn, msg, args...)
}

// Error 记录Error级别的日志。
func (l *Logger) Error(msg string, args ...any) {
	l.logWithLevel(LevelError, msg, args...)
}

// Fatal 记录Fatal级别的日志，并退出程序。
func (l *Logger) Fatal(msg string, args ...any) {
	l.logWithLevel(LevelFatal, msg, args...)
}

// Trace 记录Trace级别的日志。
func (l *Logger) Trace(msg string, args ...any) {
	l.logWithLevel(LevelTrace, msg, args...)
}

// Debugf 记录格式化的Debug级别的日志。
func (l *Logger) Debugf(format string, args ...any) {
	l.logfWithLevel(LevelDebug, format, args...)
}

// Infof 记录格式化的Info级别的日志。
func (l *Logger) Infof(format string, args ...any) {
	l.logfWithLevel(LevelInfo, format, args...)
}

// Warnf 记录格式化的Warn级别的日志。
func (l *Logger) Warnf(format string, args ...any) {
	l.logfWithLevel(LevelWarn, format, args...)
}

// Errorf 记录格式化的Error级别的日志。
func (l *Logger) Errorf(format string, args ...any) {
	l.logfWithLevel(LevelError, format, args...)
}

// Fatalf 记录格式化的Fatal级别的日志，并退出程序。
func (l *Logger) Fatalf(format string, args ...any) {
	l.logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Tracef 记录格式化的Trace级别的日志。
func (l *Logger) Tracef(format string, args ...any) {
	l.logfWithLevel(LevelTrace, format, args...)
}

// Printf 为了兼容fmt.Printf风格输出
func (l *Logger) Printf(format string, args ...any) {
	l.logWithLevel(LevelInfo, format, args...)
}

// Println 为了兼容fmt.Println风格输出
func (l *Logger) Println(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// logRecord 创建并记录日志记录。
func (l *Logger) logRecord(level Level, msg string, sprintf bool, args ...any) {
	r := newRecord(level, msg)
	if !sprintf && formatLog(msg, args...) {
		r = newRecord(level, msg, args...)
	} else if !sprintf {
		r.Add(args...)
	}
	l.handleRecord(r, level)
}

// handleRecord 处理并记录日志记录。
func (l *Logger) handleRecord(r slog.Record, level slog.Level) {
	if l.text == nil || l.json == nil {
		return
	}
	if l.level != level {
		levelVar.Set(l.level)
	}
	if textEnabled && l.text.Enabled(l.ctx, level) {
		_ = l.text.Handler().Handle(l.ctx, r)
	}
	if jsonEnabled && l.json.Enabled(l.ctx, level) {
		_ = l.json.Handler().Handle(l.ctx, r)
	}
}

// isTerminal 判断当前是否在终端环境中运行。
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

// formatLog 判断消息是否需要格式化。
func formatLog(msg string, args ...any) bool {
	//placeholders := []string{
	//	"%v", "%+v", "%#v", "%T", "%t", // 通用占位符
	//	"%b", "%c", "%d", "%o", "%O", "%q", "%x", "%X", // 整数占位符
	//	"%U", "%e", "%E", "%f", "%F", "%g", "%G", // 浮点数占位符
	//	"%s", "%q", "%x", "%X", // 字符串占位符
	//	"%p", // 指针占位符
	//}
	//
	//for _, ph := range placeholders {
	//	if strings.Contains(msg, ph) {
	//		return true
	//	}
	//}
	return len(args) > 0 && strings.Contains(msg, "%") && !strings.Contains(msg, "%%")
}

// newRecord 创建一个新的日志记录。
func newRecord(level Level, format string, args ...any) slog.Record {
	t := time.Now()
	var pcs [1]uintptr
	runtime.Callers(5, pcs[:]) // skip [runtime.Callers, this function, this function's caller]

	if args == nil {
		return slog.NewRecord(t, level, format, pcs[0])
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pcs[0])
}
