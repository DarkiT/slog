# API surface and standard slog compatibility

Use this reference when the user wants callers to import only `github.com/darkit/slog`, or when translating standard `log/slog` examples.

## Import rule

User-facing code should normally import only:

```go
import "github.com/darkit/slog"
```

The root package maps common standard `log/slog` names.

## Type aliases

Use these root names instead of `log/slog` imports:

| Standard concept | darkit/slog root name |
| --- | --- |
| `slog.Level` | `slog.Level` |
| `slog.Attr` | `slog.Attr` |
| `slog.Value` | `slog.Value` |
| `slog.Record` | `slog.Record` |
| `slog.Handler` | `slog.Handler` |
| `slog.HandlerOptions` | `slog.HandlerOptions` |
| `slog.LevelVar` | `slog.LevelVar` |
| `slog.Kind` | `slog.Kind` |
| standard logger | `*slog.SlogLogger` or `*slog.StdLogger` |

## Constructors and helpers

Root-level constructors include:

```go
slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true})
slog.New(handler)
slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
slog.NewLogLogger(handler, slog.LevelInfo)
slog.SetLogLoggerLevel(slog.LevelWarn)
```

Attr and value helpers include:

```go
slog.String("key", "value")
slog.Int("count", 3)
slog.Duration("latency", elapsed)
slog.GroupAttrs("http", slog.String("method", "GET"))
slog.StringValue("value")
slog.AnyValue(v)
```

## Default logger behavior

`SetDefault` accepts both enhanced `*slog.Logger` and standard-compatible `*slog.SlogLogger`.

```go
enhanced := slog.NewLoggerBuilder().EnableJSON(true).Build()
slog.SetDefault(enhanced)

std := slog.New(slog.NewTextHandler(os.Stdout, nil))
slog.SetDefault(std)
```

Use `slog.Default()` for the enhanced logger and `slog.StdDefault()` when a standard-compatible logger is required.

## MultiHandler

Use `NewMultiHandler` to fan one standard record to several handlers.

```go
handler := slog.NewMultiHandler(
    slog.NewTextHandler(os.Stdout, nil),
    slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug}),
)
logger := slog.New(handler)
slog.SetDefault(logger)
```

## Custom formatter

`RegisterFormatter` receives root `slog.Attr` and returns root `slog.Value`.

```go
id := slog.RegisterFormatter("mask-secret", func(groups []string, attr slog.Attr) (slog.Value, bool) {
    if attr.Key == "secret" {
        return slog.StringValue("[REDACTED]"), true
    }
    return attr.Value, false
})
defer slog.RemoveFormatter(id)
```

## Suggested assets

- `assets/examples/standard-slog-compat.go.tmpl`
- `assets/examples/custom-formatter.go.tmpl`
