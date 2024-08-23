package slog

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/dlp/dlpheader"
	"github.com/darkit/slog/formatter"
)

var (
	once      sync.Once
	dlpEngine dlpheader.EngineAPI
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
}

// newAddonsHandler 创建一个新的前缀日志处理器。
// 新处理器会在将每条日志消息传递给下一个处理器之前，从日志记录的属性中获取前缀并添加到消息前。
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
	once.Do(func() {
		// 自动初始化DLP脱敏引擎
		if h.opts.formatters != nil {
			if engine, err := dlp.DlpInit("slog.caller"); err == nil {
				dlpEngine = engine
			}
		}
	})

	if v, ok := ctx.Value(fields).(*sync.Map); ok {
		v.Range(func(key, val any) bool {
			if keyString, ok := key.(string); ok {
				// r.AddAttrs(h.transformAttr(h.groups, slog.Any(keyString, val)))
				r.AddAttrs(slog.Any(keyString, val))
			}
			return true
		})
	}

	prefixes := h.prefixes

	if r.NumAttrs() > 0 {
		nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		attrs := make([]slog.Attr, 0, r.NumAttrs())
		r.Attrs(func(attr slog.Attr) bool {
			nr.AddAttrs(h.transformAttr(h.groups, attr))
			attrs = append(attrs, attr)
			return true
		})
		if p, changed := h.extractPrefixes(attrs); changed {
			nr.AddAttrs(attrs...)
			prefixes = p
		}
		r = nr
	}

	if dlpEngine != nil {
		r.Message = DlpMask(r.Message)
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
	attrs = h.transformAttrs(h.groups, attrs)
	p, _ := h.extractPrefixes(attrs)
	return &eHandler{
		handler:  h.handler.WithAttrs(attrs),
		groups:   h.groups,
		opts:     h.opts,
		prefixes: p,
	}
}

func (h *eHandler) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return h
	}

	return &eHandler{
		groups:   append(h.groups, name),
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
		idx := slices.IndexFunc(h.opts.PrefixKeys, func(s string) bool {
			return s == attr.Key
		})
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

	for _, formatter := range h.opts.formatters {
		if v, ok := formatter(groups, attr); ok {
			attr.Value = v
		}
	}

	return attr
}

// DlpMask 脱敏传入的字符串并且返回脱敏后的结果,这里用godlp实现，所有的识别及脱敏算法全都用godlp的开源内容，当然也可以自己写或者扩展
func DlpMask(inStr string, model ...string) (outStr string) {
	if dlpEngine == nil {
		return inStr
	}
	var err error
	if len(model) > 0 {
		outStr, err = dlpEngine.Mask(inStr, model[0])
		if err != nil {
			outStr = inStr
		}
		return
	}

	outStr, _, err = dlpEngine.Deidentify(inStr)
	if err != nil {
		outStr = inStr
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
