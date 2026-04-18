// Package slog provides a high-performance, feature-rich structured logging library
// for Go, extending the standard log/slog package.
//
// slog is built on Go 1.23+'s official log/slog package and provides enhanced
// features including flexible log level control, colored output, structured
// logging, and data loss prevention (DLP) capabilities.
//
// # Installation
//
//	go get github.com/darkit/slog@latest
//
// # Quick Start
//
//	package main
//
//	import (
//	    "os"
//	    "github.com/darkit/slog"
//	)
//
//	func main() {
//	    logger := slog.NewLogger(os.Stdout, false, false)
//	    logger.Info("Hello Slog!")
//
//	    // Structured logging
//	    logger.Info("User logged in",
//	        "user_id", 123,
//	        "action", "login",
//	    )
//	}
//
// # Log Levels
//
// slog supports six log levels, from lowest to highest:
//
//	LevelTrace = -8  // Most detailed
//	LevelDebug = -4  // Debug information
//	LevelInfo  = 0   // General info (default)
//	LevelWarn  = 4   // Warnings
//	LevelError = 8   // Errors
//	LevelFatal = 12  // Fatal errors (exits program)
//
// Set levels globally:
//
//	slog.SetLevelDebug()
//	slog.SetLevel("info")
//	slog.SetLevel(slog.LevelWarn)
//
// # Creating Loggers
//
// Basic creation:
//
//	logger := slog.NewLogger(os.Stdout, noColor, addSource)
//
// With configuration:
//
//	cfg := slog.DefaultConfig()
//	cfg.SetEnableText(true)
//	cfg.SetEnableJSON(true)
//	logger := slog.NewLoggerWithConfig(os.Stdout, cfg)
//
// Builder pattern:
//
//	logger := slog.NewLoggerBuilder().
//	    WithModule("order").
//	    WithGroup("api").
//	    EnableJSON(true).
//	    Build()
//
// # Structured Logging
//
//	logger.Info("Request processed",
//	    "method", "GET",
//	    "path", "/users",
//	    "duration", 150*time.Millisecond,
//	)
//
// # Data Loss Prevention (DLP)
//
// Enable DLP to automatically desensitize sensitive data:
//
//	slog.EnableDLPLogger()
//
//	type User struct {
//	    Name  string `dlp:"chinese_name"`
//	    Phone string `dlp:"mobile_phone"`
//	    Email string `dlp:"email"`
//	}
//
// Supported DLP types: Chinese name, ID card, phone number, email, bank card,
// address, password, license plate, IPv4/IPv6, JWT, URL.
//
// # File Logging
//
//	writer := slog.NewWriter("logs/app.log").
//	    SetMaxSize(100).      // MB
//	    SetMaxAge(7).         // Days
//	    SetMaxBackups(10).
//	    SetCompress(true)
//
//	logger := slog.NewLogger(writer, false, false)
//
// # Runtime Control
//
//	snapshot := slog.GetRuntimeSnapshot()
//	slog.ApplyRuntimeOption("level", "warn")
//	slog.ApplyRuntimeOption("json", "on")
//
// # Subscription
//
// Subscribe to log records for external processing:
//
//	ch, cancel := slog.Subscribe(1000)
//	defer cancel()
//
//	go func() {
//	    for event := range ch {
//	        _ = event.Record   // 结构化视图
//	        _ = event.Rendered // 当前激活输出对应的最终渲染结果
//	    }
//	}()
//
// # Thread Safety
//
// All slog operations are thread-safe. Logger instances can be safely shared
// across goroutines. Global configuration changes are also concurrency-safe.
//
// # Performance
//
// slog is optimized for high-performance scenarios:
//   - Tiered buffer pools for memory reuse
//   - LRU cache for format string detection
//   - xxhash64 for cache key generation
//   - Atomic operations for minimal lock contention
//
// For more information, see https://pkg.go.dev/github.com/darkit/slog
package slog
