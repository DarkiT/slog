package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/modules"
)

// FormatterFunc 内部格式化器接口，避免直接依赖formatter包
type FormatterFunc func(groups []string, attr Attr) (Value, bool)

// RegisterFormatter 在运行时注册新的格式化函数，返回可用于移除的 ID。
func RegisterFormatter(name string, fn FormatterFunc) string {
	if ext != nil {
		return ext.registerFormatterInternal(name, fn)
	}
	return ""
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
		return
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
	dlpEngine         *dlp.DlpEngine
	moduleRegistry    *modules.Registry
	registeredModules []modules.Module
	formatterMu       sync.RWMutex
	modulesMu         sync.RWMutex
	dlpMu             sync.Mutex // 保护 dlpEngine 初始化
	nextFormatterID   atomic.Int64
	formatterCount    atomic.Int64
	dlpEnabled        atomic.Bool
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
	if e == nil {
		return
	}
	e.dlpMu.Lock()
	if e.dlpEngine == nil {
		e.dlpEngine = dlp.NewDlpEngine()
	}
	e.dlpMu.Unlock()
	if e.dlpEngine != nil {
		e.dlpEngine.Enable()
	}
	e.dlpEnabled.Store(true)
}

// disableDLP 禁用日志脱敏功能
func (e *extensions) disableDLP() {
	e.dlpEnabled.Store(false)
	if e.dlpEngine != nil {
		e.dlpEngine.Disable()
	}
}

// registerModule 注册模块到扩展系统
func (e *extensions) registerModule(module modules.Module) error {
	if e == nil || module == nil {
		return fmt.Errorf("invalid extensions or module")
	}

	// 在锁内初始化 moduleRegistry（一次性操作）
	e.modulesMu.Lock()
	if e.moduleRegistry == nil {
		e.moduleRegistry = modules.GetRegistry()
	}
	if e.moduleIndex == nil {
		e.moduleIndex = make(map[string]modules.Module)
	}
	e.modulesMu.Unlock()

	// 将模块添加到注册表
	if err := e.moduleRegistry.Register(module); err != nil {
		return err
	}

	// 添加到本地模块列表
	e.modulesMu.Lock()
	e.registeredModules = append(e.registeredModules, module)
	e.moduleIndex[module.Name()] = module
	e.modulesMu.Unlock()

	// 根据模块类型进行相应的处理
	switch module.Type() {
	case modules.TypeFormatter:
		if provider, ok := module.(modules.FormatterProvider); ok {
			e.addFormatterFuncs(provider.FormatterFunctions())
		} else if adapter, ok := module.(interface{ GetFormatters() any }); ok {
			if formatters := adapter.GetFormatters(); formatters != nil {
				e.addFormattersFromModule(formatters)
			}
		}
	}

	return nil
}

func (e *extensions) addFormatterFuncs(funcs []func([]string, slog.Attr) (slog.Value, bool)) {
	for _, f := range funcs {
		if f == nil {
			continue
		}
		e.registerFormatterInternal("module", FormatterFunc(f))
	}
}

