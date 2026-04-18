package webhook

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/darkit/slog/internal/common"
	"github.com/darkit/slog/modules"
)

type Option struct {
	// log level (default: info)
	Level slog.Leveler

	// URL
	Endpoint string
	Timeout  time.Duration // default: 10s

	// optional: fetch attributes from context
	AttrFromContext []func(ctx context.Context) []slog.Attr

	// optional: codec and transport
	Codec     Codec
	Transport Transport
}

func (o Option) NewWebhookHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelInfo
	}
	o.Timeout = modules.DefaultTimeout(o.Timeout, 10*time.Second)
	o.AttrFromContext = modules.DefaultAttrFromContext(o.AttrFromContext)
	if o.Codec == nil {
		codec, _ := GetCodec("default")
		o.Codec = codec
	}
	if o.Transport == nil {
		o.Transport = &HTTPTransport{
			Endpoint: o.Endpoint,
			Timeout:  o.Timeout,
		}
	}
	return &WebhookHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

var _ slog.Handler = (*WebhookHandler)(nil)

type WebhookHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *WebhookHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *WebhookHandler) Handle(ctx context.Context, record slog.Record) error {
	fromContext := common.ContextExtractor(ctx, h.option.AttrFromContext)
	allAttrs := append(slices.Clone(h.attrs), fromContext...)
	payload, err := h.option.Codec.Encode(ctx, &record, allAttrs, h.groups)
	if err != nil {
		return err
	}

	modules.RunAsync("webhook", func() error {
		return h.option.Transport.Send(context.Background(), payload)
	})
	return nil
}

func (h *WebhookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := common.AppendAttrsToGroup(h.groups, h.attrs, attrs...)
	return &WebhookHandler{
		option: h.option,
		attrs:  next,
		groups: slices.Clone(h.groups),
	}
}

func (h *WebhookHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &WebhookHandler{
		option: h.option,
		attrs:  slices.Clone(h.attrs),
		groups: append(slices.Clone(h.groups), name),
	}
}
