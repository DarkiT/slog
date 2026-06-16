package slog

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/darkit/slog/modules"
	"github.com/darkit/slog/modules/multi"
)

// RecordRouter 定义模块路由策略，返回要接收当前记录的模块名列表。
type RecordRouter func(record Record) []string

var recordRouter atomic.Value

// SetRecordRouter 自定义模块路由策略。
func SetRecordRouter(router RecordRouter) {
	recordRouter.Store(router)
}

// Use 为 Logger 添加模块实例。
func (l *Logger) Use(module modules.Module) *Logger {
	_ = l.UseWithError(module)
	return l
}

// UseWithError 为 Logger 添加模块实例，并向调用方返回注册错误。
func (l *Logger) UseWithError(module modules.Module) error {
	if module == nil || !module.Enabled() {
		return nil
	}
	if ext == nil {
		return errors.New("slog extensions are not initialized")
	}
	return ext.registerModule(module)
}

// WithModules 便捷添加多个模块。
func (l *Logger) WithModules(modules ...modules.Module) *Logger {
	for _, module := range modules {
		l.Use(module)
	}
	return l
}

// UseModule 全局方法：使用模块实例。
func UseModule(module modules.Module) *Logger {
	return GetGlobalLogger().Use(module)
}

// UseModuleWithError 全局注册模块，并返回注册错误，便于第三方模块接入时显式处理失败。
func UseModuleWithError(module modules.Module) error {
	return GetGlobalLogger().UseWithError(module)
}

// UpdateModuleConfig 热更新已注册模块的配置。
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

// ApplyModulesToHandler 将模块处理器应用到基础处理器上。
func ApplyModulesToHandler(baseHandler Handler, moduleList []modules.Module) Handler {
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
		return multi.Fanout(handlers...)
	}
	return newRoutingHandler(baseHandler, moduleHandlers, router)
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
	var errs []error
	if err := h.base.Handle(ctx, r); err != nil {
		errs = append(errs, err)
	}
	if h.router == nil {
		return errors.Join(errs...)
	}
	targets := h.router(r)
	if len(targets) == 0 {
		return errors.Join(errs...)
	}
	for _, name := range targets {
		handler, ok := h.modules[name]
		if !ok || handler == nil {
			continue
		}
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
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
