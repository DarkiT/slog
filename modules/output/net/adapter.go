package outputnet

import (
	"log/slog"
	"time"

	"github.com/darkit/slog/modules"
)

// NetAdapter sends records to generic TCP/UDP endpoints.
type NetAdapter struct {
	*modules.BaseModule

	option *SenderOption
	sender *Sender
}

func NewNetAdapter() *NetAdapter {
	return &NetAdapter{
		BaseModule: modules.NewBaseModule("output.net", modules.TypeSink, 100),
		option:     &SenderOption{},
	}
}

func (a *NetAdapter) Configure(config modules.Config) error {
	if err := a.BaseModule.Configure(config); err != nil {
		return err
	}

	var cfg struct {
		Network      string `json:"network"`
		Addr         string `json:"addr"`
		Level        string `json:"level"`
		Codec        string `json:"codec"`
		DialTimeout  string `json:"dial_timeout"`
		WriteTimeout string `json:"write_timeout"`
		Delimiter    string `json:"delimiter"`
	}
	if err := config.Bind(&cfg); err != nil {
		return err
	}

	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.DialTimeout == "" {
		cfg.DialTimeout = "3s"
	}
	if cfg.WriteTimeout == "" {
		cfg.WriteTimeout = "3s"
	}
	if cfg.Delimiter == "" {
		cfg.Delimiter = "\n"
	}

	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		dialTimeout = 3 * time.Second
	}
	writeTimeout, err := time.ParseDuration(cfg.WriteTimeout)
	if err != nil {
		writeTimeout = 3 * time.Second
	}

	a.option = &SenderOption{
		Network:      cfg.Network,
		Addr:         cfg.Addr,
		DialTimeout:  dialTimeout,
		WriteTimeout: writeTimeout,
		Delimiter:    []byte(cfg.Delimiter),
	}
	a.sender = NewSender(*a.option)

	level := parseLevel(cfg.Level)
	codec, ok := GetCodec(cfg.Codec)
	if !ok {
		return errInvalidCodec
	}
	handler := NewRawHandlerWithOption(RawOption{
		Level:  level,
		Writer: a.sender,
		Codec:  codec,
	})
	a.SetHandler(handler)
	return nil
}

func parseLevel(level string) slog.Leveler {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

func init() {
	if err := modules.RegisterFactory("output.net", func(config modules.Config) (modules.Module, error) {
		adapter := NewNetAdapter()
		return adapter, adapter.Configure(config)
	}); err != nil {
		modules.ReportAsyncError("registry.output.net", err)
	}
}
