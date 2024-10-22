package main

import (
	"context"
	"os"
	"time"

	"github.com/darkit/slog"
)

func main() {
	// 1. 创建日志记录器
	// noColor: false 表示启用颜色输出
	// addSource: true 表示显示源代码位置
	logger := slog.NewLogger(os.Stdout, false, true)

	//logger := slog.Default("DEMO")

	// 2. 日志级别控制
	demoLogLevels()

	// 3. 基本日志记录
	demoBasicLogging(logger)

	// 4. 格式化日志
	demoFormattedLogging(logger)

	// 5. 结构化日志
	demoStructuredLogging(logger)

	// 6. 日志分组和模块化
	demoGroupsAndModules()

	// 7. 上下文和值传递
	demoContextAndValues(logger)

	// 8. 输出格式控制
	demoOutputFormat()

	// 9. 日志脱敏
	demoSensitiveData()

	// 10. 日志监控
	demoLogMonitoring()

	// 11. 高级特性
	demoAdvancedFeatures(logger)

	// 添加前缀示例
	demoPrefixAndFormatting()
}

// demoLogLevels 演示日志级别控制
func demoLogLevels() {
	// 设置全局日志级别
	slog.SetLevelDebug() // 设置为Debug级别

	// 动态更新日志级别
	// 方式1：使用字符串
	_ = slog.UpdateLogLevel("debug")
	// 方式2：使用Level类型
	_ = slog.UpdateLogLevel(slog.LevelDebug)
	// 方式3：使用数字（-8: Trace, -4: Debug, 0: Info, 4: Warn, 8: Error, 12: Fatal）
	_ = slog.UpdateLogLevel(-4)

	// 获取当前日志级别
	currentLevel := slog.GetLevel()
	slog.Info("Current log level", "level", currentLevel)
}

// demoBasicLogging 演示基本日志记录
func demoBasicLogging(logger *slog.Logger) {
	_ = slog.UpdateLogLevel(slog.LevelTrace)
	// 不同级别的日志记录
	logger.Trace("这是一条跟踪日志") // 最详细的日志级别
	logger.Debug("这是一条调试日志") // 用于调试信息
	logger.Info("这是一条信息日志")  // 普通信息
	logger.Warn("这是一条警告日志")  // 警告信息
	logger.Error("这是一条错误日志") // 错误信息
	logger.Trace("这是一条路由日志") // 路由日志
	// logger.Fatal("这是一条致命错误日志") // 致命错误，会导致程序退出

	// 使用全局方法记录日志
	slog.Debug("使用全局Debug记录")
	slog.Info("使用全局Info记录")
	slog.Warn("使用全局Warn记录")
	slog.Error("使用全局Error记录")
}

// demoFormattedLogging 演示格式化日志
func demoFormattedLogging(logger *slog.Logger) {
	username := "张三"
	ip := "192.168.1.1"
	duration := 100

	// 使用格式化方法记录日志
	logger.Debugf("用户 %s 从 %s 登录", username, ip)
	logger.Infof("处理耗时 %d ms", duration)
	logger.Warnf("CPU使用率: %.2f%%", 75.5)
	logger.Errorf("连接失败: %v", "超时")

	// 兼容fmt风格的方法
	logger.Printf("这是一条Printf格式的日志 - %s", "测试")
	logger.Println("这是一条Println格式的日志")
}

// demoStructuredLogging 演示结构化日志
func demoStructuredLogging(logger *slog.Logger) {
	// 使用键值对记录结构化信息
	logger.Info("数据库连接成功",
		"host", "localhost",
		"port", 5432,
		"user", "admin",
		"database", "test_db",
	)

	// 使用slog.Attr类型记录
	logger.Info("用户操作",
		slog.String("action", "login"),
		slog.Int("user_id", 12345),
		slog.Time("timestamp", time.Now()),
		slog.Duration("process_time", time.Second*2),
		slog.Bool("success", true),
	)

	// 使用With方法添加固定字段
	dbLogger := logger.With(
		"component", "database",
		"version", "1.0",
	)
	dbLogger.Info("执行查询", "sql", "SELECT * FROM users")
}

// demoGroupsAndModules 演示日志分组和模块化
func demoGroupsAndModules() {
	// 创建不同模块的日志记录器
	userLogger := slog.Default("user")
	authLogger := slog.Default("auth")
	dbLogger := slog.Default("database")

	// 使用不同模块记录日志
	userLogger.Info("用户模块日志")
	authLogger.Warn("认证模块警告")
	dbLogger.Error("数据库模块错误")

	// 使用分组功能
	apiLogger := slog.WithGroup("api")
	apiLogger.Info("接收到请求",
		"method", "GET",
		"path", "/users",
		"ip", "127.0.0.1",
	)

	// 链式调用分组
	apiLogger.WithGroup("auth").With(
		"request_id", "req-123",
	).Info("用户认证成功")
}

// demoContextAndValues 演示上下文和值传递
func demoContextAndValues(logger *slog.Logger) {
	// 创建上下文
	ctx := context.Background()

	// 使用上下文
	ctxLogger := logger.WithContext(ctx)

	// 添加值到日志上下文
	ctxLogger = ctxLogger.WithValue("trace_id", "trace-123")
	ctxLogger = ctxLogger.WithValue("request_id", "req-456")

	// 使用带上下文的日志记录
	ctxLogger.Info("处理请求")
	ctxLogger.Debug("详细信息")
}

