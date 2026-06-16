# Troubleshooting

Use this reference for debugging integration failures and behavior drift.

## No logs appear

Check in order:

1. Current level permits the record: `slog.SetLevelDebug()` for diagnosis.
2. At least one format is enabled: `EnableTextLogger()` or `EnableJSONLogger()`.
3. Writer is not nil and does not return errors.
4. Global logger was not replaced unexpectedly by `SetDefault`.
5. Rate limiter is not dropping records: temporarily `ConfigureRecordLimiter(0, 0)`.

## Logs duplicated

Likely causes:

- Both Text and JSON are enabled and write to the same sink.
- A multi-handler fans out to the same writer twice.
- A caller logs once through standard-compatible `slog.New(...)` and once through enhanced `Default()`.

## DLP false positives

Check matcher state first:

```go
engine := dlp.NewDlpEngine()
_ = engine.DisabledMatchers()
_ = engine.EnabledMatchers()
```

Then prefer one of:

- Disable noisy free-text matchers: `engine.DisableMatchers("ipv4", "domain")`.
- Use struct tags for deterministic fields.
- Add a custom matcher with `FastFilters` or a validator when available.

## Source points to logging wrapper

Register wrapper prefixes:

```go
slog.RegisterCallerSkipPrefix("example.com/myapp/logging")
```

Or disable source in hot paths:

```go
cfg := slog.DefaultConfig()
cfg.AddSource = false
```

## Subscription drops records

- Increase `BufferSize` only after checking consumer speed.
- Pick the right policy: `DropOldest` for live views, `DropNewest` for ordered queues, `BlockWithTimeout` for low-loss consumers.
- Check `GetSubscriptionStats()` and per-subscriber stats.

## Template or docs drift

For Skill maintenance:

```bash
python /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/slog
find docs/slog -type f | sort
find docs/slog -type f -name '*.go' -print -quit | grep -q . && echo 'unexpected source file under skill docs'
find docs/slog -maxdepth 1 -type f ! -name 'SKILL.md' -print -quit | grep -q . && echo 'unexpected top-level process doc'
```

No source `.go` files should remain under the skill directory; reusable examples are `*.go.tmpl` assets.
