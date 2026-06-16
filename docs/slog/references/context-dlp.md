# Context propagation and DLP

Use this reference for trace fields, request/user propagation, PII masking, and matcher tuning.

## Context propagation

Register one propagator during application initialization.

```go
type contextKey string

const (
    traceIDKey contextKey = "trace_id"
    userIDKey  contextKey = "user_id"
)

slog.SetContextPropagator(func(ctx context.Context) []slog.Attr {
    attrs := make([]slog.Attr, 0, 2)
    if v, ok := ctx.Value(traceIDKey).(string); ok && v != "" {
        attrs = append(attrs, slog.String("trace_id", v))
    }
    if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
        attrs = append(attrs, slog.String("user_id", v))
    }
    return attrs
})
```

Then log with context-aware methods:

```go
logger.InfoContext(ctx, "request completed", "status", 200)
logger.WithContext(ctx).Warn("slow request", "duration", elapsed)
```

## DLP activation

```go
slog.EnableDLPLogger()

logger := slog.NewLoggerBuilder().
    EnableJSON(true).
    EnableDLP(true).
    Build()
```

DLP applies to logged messages and attributes in the enhanced logger path.

## Struct tag masking

Use explicit tags for deterministic masking.

```go
type User struct {
    Name  string `dlp:"chinese_name"`
    Phone string `dlp:"mobile_phone"`
    Email string `dlp:"email"`
    Token string `dlp:"access_token"`
    Meta  Meta   `dlp:"type,recursive"`
    Skip  string `dlp:"-"`
}

engine := dlp.NewDlpEngine()
engine.Enable()
if err := engine.DesensitizeStructAdvanced(&user); err != nil {
    return err
}
```

## Matcher defaults

Free-text scanning disables overly broad matchers by default to reduce false positives:

- `username`
- `api_key`
- `access_token`
- `password`

They remain available through explicit struct tags and per-type calls.

```go
engine.DisableMatchers("ipv4", "ipv6")
engine.EnableMatchers("ipv4")
engine.SetMatcherEnabled("email", false)

_ = engine.DisabledMatchers()
_ = engine.EnabledMatchers()
_ = engine.GetSupportedTypes()
```

## Supported public type constants

The `dlp` package exposes constants such as:

```text
chinese_name id_card passport social_security license_number
mobile_phone landline email address postal_code
bank_card credit_card iban swift
ipv4 ipv6 mac url domain
imei plate vin device_id uuid
api_key jwt access_token password username
md5 sha1 sha256 lat_lng medical_id company_id git_repo
```

When exact coverage matters, inspect `dlp/const.go` and `dlp/regexp.go` in live code.

## Custom matcher

```go
err := engine.RegisterCustomMatcher(&dlp.Matcher{
    Name:    "tenant_id",
    Pattern: `TEN-[0-9]{8}`,
    Transformer: func(s string) string {
        if len(s) <= 4 {
            return "****"
        }
        return s[:4] + "****"
    },
})
if err != nil {
    return err
}
```

## Suggested assets

- `assets/examples/dlp-sensitive.go.tmpl`
- `assets/examples/context-propagation.go.tmpl`
- `assets/examples/http-middleware.go.tmpl`
