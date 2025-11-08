package slog

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/darkit/slog/modules"
)

type RecordRouter func(record slog.Record) []string

var recordRouter atomic.Value

// SetRecordRouter 自定义模块路由策略（返回模块名称列表）。
func SetRecordRouter(router RecordRouter) {
	recordRouter.Store(router)
}

// ModuleBuilder 模块构建器，与现有系统集成
type ModuleBuilder struct {
	logger         *Logger
	pendingModules []modules.Module
}

// NewModuleBuilder 创建新的模块构建器
func NewModuleBuilder(logger *Logger) *ModuleBuilder {
	return &ModuleBuilder{
		logger:         logger,
		pendingModules: make([]modules.Module, 0),
	}
}

// UseModule 添加模块实例
func (mb *ModuleBuilder) UseModule(module modules.Module) *ModuleBuilder {
	if module.Enabled() {
		mb.pendingModules = append(mb.pendingModules, module)
	}
	return mb
}

// UseFactory 通过工厂创建并使用模块
func (mb *ModuleBuilder) UseFactory(name string, config modules.Config) *ModuleBuilder {
	if module, err := modules.CreateModule(name, config); err == nil {
		return mb.UseModule(module)
	}
	return mb
}

// UseConfig 使用配置批量创建模块
func (mb *ModuleBuilder) UseConfig(configs []modules.ModuleConfig) *ModuleBuilder {
	for _, cfg := range configs {
		if cfg.Enabled {
			if module, err := modules.CreateModule(cfg.Type, cfg.Config); err == nil {
				mb.UseModule(module)
			}
		}
	}
	return mb
}

// Build 构建并应用所有模块
func (mb *ModuleBuilder) Build() *Logger {
	// 将模块注册到扩展系统
	for _, module := range mb.pendingModules {
		ext.registerModule(module)
	}

	// 返回原logger，因为扩展已经应用到全局ext中
	return mb.logger
}

// Logger扩展方法

// Use 为Logger添加模块实例
func (l *Logger) Use(module modules.Module) *ModuleBuilder {
	return NewModuleBuilder(l).UseModule(module)
}

// UseFactory 为Logger添加通过工厂创建的模块
func (l *Logger) UseFactory(name string, config modules.Config) *ModuleBuilder {
	return NewModuleBuilder(l).UseFactory(name, config)
}

// UseConfig 为Logger添加批量配置模块
func (l *Logger) UseConfig(configs []modules.ModuleConfig) *ModuleBuilder {
	return NewModuleBuilder(l).UseConfig(configs)
}

// WithModules 便捷方法：一次性添加多个模块
func (l *Logger) WithModules(modules ...modules.Module) *ModuleBuilder {
	builder := NewModuleBuilder(l)
	for _, module := range modules {
		builder.UseModule(module)
	}
	return builder
}

// 全局便捷方法

// UseModule 全局方法：使用模块实例
func UseModule(module modules.Module) *ModuleBuilder {
	return GetGlobalLogger().Use(module)
}

// UseFactory 全局方法：通过工厂创建并使用模块
func UseFactory(name string, config modules.Config) *ModuleBuilder {
	return GetGlobalLogger().UseFactory(name, config)
}

// UseConfig 全局方法：使用配置批量创建模块
func UseConfig(configs []modules.ModuleConfig) *ModuleBuilder {
	return GetGlobalLogger().UseConfig(configs)
}

// UpdateModuleConfig 热更新已注册模块的配置
func UpdateModuleConfig(name string, config modules.Config) error {
	return modules.UpdateModuleConfig(name, config)
}

// RegisteredModules 返回当前已注册的模块名称。
func RegisteredModules() []string {
	if ext == nil {
		return nil
	}
	mods := ext.snapshotModules()
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		names = append(names, m.Name())
	}
	return names
}

// 便捷方法：通过工厂创建具体模块

// EnableWebhook 全局方法：启用Webhook模块
func EnableWebhook(endpoint string, options ...modules.Config) *ModuleBuilder {
	config := modules.Config{"endpoint": endpoint}
	if len(options) > 0 {
		for k, v := range options[0] {
			config[k] = v
		}
	}
	return UseFactory("webhook", config)
}

// EnableSyslog 全局方法：启用Syslog模块
func EnableSyslog(network, addr string, options ...modules.Config) *ModuleBuilder {
	config := modules.Config{
		"network": network,
		"addr":    addr,
	}
	if len(options) > 0 {
		for k, v := range options[0] {
			config[k] = v
		}
	}
	return UseFactory("syslog", config)
}

