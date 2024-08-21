package slog

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
)

type addons struct {
	PrefixKeys []string
}

// eHandler 是一个自定义的 slog 处理器，用于在日志消息前添加前缀，并将其传递给下一个处理器。
// 前缀从日志记录的属性中获取，使用 PrefixKeys 中指定的键。
type eHandler struct {
	handler  slog.Handler // 链中的下一个日志处理器。
	opts     addons       // 此处理器的配置选项。
	prefixes []slog.Value // 前缀值的缓存列表。
}

// newAddonsHandler 创建一个新的前缀日志处理器。
// 新处理器会在将每条日志消息传递给下一个处理器之前，从日志记录的属性中获取前缀并添加到消息前。
func newAddonsHandler(next slog.Handler, opts *addons) *eHandler {
	if opts == nil {
		opts = &addons{
			PrefixKeys: []string{"module"},
		}
	}
	return &eHandler{
		handler:  next,
		opts:     *opts,
		prefixes: make([]slog.Value, len(opts.PrefixKeys)),
	}
}

func (h *eHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录，如果需要，将前缀添加到消息，并将记录传递给下一个处理器。
func (h *eHandler) Handle(ctx context.Context, r slog.Record) error {
	if v, ok := ctx.Value(fields).(*sync.Map); ok {
		v.Range(func(key, val any) bool {
			if keyString, ok := key.(string); ok {
				r.AddAttrs(slog.Any(keyString, val))
			}
			return true
		})
	}

	prefixes := h.prefixes

	if r.NumAttrs() > 0 {
		nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		attrs := make([]slog.Attr, 0, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a)
			return true
		})
		if p, changed := h.extractPrefixes(attrs); changed {
			nr.AddAttrs(attrs...)
			r = nr
			prefixes = p
		}
	}

	r.Message = defaultPrefixFormatter(prefixes) + r.Message

	if recordChan != nil {
		select {
		case recordChan <- r:
		default:
		}
	}

	return h.handler.Handle(ctx, r)
}

func (h *eHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	p, _ := h.extractPrefixes(attrs)
	return &eHandler{
		handler:  h.handler.WithAttrs(attrs),
		opts:     h.opts,
		prefixes: p,
	}
}

func (h *eHandler) WithGroup(name string) slog.Handler {
	return &eHandler{
		handler:  h.handler.WithGroup(name),
		opts:     h.opts,
		prefixes: h.prefixes,
	}
}

// extractPrefixes 从属性列表中扫描 PrefixKeys 指定的键。
// 如果找到，它们的值将保存在新的前缀列表中。
// 原始属性列表将被修改以删除提取的前缀属性。
func (h *eHandler) extractPrefixes(attrs []slog.Attr) (prefixes []slog.Value, changed bool) {
	prefixes = h.prefixes
	for i, attr := range attrs {
		idx := slices.IndexFunc(h.opts.PrefixKeys, func(s string) bool { return s == attr.Key })
		if idx >= 0 {
			if !changed {
				// 复制前缀列表：
				prefixes = make([]slog.Value, len(h.prefixes))
				copy(prefixes, h.prefixes)
			}
			prefixes[idx] = attr.Value
			attrs[i] = slog.Attr{} // 移除前缀属性
			changed = true
		}
	}
	return
}

// defaultPrefixFormatter 通过使用 ":" 连接所有检测到的前缀值来构造前缀字符串。
func defaultPrefixFormatter(prefixes []slog.Value) string {
	p := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix.Any() == nil || prefix.String() == "" {
			continue // 跳过空前缀
		}
		p = append(p, prefix.String())
	}
	if len(p) == 0 {
		return ""
	}
	return "[" + strings.Join(p, ":") + "] "
}
