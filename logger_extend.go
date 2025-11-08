package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/modules"
)

// FormatterFunc 内部格式化器接口，避免直接依赖formatter包
type FormatterFunc func(groups []string, attr slog.Attr) (slog.Value, bool)

// RegisterFormatter 在运行时注册新的格式化函数，返回可用于移除的 ID。
func RegisterFormatter(name string, fn FormatterFunc) string {
	if ext == nil {
		ext = &extensions{}
	}
	return ext.registerFormatterInternal(name, fn)
}

// RemoveFormatter 根据 ID 移除先前注册的格式化函数。
func RemoveFormatter(id string) bool {
	if ext == nil {
		return false
	}
	return ext.removeFormatterInternal(id)
}

// ListFormatters 返回当前激活的格式化器名称列表。
func ListFormatters() []string {
	if ext == nil {
		return nil
	}
	return ext.listFormatterNames()
}

// EnableDiagnosticsLogging 控制扩展管线的调试输出，可选自定义输出目标。
func EnableDiagnosticsLogging(on bool, writer ...io.Writer) {
	if ext == nil {
		ext = &extensions{}
	}
	if !on {
		ext.diagnostics.Store(false)
		return
	}
	if len(writer) > 0 && writer[0] != nil {
		ext.diagnosticsWriter.Store(&writer[0])
	} else if ext.diagnosticsWriter.Load() == nil {
		w := io.Writer(os.Stderr)
		ext.diagnosticsWriter.Store(&w)
	}
	ext.diagnostics.Store(true)
}

type extensions struct {
	prefixKeys        []string
	formatters        []formatterEntry
	dlpEnabled        bool
	dlpEngine         *dlp.DlpEngine
	moduleRegistry    *modules.Registry
	registeredModules []modules.Module
	formatterMu       sync.RWMutex
	modulesMu         sync.RWMutex
	nextFormatterID   atomic.Int64
	moduleIndex       map[string]modules.Module
	diagnostics       atomic.Bool
	diagnosticsWriter atomic.Pointer[io.Writer]
}

type formatterEntry struct {
	id   string
	name string
	f    FormatterFunc
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
	e.modulesMu.Lock()
	if e.moduleIndex == nil {
		e.moduleIndex = make(map[string]modules.Module)
	}
	e.registeredModules = append(e.registeredModules, module)
	e.moduleIndex[module.Name()] = module
	e.modulesMu.Unlock()

	// 根据模块类型进行相应的处理
	switch module.Type() {
	case modules.TypeFormatter:
		// 优先使用强类型接口，避免反射带来的额外开销
		if provider, ok := module.(modules.FormatterProvider); ok {
			e.addFormatterFuncs(provider.FormatterFunctions())
			break
		}
		// 向后兼容旧接口
		if adapter, ok := module.(interface{ GetFormatters() interface{} }); ok {
			if formatters := adapter.GetFormatters(); formatters != nil {
				e.addFormattersFromModule(formatters)
			}
		}
	}

	return nil
}

func (e *extensions) addFormattersFromModule(formatters interface{}) {
	switch v := formatters.(type) {
	case []func([]string, slog.Attr) (slog.Value, bool):
		e.addFormatterFuncs(v)
	case []interface{}:
		for _, item := range v {
			if f, ok := item.(func([]string, slog.Attr) (slog.Value, bool)); ok {
				e.addFormatterFuncs([]func([]string, slog.Attr) (slog.Value, bool){f})
			}
		}
	}
}

func (e *extensions) addFormatterFuncs(funcs []func([]string, slog.Attr) (slog.Value, bool)) {
	for _, f := range funcs {
		if f == nil {
			continue
		}
		e.registerFormatterInternal("module", FormatterFunc(f))
	}
}

func (e *extensions) registerFormatterInternal(name string, fn FormatterFunc) string {
	if fn == nil {
		return ""
	}
	id := fmt.Sprintf("%s-%d", name, e.nextFormatterID.Add(1))
	entry := formatterEntry{id: id, name: name, f: fn}
	e.formatterMu.Lock()
	e.formatters = append(e.formatters, entry)
	e.formatterMu.Unlock()
	return id
}

