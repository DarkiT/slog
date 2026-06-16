# Quickstart

Use this reference for basic integration, production setup, and file logging.

## Minimal import

```go
import "github.com/darkit/slog"
```

Use the root package for both enhanced APIs and standard `log/slog`-style attrs.

```go
slog.SetLevelDebug()
slog.Info("service started", "port", 8080, "env", "dev")
slog.Warn("cache cold", "name", "user-cache")
slog.Error("request failed", "error", err)
```

## Production builder

Prefer the builder for applications and libraries that need explicit writer, module name, and DLP settings.

```go
logger := slog.NewLoggerBuilder().
    WithWriter(os.Stdout).
    WithModule("order-service").
    WithAttrs(slog.String("env", "production")).
    EnableText(false).
    EnableJSON(true).
    EnableDLP(true).
    Build()

slog.SetDefault(logger)
```

## Library-friendly pattern

For reusable modules, accept a logger instead of hard-coding global state.

```go
type Service struct {
    log *slog.Logger
}

func NewService(log *slog.Logger) *Service {
    if log == nil {
        log = slog.Default("service")
    }
    return &Service{log: log}
}
```

## Structured fields

Prefer stable keys and typed attrs.

```go
logger.Info("payment captured",
    "order_id", orderID,
    "amount_cents", amount,
    "duration", elapsed,
)
```

Avoid `Infof` for events that should be indexed by log systems. Keep formatted logs for CLI-like human messages.

## File writer with rotation

```go
writer := slog.NewWriter("logs/app.log").
    SetMaxSize(100).
    SetMaxAge(7).
    SetMaxBackups(10).
    SetCompress(true)

logger := slog.NewLoggerBuilder().
    WithWriter(writer).
    EnableJSON(true).
    EnableDLP(true).
    Build()
```

## Runtime switches

```go
_ = slog.SetLevel("warn")
slog.EnableJSONLogger()
slog.DisableTextLogger()
slog.EnableDLPLogger()

snapshot := slog.GetRuntimeSnapshot()
_ = snapshot
```

## Suggested assets

- `assets/examples/basic-logging.go.tmpl`
- `assets/examples/production-json-dlp.go.tmpl`
