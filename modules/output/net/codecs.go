package outputnet

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/darkit/slog/internal/common"
)

// Codec converts slog records into bytes for network transport.
type Codec interface {
	Name() string
	Encode(record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error)
}

type codecRegistry struct {
	mu     sync.RWMutex
	codecs map[string]Codec
}

var globalCodecs = &codecRegistry{codecs: map[string]Codec{}}

func init() {
	_ = RegisterCodec(rawCodec{})
	_ = RegisterCodec(jsonCodec{})
}

// RegisterCodec registers a codec by name.
func RegisterCodec(codec Codec) error {
	if codec == nil || strings.TrimSpace(codec.Name()) == "" {
		return errInvalidCodec
	}
	name := strings.ToLower(strings.TrimSpace(codec.Name()))
	globalCodecs.mu.Lock()
	defer globalCodecs.mu.Unlock()
	globalCodecs.codecs[name] = codec
	return nil
}

// GetCodec returns a registered codec. Empty name defaults to raw.
func GetCodec(name string) (Codec, bool) {
	if strings.TrimSpace(name) == "" {
		name = "raw"
	}
	name = strings.ToLower(strings.TrimSpace(name))
	globalCodecs.mu.RLock()
	defer globalCodecs.mu.RUnlock()
	codec, ok := globalCodecs.codecs[name]
	return codec, ok
}

type rawCodec struct{}

func (c rawCodec) Name() string { return "raw" }

func (c rawCodec) Encode(record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error) {
	if record == nil {
		return []byte("level=INFO msg="), nil
	}
	allAttrs := common.AppendRecordAttrsToAttrs(attrs, groups, record)
	allAttrs = common.RemoveEmptyAttrs(allAttrs)

	var buf bytes.Buffer
	buf.WriteString("level=")
	buf.WriteString(record.Level.String())
	buf.WriteString(" msg=")
	buf.WriteString(record.Message)
	for _, attr := range allAttrs {
		buf.WriteByte(' ')
		buf.WriteString(attr.Key)
		buf.WriteByte('=')
		buf.WriteString(attr.Value.String())
	}
	return buf.Bytes(), nil
}

type jsonCodec struct{}

func (c jsonCodec) Name() string { return "json" }

func (c jsonCodec) Encode(record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error) {
	if record == nil {
		return json.Marshal(map[string]any{
			"level":   "INFO",
			"message": "",
		})
	}
	flat := common.AttrsToMap(common.RemoveEmptyAttrs(common.AppendRecordAttrsToAttrs(attrs, groups, record))...)
	payload := map[string]any{
		"level":   record.Level.String(),
		"message": record.Message,
	}
	if !record.Time.IsZero() {
		payload["timestamp"] = record.Time.UTC().Format(time.RFC3339Nano)
	}
	for k, v := range flat {
		payload[k] = v
	}
	return json.Marshal(payload)
}
