package slog

import (
	"log/slog"
	"time"
)

type Attr = slog.Attr

func Any(key string, value any) Attr {
	return slog.Any(key, value)
}

func Bool(key string, v bool) Attr {
	return slog.Bool(key, v)
}

func Duration(key string, v time.Duration) Attr {
	return slog.Duration(key, v)
}

func Float64(key string, v float64) Attr {
	return slog.Float64(key, v)
}

func Group(key string, args ...any) Attr {
	return slog.Group(key, args...)
}

func Int(key string, v int) Attr {
	return slog.Int(key, v)
}

func Int64(key string, v int64) Attr {
	return slog.Int64(key, v)
}

func String(key string, v string) Attr {
	return slog.String(key, v)
}

func Time(key string, v time.Time) Attr {
	return slog.Time(key, v)
}

func Uint64(key string, v uint64) Attr {
	return slog.Uint64(key, v)
}
