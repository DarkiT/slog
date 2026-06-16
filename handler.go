package slog

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"
)

var (
	defaultLevel  = LevelInfo
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
	state              *handlerState
	level              slog.Leveler
	groupPrefix        groupState
	attrs              string
	timeFormat         string
	replaceAttr        func(groups []string, a slog.Attr) slog.Attr
	addSource, noColor bool
	dynamicTTY         bool
}

type handlerState struct {
	mu            sync.Mutex
	dynamicActive bool
}

type groupState struct {
	names  []string
	joined []string
}

func (s *groupState) clone() groupState {
	return groupState{
		names:  slices.Clone(s.names),
		joined: slices.Clone(s.joined),
	}
}

func (s *groupState) push(name string) {
	if name == "" {
		return
	}
	s.names = append(s.names, name)
	if len(s.joined) == 0 {
		s.joined = append(s.joined, name)
		return
	}
	s.joined = append(s.joined, s.joined[len(s.joined)-1]+"."+name)
}

func (s *groupState) pop() {
	if len(s.names) == 0 {
		return
	}
	s.names = s.names[:len(s.names)-1]
	s.joined = s.joined[:len(s.joined)-1]
}

func (s *groupState) prefix() string {
	if len(s.joined) == 0 {
		return ""
	}
	return s.joined[len(s.joined)-1]
}

func (s *groupState) values() []string {
	return s.names
}

// NewConsoleHandler returns a [log/slog.Handler] using the receiver's options.
// Default options are used if opts is nil.
func NewConsoleHandler(w io.Writer, noColor bool, opts *HandlerOptions) Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	h := &handler{
		w:           w,
		state:       &handlerState{},
		level:       opts.Level,
		replaceAttr: opts.ReplaceAttr,
		addSource:   opts.AddSource,
		dynamicTTY:  isInteractiveWriter(w),
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
func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	sb := newBuffer()
	defer sb.Free()

	rep := h.replaceAttr

	groups := h.groupPrefix.clone()

	if !r.Time.IsZero() {
		val := r.Time.Round(0)
		if rep == nil {
			t := r.Time
			sb.AppendString(t.Format(h.timeFormat))
		} else if a := rep(nil, slog.Time(slog.TimeKey, val)); a.Key != "" {
			if a.Value.Kind() == slog.KindTime {
				sb.AppendString(a.Value.Time().Format(h.timeFormat))
			} else if a.Value.Kind() == slog.KindString {
				sb.AppendString(a.Value.String())
			}
			sb.AppendByte(' ')
		}
	}

	h.appendLevel(sb, r.Level)
	sb.AppendByte(' ')

	if h.addSource && r.PC != 0 {
		sb.AppendString(h.newSourceAttr(r.PC))
		sb.AppendByte(' ')
	}

	sb.AppendString(r.Message)
	if h.attrs != "" {
		sb.AppendString(h.attrs)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(sb, &groups, a)
		return true
	})
	sb.AppendByte('\n')

	if renderState, ok := dynamicRenderFromContext(ctx); ok && h.dynamicTTY {
		return h.writeDynamicBuffer(sb, renderState.final)
	}

	return h.writeBuffer(sb)
}

// WithAttrs returns a new [log/slog.Handler] that has the receiver's attributes plus attrs.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	sb := newBuffer()
	defer sb.Free()

	state := h2.groupPrefix.clone()

	for _, a := range attrs {
		h2.appendAttr(sb, &state, a)
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
	h2.groupPrefix.push(name)
	return h2
}

func (h *handler) clone() *handler {
	return &handler{
		w:           h.w,
		state:       h.state,
		level:       h.level,
		groupPrefix: h.groupPrefix.clone(),
		attrs:       h.attrs,
		timeFormat:  h.timeFormat,
		replaceAttr: h.replaceAttr,
		addSource:   h.addSource,
		noColor:     h.noColor,
		dynamicTTY:  h.dynamicTTY,
	}
}

func (h *handler) writeBuffer(buf *buffer) error {
	if h.state == nil {
		h.state = &handlerState{}
	}

	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	if h.state.dynamicActive {
		if _, err := io.WriteString(h.w, "\n"); err != nil {
			return err
		}
		h.state.dynamicActive = false
	}

	_, err := buf.WriteTo(h.w)
	return err
}

func (h *handler) writeDynamicBuffer(buf *buffer, final bool) error {
	if h.state == nil {
		h.state = &handlerState{}
	}

	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	if _, err := io.WriteString(h.w, "\r\x1b[K"); err != nil {
		return err
	}
	message := buf.buf
	if len(message) > 0 && message[len(message)-1] == '\n' {
		message = message[:len(message)-1]
	}
	if len(message) > 0 {
		if _, err := h.w.Write(message); err != nil {
			return err
		}
	}
	if final {
		h.state.dynamicActive = false
		_, err := io.WriteString(h.w, "\n")
		return err
	}

	h.state.dynamicActive = true
	return nil
}

func isInteractiveWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok || file == nil {
		return false
	}

	if term := os.Getenv("TERM"); term == "" || term == "dumb" {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func (h *handler) appendLevel(sb *buffer, level slog.Level) {
	color, ok := levelColorMap[level]
	if !ok {
		color = ansiBrightRed
	}

	sb.AppendStringIf(!h.noColor, color)
	sb.AppendString("[")
	sb.AppendString(levelTextName(level))
	sb.AppendString("]")
	sb.AppendStringIf(!h.noColor, ansiReset)
}

func levelTextName(level slog.Level) string {
	if name, ok := levelTextNames[level]; ok {
		return name
	}
	return level.String()
}

func (h *handler) appendAttr(sb *buffer, groups *groupState, a slog.Attr) {
	if a.Value.Kind() == slog.KindLogValuer {
		a.Value = a.Value.Resolve()
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			groups.push(a.Key)
		}
		for _, child := range attrs {
			h.appendAttr(sb, groups, child)
		}
		if a.Key != "" {
			groups.pop()
		}
		return
	}
	if h.replaceAttr != nil {
		a = h.replaceAttr(groups.values(), a)
	}
	if !a.Equal(slog.Attr{}) {
		appendKey(sb, groups.prefix(), a.Key)
		h.appendVal(sb, a.Value)
	}
}

func appendKey(sb *buffer, prefix string, key string) {
	sb.AppendByte(' ')
	if prefix != "" {
		if key != "" {
			key = prefix + "." + key
		} else {
			key = prefix
		}
	}
	if needsQuoting(key) {
		sb.AppendString(strconv.Quote(key))
	} else {
		sb.AppendString(key)
	}
	sb.AppendByte('=')
}

func (h *handler) appendVal(sb *buffer, val slog.Value) {
	switch val.Kind() {
	case slog.KindString:
		appendString(sb, val.String())
	case slog.KindInt64:
		sb.AppendString(strconv.FormatInt(val.Int64(), 10))
	case slog.KindUint64:
		sb.AppendString(strconv.FormatUint(val.Uint64(), 10))
	case slog.KindFloat64:
		sb.AppendString(strconv.FormatFloat(val.Float64(), 'g', -1, 64))
	case slog.KindBool:
		sb.AppendString(strconv.FormatBool(val.Bool()))
	case slog.KindDuration:
		appendString(sb, val.Duration().String())
	case slog.KindTime:
		quoteTime := needsQuoting(h.timeFormat)
		if quoteTime {
			sb.AppendByte(' ')
		}
		sb.AppendString(val.Time().Format(h.timeFormat))
		if quoteTime {
			sb.AppendByte(' ')
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
		sb.AppendString(strconv.Quote(s))
	} else {
		sb.AppendString(s)
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
