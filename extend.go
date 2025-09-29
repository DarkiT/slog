package slog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/modules"
)

// FormatterFunc 内部格式化器接口，避免直接依赖formatter包
type FormatterFunc func(groups []string, attr slog.Attr) (slog.Value, bool)

type extensions struct {
	prefixKeys        []string
	formatters        []FormatterFunc
	dlpEnabled        bool
	dlpEngine         *dlp.DlpEngine
	moduleRegistry    *modules.Registry
	registeredModules []modules.Module
}

// enableDLP 启用日志脱敏功能
func (e *extensions) enableDLP() {
	e.dlpEnabled = true
	if e.dlpEngine == nil {
		e.dlpEngine = dlp.NewDlpEngine()
	}
	e.dlpEngine.Enable()
}

// disableDLP 禁用日志脱敏功能
func (e *extensions) disableDLP() {
	e.dlpEnabled = false
	if e.dlpEngine != nil {
		e.dlpEngine.Disable()
	}
}

// registerModule 注册模块到扩展系统
func (e *extensions) registerModule(module modules.Module) error {
	if e.moduleRegistry == nil {
		e.moduleRegistry = modules.GetRegistry()
	}

	// 将模块添加到注册表
	if err := e.moduleRegistry.Register(module); err != nil {
		return err
	}

	// 添加到本地模块列表
	e.registeredModules = append(e.registeredModules, module)

	// 根据模块类型进行相应的处理
	switch module.Type() {
	case modules.TypeFormatter:
		// 如果是格式化器模块，通过反射提取其格式化器并转换为内部接口
		if adapter, ok := module.(interface{ GetFormatters() interface{} }); ok {
			if formatters := adapter.GetFormatters(); formatters != nil {
				// 使用反射来转换格式化器函数
				e.addFormattersFromModule(formatters)
			}
		}
	}

	return nil
}

// addFormattersFromModule 从模块中添加格式化器（通过反射处理类型转换）
func (e *extensions) addFormattersFromModule(formatters interface{}) {
	// 这里使用类型断言来处理不同类型的格式化器
	// 由于我们无法直接导入formatter包，我们使用interface{}并在运行时处理

	// 如果适配器提供了正确的格式化器函数，我们将其转换为内部格式
	// 这是一个简化的实现，实际的格式化器转换会在适配器中处理
	switch v := formatters.(type) {
	case []func([]string, slog.Attr) (slog.Value, bool):
		// 直接兼容的格式化器函数
		for _, f := range v {
			e.formatters = append(e.formatters, FormatterFunc(f))
		}
	case []interface{}:
		// 通用接口列表，需要进一步转换
		for _, item := range v {
			if f, ok := item.(func([]string, slog.Attr) (slog.Value, bool)); ok {
				e.formatters = append(e.formatters, FormatterFunc(f))
			}
		}
	}
}

// eHandler 是一个自定义的 slog 处理器，用于在日志消息前添加前缀，并将其传递给下一个处理器。
// 前缀从日志记录的属性中获取，使用 prefixKeys 中指定的键。
type eHandler struct {
	handler  slog.Handler // 链中的下一个日志处理器。
	opts     extensions   // 此处理器的配置选项。
	prefixes []slog.Value // 前缀值的缓存列表。
	groups   []string
	ctx      context.Context
}

// newAddonsHandler 创建一个新的前缀日志处理器。
// 新处理器会在将每条日志消息传递给下一个处理器之前，从日志记录的属性中获取前缀并添加到消息前。
// newAddonsHandler 创建新的处理器实例
func newAddonsHandler(next slog.Handler, opts *extensions) *eHandler {
	if opts == nil {
		opts = ext
	}

	return &eHandler{
		handler:  next,
		opts:     *opts,
		groups:   []string{},
		prefixes: make([]slog.Value, len(opts.prefixKeys)),
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
			// 创建已存在属性的映射
			seen := make(map[string]bool)
			r.Attrs(func(attr slog.Attr) bool {
				seen[attr.Key] = true
				return true
			})

			// 从 Fields 中获取并添加属性
			fields.mu.RLock()
			for key, val := range fields.values {
				if !seen[key] && key != "$module" { // 排除 module 属性
					attr := slog.Any(key, val)
					// 应用转换（包括 DLP 处理）
					attr = h.transformAttr(h.groups, attr)
					nr.AddAttrs(attr)
				}
			}
			fields.mu.RUnlock()
		}
	}

	// 添加原始记录的属性（排除 module）
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "$module" {
			nr.AddAttrs(h.transformAttr(h.groups, attr))
		}
		return true
	})

	return h.handler.Handle(ctx, nr)
}

// WithAttrs 方法，正确使用模块值
func (h *eHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 复制处理器实例
	newHandler := &eHandler{
		handler:  h.handler.WithAttrs(h.transformAttrs(h.groups, attrs)),
		opts:     h.opts,
		groups:   slices.Clone(h.groups),
		prefixes: slices.Clone(h.prefixes), // 复制现有前缀
	}

	// 检查是否有前缀键
	for _, attr := range attrs {
		for i, key := range h.opts.prefixKeys {
			if attr.Key == key && i < len(newHandler.prefixes) {
				// 存储前缀值
				newHandler.prefixes[i] = attr.Value
			}
		}
	}

	return newHandler
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
	// 先处理LogValuer
	for attr.Value.Kind() == slog.KindLogValuer {
		attr.Value = attr.Value.LogValuer().LogValue()
	}
	// 应用所有formatters
	for _, f := range h.opts.formatters {
		if v, ok := f(groups, attr); ok {
			attr.Value = v
		}
	}

	// DLP处理
	if h.opts.dlpEnabled && h.opts.dlpEngine != nil {
		switch attr.Value.Kind() {
		case slog.KindString:
			attr.Value = slog.StringValue(h.opts.dlpEngine.DesensitizeText(attr.Value.String()))
		case slog.KindGroup:
			attrs := attr.Value.Group()
			newAttrs := make([]slog.Attr, len(attrs))
			for i, a := range attrs {
				newAttrs[i] = h.transformAttr(append(groups, attr.Key), a)
			}
			attr.Value = slog.GroupValue(newAttrs...)
		}
	}

	return attr
}
