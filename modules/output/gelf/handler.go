package gelf

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/darkit/slog/modules"
)

// Options 控制 GELF 输出。
type Options struct {
	Writer      modules.WriteSyncer
	Level       slog.Leveler
	Host        string
	Facility    string
	AddSource   bool
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// Handler 兼容 GELF 1.1。
type Handler struct {
	w           modules.WriteSyncer
	mu          sync.Mutex
	level       slog.Leveler
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	addSource   bool
	host        string
	facility    string
}

// New 创建 handler。
func New(opt Options) *Handler {
	if opt.Writer == nil {
		opt.Writer = modules.NewStdWriter()
	}
	if opt.Level == nil {
		opt.Level = slog.LevelInfo
	}
	host := opt.Host
	if host == "" {
		if h, err := os.Hostname(); err == nil {
			host = h
		}
	}
	return &Handler{
		w:           opt.Writer,
		level:       opt.Level,
		replaceAttr: opt.ReplaceAttr,
		addSource:   opt.AddSource,
		host:        host,
		facility:    opt.Facility,
	}
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l.Level() >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	payload := make(map[string]any, 16)
	payload["version"] = "1.1"
	if h.host != "" {
		payload["host"] = h.host
	}
	payload["short_message"] = r.Message
	payload["timestamp"] = float64(r.Time.UnixNano()) / 1e9
	payload["level"] = h.levelToSyslog(r.Level)
	if h.facility != "" {
		payload["facility"] = h.facility
	}

	if h.addSource && r.PC != 0 {
		payload["_source"] = modules.SourceLabel(modules.Frame(r.PC))
	}

	groups := groupState{}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(payload, &groups, a)
		return true
	})

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err = h.w.Write(data); err != nil {
		return err
	}
	_, _ = h.w.Write([]byte{'\n'})
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler { _ = attrs; return h }
func (h *Handler) WithGroup(name string) slog.Handler       { _ = name; return h }

func (h *Handler) appendAttr(payload map[string]any, groups *groupState, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if h.replaceAttr != nil {
		attr = h.replaceAttr(groups.Values(), attr)
		if attr.Key == "" {
			return
		}
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		groups.Push(attr.Key)
		for _, a := range attr.Value.Group() {
			h.appendAttr(payload, groups, a)
		}
		groups.Pop()
	default:
		key := "_" + strings.Join(append(groups.Values(), attr.Key), ".")
		payload[key] = attr.Value.Any()
	}
}

func (h *Handler) levelToSyslog(level slog.Level) int {
	switch level {
	case slog.LevelError:
		return 3
	case slog.LevelWarn:
		return 4
	case slog.LevelInfo:
		return 6
	default:
		return 7
	}
}

// 模块注册：gelf
func init() {
	modules.RegisterFactory("gelf", func(config modules.Config) (modules.Module, error) {
		h := New(Options{})
		return modules.NewHandlerModule("gelf", h), nil
	})
}

// --- helpers ---

type groupState struct {
	names []string
}

func (g *groupState) Push(name string) {
	if name == "" {
		return
	}
	g.names = append(g.names, name)
}

func (g *groupState) Pop() {
	if len(g.names) == 0 {
		return
	}
	g.names = g.names[:len(g.names)-1]
}

func (g *groupState) Values() []string { return g.names }
