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

	// 配置webhook选项
	if endpoint, ok := config["endpoint"].(string); ok {
		w.option.Endpoint = endpoint
	}

	if timeout, ok := config["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			w.option.Timeout = d
		}
	} else {
		w.option.Timeout = 10 * time.Second
	}

	if level, ok := config["level"].(string); ok {
		switch level {
		case "debug":
			w.option.Level = slog.LevelDebug
		case "info":
			w.option.Level = slog.LevelInfo
		case "warn":
			w.option.Level = slog.LevelWarn
		case "error":
			w.option.Level = slog.LevelError
		default:
			w.option.Level = slog.LevelDebug
		}
	} else {
		w.option.Level = slog.LevelDebug
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
