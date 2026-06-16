package outputnet

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"

	"github.com/darkit/slog/modules"
)

var (
	errInvalidCodec = errors.New("outputnet: invalid codec")
	errNilWriter    = errors.New("outputnet: writer cannot be nil")
)

// RawOption controls generic output.net formatting.
type RawOption struct {
	Level  slog.Leveler
	Writer io.Writer
	Codec  Codec
}

// RawHandler is a generic network output handler with async transport.
type RawHandler struct {
	option RawOption
	attrs  []slog.Attr
	groups []string
}

func NewRawHandler(w io.Writer, level slog.Leveler) slog.Handler {
	codec, _ := GetCodec("raw")
	return NewRawHandlerWithOption(RawOption{
		Level:  level,
		Writer: w,
		Codec:  codec,
	})
}

func NewRawHandlerWithOption(opt RawOption) slog.Handler {
	if opt.Level == nil {
		opt.Level = slog.LevelInfo
	}
	if opt.Codec == nil {
		codec, _ := GetCodec("raw")
		opt.Codec = codec
	}
	return &RawHandler{
		option: opt,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

func (h *RawHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *RawHandler) Handle(_ context.Context, record slog.Record) error {
	if h.option.Writer == nil {
		return errNilWriter
	}

	payload, err := h.option.Codec.Encode(&record, h.attrs, h.groups)
	if err != nil {
		return err
	}

	modules.RunAsync("output.net", func() error {
		_, err := h.option.Writer.Write(payload)
		return err
	})

	return nil
}

func (h *RawHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nextAttrs := append(slices.Clone(h.attrs), attrs...)
	return &RawHandler{
		option: h.option,
		attrs:  nextAttrs,
		groups: slices.Clone(h.groups),
	}
}

func (h *RawHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nextGroups := append(slices.Clone(h.groups), name)
	return &RawHandler{
		option: h.option,
		attrs:  slices.Clone(h.attrs),
		groups: nextGroups,
	}
}