// EnableFormatter 全局方法：启用格式化器模块
func EnableFormatter(formatterType string, options ...modules.Config) *ModuleBuilder {
	config := modules.Config{"type": formatterType}
	if len(options) > 0 {
		for k, v := range options[0] {
			config[k] = v
		}
	}
	return UseFactory("formatter", config)
}

// EnableMulti 全局方法：启用多处理器模块
func EnableMulti(strategy string, options ...modules.Config) *ModuleBuilder {
	config := modules.Config{"strategy": strategy}
	if len(options) > 0 {
		for k, v := range options[0] {
			config[k] = v
		}
	}
	return UseFactory("multi", config)
}

// 辅助函数：从模块中获取处理器并应用到现有系统

// ApplyModulesToHandler 将模块处理器应用到基础处理器上
func ApplyModulesToHandler(baseHandler slog.Handler, moduleList []modules.Module) slog.Handler {
	router, _ := recordRouter.Load().(RecordRouter)
	moduleHandlers := make(map[string]slog.Handler)
	handlers := []slog.Handler{baseHandler}

	for _, module := range moduleList {
		if !module.Enabled() {
			continue
		}
		h := module.Handler()
		if h == nil {
			continue
		}
		moduleHandlers[module.Name()] = h
		if router == nil {
			handlers = append(handlers, h)
		}
	}

	if router == nil {
		return NewFanoutHandler(handlers...)
	}
	return newRoutingHandler(baseHandler, moduleHandlers, router)
}

// ChainHandler 链式处理器
type ChainHandler struct {
	middleware slog.Handler
	next       slog.Handler
}

func NewChainHandler(middleware, next slog.Handler) *ChainHandler {
	return &ChainHandler{
		middleware: middleware,
		next:       next,
	}
}

func (h *ChainHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.middleware.Enabled(ctx, level) || h.next.Enabled(ctx, level)
}

func (h *ChainHandler) Handle(ctx context.Context, r slog.Record) error {
	// 先通过中间件处理
	if err := h.middleware.Handle(ctx, r); err != nil {
		return err
	}
	// 再传递给下一个处理器
	return h.next.Handle(ctx, r)
}

func (h *ChainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewChainHandler(
		h.middleware.WithAttrs(attrs),
		h.next.WithAttrs(attrs),
	)
}

func (h *ChainHandler) WithGroup(name string) slog.Handler {
	return NewChainHandler(
		h.middleware.WithGroup(name),
		h.next.WithGroup(name),
	)
}

// FanoutHandler 扇出处理器
type FanoutHandler struct {
	handlers []slog.Handler
}

func NewFanoutHandler(handlers ...slog.Handler) *FanoutHandler {
	return &FanoutHandler{handlers: handlers}
}

func (h *FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *FanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			// 克隆记录以避免并发问题
			go handler.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (h *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return NewFanoutHandler(newHandlers...)
}

func (h *FanoutHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return NewFanoutHandler(newHandlers...)
}

type routingHandler struct {
	base    slog.Handler
	modules map[string]slog.Handler
	router  RecordRouter
}

func newRoutingHandler(base slog.Handler, modules map[string]slog.Handler, router RecordRouter) slog.Handler {
	return &routingHandler{
		base:    base,
		modules: modules,
		router:  router,
	}
}

func (h *routingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *routingHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.base.Handle(ctx, r); err != nil {
		return err
	}
	if h.router == nil {
		return nil
	}
	targets := h.router(r)
	if len(targets) == 0 {
		return nil
	}
	for _, name := range targets {
		handler, ok := h.modules[name]
		if !ok || handler == nil {
			continue
		}
		if handler.Enabled(ctx, r.Level) {
			go handler.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (h *routingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newRoutingHandler(
		h.base.WithAttrs(attrs),
		cloneHandlerMap(h.modules, func(handler slog.Handler) slog.Handler { return handler.WithAttrs(attrs) }),
		h.router,
	)
}

func (h *routingHandler) WithGroup(name string) slog.Handler {
	return newRoutingHandler(
		h.base.WithGroup(name),
		cloneHandlerMap(h.modules, func(handler slog.Handler) slog.Handler { return handler.WithGroup(name) }),
		h.router,
	)
}

func cloneHandlerMap(src map[string]slog.Handler, transform func(slog.Handler) slog.Handler) map[string]slog.Handler {
	if src == nil {
		return nil
	}
	dst := make(map[string]slog.Handler, len(src))
	for k, v := range src {
		if transform != nil {
			dst[k] = transform(v)
		} else {
			dst[k] = v
		}
	}
	return dst
}
