# Performance and production guidance

Use this reference when tuning high-throughput logging, production JSON output, file rotation, or DLP overhead.

## Defaults to prefer

```go
cfg := slog.DefaultConfig()
cfg.NoColor = true
cfg.AddSource = false
cfg.SetEnableText(false)
cfg.SetEnableJSON(true)
logger := slog.NewLoggerWithConfig(os.Stdout, cfg)
slog.SetDefault(logger)
```

## DLP performance rules

- Prefer struct tags for known fields; they are more deterministic than free-text scanning.
- Keep broad matchers disabled for free text unless the user explicitly accepts false positives.
- Measure cache hit rate before increasing cache sizes.
- Avoid logging large raw payloads; log stable IDs, counts, hashes, or summarized metadata.

## Allocation rules

- Prefer typed attrs for hot paths.
- Avoid building `[]any` repeatedly inside tight loops when attrs can be prepared once.
- Avoid `fmt.Sprintf` before logging; let structured attrs carry values.
- Disable source locations on hot production paths unless auditing requires them.

## Rate limiting

Use token-bucket limiting to prevent log storms.

```go
slog.ConfigureRecordLimiter(1000, 200) // rate, burst
slog.ConfigureRecordLimiter(0, 0)      // disable
```

## File output

```go
writer := slog.NewWriter("logs/app.log").
    SetMaxSize(100).
    SetMaxAge(14).
    SetMaxBackups(20).
    SetCompress(true)
```

Prefer stdout JSON in containers; prefer rotating files for host-based daemons.

## Validation for production changes

Run:

```bash
go test ./...
go test -race ./...
gosec ./...
```

If `govulncheck` reports Go standard-library findings, verify the local toolchain version and recommend upgrading the toolchain rather than patching project code blindly.
