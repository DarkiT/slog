package slog

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/darkit/slog/formatter"
)

type extends struct {
	PrefixKeys []string
	formatters []formatter.Formatter
}

// eHandler 是一个自定义的 slog 处理器，用于在日志消息前添加前缀，并将其传递给下一个处理器。
// 前缀从日志记录的属性中获取，使用 PrefixKeys 中指定的键。
type eHandler struct {
	handler  slog.Handler // 链中的下一个日志处理器。
	opts     extends      // 此处理器的配置选项。
	prefixes []slog.Value // 前缀值的缓存列表。
	groups   []string
	ctx      context.Context
	module   string
}

// newAddonsHandler 创建一个新的前缀日志处理器。
// 新处理器会在将每条日志消息传递给下一个处理器之前，从日志记录的属性中获取前缀并添加到消息前。
// newAddonsHandler 创建新的处理器实例
func newAddonsHandler(next slog.Handler, opts *extends) *eHandler {
	if opts == nil {
		opts = &extends{
			PrefixKeys: []string{"module"},
		}
	}

	return &eHandler{
		handler:  next,
		opts:     *opts,
		groups:   []string{},
		prefixes: make([]slog.Value, len(opts.PrefixKeys)),
	}
}

func (h *eHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录，如果需要，将前缀添加到消息，并将记录传递给下一个处理器。
func (h *eHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		ctx = h.ctx
	}

	// 构建带前缀的消息
	prefix := ""
	if len(h.prefixes) > 0 && h.prefixes[0].Any() != nil {
		prefix = "[" + h.prefixes[0].String() + "] " // 添加方括号和空格
	}

	// 创建新记录，使用带前缀的消息
	nr := slog.NewRecord(r.Time, r.Level, prefix+r.Message, r.PC)

	// 处理上下文字段
	if v := ctx.Value(fieldsKey); v != nil {
		if fields, ok := v.(*Fields); ok {
			if m := fields.store.Load().(*sync.Map); m != nil {
				seen := make(map[string]bool)
				r.Attrs(func(attr slog.Attr) bool {
					seen[attr.Key] = true
					return true
				})

				m.Range(func(key, val interface{}) bool {
					if keyStr, ok := key.(string); ok {
						if !seen[keyStr] && keyStr != "module" { // 排除 module 属性
							nr.AddAttrs(slog.Any(keyStr, val))
						}
					}
					return true
				})
			}
		}
	}

	// 添加原始记录的属性（排除 module）
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "module" {
			nr.AddAttrs(h.transformAttr(h.groups, attr))
		}
		return true
	})

	return h.handler.Handle(ctx, nr)
}

// WithAttrs 方法，正确使用模块值
func (h *eHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &eHandler{
		handler:  h.handler.WithAttrs(h.transformAttrs(h.groups, attrs)),
		opts:     h.opts,
		groups:   slices.Clone(h.groups),
		prefixes: slices.Clone(h.prefixes), // 保持现有前缀
	}
}

func (h *eHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &eHandler{
		handler:  h.handler.WithGroup(name),
		opts:     h.opts,
		groups:   append(slices.Clone(h.groups), name),
		prefixes: slices.Clone(h.prefixes),
	}
}

func (h *eHandler) transformAttrs(groups []string, attrs []slog.Attr) []slog.Attr {
	for i := range attrs {
		attrs[i] = h.transformAttr(groups, attrs[i])
	}
	return attrs
}

func (h *eHandler) transformAttr(groups []string, attr slog.Attr) slog.Attr {
	for attr.Value.Kind() == slog.KindLogValuer {
		attr.Value = attr.Value.LogValuer().LogValue()
	}

	for _, _formatter := range h.opts.formatters {
		if v, ok := _formatter(groups, attr); ok {
			attr.Value = v
		}
	}

	return attr
}

// defaultPrefixFormatter 通过使用 ":" 连接所有检测到的前缀值来构造前缀字符串。
func defaultPrefixFormatter(prefixes []slog.Value) string {
	p := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix.Any() == nil {
			continue
		}
		str := prefix.String()
		if str == "" {
			continue
		}
		p = append(p, str)
	}
	if len(p) == 0 {
		return ""
	}

	return "[" + strings.Join(p, ":") + "] "
}
