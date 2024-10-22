package dlp

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Formatter 定义日志格式化器接口
type Formatter func(groups []string, attr slog.Attr) (slog.Value, bool)

// DLPFormatter slog的DLP格式化器
type DLPFormatter struct {
	engine *DlpEngine
}

// DLPLevel 实现 slog.Leveler 接口
type DLPLevel slog.Level

func (l DLPLevel) Level() slog.Level {
	return slog.Level(l)
}

// NewDLPFormatter 创建新的DLP格式化器
func NewDLPFormatter(engine *DlpEngine) Formatter {
	return func(groups []string, attr slog.Attr) (slog.Value, bool) {
		if !engine.config.IsEnabled() {
			return attr.Value, false
		}

		switch attr.Value.Kind() {
		case slog.KindString:
			desensitized := engine.DesensitizeText(attr.Value.String())
			return slog.StringValue(desensitized), true

		case slog.KindGroup:
			attrs := attr.Value.Group()
			newAttrs := make([]slog.Attr, len(attrs))
			for i, a := range attrs {
				if v, ok := engine.formatAttr(append(groups, attr.Key), a); ok {
					newAttrs[i] = slog.Attr{Key: a.Key, Value: v}
				} else {
					newAttrs[i] = a
				}
			}
			return slog.GroupValue(newAttrs...), true

		case slog.KindAny:
			if str, ok := attr.Value.Any().(string); ok {
				desensitized := engine.DesensitizeText(str)
				return slog.AnyValue(desensitized), true
			}
		}

		return attr.Value, false
	}
}

// formatAttr 格式化单个属性
func (e *DlpEngine) formatAttr(groups []string, attr slog.Attr) (slog.Value, bool) {
	for _, formatter := range e.config.formatters {
		if v, ok := formatter(groups, attr); ok {
			return v, true
		}
	}
	return attr.Value, false
}

// EnableDLPHandler 创建一个支持DLP的Handler
func EnableDLPHandler(w io.Writer, engine *DlpEngine, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	// 添加DLP格式化器
	originalReplace := opts.ReplaceAttr
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if originalReplace != nil {
			a = originalReplace(groups, a)
		}

		if v, ok := NewDLPFormatter(engine)(groups, a); ok {
			a.Value = v
		}

		return a
	}

	return slog.NewTextHandler(w, opts)
}

// EnableDLPForLogger 为现有的logger启用DLP功能
func EnableDLPForLogger(logger *slog.Logger, engine *DlpEngine) *slog.Logger {
	var level DLPLevel

	// 检查现有logger的级别
	if logger.Handler().Enabled(context.Background(), slog.LevelInfo) {
		level = DLPLevel(slog.LevelInfo)
	} else if logger.Handler().Enabled(context.Background(), slog.LevelDebug) {
		level = DLPLevel(slog.LevelDebug)
	} else {
		level = DLPLevel(slog.LevelError)
	}

	// 创建新的handler，仅设置级别
	handler := EnableDLPHandler(os.Stdout, engine, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

// 默认的格式化器
func defaultFormatters() []Formatter {
	return []Formatter{
		// 敏感信息格式化器
		func(groups []string, attr slog.Attr) (slog.Value, bool) {
			sensitiveFields := map[string]bool{
				"password":   true,
				"token":      true,
				"key":        true,
				"secret":     true,
				"credential": true,
			}

			if sensitiveFields[attr.Key] && attr.Value.Kind() == slog.KindString {
				return slog.StringValue("******"), true
			}
			return attr.Value, false
		},

		// 个人信息格式化器
		func(groups []string, attr slog.Attr) (slog.Value, bool) {
			personalFields := map[string]DesensitizeFunc{
				"name":    ChineseNameDesensitize,
				"phone":   MobilePhoneDesensitize,
				"email":   EmailDesensitize,
				"address": AddressDesensitize,
				"id_card": IdCardDesensitize,
			}

			if desensitizer, ok := personalFields[attr.Key]; ok && attr.Value.Kind() == slog.KindString {
				return slog.StringValue(desensitizer(attr.Value.String())), true
			}
			return attr.Value, false
		},
	}
}
