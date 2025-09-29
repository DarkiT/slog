package webhook

import (
	"log/slog"
	"net/http"

	svr "github.com/darkit/slog"
	"github.com/darkit/slog/common"
)

var (
	SourceKey            = "source"
	ContextKey           = "extra"
	ErrorKeys            = []string{"error", "err"}
	RequestKey           = "request"
	RequestIgnoreHeaders = false
)

type Converter func(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) map[string]any

func DefaultConverter(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) map[string]any {
	// aggregate all attributes
	attrs := common.AppendRecordAttrsToAttrs(loggerAttr, groups, record)

	// developer formatters
	attrs = common.ReplaceError(attrs, ErrorKeys...)
	if addSource {
		attrs = append(attrs, common.Source(SourceKey, record))
	}
	attrs = common.ReplaceAttrs(replaceAttr, []string{}, attrs...)
	attrs = common.RemoveEmptyAttrs(attrs)

	// handler formatter
	extra := common.AttrsToMap(attrs...)

	payload := map[string]any{
		"logger.name":    svr.Name,
		"logger.version": svr.Version,
		"timestamp":      record.Time.UTC(),
		"level":          record.Level.String(),
		"message":        record.Message,
	}

	for _, errorKey := range ErrorKeys {
		if v, ok := extra[errorKey]; ok {
			if err, ok := v.(error); ok {
				payload[errorKey] = common.FormatError(err)
				delete(extra, errorKey)
				break
			}
		}
	}

	if v, ok := extra[RequestKey]; ok {
		if req, ok := v.(*http.Request); ok {
			payload[RequestKey] = common.FormatRequest(req, RequestIgnoreHeaders)
			delete(extra, RequestKey)
		}
	}

	if user, ok := extra["user"]; ok {
		payload["user"] = user
		delete(extra, "user")
	}

	payload[ContextKey] = extra

	return payload
}
