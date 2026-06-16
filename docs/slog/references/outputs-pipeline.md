# Outputs, subscriptions, and modules

Use this reference for event subscription, backpressure, multi-handler fanout, and output module selection.

## Subscription model

`Subscribe` receives published log events after formatter, DLP, and context enrichment.

```go
ch, cancel := slog.Subscribe(1000)
defer cancel()

for event := range ch {
    _ = event.Record   // structured slog.Record alias
    _ = event.Rendered // final text/json rendering when available
    _ = event.Format   // "text", "json", or ""
}
```

## Backpressure policies

Use `SubscribeWithOptions` for high-throughput consumers.

```go
ch, cancel := slog.SubscribeWithOptions(slog.SubscribeOptions{
    BufferSize:   4096,
    Backpressure: slog.SubscriptionDropOldest,
    BlockTimeout: 5 * time.Millisecond,
})
defer cancel()
_ = ch
```

Policies:

- `SubscriptionDropOldest`: preserve recent logs; good for dashboards.
- `SubscriptionDropNewest`: preserve queue order; good for archival consumers.
- `SubscriptionBlockWithTimeout`: wait briefly; good when loss should be rare but logging must not hang forever.

Inspect health:

```go
stats := slog.GetSubscriptionStats()
all := slog.ListSubscriberStats()
_ = stats
_ = all
```

## Multi-handler fanout

For standard-compatible fanout, use root handler mappings.

```go
handler := slog.NewMultiHandler(
    slog.NewTextHandler(os.Stdout, nil),
    slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug}),
)
logger := slog.New(handler)
slog.SetDefault(logger)
```

## Builder output modes

```go
logger := slog.NewLoggerBuilder().UseLogfmt().Build()
logger.Info("event", "component", "worker")
```

For GELF or raw TCP/UDP, import only the specific module package needed:

```go
import outputnet "github.com/darkit/slog/modules/output/net"

logger := slog.NewLoggerBuilder().
    UseNetOutput(&outputnet.SenderOption{
        Network: "tcp",
        Addr:    "logs.example.internal:1514",
    }).
    Build()
```

## Module guardrails

- Prefer async external delivery for webhooks/syslog to avoid blocking request paths.
- Always bound subscription buffers and choose a backpressure strategy.
- Use JSON/logfmt for collectors; use text only for humans.
- Never put real webhook URLs or credentials in examples or tests.

## Suggested assets

- `assets/examples/subscription.go.tmpl`
- `assets/examples/multi-output.go.tmpl`
