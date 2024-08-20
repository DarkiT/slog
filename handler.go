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

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.Level(-4)
	LevelInfo  = slog.Level(0)
	LevelWarn  = slog.Level(4)
	LevelError = slog.Level(8)
	LevelFatal = slog.Level(12)
)

var (
	disableColor   = false
	TimeFormat     = "2006/01/02 15:04.05.000"
	defaultLevel   = LevelError
	prefixKeys     = []string{"$service"}
	levelTextNames = map[slog.Leveler]string{
		LevelInfo:  "I",
		LevelDebug: "D",
		LevelWarn:  "W",
		LevelError: "E",
		LevelTrace: "T",
		LevelFatal: "F",
	}
)

type handler struct {
	w                  io.Writer
	mu                 *sync.Mutex
	level              slog.Leveler
	groups             []string
	attrs              string
	timeFormat         string
	replaceAttr        func(groups []string, a slog.Attr) slog.Attr
	prefixes           []slog.Value // Cached list of prefix values.
	addSource, noColor bool
}

// NewConsoleHandler returns a [log/slog.Handler] using the receiver's options.
// Default options are used if opts is nil.
func NewConsoleHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	h := &handler{
		w:           w,
		mu:          &sync.Mutex{},
		level:       opts.Level,
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
		prefixes:    make([]slog.Value, len(prefixKeys)),
	}
	h.groups = make([]string, 0, 10)

	if opts.Level == nil {
		h.level = defaultLevel
	}
	if h.timeFormat == "" {
		h.timeFormat = TimeFormat
	}
	if disableColor {
		h.noColor = true
	}
	return h
}

// Users should not call the following methods directly on a handler. Instead,
// users should create a logger and call methods on the logger. The logger will
// create a record and invoke the handler's methods.

// Enabled indicates whether the receiver logs at the given level.
func (h *handler) Enabled(_ context.Context, l slog.Level) bool {
	// level := h.level.Level()
	// return level <= LevelDebug || l >= level
	return l.Level() >= h.level.Level()
}

// Handle formats a given record in a human-friendly but still largely
// structured way.
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	buf := newBuffer()
	defer buf.Free()

	timeAttr := slog.Time(slog.TimeKey, r.Time)
	if h.replaceAttr != nil {
		timeAttr = h.replaceAttr(nil, timeAttr)
	}
	if !r.Time.IsZero() && !timeAttr.Equal(slog.Attr{}) {
		buf.WriteString(timeAttr.Value.Time().Format(h.timeFormat))
		buf.WriteByte(' ')
	}

	h.appendLevel(buf, r.Level)
	buf.WriteByte(' ')

	if h.addSource && r.PC != 0 {
		buf.WriteString(h.newSourceAttr(r.PC))
		buf.WriteByte(' ')
	}

	prefixes := h.prefixes

	if r.NumAttrs() > 0 {
		nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		attrs := make([]slog.Attr, 0, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a)
			return true
		})
		if p, changed := h.extractPrefixes(attrs); changed {
			nr.AddAttrs(attrs...)
			r = nr
			prefixes = p
		}
	}
	h.formatterPrefix(buf, prefixes)

	buf.WriteString(r.Message)
	if h.attrs != "" {
		buf.WriteString(h.attrs)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(buf, a)
		return true
	})
	buf.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*buf)
	return err
}

// WithAttrs returns a new [log/slog.Handler] that has the receiver's
// attributes plus attrs.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h.prefixes, _ = h.extractPrefixes(attrs)
	h2 := h.clone()
	buf := newBuffer()
	defer buf.Free()
	for _, a := range attrs {
		h2.appendAttr(buf, a)
	}
	h2.attrs += string(*buf)
	return h2
}

// WithGroup returns a new [log/slog.Handler] with name appended to the
// receiver's groups.
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
		mu:          h.mu,
		level:       h.level,
		groups:      slices.Clip(h.groups),
		attrs:       h.attrs,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
	}
}

