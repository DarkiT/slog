package slog

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

var (
	defaultLevel   = LevelError
	levelTextNames = map[slog.Leveler]string{
		LevelInfo:  "I",
		LevelDebug: "D",
		LevelWarn:  "W",
		LevelError: "E",
		LevelTrace: "T",
		LevelFatal: "F",
	}
	levelColorMap = map[slog.Level]string{
		LevelDebug: ansiBrightBlue,
		LevelInfo:  ansiBrightGreen,
		LevelWarn:  ansiBrightYellow,
		LevelError: ansiBrightRed,
		LevelTrace: ansiBrightPurple,
		LevelFatal: ansiBrightRed,
	}
)

type handler struct {
	w                  io.Writer
	mu                 sync.Mutex
	level              slog.Leveler
	groups             []string
	attrs              string
	timeFormat         string
	replaceAttr        func(groups []string, a slog.Attr) slog.Attr
	addSource, noColor bool
}

// NewConsoleHandler returns a [log/slog.Handler] using the receiver's options.
// Default options are used if opts is nil.
func NewConsoleHandler(w io.Writer, noColor bool, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	h := &handler{
		w:           w,
		level:       opts.Level,
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
	}

	if opts.Level == nil {
		h.level = defaultLevel
	}
	if h.timeFormat == "" {
		h.timeFormat = TimeFormat
	}

	h.noColor = noColor
	return h
}

// Enabled indicates whether the receiver logs at the given level.
func (h *handler) Enabled(_ context.Context, l slog.Level) bool {
	return l.Level() >= h.level.Level()
}

// Handle formats a given record in a human-friendly but still largely structured way.
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	var sb buffer

	if !r.Time.IsZero() {
		sb.WriteString(r.Time.Format(TimeFormat))
		_ = sb.WriteByte(' ')
	}

	h.appendLevel(&sb, r.Level)
	_ = sb.WriteByte(' ')

	if h.addSource && r.PC != 0 {
		sb.WriteString(h.newSourceAttr(r.PC))
		_ = sb.WriteByte(' ')
	}

	sb.WriteString(r.Message)
	if h.attrs != "" {
		sb.WriteString(h.attrs)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&sb, a)
		return true
	})
	_ = sb.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write([]byte(sb.String()))
	return err
}

// WithAttrs returns a new [log/slog.Handler] that has the receiver's attributes plus attrs.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()
	var sb buffer
	for _, a := range attrs {
		h2.appendAttr(&sb, a)
	}
	h2.attrs += sb.String()
	return h2
}

// WithGroup returns a new [log/slog.Handler] with name appended to the receiver's groups.
func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		w:           h.w,
		mu:          h.mu, // Reuse the same mutex
		level:       h.level,
		groups:      slices.Clip(h.groups),
		attrs:       h.attrs,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
		noColor:     h.noColor,
	}
}

func (h *handler) appendLevel(sb *buffer, level slog.Level) {
	color, ok := levelColorMap[level]
	if !ok {
		color = ansiBrightRed
	}

	sb.WriteStringIf(!h.noColor, color)
	sb.WriteString("[")
	sb.WriteString(levelTextNames[level])
	sb.WriteString("]")
	sb.WriteStringIf(!h.noColor, ansiReset)
}

func (h *handler) appendAttr(sb *buffer, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			h.groups = append(h.groups, a.Key)
		}
		for _, a := range attrs {
			h.appendAttr(sb, a)
		}
		if a.Key != "" {
			h.groups = h.groups[:len(h.groups)-1]
		}
		return
	}
	if h.replaceAttr != nil {
		a = h.replaceAttr(h.groups, a)
	}
	if !a.Equal(slog.Attr{}) {
		appendKey(sb, h.groups, a.Key)
		h.appendVal(sb, a.Value)
	}
}

func appendKey(sb *buffer, groups []string, key string) {
	_ = sb.WriteByte(' ')
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	if needsQuoting(key) {
		sb.WriteString(strconv.Quote(key))
	} else {
		sb.WriteString(key)
	}
	_ = sb.WriteByte('=')
}

func (h *handler) appendVal(sb *buffer, val slog.Value) {
	switch val.Kind() {
	case slog.KindString:
		appendString(sb, val.String())
	case slog.KindInt64:
		sb.WriteString(strconv.FormatInt(val.Int64(), 10))
	case slog.KindUint64:
		sb.WriteString(strconv.FormatUint(val.Uint64(), 10))
	case slog.KindFloat64:
		sb.WriteString(strconv.FormatFloat(val.Float64(), 'g', -1, 64))
	case slog.KindBool:
		sb.WriteString(strconv.FormatBool(val.Bool()))
	case slog.KindDuration:
		appendString(sb, val.Duration().String())
	case slog.KindTime:
		quoteTime := needsQuoting(h.timeFormat)
		if quoteTime {
			_ = sb.WriteByte(' ')
		}
		sb.WriteString(val.Time().Format(h.timeFormat))
		if quoteTime {
			_ = sb.WriteByte(' ')
		}
	case slog.KindGroup, slog.KindLogValuer:
		if tm, ok := val.Any().(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err == nil {
				appendString(sb, string(data))
			}
			return
		}
		appendString(sb, fmt.Sprint(val.Any()))
	case slog.KindAny:
		switch cv := val.Any().(type) {
		case slog.Level:
			h.appendLevel(sb, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err == nil {
				appendString(sb, string(data))
			}
		default:
			appendString(sb, fmt.Sprint(val.Any()))
		}
	}
}

func appendString(sb *buffer, s string) {
	if needsQuoting(s) {
		sb.WriteString(strconv.Quote(s))
	} else {
		sb.WriteString(s)
	}
}

func (h *handler) newSourceAttr(pc uintptr) string {
	source := frame(pc)
	return fmt.Sprintf("[%s:%d]", filepath.Base(source.File), source.Line)
}

func frame(pc uintptr) runtime.Frame {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return f
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if unsafe[b] {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

var unsafe = [utf8.RuneSelf]bool{
	' ': true,
	'"': true,
	'=': true,
}
