package syslog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"

	"github.com/darkit/slog/internal/common"
	"github.com/darkit/slog/modules"
)

const ceePrefix = "@cee: "

var errNilWriter = errors.New("syslog: writer cannot be nil")

type Option struct {
	// log level (default: info)
	Level slog.Leveler

	// connection target writer
	Writer io.Writer

	// optional: fetch attributes from context
	AttrFromContext []func(ctx context.Context) []slog.Attr

	// optional: codec
	Codec Codec
}

func NewSyslogHandler(w io.Writer, o *Option) slog.Handler {
	if o == nil {
		o = &Option{}
	}
	if o.Level == nil {
		o.Level = slog.LevelInfo
	}
	o.Writer = w
	o.AttrFromContext = modules.DefaultAttrFromContext(o.AttrFromContext)
	if o.Codec == nil {
		codec, _ := GetCodec("default")
		o.Codec = codec
	}

	return &SyslogHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

var _ slog.Handler = (*SyslogHandler)(nil)

type SyslogHandler struct {
	option *Option
	attrs  []slog.Attr
	groups []string
}

func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *SyslogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.option.Writer == nil {
		return errNilWriter
	}

	fromContext := common.ContextExtractor(ctx, h.option.AttrFromContext)
	allAttrs := append(slices.Clone(h.attrs), fromContext...)
	payload, err := h.option.Codec.Encode(ctx, &record, allAttrs, h.groups)
	if err != nil {
		return err
	}

	modules.RunAsync("syslog", func() error {
		_, err := h.option.Writer.Write(append([]byte(ceePrefix), payload...))
		return err
	})
	return nil
}

func (h *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SyslogHandler{
		option: h.option,
		attrs:  common.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *SyslogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &SyslogHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}
