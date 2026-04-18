package slog

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
)

func (l *Logger) subscriptionEvent(ctx context.Context, r slog.Record) SubscriptionEvent {
	published := l.publishedRecord(ctx, r)
	rendered, format := l.renderSubscription(ctx, r, published)

	return SubscriptionEvent{
		Record:   published,
		Rendered: rendered,
		Format:   format,
	}
}

func (l *Logger) publishedRecord(ctx context.Context, r slog.Record) slog.Record {
	if eh := l.observerHandler(); eh != nil {
		return eh.normalizedObserverRecord(ctx, r)
	}

	fallback := &eHandler{
		opts: ext,
		ctx:  l.ctx,
	}
	return fallback.normalizedObserverRecord(ctx, r)
}

func (l *Logger) observerHandler() *eHandler {
	if l == nil {
		return nil
	}
	if eh := unwrapObserverHandler(l.text); eh != nil {
		return eh
	}
	return unwrapObserverHandler(l.json)
}

func unwrapObserverHandler(logger *slog.Logger) *eHandler {
	if logger == nil {
		return nil
	}
	if eh, ok := logger.Handler().(*eHandler); ok {
		return eh
	}
	return nil
}

func (l *Logger) preferredRenderedFormat() string {
	textOn, jsonOn := l.outputEnabled()
	if jsonOn && !textOn {
		return "json"
	}
	if textOn {
		return "text"
	}
	if jsonOn {
		return "json"
	}
	return ""
}

func (l *Logger) renderSubscription(ctx context.Context, raw slog.Record, published slog.Record) (string, string) {
	switch l.preferredRenderedFormat() {
	case "json":
		return l.renderSubscriptionJSON(ctx, raw, published), "json"
	case "text":
		return l.renderSubscriptionText(ctx, raw, published), "text"
	default:
		return "", ""
	}
}

func (l *Logger) renderSubscriptionText(ctx context.Context, raw slog.Record, published slog.Record) string {
	if l != nil && l.text != nil {
		return l.renderWithHandlerChain(ctx, raw, l.text.Handler(), func(buf *bytes.Buffer) slog.Handler {
			return NewConsoleHandler(buf, l.noColor, l.subscriptionHandlerOptions())
		})
	}
	return l.renderPublishedText(published)
}

func (l *Logger) renderSubscriptionJSON(ctx context.Context, raw slog.Record, published slog.Record) string {
	if l != nil && l.json != nil {
		return l.renderWithHandlerChain(ctx, raw, l.json.Handler(), func(buf *bytes.Buffer) slog.Handler {
			return NewJSONHandler(buf, l.subscriptionHandlerOptions())
		})
	}
	return l.renderPublishedJSON(published)
}

func (l *Logger) renderPublishedText(record slog.Record) string {
	var buf bytes.Buffer
	handler := NewConsoleHandler(&buf, l.noColor, l.subscriptionHandlerOptions())
	if err := handler.Handle(context.Background(), record); err != nil {
		return ""
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func (l *Logger) renderWithHandlerChain(ctx context.Context, raw slog.Record, original slog.Handler, baseFactory func(*bytes.Buffer) slog.Handler) string {
	var buf bytes.Buffer
	renderer := cloneRenderChain(original, baseFactory(&buf))
	if renderer == nil {
		return ""
	}
	if err := renderer.Handle(ctx, raw); err != nil {
		return ""
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func cloneRenderChain(original slog.Handler, leaf slog.Handler) slog.Handler {
	if original == nil {
		return leaf
	}
	eh, ok := original.(*eHandler)
	if !ok {
		return leaf
	}

	next := cloneRenderChain(eh.handler, leaf)
	return &eHandler{
		handler:     next,
		opts:        eh.opts,
		prefixes:    slices.Clone(eh.prefixes),
		groups:      slices.Clone(eh.groups),
		observerOps: cloneObserverOperations(eh.observerOps),
		ctx:         eh.ctx,
	}
}

func (l *Logger) renderPublishedJSON(record slog.Record) string {
	var buf bytes.Buffer
	handler := NewJSONHandler(&buf, l.subscriptionHandlerOptions())
	if err := handler.Handle(context.Background(), record); err != nil {
		return ""
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func (l *Logger) subscriptionHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource:   l.renderConfig.addSource,
		ReplaceAttr: l.renderConfig.replaceAttr,
	}
}
