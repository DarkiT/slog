package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/darkit/slog"
	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/modules"
	_ "github.com/darkit/slog/modules/formatter" // 自动注册formatter模块
	_ "github.com/darkit/slog/modules/multi"     // 自动注册multi模块
	_ "github.com/darkit/slog/modules/syslog"    // 自动注册syslog模块
	_ "github.com/darkit/slog/modules/webhook"   // 自动注册webhook模块
)

func init() {
	// 初始化日志设置
	slog.EnableTextLogger()           // 启用文本日志
	time.Sleep(50 * time.Millisecond) // 等待初始化完成
}

func main() {
	fmt.Println("🚀 darkit/slog 综合功能演示")
	fmt.Println(strings.Repeat("=", 60))

	// 创建主logger
	logger := slog.NewLogger(os.Stdout, false, false)

	// 定义演示项目
	demos := []struct {
		name        string
		description string
		fn          func()
	}{
		{
			name:        "基础日志功能",
			description: "演示所有日志级别和基本功能",
			fn:          demoBasicLogging,
		},
		{
			name:        "结构化日志",
			description: "演示结构化字段和格式化日志",
			fn:          func() { demoStructuredLogging(logger) },
		},
		{
			name:        "动态级别控制",
			description: "演示生产环境动态级别切换",
			fn:          demoDynamicLevelControl,
		},
		{
			name:        "DLP数据脱敏",
			description: "演示敏感信息脱敏功能",
			fn:          demoDLPMasking,
		},
		{
			name:        "模块注册系统",
			description: "演示模块注册中心和各种使用方式",
			fn:          demoModuleSystem,
		},
		{
			name:        "异步日志处理",
			description: "演示异步日志和订阅者模式",
			fn:          demoAsyncLogging,
		},
		{
			name:        "性能基准测试",
			description: "演示各种场景下的性能表现",
			fn:          demoPerformanceTests,
		},
		{
			name:        "上下文和追踪",
			description: "演示上下文传递和链路追踪",
			fn:          func() { demoContextAndTracing(logger) },
		},
		{
			name:        "错误处理",
			description: "演示错误日志和异常处理",
			fn:          func() { demoErrorHandling(logger) },
		},
		{
			name:        "生产环境场景",
			description: "演示真实生产环境使用场景",
			fn:          demoProductionScenarios,
		},
	}

	// 执行所有演示
	for i, demo := range demos {
		fmt.Printf("\n📋 [%d/%d] %s\n", i+1, len(demos), demo.name)
		fmt.Printf("📝 %s\n", demo.description)
		fmt.Println(strings.Repeat("-", 40))

		demo.fn()

		fmt.Printf("✅ %s 演示完成\n", demo.name)
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Printf("\n🎉 所有演示完成！\n")
	fmt.Printf("📚 更多信息请查看项目文档和代码注释\n")
}

// 基础日志功能演示
func demoBasicLogging() {
	fmt.Println("🎯 日志级别演示:")

	// 设置为最详细级别
	slog.SetLevelTrace()

	// 不同级别的日志
	slog.Trace("最详细的追踪信息 - 通常用于复杂问题诊断")
	slog.Debug("调试信息 - 开发阶段使用")
	slog.Info("普通信息 - 业务流程记录")
	slog.Warn("警告信息 - 需要注意但不影响运行")
	slog.Error("错误信息 - 发生错误但程序可继续")

	fmt.Println("\n🎨 格式化日志:")
	username := "张三"
	userID := 12345
	loginTime := time.Now()

	slog.Infof("用户 %s (ID: %d) 在 %s 登录成功",
		username, userID, loginTime.Format("15:04:05"))
	slog.Warnf("用户 %s 连续登录失败 %d 次", username, 3)
	slog.Errorf("用户 %s 权限验证失败: %v", username, "无效令牌")

	fmt.Println("\n✨ 动态效果:")
	slog.Dynamic("系统初始化", 8, 200)
	slog.Progress("加载配置", 1500)
	slog.Countdown("服务启动", 3)
	slog.Loading("连接数据库", 2)
}

// 结构化日志演示
func demoStructuredLogging(logger *slog.Logger) {
	fmt.Println("🏗️ 结构化字段:")

	// 用户操作场景
	logger.Info("用户操作事件",
		"user_id", 12345,
		"username", "张三",
		"action", "查询订单",
		"ip_address", "192.168.1.100",
		"user_agent", "Mozilla/5.0 Chrome/91.0",
		"timestamp", time.Now(),
		"session_id", "sess_abc123",
	)

	// API请求场景
	logger.Info("API请求处理",
		"method", "POST",
		"endpoint", "/api/orders",
		"status_code", 200,
		"response_time_ms", 245,
		"request_size", 1024,
		"response_size", 2048,
	)

	// 系统监控场景
	logger.Warn("系统资源监控",
		"cpu_usage", 78.5,
		"memory_usage", 65.2,
		"disk_usage", 45.8,
		"active_connections", 150,
		"queue_size", 25,
	)

	fmt.Println("\n📊 业务指标:")
	logger.Info("订单处理完成",
		"order_id", "ORD-2024-001",
		"customer_id", 9876,
		"amount", 299.99,
		"currency", "CNY",
		"payment_method", "微信支付",
		"processing_time", 3.2,
	)
}

// 动态级别控制演示
func demoDynamicLevelControl() {
	fmt.Println("🎚️ 生产环境级别切换场景:")

	// 1. 生产模式 - 只记录重要信息
	fmt.Println("\n1. 生产模式启动 (level: error)")
	slog.SetLevel("error")
	fmt.Printf("   当前级别: %v\n", slog.GetLevel())

	fmt.Println("   正常业务运行:")
	simulateBusinessOperations("生产模式")

	// 2. 发现异常 - 开启详细日志
	fmt.Println("\n2. 发现异常，开启调试模式 (level: debug)")
	slog.SetLevel("debug")
	fmt.Printf("   当前级别: %v\n", slog.GetLevel())

	fmt.Println("   详细排查模式:")
	simulateBusinessOperations("调试模式")

	// 3. 恢复生产模式
	fmt.Println("\n3. 问题解决，恢复生产模式 (level: error)")
	slog.SetLevel("error")
	fmt.Printf("   当前级别: %v\n", slog.GetLevel())

	fmt.Println("   恢复正常运行:")
	simulateBusinessOperations("恢复模式")
}

// DLP数据脱敏演示
func demoDLPMasking() {
	fmt.Println("🔒 敏感信息脱敏功能:")

	// 启用DLP引擎
	dlpEngine := dlp.NewDlpEngine()
	dlpEngine.Enable()

	// 清除可能存在的缓存，确保测试准确
	dlpEngine.ClearCache()

	// 测试各种敏感信息
	testData := []struct {
		name string
		data string
	}{
		{"手机号", "用户手机号：13812345678"},
		{"邮箱地址", "邮箱：zhangsan@company.com"},
		{"身份证号", "身份证：110101199001011237"},
		{"银行卡号", "工商银行卡：6222021234567890123"},
		{"IP地址", "客户端IP：192.168.1.100"},
		{"网址链接", "访问地址：https://www.example.com/api?token=123456789"},
		{"中文姓名", "客户姓名：张三丰"},
		{"综合信息", "张三(13812345678)使用银行卡6222021234567890123支付"},
	}

	fmt.Println("\n脱敏前后对比:")
	for _, test := range testData {
		original := test.data
		masked := dlpEngine.DesensitizeText(original)

		fmt.Printf("   %s:\n", test.name)
		fmt.Printf("     原文: %s\n", original)
		fmt.Printf("     脱敏: %s\n", masked)
		fmt.Println()
	}

	// 结构体脱敏演示
	fmt.Println("📋 结构体脱敏:")
	type UserInfo struct {
		Name     string `dlp:"chinese_name"`
		Phone    string `dlp:"mobile_phone"`
		Email    string `dlp:"email"`
		BankCard string `dlp:"bank_card"`
		IDCard   string `dlp:"id_card"`
	}

	user := UserInfo{
		Name:     "李四",
		Phone:    "13987654321",
		Email:    "lisi@example.com",
		BankCard: "6217001234567890123",
		IDCard:   "440301199001011234",
	}

	fmt.Printf("   脱敏前: %+v\n", user)
	dlpEngine.DesensitizeStruct(&user)
	fmt.Printf("   脱敏后: %+v\n", user)
}

// 模块注册系统演示
func demoModuleSystem() {
	fmt.Println("🔧 模块注册中心:")

	// 查看已注册的模块工厂
	registry := modules.GetRegistry()
	factories := registry.ListFactories()
	fmt.Printf("   已注册工厂数量: %d\n", len(factories))
	for _, name := range factories {
		fmt.Printf("     ✓ %s\n", name)
	}

	// 查看已创建的模块实例
	moduleInstances := registry.List()
	fmt.Printf("   已创建模块实例数量: %d\n", len(moduleInstances))

	fmt.Println("\n🚀 模块使用方式:")

	// 方式1: 快速启用单个模块
	fmt.Println("   1. 快速启用:")
	logger1 := slog.UseFactory("formatter", modules.Config{
		"type": "time",
	}).Build()
	if logger1 != nil {
		logger1.Info("使用时间格式化模块",
			"timestamp", time.Now().Format("2006-01-02 15:04:05"))
	}

	// 方式2: 配置驱动方式
	fmt.Println("\n   2. 配置驱动:")
	configs := []modules.ModuleConfig{
		{
			Type:     "formatter",
			Name:     "time-fmt",
			Enabled:  true,
			Priority: 10,
			Config: modules.Config{
				"type": "time",
			},
		},
		{
			Type:     "multi",
			Name:     "multi-output",
			Enabled:  true,
			Priority: 20,
			Config: modules.Config{
				"outputs": []string{"stdout", "file"},
			},
		},
	}

	logger2 := slog.UseConfig(configs).Build()
	if logger2 != nil {
		logger2.Info("配置驱动创建的Logger")
		logger2.Warn("支持多种模块组合")
	}

	fmt.Println("\n   3. 链式调用语法示例:")
	fmt.Println("      slog.EnableFormatter(\"time\").EnableMulti(config).Build()")
	fmt.Println("      (为避免副作用，此处仅展示语法)")
}

// 异步日志处理演示
func demoAsyncLogging() {
	fmt.Println("⚡ 异步日志处理:")

	// 创建订阅者
	records, cancel := slog.Subscribe(1000)
	defer cancel()

	// 启动异步消费者
	var wg sync.WaitGroup
	processedCount := 0

	wg.Add(1)
	go func() {
		defer wg.Done()
		for record := range records {
			processedCount++
			// 模拟处理日志记录
			if processedCount <= 5 { // 只打印前5条
				fmt.Printf("   处理日志: [%s] %s\n",
					record.Level, record.Message)
			}
		}
	}()

	// 生产日志记录
	logger := slog.NewLogger(&bytes.Buffer{}, false, false)

	fmt.Println("   生产日志记录...")
	for i := 0; i < 20; i++ {
		logger.Info("异步处理测试",
			"序号", i,
			"时间", time.Now().Format("15:04:05.000"))
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)
	cancel() // 关闭通道
	wg.Wait()

	fmt.Printf("   ✅ 共处理 %d 条日志记录\n", processedCount)
}

// 性能基准测试演示
func demoPerformanceTests() {
	fmt.Println("📊 性能基准测试:")

	// 基础性能测试
	fmt.Println("\n   1. 基础日志性能:")
	testBasicPerformance()

	// 并发性能测试
	fmt.Println("\n   2. 并发性能测试:")
	testConcurrencyPerformance()

	// 内存使用测试
	fmt.Println("\n   3. 内存使用测试:")
	testMemoryUsage()
}

// 上下文和追踪演示
func demoContextAndTracing(logger *slog.Logger) {
	fmt.Println("📋 上下文传递和链路追踪:")

	// 设置合适的日志级别以显示所有日志
	slog.SetLevel("debug")

	// 创建带追踪ID的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "trace-"+fmt.Sprintf("%d", time.Now().Unix()))
	ctx = context.WithValue(ctx, "user_id", "user-12345")

	// 模拟API调用链
	fmt.Println("\n   API调用链路:")

	// 1. 接收请求
	logger.Info("收到API请求",
		"trace_id", ctx.Value("trace_id"),
		"user_id", ctx.Value("user_id"),
		"endpoint", "/api/orders",
		"method", "POST",
	)

	// 2. 验证权限
	logger.Debug("验证用户权限",
		"trace_id", ctx.Value("trace_id"),
		"user_id", ctx.Value("user_id"),
		"permission", "order:create",
	)

	// 3. 数据库操作
	logger.Debug("执行数据库操作",
		"trace_id", ctx.Value("trace_id"),
		"user_id", ctx.Value("user_id"),
		"operation", "INSERT INTO orders",
		"duration_ms", 45,
	)

	// 4. 返回响应
	logger.Info("API请求完成",
		"trace_id", ctx.Value("trace_id"),
		"user_id", ctx.Value("user_id"),
		"status_code", 201,
		"total_duration_ms", 128,
	)
}

// 错误处理演示
func demoErrorHandling(logger *slog.Logger) {
	fmt.Println("🚨 错误处理和日志:")

	// 模拟各种错误场景
	errors := []struct {
		scenario string
		err      error
		context  map[string]interface{}
	}{
		{
			scenario: "数据库连接失败",
			err:      fmt.Errorf("connection timeout after 5s"),
			context: map[string]interface{}{
				"host":     "db.example.com",
				"port":     3306,
				"database": "orders",
				"retries":  3,
			},
		},
		{
			scenario: "API调用失败",
			err:      fmt.Errorf("HTTP 503 Service Unavailable"),
			context: map[string]interface{}{
				"url":           "https://api.payment.com/charge",
				"method":        "POST",
				"timeout":       "30s",
				"response_code": 503,
			},
		},
		{
			scenario: "文件操作失败",
			err:      fmt.Errorf("permission denied"),
			context: map[string]interface{}{
				"file_path":  "/var/log/app.log",
				"operation":  "write",
				"file_size":  "125MB",
				"free_space": "256MB",
			},
		},
	}

	fmt.Println("\n   错误场景处理:")
	for i, errCase := range errors {
		fmt.Printf("\n   场景 %d: %s\n", i+1, errCase.scenario)

		// 记录错误日志，包含丰富的上下文信息
		fields := []interface{}{"error", errCase.err.Error()}
		for key, value := range errCase.context {
			fields = append(fields, key, value)
		}

		logger.Error(errCase.scenario, fields...)

		// 记录恢复操作
		logger.Info("错误恢复操作",
			"action", "fallback_mechanism",
			"status", "attempting_recovery",
		)
	}
}

// 生产环境场景演示
func demoProductionScenarios() {
	fmt.Println("🏭 生产环境真实场景:")

	// 设置合适的日志级别以显示所有日志
	slog.SetLevel("debug")

	scenarios := []struct {
		name string
		fn   func()
	}{
		{"Web服务请求处理", simulateWebRequest},
		{"数据库事务处理", simulateDatabaseTransaction},
		{"微服务通信", simulateMicroserviceCall},
		{"定时任务执行", simulateScheduledJob},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n   📋 %s:\n", scenario.name)
		scenario.fn()
	}
}

// 辅助函数实现

func simulateBusinessOperations(mode string) {
	slog.Debug("解析请求参数", "mode", mode, "user_id", 12345)
	slog.Info("处理业务逻辑", "mode", mode, "operation", "query_orders")
	slog.Warn("系统负载较高", "mode", mode, "cpu_usage", "85%")
	slog.Error("处理失败", "mode", mode, "error", "database_timeout")
}

func testBasicPerformance() {
	logger := slog.NewLogger(io.Discard, false, false)
	iterations := 10000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		logger.Info("性能测试", "iteration", i, "data", "test_payload")
	}
	duration := time.Since(start)

	opsPerSec := float64(iterations) / duration.Seconds()
	fmt.Printf("     %d 次操作耗时: %v\n", iterations, duration)
	fmt.Printf("     性能: %.0f ops/sec\n", opsPerSec)
}

