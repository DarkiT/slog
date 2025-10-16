package webhook

import (
	"log/slog"
	"time"

	"github.com/darkit/slog/modules"
)

// WebhookAdapter Webhook模块适配器
type WebhookAdapter struct {
	*modules.BaseModule
	option *Option
}

// NewWebhookAdapter 创建Webhook适配器
func NewWebhookAdapter() *WebhookAdapter {
	return &WebhookAdapter{
		BaseModule: modules.NewBaseModule("webhook", modules.TypeSink, 100),
		option:     &Option{},
	}
}

// Configure 配置Webhook模块
func (w *WebhookAdapter) Configure(config modules.Config) error {
	if err := w.BaseModule.Configure(config); err != nil {
		return err
	}

	var cfg struct {
		Endpoint  string        `json:"endpoint"`
		Timeout   time.Duration `json:"timeout"`
		Level     string        `json:"level"`
		AddSource bool          `json:"add_source"`
	}

	if err := config.Bind(&cfg); err != nil {
		return err
	}

	if cfg.Endpoint != "" {
		w.option.Endpoint = cfg.Endpoint
	}

	if cfg.Timeout > 0 {
		w.option.Timeout = cfg.Timeout
	} else {
		w.option.Timeout = 10 * time.Second
	}

	switch cfg.Level {
	case "debug":
		w.option.Level = slog.LevelDebug
	case "info":
		w.option.Level = slog.LevelInfo
	case "warn":
		w.option.Level = slog.LevelWarn
	case "error":
		w.option.Level = slog.LevelError
	case "":
		w.option.Level = slog.LevelDebug
	default:
		w.option.Level = slog.LevelDebug
	}

	if cfg.AddSource {
		w.option.AddSource = true
	}

	// 创建处理器
	w.SetHandler(w.option.NewWebhookHandler())
	return nil
}

// init 注册webhook模块工厂
func init() {
	modules.RegisterFactory("webhook", func(config modules.Config) (modules.Module, error) {
		adapter := NewWebhookAdapter()
		return adapter, adapter.Configure(config)
	})
}
