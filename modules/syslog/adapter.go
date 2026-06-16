package syslog

import (
	"log/slog"
	"time"

	"github.com/darkit/slog/modules"
	outputnet "github.com/darkit/slog/modules/output/net"
)

// SyslogAdapter Syslog模块适配器
type SyslogAdapter struct {
	*modules.BaseModule
	option *Option
	sender *outputnet.Sender
}

// NewSyslogAdapter 创建Syslog适配器
func NewSyslogAdapter() *SyslogAdapter {
	return &SyslogAdapter{
		BaseModule: modules.NewBaseModule("syslog", modules.TypeSink, 100),
		option:     &Option{},
	}
}

// Configure 配置Syslog模块
func (s *SyslogAdapter) Configure(config modules.Config) error {
	if err := s.BaseModule.Configure(config); err != nil {
		return err
	}

	var cfg struct {
		Network string `json:"network"`
		Addr    string `json:"addr"`
		Level   string `json:"level"`
		Codec   string `json:"codec"`
	}

	if err := config.Bind(&cfg); err != nil {
		return err
	}

	// 配置syslog选项
	if cfg.Network != "" && cfg.Addr != "" {
		s.sender = outputnet.NewSender(outputnet.SenderOption{
			Network:      cfg.Network,
			Addr:         cfg.Addr,
			Delimiter:    []byte("\n"),
			DialTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		s.option.Writer = s.sender
	}

	switch cfg.Level {
	case "debug":
		s.option.Level = slog.LevelDebug
	case "info":
		s.option.Level = slog.LevelInfo
	case "warn":
		s.option.Level = slog.LevelWarn
	case "error":
		s.option.Level = slog.LevelError
	case "":
		s.option.Level = slog.LevelDebug
	default:
		s.option.Level = slog.LevelDebug
	}
	if codec, ok := GetCodec(cfg.Codec); ok {
		s.option.Codec = codec
	} else if cfg.Codec != "" {
		return errInvalidCodec
	}

	// 创建处理器
	if s.option.Writer != nil {
		s.SetHandler(NewSyslogHandler(s.option.Writer, s.option))
	}
	return nil
}

// init 注册syslog模块工厂
func init() {
	if err := modules.RegisterFactory("syslog", func(config modules.Config) (modules.Module, error) {
		adapter := NewSyslogAdapter()
		return adapter, adapter.Configure(config)
	}); err != nil {
		modules.ReportAsyncError("registry.syslog", err)
	}
}