func testConcurrencyPerformance() {
	logger := slog.NewLogger(io.Discard, false, false)
	goroutines := 4
	iterationsPerGoroutine := 2500

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				logger.Info("并发测试", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOps := goroutines * iterationsPerGoroutine

	fmt.Printf("     %d 协程 x %d 操作耗时: %v\n",
		goroutines, iterationsPerGoroutine, duration)
	fmt.Printf("     并发性能: %.0f ops/sec\n",
		float64(totalOps)/duration.Seconds())
}

func testMemoryUsage() {
	logger := slog.NewLogger(io.Discard, false, false)
	iterations := 5000

	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < iterations; i++ {
		logger.Info("内存测试",
			"iteration", i,
			"timestamp", time.Now(),
			"data", fmt.Sprintf("payload_%d", i),
		)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	memUsed := m2.TotalAlloc - m1.TotalAlloc
	memUsedMB := float64(memUsed) / 1024 / 1024
	bytesPerOp := float64(memUsed) / float64(iterations)

	fmt.Printf("     %d 次操作内存分配: %.2f MB\n", iterations, memUsedMB)
	fmt.Printf("     平均每次操作: %.2f bytes\n", bytesPerOp)
}

func simulateWebRequest() {
	logger := slog.NewLogger(os.Stdout, false, false)

	requestID := fmt.Sprintf("req_%d", time.Now().Unix())

	logger.Info("接收HTTP请求",
		"request_id", requestID,
		"method", "POST",
		"path", "/api/users",
		"ip", "192.168.1.100",
		"user_agent", "Chrome/91.0",
	)

	logger.Debug("路由匹配",
		"request_id", requestID,
		"handler", "UserController.Create",
		"middleware", []string{"auth", "ratelimit", "cors"},
	)

	logger.Info("请求处理完成",
		"request_id", requestID,
		"status_code", 201,
		"response_time_ms", 156,
		"response_size", 245,
	)
}

func simulateDatabaseTransaction() {
	logger := slog.NewLogger(os.Stdout, false, false)

	txID := fmt.Sprintf("tx_%d", time.Now().Unix())

	logger.Debug("开启数据库事务",
		"transaction_id", txID,
		"isolation_level", "READ_COMMITTED",
	)

	logger.Debug("执行SQL语句",
		"transaction_id", txID,
		"sql", "INSERT INTO users (name, email) VALUES (?, ?)",
		"params", []string{"张三", "zhangsan@example.com"},
		"execution_time_ms", 23,
	)

	logger.Info("事务提交成功",
		"transaction_id", txID,
		"affected_rows", 1,
		"total_time_ms", 45,
	)
}

func simulateMicroserviceCall() {
	logger := slog.NewLogger(os.Stdout, false, false)

	callID := fmt.Sprintf("call_%d", time.Now().Unix())

	logger.Info("调用下游服务",
		"call_id", callID,
		"service", "user-service",
		"endpoint", "/internal/users/validate",
		"timeout", "5s",
	)

	logger.Debug("服务响应",
		"call_id", callID,
		"status_code", 200,
		"response_time_ms", 89,
		"cache_hit", true,
	)
}

func simulateScheduledJob() {
	logger := slog.NewLogger(os.Stdout, false, false)

	jobID := fmt.Sprintf("job_%d", time.Now().Unix())

	logger.Info("定时任务开始",
		"job_id", jobID,
		"job_name", "数据同步任务",
		"schedule", "0 */10 * * * *", // 每10分钟
		"trigger", "cron",
	)

	logger.Debug("处理数据批次",
		"job_id", jobID,
		"batch_size", 1000,
		"processed", 856,
		"errors", 3,
	)

	logger.Info("定时任务完成",
		"job_id", jobID,
		"duration_sec", 45,
		"status", "success",
		"next_run", time.Now().Add(10*time.Minute).Format("15:04:05"),
	)
}
