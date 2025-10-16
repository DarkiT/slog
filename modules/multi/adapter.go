package multi

import (
	"log/slog"

	"github.com/darkit/slog/modules"
)

// MultiAdapter Multi模块适配器
type MultiAdapter struct {
	*modules.BaseModule
	strategy string
	handlers []slog.Handler
}

// NewMultiAdapter 创建Multi适配器
func NewMultiAdapter() *MultiAdapter {
	return &MultiAdapter{
		BaseModule: modules.NewBaseModule("multi", modules.TypeHandler, 50),
		handlers:   make([]slog.Handler, 0),
	}
}

// Configure 配置Multi模块
func (m *MultiAdapter) Configure(config modules.Config) error {
	if err := m.BaseModule.Configure(config); err != nil {
		return err
	}

	var cfg struct {
		Strategy string `json:"strategy"`
	}

	if err := config.Bind(&cfg); err != nil {
		return err
	}

	if cfg.Strategy != "" {
		m.strategy = cfg.Strategy
	} else {
		m.strategy = "fanout" // 默认策略
	}

	// 根据策略创建处理器
	switch m.strategy {
	case "fanout":
		if len(m.handlers) > 0 {
			m.SetHandler(Fanout(m.handlers...))
		}
		// 其他策略暂时不实现，避免API复杂性
	}

	return nil
}

// AddHandler 添加处理器
func (m *MultiAdapter) AddHandler(handler slog.Handler) {
	m.handlers = append(m.handlers, handler)
	// 根据当前策略重新创建处理器
	switch m.strategy {
	case "fanout":
		if len(m.handlers) > 0 {
			m.SetHandler(Fanout(m.handlers...))
		}
	}
}

// GetHandlers 获取处理器列表
func (m *MultiAdapter) GetHandlers() []slog.Handler {
	return m.handlers
}

// init 注册multi模块工厂
func init() {
	modules.RegisterFactory("multi", func(config modules.Config) (modules.Module, error) {
		adapter := NewMultiAdapter()
		return adapter, adapter.Configure(config)
	})
}
