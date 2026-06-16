// Package slog provides a high-performance, feature-rich structured logging library
// for Go, extending the standard log/slog package.
//
// Built on Go 1.23+'s official log/slog, slog adds enterprise-grade features:
// DLP data desensitization (36 sensitive types), tiered object pools, log subscription
// with backpressure control, modular extensions, and runtime dynamic switches.
//
// # Installation
//
//	go get github.com/darkit/slog@latest
//
// # Quick Start
//
//	logger := slog.Default()
//	logger.Info("服务启动", "port", 8080, "env", "production")
//
// # Log Levels
//
// Six levels from lowest to highest:
//
//	LevelTrace = -8   // Most detailed
//	LevelDebug = -4
//	LevelInfo  = 0    // Default
//	LevelWarn  = 4
//	LevelError = 8
//	LevelFatal = 12   // Calls os.Exit(1)
//
// Set levels globally:
//
//	slog.SetLevelDebug()
//	slog.SetLevel("info")
//	slog.SetLevel(slog.LevelWarn)
//
// # Creating Loggers
//
// Builder pattern (recommended):
//
//	logger := slog.NewLoggerBuilder().
//	    WithModule("order-service").
//	    WithGroup("http").
//	    EnableJSON(true).
//	    EnableDLP(true).
//	    Build()
//
// Direct creation:
//
//	logger := slog.NewLoggerWithConfig(os.Stdout, &slog.Config{
//	    EnableText: boolPtr(true),
//	    EnableJSON: boolPtr(true),
//	    NoColor:    true,
//	})
//
// Output format variants:
//
//	logger := slog.NewLoggerBuilder().UseLogfmt().Build()    // Loki / Vector
//	logger := slog.NewLoggerBuilder().UseGELF(nil).Build()   // Graylog
//	logger := slog.NewLoggerBuilder().UseNetOutput(opts).Build() // TCP/UDP
//
// # DLP Data Desensitization
//
// Enable globally:
//
//	slog.EnableDLPLogger()
//
// Engine-level usage:
//
//	engine := dlp.NewDlpEngine()
//	engine.Enable()
//	masked := engine.DesensitizeText("手机号：13812345678") // → 手机号：138****5678
//
// Struct tag desensitization:
//
//	type User struct {
//	    Name  string `dlp:"chinese_name"`
//	    Phone string `dlp:"mobile_phone"`
//	    Email string `dlp:"email"`
//	}
//
// Matcher management:
//
//	engine.DisableMatchers("ipv4", "ipv6")  // Disable (variadic)
//	engine.EnableMatchers("ipv4")            // Re-enable
//	engine.SetMatcherEnabled("email", false) // Toggle single
//	engine.EnabledMatchers()                 // List all enabled
//
// Supported types: chinese_name, id_card, mobile_phone, email, bank_card,
// ipv4, ipv6, mac, url, domain, jwt, uuid, md5, sha256, plate, vin,
// imei, license_plate, postal_code, address, password, username,
// api_key, access_token, passport, social_security, credit_card,
// iban, swift, lat_lng, medical_id, company_id, git_repo, device_id.
//
// # Runtime Control
//
//	snapshot := slog.GetRuntimeSnapshot()
//	slog.ApplyRuntimeOption("level", "warn")
//	slog.ApplyRuntimeOption("json", "on")
//	slog.ApplyRuntimeOption("dlp", "on")
//
// # Subscription
//
//	ch, cancel := slog.Subscribe(1000)
//	defer cancel()
//
//	ch, cancel = slog.SubscribeWithOptions(slog.SubscribeOptions{
//	    BufferSize:   1000,
//	    Backpressure: slog.SubscriptionDropOldest, // DropOldest / DropNewest / BlockWithTimeout
//	})
//
// # Context Propagation
//
//	slog.SetContextPropagator(func(ctx context.Context) []slog.Attr {
//	    if v, ok := ctx.Value("trace_id").(string); ok {
//	        return []slog.Attr{slog.String("trace_id", v)}
//	    }
//	    return nil
//	})
//
// # Thread Safety
//
// All operations are goroutine-safe. Logger instances can be freely shared.
//
// # Performance
//
//	DLP cache hit:       ~46 ns/op
//	Cache key (xxhash64): ~314 ns/op
//	Memory reuse rate:   95%+ (tiered object pools)
//
// For more information, see https://pkg.go.dev/github.com/darkit/slog
package slog