// demoOutputFormat 演示输出格式控制
func demoOutputFormat() {
	// 控制日志格式
	slog.EnableTextLogger()  // 启用文本日志
	slog.EnableJsonLogger()  // 启用JSON日志
	slog.DisableTextLogger() // 禁用文本日志
	slog.DisableJsonLogger() // 禁用JSON日志

	// 重新启用文本日志用于演示
	slog.EnableTextLogger()
}

// demoLogMonitoring 演示日志监控
func demoLogMonitoring() {
	// 监听日志级别变化
	slog.WatchLevel("monitor1", func(level slog.Level) {
		slog.Info("日志级别变化", "new_level", level)
	})

	// 触发级别变化
	slog.SetLevelDebug()
	slog.SetLevelInfo()

	// 取消监听
	slog.UnwatchLevel("monitor1")
}

// demoSensitiveData 演示日志脱敏功能
func demoSensitiveData() {
	// 启用日志脱敏功能
	slog.EnableDLP()

	// 注册自定义脱敏策略
	slog.RegisterDLPStrategy("custom", func(text string) string {
		if len(text) > 4 {
			return text[:2] + "***" + text[len(text)-2:]
		}
		return text
	})

	// 记录包含敏感信息的日志
	slog.Info("用户信息",
		"user_id", "12345678", // 会被自定义脱敏规则处理
		"phone", "13800138000", // 会被默认脱敏规则处理
		"normal", "public-info", // 普通信息不会被脱敏
	)

	// 检查脱敏功能是否启用
	if slog.IsDLPEnabled() {
		slog.Info("DLP is enabled")
	}

	// 手动执行文本脱敏
	maskedText := slog.DlpMask("13800138000")
	slog.Info("Masked text", "result", maskedText)

	// 禁用日志脱敏
	slog.DisableDLP()
}

// demoAdvancedFeatures 演示高级特性
func demoAdvancedFeatures(logger *slog.Logger) {
	// 获取原始slog.Logger
	slogLogger := logger.GetSlogLogger()
	slogLogger.Info("使用原始slog记录日志")

	// 获取日志记录通道
	recordChan := slog.GetChanRecord(1000) // 指定缓冲大小
	_ = recordChan                         // 在实际应用中可以用于异步处理日志

	// 使用Group和With的组合
	logger.WithGroup("api").
		With("version", "v1").
		WithGroup("auth").
		With("client_id", "123").
		Info("API调用")
}

// demoPrefixAndFormatting 演示日志前缀和格式化功能
func demoPrefixAndFormatting() {
	// 1. 基本前缀使用
	// Default方法会自动添加module前缀
	userLogger := slog.Default("user")          // [user] 日志内容
	apiLogger := slog.Default("api")            // [api] 日志内容
	dbLogger := slog.Default("db", "mysql")     // [db:mysql] 日志内容
	authLogger := slog.Default("auth", "oauth") // [auth:oauth] 日志内容

	// 记录带前缀的日志
	userLogger.Info("用户登录")    // 输出: [user] 用户登录
	apiLogger.Info("接收到请求")    // 输出: [api] 接收到请求
	dbLogger.Info("执行SQL查询")   // 输出: [db:mysql] 执行SQL查询
	authLogger.Info("验证token") // 输出: [auth:oauth] 验证token

	// 2. 多级模块前缀
	// 使用多个参数创建更具体的模块路径
	serviceLogger := slog.Default("service", "user", "register") // [service:user:register]
	serviceLogger.Info("处理注册请求")                                 // 输出: [service:user:register] 处理注册请求

	// 3. 组合使用前缀和结构化字段
	userLogger.Info("创建新用户",
		"user_id", 12345,
		"username", "zhangsan",
	)
	// 输出: [user] 创建新用户 user_id=12345 username=zhangsan

	// 4. 组合使用前缀和分组
	apiLogger.WithGroup("request").Info("处理API请求",
		"method", "POST",
		"path", "/api/v1/users",
		"duration", "100ms",
	)
	// 输出: [api] 处理API请求 request.method=POST request.path=/api/v1/users request.duration=100ms

	// 5. 前缀和上下文结合
	authLogger.WithValue("request_id", "req-123").
		Info("验证用户身份")
	// 输出: [auth:oauth] 验证用户身份 request_id=req-123

	// 6. 不同日志级别的前缀使用
	dbLogger.Debug("连接数据库") // 输出: [D] [db:mysql] 连接数据库
	dbLogger.Info("查询成功")   // 输出: [I] [db:mysql] 查询成功
	dbLogger.Warn("连接超时")   // 输出: [W] [db:mysql] 连接超时
	dbLogger.Error("查询失败")  // 输出: [E] [db:mysql] 查询失败

	// 7. 在其他功能中使用前缀
	serviceLogger.With(
		"trace_id", "trace-456",
		"user_id", "user-789",
	).Info("处理业务逻辑")
	// 输出: [service:user:register] 处理业务逻辑 trace_id=trace-456 user_id=user-789

	// 8. 前缀和格式化日志
	userLogger.Infof("用户 %s 的登录次数: %d", "zhangsan", 5)
	// 输出: [user] 用户 zhangsan 的登录次数: 5

	// 9. 链式调用中使用前缀
	apiLogger.
		WithGroup("metrics").
		With("latency", "50ms").
		Info("API性能统计")
	// 输出: [api] API性能统计 metrics.latency=50ms
}
