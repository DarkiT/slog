package slog

import (
	"context"
	"errors"
	"io"
	"log"
	stdslog "log/slog"
	"time"
)

const (
	// TimeKey 是标准 slog 内置时间字段名。
	TimeKey = stdslog.TimeKey
	// LevelKey 是标准 slog 内置级别字段名。
	LevelKey = stdslog.LevelKey
	// MessageKey 是标准 slog 内置消息字段名。
	MessageKey = stdslog.MessageKey
	// SourceKey 是标准 slog 内置调用源字段名。
	SourceKey = stdslog.SourceKey
)

const (
	// KindAny 表示任意 Go 值。
	KindAny = stdslog.KindAny
	// KindBool 表示 bool 值。
	KindBool = stdslog.KindBool
	// KindDuration 表示 time.Duration 值。
	KindDuration = stdslog.KindDuration
	// KindFloat64 表示 float64 值。
	KindFloat64 = stdslog.KindFloat64
	// KindInt64 表示 int64 值。
	KindInt64 = stdslog.KindInt64
	// KindString 表示 string 值。
	KindString = stdslog.KindString
	// KindTime 表示 time.Time 值。
	KindTime = stdslog.KindTime
	// KindUint64 表示 uint64 值。
	KindUint64 = stdslog.KindUint64
	// KindGroup 表示属性组。
	KindGroup = stdslog.KindGroup
	// KindLogValuer 表示延迟求值的 LogValuer。
	KindLogValuer = stdslog.KindLogValuer
)

// Level 映射标准库 log/slog.Level。
type Level = stdslog.Level

// Attr 映射标准库 log/slog.Attr。
type Attr = stdslog.Attr

// Value 映射标准库 log/slog.Value。
type Value = stdslog.Value

// Record 映射标准库 log/slog.Record。
type Record = stdslog.Record

// Handler 映射标准库 log/slog.Handler。
type Handler = stdslog.Handler

// HandlerOptions 映射标准库 log/slog.HandlerOptions。
type HandlerOptions = stdslog.HandlerOptions

// TextHandler 映射标准库 log/slog.TextHandler。
type TextHandler = stdslog.TextHandler

// JSONHandler 映射标准库 log/slog.JSONHandler。
type JSONHandler = stdslog.JSONHandler

// Kind 映射标准库 log/slog.Kind。
type Kind = stdslog.Kind

// LevelVar 映射标准库 log/slog.LevelVar。
type LevelVar = stdslog.LevelVar

// Leveler 映射标准库 log/slog.Leveler。
type Leveler = stdslog.Leveler

// LogValuer 映射标准库 log/slog.LogValuer。
type LogValuer = stdslog.LogValuer

// Source 映射标准库 log/slog.Source。
type Source = stdslog.Source

// SlogLogger 映射标准库 log/slog.Logger，用于避免与本包增强 Logger 重名。
type SlogLogger = stdslog.Logger

// StdLogger 是 SlogLogger 的语义别名，便于需要强调标准库 logger 的调用方使用。
type StdLogger = stdslog.Logger

// DiscardHandler 丢弃所有日志输出。
var DiscardHandler Handler = discardHandler{}

// NewTextHandler 映射标准库 log/slog.NewTextHandler。
func NewTextHandler(w io.Writer, opts *HandlerOptions) *TextHandler {
	return stdslog.NewTextHandler(w, opts)
}

// NewJSONHandler 映射标准库 log/slog.NewJSONHandler。
func NewJSONHandler(w io.Writer, opts *HandlerOptions) *JSONHandler {
	return stdslog.NewJSONHandler(w, opts)
}

// NewRecord 映射标准库 log/slog.NewRecord。
func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return stdslog.NewRecord(t, level, msg, pc)
}

// NewLogLogger 映射标准库 log/slog.NewLogLogger。
func NewLogLogger(h Handler, level Level) *log.Logger {
	return stdslog.NewLogLogger(h, level)
}

// SetLogLoggerLevel 映射标准库 log/slog.SetLogLoggerLevel。
func SetLogLoggerLevel(level Level) (oldLevel Level) {
	return stdslog.SetLogLoggerLevel(level)
}

// SetDefault 设置默认 Logger，并同步标准 log/slog 与本包顶层日志入口。
//
// logger 支持 *SlogLogger 与本包增强 *Logger，便于兼容标准库示例和 darkit/slog 的增强入口。
func SetDefault(logger any) {
	switch l := logger.(type) {
	case *SlogLogger:
		if l == nil {
			panic("slog.SetDefault: nil *SlogLogger")
		}
		stdslog.SetDefault(l)
		globalManager.setDefaultSlogLogger(l)
	case *Logger:
		if l == nil {
			panic("slog.SetDefault: nil *Logger")
		}
		if stdLogger := l.GetSlogLogger(); stdLogger != nil {
			stdslog.SetDefault(stdLogger)
		}
		globalManager.setDefaultLogger(l)
	default:
		panic("slog.SetDefault: logger must be *SlogLogger or *Logger")
	}
}

// StdDefault 返回标准库 log/slog 当前默认 logger。
func StdDefault() *SlogLogger {
	return stdslog.Default()
}

