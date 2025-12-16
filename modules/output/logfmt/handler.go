package logfmt

import (
	"bytes"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkit/slog/modules"
)

// Handler 以 logfmt 形式输出，便于 Loki/Vector 等收集器解析。
type Handler struct {
	w           modules.WriteSyncer
	mu          sync.Mutex
	level       slog.Leveler
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	addSource   bool
	timeFormat  string
}

// Option 用于创建 Handler。
type Option struct {
	Writer      modules.WriteSyncer
	Level       slog.Leveler
	AddSource   bool
	TimeFormat  string
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// New 创建 Handler。
func New(opt Option) *Handler {
	if opt.Writer == nil {
		opt.Writer = modules.NewStdWriter()
	}
	if opt.Level == nil {
		opt.Level = slog.LevelInfo
	}
	if opt.TimeFormat == "" {
		opt.TimeFormat = time.DateTime
	}
	return &Handler{
		w:           opt.Writer,
		level:       opt.Level,
		replaceAttr: opt.ReplaceAttr,
		addSource:   opt.AddSource,
		timeFormat:  opt.TimeFormat,
	}
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l.Level() >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	rep := h.replaceAttr
	groups := groupState{}

	if !r.Time.IsZero() {
		ts := r.Time.Round(0)
		if rep != nil {
			if a := rep(nil, slog.Time(slog.TimeKey, ts)); a.Key != "" {
				switch a.Value.Kind() {
				case slog.KindTime:
					ts = a.Value.Time()
				case slog.KindString:
					if parsed, err := time.Parse(h.timeFormat, a.Value.String()); err == nil {
						ts = parsed
					}
				}
			}
		}
		buf.WriteString("time=")
		buf.WriteString(ts.Format(h.timeFormat))
		buf.WriteByte(' ')
	}

	buf.WriteString("level=")
	buf.WriteString(levelString(r.Level))
	buf.WriteByte(' ')

	if h.addSource && r.PC != 0 {
		buf.WriteString("source=")
		buf.WriteString(sourceLabel(r.PC))
		buf.WriteByte(' ')
	}

	buf.WriteString("msg=")
	writeStringValue(&buf, r.Message)

	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&buf, &groups, a)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	_ = attrs
	return h
}

func (h *Handler) WithGroup(name string) slog.Handler {
	_ = name
	return h
}

// 模块注册
func init() {
	modules.RegisterFactory("logfmt", func(config modules.Config) (modules.Module, error) {
		h := New(Option{})
		return modules.NewHandlerModule("logfmt", h), nil
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

func (g *groupState) Values() []string {
	return g.names
}

func (h *Handler) appendAttr(buf *bytes.Buffer, groups *groupState, attr slog.Attr) {
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
			h.appendAttr(buf, groups, a)
		}
		groups.Pop()
	default:
		key := strings.Join(append(groups.Values(), attr.Key), ".")
		buf.WriteByte(' ')
		buf.WriteString(key)
		buf.WriteByte('=')
		writeValue(buf, attr.Value, h.timeFormat)
	}
}

func writeValue(buf *bytes.Buffer, v slog.Value, timeFmt string) {
	switch v.Kind() {
	case slog.KindString:
		writeStringValue(buf, v.String())
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64:
		buf.WriteString(v.String())
	case slog.KindBool:
		buf.WriteString(strconv.FormatBool(v.Bool()))
	case slog.KindTime:
		buf.WriteString(v.Time().Format(timeFmt))
	case slog.KindDuration:
		buf.WriteString(v.Duration().String())
	default:
		writeStringValue(buf, v.String())
	}
}

func writeStringValue(buf *bytes.Buffer, s string) {
	needsQuote := strings.ContainsAny(s, " \t\n\r\"=")
	if !needsQuote {
		buf.WriteString(s)
		return
	}
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"':
			buf.WriteByte('\\')
			buf.WriteByte(s[i])
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			buf.WriteByte(s[i])
		}
	}
	buf.WriteByte('"')
}

func sourceLabel(pc uintptr) string {
	f := modules.Frame(pc)
	return modules.SourceLabel(f)
}

func levelString(l slog.Level) string {
	switch l {
	case slog.LevelDebug:
		return "Debug"
	case slog.LevelWarn:
		return "Warn"
	case slog.LevelError:
		return "Error"
	case slog.LevelInfo:
		fallthrough
	default:
		return "Info"
	}
}