func (e *extensions) removeFormatterInternal(id string) bool {
	if id == "" {
		return false
	}
	e.formatterMu.Lock()
	defer e.formatterMu.Unlock()
	for i, entry := range e.formatters {
		if entry.id == id {
			e.formatters = append(e.formatters[:i], e.formatters[i+1:]...)
			return true
		}
	}
	return false
}

func (e *extensions) listFormatterNames() []string {
	e.formatterMu.RLock()
	defer e.formatterMu.RUnlock()
	names := make([]string, len(e.formatters))
	for i, entry := range e.formatters {
		names[i] = entry.name
	}
	return names
}

func (e *extensions) snapshotModules() []modules.Module {
	e.modulesMu.RLock()
	defer e.modulesMu.RUnlock()
	return append([]modules.Module(nil), e.registeredModules...)
}

func (e *extensions) getModule(name string) (modules.Module, bool) {
	e.modulesMu.RLock()
	defer e.modulesMu.RUnlock()
	if e.moduleIndex == nil {
		return nil, false
	}
	m, ok := e.moduleIndex[name]
	return m, ok
}

func (e *extensions) applyFormatters(groups []string, attr slog.Attr) slog.Attr {
	e.formatterMu.RLock()
	defer e.formatterMu.RUnlock()
	for _, entry := range e.formatters {
		if entry.f == nil {
			continue
		}
		if v, ok := entry.f(groups, attr); ok {
			attr.Value = v
		}
	}
	return attr
}

func (e *extensions) emitDiagnostics(stage string, groups []string, before, after slog.Attr) {
	if e == nil || !e.diagnostics.Load() || !attrChanged(before, after) {
		return
	}
	writerPtr := e.diagnosticsWriter.Load()
	if writerPtr == nil {
		w := io.Writer(os.Stderr)
		e.diagnosticsWriter.Store(&w)
		writerPtr = &w
	}
	w := *writerPtr
	if w == nil {
		return
	}
	fmt.Fprintf(w, "[slog-diagnostics] stage=%s groups=%v key=%s before=%s after=%s\n", stage, groups, after.Key, before.Value, after.Value)
}

func attrChanged(before, after slog.Attr) bool {
	if before.Key != after.Key {
		return true
	}
	return before.Value.String() != after.Value.String()
}

// eHandler 是一个自定义的 slog 处理器，用于在日志消息前添加前缀，并将其传递给下一个处理器。
// 前缀从日志记录的属性中获取，使用 prefixKeys 中指定的键。
type eHandler struct {
	handler  slog.Handler // 链中的下一个日志处理器。
	opts     *extensions  // 此处理器的配置选项。
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
		opts:     opts,
		groups:   []string{},
		prefixes: make([]slog.Value, len(opts.prefixKeys)),
	}
}

func cloneHandlerWithContext(handler slog.Handler, ctx context.Context) slog.Handler {
	if eh, ok := handler.(*eHandler); ok {
		clone := &eHandler{
			handler:  eh.handler,
			opts:     eh.opts,
			groups:   slices.Clone(eh.groups),
			prefixes: slices.Clone(eh.prefixes),
			ctx:      ctx,
		}
		return clone
	}
	return newAddonsHandler(handler, ext)
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

	if propagator := currentContextPropagator(); propagator != nil {
		if attrs := propagator(ctx); len(attrs) > 0 {
			for _, attr := range attrs {
				nr.AddAttrs(h.transformAttr(h.groups, attr))
			}
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
	if h.opts != nil {
		before := attr
		attr = h.opts.applyFormatters(groups, attr)
		h.opts.emitDiagnostics("formatter", groups, before, attr)
	}

	// DLP处理
	if h.opts.dlpEnabled && h.opts.dlpEngine != nil {
		before := attr
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
		h.opts.emitDiagnostics("dlp", groups, before, attr)
	}

	return attr
}