func (e *extensions) addFormattersFromModule(formatters any) {
	switch v := formatters.(type) {
	case []func([]string, slog.Attr) (slog.Value, bool):
		e.addFormatterFuncs(v)
	case []any:
		for _, item := range v {
			if f, ok := item.(func([]string, slog.Attr) (slog.Value, bool)); ok {
				e.addFormatterFuncs([]func([]string, slog.Attr) (slog.Value, bool){f})
			}
		}
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
	e.formatterCount.Add(1)
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
			e.formatterCount.Add(-1)
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

func (e *extensions) applyFormatters(groups []string, attr slog.Attr) slog.Attr {
	if e == nil || e.formatterCount.Load() == 0 {
		return attr
	}
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

// hasAttrTransformers 判断当前扩展链是否会改写属性值。
func (e *extensions) hasAttrTransformers() bool {
	if e == nil {
		return false
	}
	if e.dlpEnabled.Load() && e.dlpEngine != nil {
		return true
	}
	return e.formatterCount.Load() > 0
}

// hasMessageTransformer 判断消息正文是否需要在扩展链中被改写。
func (e *extensions) hasMessageTransformer() bool {
	return e != nil && e.dlpEnabled.Load() && e.dlpEngine != nil
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

func (e *extensions) transformMessage(msg string) string {
	if e == nil || !e.dlpEnabled.Load() || e.dlpEngine == nil || msg == "" {
		return msg
	}
	// 快速路径：短消息通常不包含敏感信息，跳过脱敏以提高性能
	if len(msg) < 8 {
		return msg
	}
	return e.dlpEngine.DesensitizeText(msg)
}

// eHandler 是一个自定义的 slog 处理器，用于在日志消息前添加前缀，并将其传递给下一个处理器。
// 前缀从日志记录的属性中获取，使用 prefixKeys 中指定的键。
type eHandler struct {
	handler     slog.Handler // 链中的下一个日志处理器。
	opts        *extensions  // 此处理器的配置选项。
	prefixes    []slog.Value // 前缀值的缓存列表。
	groups      []string
	observerOps []observerOperation
	ctx         context.Context
}

type observerOpKind uint8

const (
	observerOpAttrs observerOpKind = iota
	observerOpGroup
)

type observerOperation struct {
	kind  observerOpKind
	group string
	attrs []slog.Attr
}

// newAddonsHandler 创建一个新的前缀日志处理器。
// 新处理器会在将每条日志消息传递给下一个处理器之前，从日志记录的属性中获取前缀并添加到消息前。
// newAddonsHandler 创建新的处理器实例
func newAddonsHandler(next slog.Handler, opts *extensions) *eHandler {
	if opts == nil {
		opts = ext
	}
	if existing, ok := next.(*eHandler); ok {
		return &eHandler{
			handler:     existing.handler,
			opts:        opts,
			prefixes:    slices.Clone(existing.prefixes),
			groups:      slices.Clone(existing.groups),
			observerOps: cloneObserverOperations(existing.observerOps),
			ctx:         existing.ctx,
		}
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
			handler:     eh.handler,
			opts:        eh.opts,
			groups:      slices.Clone(eh.groups),
			prefixes:    slices.Clone(eh.prefixes),
			observerOps: cloneObserverOperations(eh.observerOps),
			ctx:         ctx,
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
	return h.handler.Handle(ctx, h.prepareRecord(ctx, r))
}

// WithAttrs 方法，正确使用模块值
func (h *eHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 延迟到 Handle 阶段再把 attrs 注入 Record，避免每次 With 都重建整棵 handler 树。
	newHandler := &eHandler{
		handler:     h.handler,
		opts:        h.opts,
		groups:      slices.Clone(h.groups),
		prefixes:    slices.Clone(h.prefixes), // 复制现有前缀
		observerOps: append(cloneObserverOperations(h.observerOps), observerOperation{kind: observerOpAttrs, attrs: slices.Clone(attrs)}),
		ctx:         h.ctx,
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
		handler:     h.handler,
		opts:        h.opts,
		groups:      append(slices.Clone(h.groups), name),
		prefixes:    slices.Clone(h.prefixes),
		observerOps: append(cloneObserverOperations(h.observerOps), observerOperation{kind: observerOpGroup, group: name}),
		ctx:         h.ctx,
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

	if h.opts == nil {
		return attr
	}

	// 应用所有formatters
	before := attr
	attr = h.opts.applyFormatters(groups, attr)
	h.opts.emitDiagnostics("formatter", groups, before, attr)

	// DLP处理
	if h.opts.dlpEnabled.Load() && h.opts.dlpEngine != nil {
		before := attr
		switch attr.Value.Kind() {
		case slog.KindString:
			attr.Value = slog.StringValue(desensitizeAttrValue(h.opts.dlpEngine, attr.Key, attr.Value.String()))
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

func desensitizeAttrValue(engine *dlp.DlpEngine, key string, value string) string {
	if engine == nil || value == "" {
		return value
	}

	switch strings.ToLower(key) {
	case "phone", "mobile", "mobile_phone", "telephone":
		return engine.DesensitizeSpecificType(value, "mobile_phone")
	case "email", "email_address", "mail":
		return engine.DesensitizeSpecificType(value, "email")
	case "id_card", "idcard", "identity_card":
		return engine.DesensitizeSpecificType(value, "id_card")
	case "bank_card", "bankcard":
		return engine.DesensitizeSpecificType(value, "bank_card")
	case "ipv4", "ipv6", "ip", "ip_address":
		return engine.DesensitizeSpecificType(value, "ipv4")
	case "imei":
		return engine.DesensitizeSpecificType(value, "imei")
	case "url", "uri":
		return engine.DesensitizeSpecificType(value, "url")
	case "domain", "hostname":
		return engine.DesensitizeSpecificType(value, "domain")
	case "jwt":
		return engine.DesensitizeSpecificType(value, "jwt")
	case "access_token", "token":
		return engine.DesensitizeSpecificType(value, "access_token")
	}

	return engine.DesensitizeText(value)
}

// prepareRecord 根据当前扩展状态选择最轻的记录整理路径。
func (h *eHandler) prepareRecord(ctx context.Context, r slog.Record) slog.Record {
	if len(h.observerOps) > 0 {
		return h.normalizedObserverRecord(ctx, r)
	}
	if h.canPassThrough(ctx) {
		return r
	}
	return h.normalizeRuntimeRecord(ctx, r)
}

func (h *eHandler) canPassThrough(ctx context.Context) bool {
	if h == nil {
		return true
	}
	if h.hasPrefix() {
		return false
	}
	if h.opts != nil && h.opts.hasMessageTransformer() {
		return false
	}
	if h.opts != nil && h.opts.hasAttrTransformers() {
		return false
	}
	if ctx == nil {
		ctx = h.ctx
	}
	if hasContextFields(getFields(ctx)) {
		return false
	}
	return currentContextPropagator() == nil
}

func (h *eHandler) hasPrefix() bool {
	for _, prefix := range h.prefixes {
		if prefix.Any() != nil {
			return true
		}
	}
	return false
}

func (h *eHandler) normalizeRuntimeRecord(ctx context.Context, r slog.Record) slog.Record {
	nr := slog.NewRecord(r.Time, r.Level, h.prefixedMessage(r.Message), r.PC)
	for _, attr := range h.runtimeAttrs(ctx, r, h.groups) {
		nr.AddAttrs(attr)
	}
	return nr
}

func (h *eHandler) normalizedObserverRecord(ctx context.Context, r slog.Record) slog.Record {
	nr := slog.NewRecord(r.Time, r.Level, h.prefixedMessage(r.Message), r.PC)
	currentGroups := make([]string, 0, len(h.groups))
	for _, op := range h.observerOps {
		switch op.kind {
		case observerOpGroup:
			currentGroups = append(currentGroups, op.group)
		case observerOpAttrs:
			attrs := h.transformAttrs(currentGroups, slices.Clone(op.attrs))
			nr.AddAttrs(wrapAttrsWithGroups(currentGroups, attrs)...)
		}
	}
	runtimeAttrs := h.runtimeAttrs(ctx, r, currentGroups)
	nr.AddAttrs(wrapAttrsWithGroups(currentGroups, runtimeAttrs)...)
	return nr
}

func (h *eHandler) prefixedMessage(msg string) string {
	if len(h.prefixes) > 0 && h.prefixes[0].Any() != nil {
		msg = "[" + h.prefixes[0].String() + "] " + msg
	}
	if h.opts != nil {
		return h.opts.transformMessage(msg)
	}
	return msg
}

func (h *eHandler) runtimeAttrs(ctx context.Context, r slog.Record, groups []string) []slog.Attr {
	if ctx == nil {
		ctx = h.ctx
	}

	fields := getFields(ctx)
	propagator := currentContextPropagator()

	if !hasContextFields(fields) && propagator == nil {
		return h.appendRecordAttrs(nil, r, groups)
	}

	attrs := make([]slog.Attr, 0, 8)
	if hasContextFields(fields) {
		seen := make(map[string]struct{}, 8)
		r.Attrs(func(attr slog.Attr) bool {
			seen[attr.Key] = struct{}{}
			return true
		})

		fields.mu.RLock()
		for key, val := range fields.values {
			if _, exists := seen[key]; !exists && key != "$module" {
				attrs = append(attrs, h.transformAttr(groups, slog.Any(key, val)))
			}
		}
		fields.mu.RUnlock()
	}

	if propagator != nil {
		if propagated := propagator(ctx); len(propagated) > 0 {
			for _, attr := range propagated {
				attrs = append(attrs, h.transformAttr(groups, attr))
			}
		}
	}

	return h.appendRecordAttrs(attrs, r, groups)
}

func hasContextFields(fields *Fields) bool {
	if fields == nil {
		return false
	}
	fields.mu.RLock()
	defer fields.mu.RUnlock()
	return len(fields.values) > 0
}

func (h *eHandler) appendRecordAttrs(dst []slog.Attr, r slog.Record, groups []string) []slog.Attr {
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "$module" {
			dst = append(dst, h.transformAttr(groups, attr))
		}
		return true
	})
	return dst
}

func cloneObserverOperations(ops []observerOperation) []observerOperation {
	if len(ops) == 0 {
		return nil
	}
	cloned := make([]observerOperation, len(ops))
	for i, op := range ops {
		cloned[i] = observerOperation{
			kind:  op.kind,
			group: op.group,
			attrs: slices.Clone(op.attrs),
		}
	}
	return cloned
}

func wrapAttrsWithGroups(groups []string, attrs []slog.Attr) []slog.Attr {
	if len(groups) == 0 || len(attrs) == 0 {
		return attrs
	}

	wrapped := slices.Clone(attrs)
	for i := len(groups) - 1; i >= 0; i-- {
		wrapped = []slog.Attr{{Key: groups[i], Value: slog.GroupValue(wrapped...)}}
	}
	return wrapped
}
