package formatter

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/darkit/slog/internal/common"
)

// HTTPRequestFormatter transforms a *http.Request into a readable object.
func HTTPRequestFormatter(ignoreHeaders bool) Formatter {
	headers := slog.String("headers", "[hidden]")

	return FormatByType(func(req *http.Request) slog.Value {
		if !ignoreHeaders {
			headers = slog.Group(
				"headers",
				common.ToAnySlice(common.MapToSlice(req.Header, func(key string, values []string) slog.Attr {
					return slog.String(key, strings.Join(values, ","))
				}))...,
			)
		}

		return slog.GroupValue(
			slog.String("host", req.Host),
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Group(
				"url",
				slog.String("url", req.URL.String()),
				slog.String("scheme", req.URL.Scheme),
				slog.String("host", req.URL.Host),
				slog.String("path", req.URL.Path),
				slog.String("raw_query", req.URL.RawQuery),
				slog.String("fragment", req.URL.Fragment),
				slog.Group(
					"query",
					common.ToAnySlice(common.MapToSlice(req.URL.Query(), func(key string, values []string) slog.Attr {
						return slog.String(key, strings.Join(values, ","))
					}))...,
				),
			),
			headers,
		)
	})
}

// HTTPResponseFormatter transforms a *http.Response into a readable object.
func HTTPResponseFormatter(ignoreHeaders bool) Formatter {
	headers := slog.String("headers", "[hidden]")

	return FormatByType(func(res *http.Response) slog.Value {
		if !ignoreHeaders {
			headers = slog.Group(
				"headers",
				common.ToAnySlice(common.MapToSlice(res.Header, func(key string, values []string) slog.Attr {
					return slog.String(key, strings.Join(values, ","))
				}))...,
			)
		}

		return slog.GroupValue(
			slog.Int("status", res.StatusCode),
			slog.String("status_text", res.Status),
			slog.Int64("content_length", res.ContentLength),
			slog.Bool("uncompressed", res.Uncompressed),
			headers,
		)
	})
}
