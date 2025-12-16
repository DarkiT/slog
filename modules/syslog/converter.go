package syslog

import (
	"log/slog"

	svr "github.com/darkit/slog"
	"github.com/darkit/slog/internal/common"
)

var (
	SourceKey  = "source"
	ContextKey = "extra"
	ErrorKeys  = []string{"error", "err"}
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
	log := map[string]any{
		"logger.name":    svr.Name,
		"logger.version": svr.Version,
		"timestamp":      record.Time.UTC(),
		"level":          record.Level.String(),
		"message":        record.Message,
	}

	extra := common.AttrsToMap(attrs...)

	log[ContextKey] = extra

	return log
}