// AnyValue 映射标准库 log/slog.AnyValue。
func AnyValue(v any) Value {
	return stdslog.AnyValue(v)
}

// BoolValue 映射标准库 log/slog.BoolValue。
func BoolValue(v bool) Value {
	return stdslog.BoolValue(v)
}

// DurationValue 映射标准库 log/slog.DurationValue。
func DurationValue(v time.Duration) Value {
	return stdslog.DurationValue(v)
}

// Float64Value 映射标准库 log/slog.Float64Value。
func Float64Value(v float64) Value {
	return stdslog.Float64Value(v)
}

// GroupValue 映射标准库 log/slog.GroupValue。
func GroupValue(args ...Attr) Value {
	return stdslog.GroupValue(args...)
}

// Int64Value 映射标准库 log/slog.Int64Value。
func Int64Value(v int64) Value {
	return stdslog.Int64Value(v)
}

// IntValue 映射标准库 log/slog.IntValue。
func IntValue(v int) Value {
	return stdslog.IntValue(v)
}

// StringValue 映射标准库 log/slog.StringValue。
func StringValue(value string) Value {
	return stdslog.StringValue(value)
}

// TimeValue 映射标准库 log/slog.TimeValue。
func TimeValue(v time.Time) Value {
	return stdslog.TimeValue(v)
}

// Uint64Value 映射标准库 log/slog.Uint64Value。
func Uint64Value(v uint64) Value {
	return stdslog.Uint64Value(v)
}

// Any 映射标准库 log/slog.Any。
func Any(key string, v any) Attr {
	return stdslog.Any(key, v)
}

// Bool 映射标准库 log/slog.Bool。
func Bool(key string, v bool) Attr {
	return stdslog.Bool(key, v)
}

// Duration 映射标准库 log/slog.Duration。
func Duration(key string, v time.Duration) Attr {
	return stdslog.Duration(key, v)
}

// Float64 映射标准库 log/slog.Float64。
func Float64(key string, v float64) Attr {
	return stdslog.Float64(key, v)
}

// Group 映射标准库 log/slog.Group。
func Group(key string, args ...any) Attr {
	return stdslog.Group(key, args...)
}

// GroupAttrs 用已有 Attr 构造分组，等价于标准库较新版本的 log/slog.GroupAttrs。
func GroupAttrs(key string, attrs ...Attr) Attr {
	return Attr{Key: key, Value: GroupValue(attrs...)}
}

// Int 映射标准库 log/slog.Int。
func Int(key string, v int) Attr {
	return stdslog.Int(key, v)
}

// Int64 映射标准库 log/slog.Int64。
func Int64(key string, v int64) Attr {
	return stdslog.Int64(key, v)
}

// String 映射标准库 log/slog.String。
func String(key string, v string) Attr {
	return stdslog.String(key, v)
}

// Time 映射标准库 log/slog.Time。
func Time(key string, v time.Time) Attr {
	return stdslog.Time(key, v)
}

// Uint64 映射标准库 log/slog.Uint64。
func Uint64(key string, v uint64) Attr {
	return stdslog.Uint64(key, v)
}

// MultiHandler 将同一条记录分发给多个 Handler。
type MultiHandler struct {
	handlers []Handler
}

// NewMultiHandler 创建 MultiHandler，并复制输入切片以避免外部突变影响。
func NewMultiHandler(handlers ...Handler) *MultiHandler {
	copied := make([]Handler, len(handlers))
	copy(copied, handlers)
	return &MultiHandler{handlers: copied}
}

// Enabled 判断任一子 Handler 是否会处理当前级别。
func (h *MultiHandler) Enabled(ctx context.Context, level Level) bool {
	if h == nil {
		return false
	}
	for _, handler := range h.handlers {
		if handler != nil && handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle 将记录发送给所有启用的子 Handler。
func (h *MultiHandler) Handle(ctx context.Context, r Record) error {
	if h == nil {
		return nil
	}
	var errs []error
	for _, handler := range h.handlers {
		if handler == nil || !handler.Enabled(ctx, r.Level) {
			continue
		}
		if err := handler.Handle(ctx, r.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs 返回带固定属性的新 MultiHandler。
func (h *MultiHandler) WithAttrs(attrs []Attr) Handler {
	if h == nil {
		return DiscardHandler
	}
	handlers := make([]Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		if handler != nil {
			handlers = append(handlers, handler.WithAttrs(attrs))
		}
	}
	return &MultiHandler{handlers: handlers}
}

// WithGroup 返回带分组的新 MultiHandler。
func (h *MultiHandler) WithGroup(name string) Handler {
	if h == nil {
		return DiscardHandler
	}
	handlers := make([]Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		if handler != nil {
			handlers = append(handlers, handler.WithGroup(name))
		}
	}
	return &MultiHandler{handlers: handlers}
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, Level) bool { return false }

func (discardHandler) Handle(context.Context, Record) error { return nil }

func (dh discardHandler) WithAttrs([]Attr) Handler { return dh }

func (dh discardHandler) WithGroup(string) Handler { return dh }
