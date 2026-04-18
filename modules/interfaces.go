package modules

import "log/slog"

// Named 命名接口 - 只负责提供名称。
// 与 Module.Name 保持一致。
type Named interface {
	Name() string
}

// Typed 类型接口 - 只负责提供类型信息。
// 与 Module.Type 保持一致。
type Typed interface {
	Type() ModuleType
}

// Configurable 可配置接口 - 只负责配置管理。
// 与 Module.Configure 保持一致。
type Configurable interface {
	Configure(config Config) error
}

// Enableable 可启用接口 - 只负责启用状态管理。
// 与 Module.Enabled 保持一致，并补充开关能力。
type Enableable interface {
	Enabled() bool
	Enable()
	Disable()
}

// HandlerProvider 处理器提供者接口 - 只负责提供 slog.Handler。
// 与 Module.Handler 保持一致，并补充动态替换能力。
type HandlerProvider interface {
	Handler() slog.Handler
	SetHandler(handler slog.Handler)
}

// FormatterProvider 提供格式化函数，避免使用反射适配。
type FormatterProvider interface {
	FormatterFunctions() []func([]string, slog.Attr) (slog.Value, bool)
}

// Healthable 健康检查接口 - 用于诊断聚合。
type Healthable interface {
	HealthCheck() error
	IsHealthy() bool
}

// Measurable 可度量接口 - 用于诊断聚合。
type Measurable interface {
	GetMetrics() map[string]interface{}
	ResetMetrics()
}
