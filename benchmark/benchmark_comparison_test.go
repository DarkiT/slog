package slog

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	zslog "github.com/darkit/slog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BenchmarkSimpleMessage 简单消息性能对比
func BenchmarkSimpleMessage(b *testing.B) {
	message := "Simple log message for performance comparison"

	// 设置各种logger
	var darkitBuf, slogBuf, logrusBuf, zapBuf bytes.Buffer

	// darkit/slog (我们的库)
	darkitLogger := zslog.NewLogger(&darkitBuf, true, false) // noColor=true

	// log/slog (Go标准库)
	slogLogger := slog.New(slog.NewTextHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// logrus
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&logrusBuf)
	logrusLogger.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	logrusLogger.SetLevel(logrus.InfoLevel)

	// zap
	zapCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&zapBuf),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(zapCore)

	b.Run("darkit/slog", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.Info(message)
		}
	})

	b.Run("log/slog", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slogLogger.Info(message)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logrusLogger.Info(message)
		}
	})

	b.Run("zap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			zapLogger.Info(message)
		}
	})
}

// BenchmarkStructuredLogging 结构化日志性能对比
func BenchmarkStructuredLogging(b *testing.B) {
	message := "User action performed"
	userID := 12345
	action := "login"
	timestamp := time.Now()

	var darkitBuf, slogJSONBuf, logrusJSONBuf, zapJSONBuf bytes.Buffer

	// darkit/slog JSON模式
	darkitLogger := zslog.NewLogger(&darkitBuf, true, false)
	zslog.EnableJSONLogger()
	zslog.DisableTextLogger()
	defer func() {
		zslog.EnableTextLogger()
		zslog.DisableJSONLogger()
	}()

	// log/slog JSON
	slogJSONLogger := slog.New(slog.NewJSONHandler(&slogJSONBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// logrus JSON
	logrusJSONLogger := logrus.New()
	logrusJSONLogger.SetOutput(&logrusJSONBuf)
	logrusJSONLogger.SetFormatter(&logrus.JSONFormatter{})
	logrusJSONLogger.SetLevel(logrus.InfoLevel)

	// zap JSON
	zapJSONCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapJSONBuf),
		zapcore.InfoLevel,
	)
	zapJSONLogger := zap.New(zapJSONCore)

	b.Run("darkit/slog-json", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.Info(message,
				"user_id", userID,
				"action", action,
				"timestamp", timestamp,
			)
		}
	})

	b.Run("log/slog-json", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slogJSONLogger.Info(message,
				"user_id", userID,
				"action", action,
				"timestamp", timestamp,
			)
		}
	})

	b.Run("logrus-json", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logrusJSONLogger.WithFields(logrus.Fields{
				"user_id":   userID,
				"action":    action,
				"timestamp": timestamp,
			}).Info(message)
		}
	})

	b.Run("zap-json", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			zapJSONLogger.Info(message,
				zap.Int("user_id", userID),
				zap.String("action", action),
				zap.Time("timestamp", timestamp),
			)
		}
	})
}

// BenchmarkDLPFeature DLP功能性能影响测试
func BenchmarkDLPFeature(b *testing.B) {
	var darkitBuf, slogBuf bytes.Buffer

	darkitLogger := zslog.NewLogger(&darkitBuf, true, false)
	slogLogger := slog.New(slog.NewJSONHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// 包含敏感信息的数据
	sensitiveMessage := "User registration"
	phone := "13812345678"
	email := "user@example.com"
	idCard := "123456789012345678"

	b.Run("darkit/slog-without-dlp", func(b *testing.B) {
		zslog.DisableDLPLogger() // 确保DLP关闭
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.Info(sensitiveMessage,
				"phone", phone,
				"email", email,
				"id_card", idCard,
			)
		}
	})

	b.Run("darkit/slog-with-dlp", func(b *testing.B) {
		zslog.EnableDLPLogger() // 启用DLP功能
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.Info(sensitiveMessage,
				"phone", phone,
				"email", email,
				"id_card", idCard,
			)
		}
		zslog.DisableDLPLogger() // 测试后关闭
	})

	b.Run("log/slog-baseline", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slogLogger.Info(sensitiveMessage,
				"phone", phone,
				"email", email,
				"id_card", idCard,
			)
		}
	})
}

// BenchmarkMemoryAllocation 内存分配对比
func BenchmarkMemoryAllocation(b *testing.B) {
	var darkitBuf, slogBuf, zapBuf bytes.Buffer

	darkitLogger := zslog.NewLogger(&darkitBuf, true, false)
	slogLogger := slog.New(slog.NewJSONHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapBuf),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(zapCore)

	complexData := map[string]interface{}{
		"user_id":   12345,
		"username":  "testuser",
		"email":     "user@example.com",
		"roles":     []string{"admin", "user"},
		"metadata":  map[string]string{"region": "us-east-1", "env": "prod"},
		"timestamp": time.Now(),
		"active":    true,
	}

	b.Run("darkit/slog-memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.Info("Complex data logging",
				"iteration", i,
				"data", complexData,
			)
		}
	})

	b.Run("log/slog-memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slogLogger.Info("Complex data logging",
				"iteration", i,
				"data", complexData,
			)
		}
	})

	b.Run("zap-memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			zapLogger.Info("Complex data logging",
				zap.Int("iteration", i),
				zap.Any("data", complexData),
			)
		}
	})
}

// BenchmarkConcurrentLogging 并发日志性能测试
func BenchmarkConcurrentLogging(b *testing.B) {
	message := "Concurrent logging test"

	b.Run("darkit/slog-concurrent", func(b *testing.B) {
		var darkitBuf bytes.Buffer
		darkitLogger := zslog.NewLogger(&darkitBuf, true, false)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				darkitLogger.Info(message, "worker_id", "worker-123", "iteration", 1)
			}
		})
	})

	b.Run("log/slog-concurrent", func(b *testing.B) {
		var slogBuf bytes.Buffer
		slogLogger := slog.New(slog.NewJSONHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				slogLogger.Info(message, "worker_id", "worker-123", "iteration", 1)
			}
		})
	})

	// Zap需要特殊处理并发安全
	b.Run("zap-concurrent", func(b *testing.B) {
		// 使用无缓冲输出避免并发问题
		zapLogger := zap.NewNop() // No-op logger for concurrent testing
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				zapLogger.Info(message,
					zap.String("worker_id", "worker-123"),
					zap.Int("iteration", 1),
				)
			}
		})
	})
}

// BenchmarkWithContext 上下文日志性能测试
func BenchmarkWithContext(b *testing.B) {
	var darkitBuf, slogBuf bytes.Buffer

	darkitLogger := zslog.NewLogger(&darkitBuf, true, false)
	slogLogger := slog.New(slog.NewJSONHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	message := "Context-aware logging"

	b.Run("darkit/slog-with-context", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			darkitLogger.With("request_id", "req-123", "trace_id", "trace-456").Info(message)
		}
	})

	b.Run("log/slog-with-context", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slogLogger.With("request_id", "req-123", "trace_id", "trace-456").Info(message)
		}
	})
}