func (h *handler) appendLevel(buf *buffer, level slog.Level) {
	switch {
	case level == LevelDebug:
		buf.WriteStringIf(!h.noColor, ansiBrightBlue)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelDebug])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level == LevelInfo:
		buf.WriteStringIf(!h.noColor, ansiBrightGreen)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelInfo])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level == LevelError:
		buf.WriteStringIf(!h.noColor, ansiBrightRed)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelError])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level == LevelWarn:
		buf.WriteStringIf(!h.noColor, ansiBrightYellow)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelWarn])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level == LevelTrace:
		buf.WriteStringIf(!h.noColor, ansiBrightPurple)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelTrace])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level == LevelFatal:
		buf.WriteStringIf(!h.noColor, ansiBrightRed)
		buf.WriteString("[")
		buf.WriteString(levelTextNames[LevelFatal])
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	default:
		buf.WriteStringIf(!h.noColor, ansiBrightRed)
		buf.WriteString("[")
		buf.WriteString(level.Level().String())
		buf.WriteString("]")
		buf.WriteStringIf(!h.noColor, ansiReset)
	}
}

func (h *handler) appendAttr(buf *buffer, a slog.Attr) {
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
			h.appendAttr(buf, a)
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
		appendKey(buf, h.groups, a.Key)
		h.appendVal(buf, a.Value)
	}
}

func appendKey(buf *buffer, groups []string, key string) {
	buf.WriteByte(' ')
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	if needsQuoting(key) {
		*buf = strconv.AppendQuote(*buf, key)
	} else {
		buf.WriteString(key)
	}
	buf.WriteByte('=')
}

func (h *handler) appendVal(buf *buffer, val slog.Value) {
	switch val.Kind() {
	case slog.KindString:
		appendString(buf, val.String())
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, val.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, val.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, val.Float64(), 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, val.Bool())
	case slog.KindDuration:
		appendString(buf, val.Duration().String())
	case slog.KindTime:
		quoteTime := needsQuoting(h.timeFormat)
		if quoteTime {
			buf.WriteByte('"')
		}
		*buf = val.Time().AppendFormat(*buf, h.timeFormat)
		if quoteTime {
			buf.WriteByte('"')
		}
	case slog.KindGroup, slog.KindLogValuer:
		if tm, ok := val.Any().(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				return
			}
			appendString(buf, string(data))
			return
		}
		appendString(buf, fmt.Sprint(val.Any()))
	case slog.KindAny:
		switch cv := val.Any().(type) {
		case slog.Level:
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendString(buf, string(data))
		default:
			appendString(buf, fmt.Sprint(val.Any()))
		}
	}
}

func appendString(buf *buffer, s string) {
	if needsQuoting(s) {
		*buf = strconv.AppendQuote(*buf, s)
	} else {
		buf.WriteString(s)
	}
}

func (h *handler) newSourceAttr(pc uintptr) string {
	source := frame(pc)
	// dir, file := filepath.Split(source.File)
	// filepath.Join(filepath.Base(dir), file)
	return fmt.Sprintf("[%s:%d]", filepath.Base(source.File), source.Line)
}

func (h *handler) extractPrefixes(attrs []slog.Attr) (prefixes []slog.Value, changed bool) {
	prefixes = h.prefixes
	for i, attr := range attrs {
		idx := slices.IndexFunc(prefixKeys, func(s string) bool { return s == attr.Key })
		if idx >= 0 {
			if !changed {
				// make a copy of prefixes:
				prefixes = make([]slog.Value, len(h.prefixes))
				copy(prefixes, h.prefixes)
			}
			prefixes[idx] = attr.Value
			attrs[i] = slog.Attr{} // remove the prefix attribute
			changed = true
		}
	}
	return
}

func (h *handler) formatterPrefix(buf *buffer, prefixes []slog.Value) {
	p := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix.Any() == nil || prefix.String() == "" {
			continue // skip empty prefixes
		}
		p = append(p, prefix.String())
	}
	if len(p) > 0 {
		buf.WriteString("[")
		buf.WriteString(strings.Join(p, ":"))
		buf.WriteString("]")
		buf.WriteString(" ")
	}
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

// Adapted from log/slog/json_handler.go which copied the original from
// encoding/json/tables.go.
//
// unsafe holds the value true if the ASCII character requires a logfmt key or
// value to be quoted.
//
// All values are safe except for ' ', '"', and '='. Note that a map is far slower.
var unsafe = [utf8.RuneSelf]bool{
	' ': true,
	'"': true,
	'=': true,
}
