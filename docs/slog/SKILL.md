---
name: darkit-slog
description: "Production-grade Go structured logging guidance for github.com/darkit/slog. Use when integrating, reviewing, or modifying darkit/slog logging: JSON/Text output, log/slog compatibility mapping, DLP masking, context propagation, subscriptions, runtime level control, file rotation, formatter modules, multi-output pipelines, and production performance or troubleshooting."
---

# darkit-slog

Use this skill to work with `github.com/darkit/slog`, a Go 1.23+ structured logging module built around the standard `log/slog` model plus DLP masking, runtime controls, subscriptions, and output modules.

Keep `SKILL.md` as the routing layer. Load only the reference or asset that matches the user's task.

## First move

1. If working inside a repository, inspect `go.mod` first. Prefer live code over these docs when they disagree.
2. Classify the request by scenario using the table below.
3. Read at most the minimal matching reference file before editing or answering.
4. Copy an asset template only when producing code for a user project.
5. Validate with the smallest useful Go command, then widen if the change touches concurrency, DLP, or output pipelines.

## Scenario router

| User intent | Read this | Use asset templates |
| --- | --- | --- |
| Quick integration, basic logging, builder setup | `references/quickstart.md` | `assets/examples/basic-logging.go.tmpl`, `assets/examples/production-json-dlp.go.tmpl` |
| Standard `log/slog` compatibility, no extra `log/slog` import | `references/api-surface.md` | `assets/examples/standard-slog-compat.go.tmpl` |
| DLP masking, PII, struct tags, matcher control | `references/context-dlp.md` | `assets/examples/dlp-sensitive.go.tmpl` |
| Trace/request/user propagation through `context.Context` | `references/context-dlp.md` | `assets/examples/context-propagation.go.tmpl`, `assets/examples/http-middleware.go.tmpl` |
| Subscriptions, backpressure, multi-output, modules | `references/outputs-pipeline.md` | `assets/examples/subscription.go.tmpl`, `assets/examples/multi-output.go.tmpl` |
| Production performance, allocations, DLP cache, high throughput | `references/performance-production.md` | `assets/examples/production-json-dlp.go.tmpl` |
| Bugs, missing logs, duplicated logs, DLP false positives | `references/troubleshooting.md` | choose only if code is needed |
| Custom formatter or attr rendering | `references/api-surface.md` | `assets/examples/custom-formatter.go.tmpl` |

## Core model

Think in six layers:

1. **Core Logger**: `Default()`, `SetDefault()`, `NewLoggerBuilder()`, `With()`, `WithGroup()`.
2. **Standard surface**: root package maps common `log/slog` types and constructors, so callers normally import only `github.com/darkit/slog`.
3. **Level and format**: Trace, Debug, Info, Warn, Error, Fatal plus runtime Text/JSON toggles.
4. **Context and DLP**: context propagator injects fields; DLP masks messages, attrs, and tagged structs.
5. **Pipeline**: handler, writer, formatter, subscription, and output modules fan records out.
6. **Production guardrails**: rate limiting, file rotation, object pools, cache sizing, race-safe sharing.

## Coding rules for generated examples

- Import `github.com/darkit/slog` as the only logging package unless a submodule is explicitly needed.
- Do not import `log/slog` in user-facing examples; use root aliases such as `slog.Attr`, `slog.HandlerOptions`, `slog.NewTextHandler`, `slog.NewJSONHandler`, and `slog.LevelVar`.
- Prefer structured key-value logs over formatted strings for production paths.
- Use typed attrs (`slog.String`, `slog.Int`, `slog.Duration`, `slog.GroupAttrs`) when values are known.
- Keep context keys typed in library/middleware examples to avoid collisions.
- Handle DLP and module registration errors when APIs return `error`.
- Avoid real tokens, webhook URLs, or credentials in examples. Use placeholders.

## Validation matrix

Use the smallest set that covers the risk:

| Change type | Minimum validation |
| --- | --- |
| Documentation or templates only | `python /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/slog` and grep for stale paths |
| Public API examples | `go test ./...` plus compile a copied template if practical |
| DLP behavior | `go test ./dlp ./dlp/header` and add table tests for false positives/disabled matchers |
| Concurrency/subscription/output | `go test -race ./...` |
| Security-sensitive output, DLP, file/network writer | `gosec ./...` if available |
| Module dependencies or generated code | `go mod tidy && go mod verify` |

## Resource inventory

### References

- `references/quickstart.md`: basic import, builder, JSON/Text, file writer.
- `references/api-surface.md`: root package API surface and standard `log/slog` mapping.
- `references/context-dlp.md`: context propagation, DLP engine, matcher defaults, struct tags.
- `references/outputs-pipeline.md`: subscription, backpressure, multi-handler, output modules.
- `references/performance-production.md`: production tuning, rate limiting, allocation guidance.
- `references/troubleshooting.md`: diagnosis checklist and common fixes.

### Assets

- `assets/examples/*.go.tmpl`: copy-and-adapt Go templates. Treat these as output assets; do not load all of them unless the user asks for examples.

## Live-code search anchors

When accuracy matters, search the current repository for these anchors:

- Logger creation: `NewLoggerBuilder`, `NewLoggerWithConfig`, `SetDefault`.
- Standard mapping: `type Attr =`, `NewTextHandler`, `GroupAttrs`, `NewMultiHandler`.
- Runtime control: `SetLevel`, `ApplyRuntimeOption`, `EnableJSONLogger`, `EnableDLPLogger`.
- Context: `SetContextPropagator`, `InfoContext`, `WithContext`.
- DLP: `DesensitizeStructAdvanced`, `DisableMatchers`, `EnabledMatchers`, `RegisterCustomMatcher`.
- Subscriptions: `SubscribeWithOptions`, `SubscriptionBackpressurePolicy`, `GetSubscriptionStats`.
- Output modules: `UseLogfmt`, `UseGELF`, `UseNetOutput`, `modules.RegisterFactory`.
