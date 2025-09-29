package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	darkit "github.com/darkit/slog"
)

// PerformanceDemo 性能演示
func main() {
	fmt.Println("🚀 Go日志库性能演示")
	fmt.Println("==================")

	// 1. 简单日志性能演示
	fmt.Println("\n📊 1. 简单日志性能测试")
	testSimpleLogging()

	// 2. 结构化日志演示
	fmt.Println("\n📋 2. 结构化日志测试")
	testStructuredLogging()

	// 3. DLP功能演示
	fmt.Println("\n🔒 3. DLP数据脱敏演示")
	testDLPFeature()

	// 4. 可视化功能演示
	fmt.Println("\n🎨 4. 可视化功能演示")
	testVisualizationFeatures()

	// 5. 并发性能演示
	fmt.Println("\n⚡ 5. 并发性能测试")
	testConcurrentPerformance()
}

// testSimpleLogging 简单日志性能测试
func testSimpleLogging() {
	iterations := 100000
	message := "Simple log message for performance test"

	// darkit/slog
	fmt.Print("darkit/slog: ")
	start := time.Now()
	darkitLogger := darkit.NewLogger(os.Stdout, true, false)
	for i := 0; i < iterations; i++ {
		darkitLogger.Info(message)
	}
	darkitTime := time.Since(start)
	fmt.Printf("%.2fms (%.0f ops/sec)\n",
		float64(darkitTime.Nanoseconds())/1e6,
		float64(iterations)/darkitTime.Seconds())

	// log/slog
	fmt.Print("log/slog:    ")
	start = time.Now()
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	for i := 0; i < iterations; i++ {
		slogLogger.Info(message)
	}
	slogTime := time.Since(start)
	fmt.Printf("%.2fms (%.0f ops/sec)\n",
		float64(slogTime.Nanoseconds())/1e6,
		float64(iterations)/slogTime.Seconds())

	// zap
	fmt.Print("zap:         ")
	start = time.Now()
	zapCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(zapCore)
	for i := 0; i < iterations; i++ {
		zapLogger.Info(message)
	}
	zapTime := time.Since(start)
	fmt.Printf("%.2fms (%.0f ops/sec)\n",
		float64(zapTime.Nanoseconds())/1e6,
		float64(iterations)/zapTime.Seconds())

	// logrus
	fmt.Print("logrus:      ")
	start = time.Now()
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(os.Stdout)
	logrusLogger.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	for i := 0; i < iterations; i++ {
		logrusLogger.Info(message)
	}
	logrusTime := time.Since(start)
	fmt.Printf("%.2fms (%.0f ops/sec)\n",
		float64(logrusTime.Nanoseconds())/1e6,
		float64(iterations)/logrusTime.Seconds())

	// 性能对比
	fmt.Printf("\n相对性能 (以darkit/slog为基准):\n")
	fmt.Printf("- darkit/slog: 1.00x (基准)\n")
	fmt.Printf("- log/slog:    %.2fx\n", float64(slogTime)/float64(darkitTime))
	fmt.Printf("- zap:         %.2fx\n", float64(zapTime)/float64(darkitTime))
	fmt.Printf("- logrus:      %.2fx\n", float64(logrusTime)/float64(darkitTime))
}

// testStructuredLogging 结构化日志测试
func testStructuredLogging() {
	fmt.Println("比较结构化日志性能...")

	userID := 12345
	action := "login"
	timestamp := time.Now()
	duration := 150 * time.Millisecond

	// darkit/slog 结构化日志
	fmt.Println("\ndarkit/slog 结构化日志:")
	darkitLogger := darkit.NewLogger(os.Stdout, false, false)
	darkit.EnableJSONLogger()
	darkitLogger.Info("User action performed",
		"user_id", userID,
		"action", action,
		"timestamp", timestamp,
		"duration", duration,
	)
	darkit.DisableJSONLogger()

	// log/slog 结构化日志
	fmt.Println("\nlog/slog 结构化日志:")
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slogLogger.Info("User action performed",
		"user_id", userID,
		"action", action,
		"timestamp", timestamp,
		"duration", duration,
	)

	// zap 结构化日志
	fmt.Println("\nzap 结构化日志:")
	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)
	zapLogger := zap.New(zapCore)
	zapLogger.Info("User action performed",
		zap.Int("user_id", userID),
		zap.String("action", action),
		zap.Time("timestamp", timestamp),
		zap.Duration("duration", duration),
	)
}

// testDLPFeature DLP功能演示
func testDLPFeature() {
	fmt.Println("darkit/slog 独有的DLP数据脱敏功能:")

	logger := darkit.NewLogger(os.Stdout, false, false)

	// 启用DLP功能
	darkit.EnableDLPLogger()
	defer darkit.DisableDLPLogger()

	fmt.Println("\n原始敏感数据:")
	fmt.Println("手机号: 13812345678")
	fmt.Println("邮箱: user@example.com")
	fmt.Println("身份证: 123456789012345678")
	fmt.Println("银行卡: 6222020000000000000")

	fmt.Println("\n脱敏后的日志输出:")
	logger.Info("用户注册信息",
		"phone", "13812345678",
		"email", "user@example.com",
		"id_card", "123456789012345678",
		"bank_card", "6222020000000000000",
	)

	fmt.Println("\n✨ 其他日志库都不具备此功能！")
}

// testVisualizationFeatures 可视化功能演示
func testVisualizationFeatures() {
	fmt.Println("darkit/slog 独有的可视化功能:")

	logger := darkit.NewLogger(os.Stdout, false, false)

	// 进度条演示
	fmt.Println("\n1. 进度条功能:")
	logger.ProgressBarWithValue("处理进度", 75.5, 30)

	// 动态效果演示
	fmt.Println("\n2. 动态加载效果:")
	logger.Dynamic("处理中", 10, 100) // 10帧，每帧100ms

	fmt.Println("\n✨ 其他日志库都不具备可视化功能！")
}

// testConcurrentPerformance 并发性能测试
func testConcurrentPerformance() {
	fmt.Println("并发日志性能测试...")

	const goroutines = 10
	const iterations = 1000
	message := "Concurrent logging test"

	// darkit/slog 并发测试
	fmt.Print("darkit/slog 并发: ")
	start := time.Now()
	done := make(chan bool, goroutines)
	darkitLogger := darkit.NewLogger(os.Stdout, true, false)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				darkitLogger.Info(message, "worker_id", id, "iteration", j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
	darkitConcurrentTime := time.Since(start)
	fmt.Printf("%.2fms\n", float64(darkitConcurrentTime.Nanoseconds())/1e6)

	// log/slog 并发测试
	fmt.Print("log/slog 并发:    ")
	start = time.Now()
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				slogLogger.Info(message, "worker_id", id, "iteration", j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
	slogConcurrentTime := time.Since(start)
	fmt.Printf("%.2fms\n", float64(slogConcurrentTime.Nanoseconds())/1e6)

	fmt.Printf("\n并发性能对比:\n")
	fmt.Printf("- darkit/slog: %.2fms (基准)\n", float64(darkitConcurrentTime.Nanoseconds())/1e6)
	fmt.Printf("- log/slog:    %.2fms (%.2fx)\n",
		float64(slogConcurrentTime.Nanoseconds())/1e6,
		float64(slogConcurrentTime)/float64(darkitConcurrentTime))
}
