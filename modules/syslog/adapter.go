package syslog

import (
	"log/slog"
	"net"

	"github.com/darkit/slog/modules"
)

// SyslogAdapter Syslog模块适配器
type SyslogAdapter struct {
	*modules.BaseModule
	option *Option
	conn   net.Conn
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
	}

	if err := config.Bind(&cfg); err != nil {
		return err
	}

	// 配置syslog选项
	if cfg.Network != "" && cfg.Addr != "" {
		if conn, err := net.Dial(cfg.Network, cfg.Addr); err == nil {
			s.conn = conn
			s.option.Writer = conn
		}
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

	// 创建处理器
	if s.option.Writer != nil {
		s.SetHandler(NewSyslogHandler(s.option.Writer, s.option))
	}
	return nil
}

// init 注册syslog模块工厂
func init() {
	modules.RegisterFactory("syslog", func(config modules.Config) (modules.Module, error) {
		adapter := NewSyslogAdapter()
		return adapter, adapter.Configure(config)
	})
}
