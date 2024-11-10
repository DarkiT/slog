package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/darkit/slog"
)

var filename = "logs/app.log"

func init() {
	// 在包初始化时确保基础设施准备就绪
	slog.EnableTextLogger() // 启用文本日志
	// slog.EnableJsonLogger()            // 也启用JSON日志用于演示
	time.Sleep(100 * time.Millisecond) // 给一些初始化时间
}

func main() {
	write := os.Stdout
	// write := slog.NewWriter(filename)
	logger := slog.NewLogger(write, false, true)

	// 定义所有演示项目
	demos := []struct {
		name        string
		description string
		fn          func()
	}{
		{
			name:        "日志级别控制",
			description: "演示动态调整日志级别和监听级别变化",
			fn:          demoLogLevels,
		},
		{
			name:        "基本日志记录",
			description: "演示基本的日志记录功能",
			fn:          func() { demoBasicLogging(logger) },
		},
		{
			name:        "格式化日志",
			description: "演示不同格式的日志输出",
			fn:          func() { demoFormattedLogging(logger) },
		},
		{
			name:        "结构化日志",
			description: "演示结构化日志记录",
			fn:          func() { demoStructuredLogging(logger) },
		},
		{
			name:        "日志分组和模块化",
			description: "演示日志分组和模块管理",
			fn:          demoGroupsAndModules,
		},
		{
			name:        "上下文和值传递",
			description: "演示使用上下文和值传递",
			fn:          func() { demoContextAndValues(logger) },
		},
		{
			name:        "输出格式控制",
			description: "演示不同的日志输出格式",
			fn:          func() { demoOutputFormat(logger) },
		},
		{
			name:        "日志脱敏",
			description: "演示敏感信息脱敏功能",
			fn:          func() { demoSensitiveData(logger) },
		},
		{
			name:        "高级特性",
			description: "演示高级日志特性",
			fn:          func() { demoAdvancedFeatures(logger) },
		},
		{
			name:        "前缀和格式化",
			description: "演示日志前缀和格式化功能",
			fn:          demoPrefixAndFormatting,
		},
		{
			name:        "异步日志处理",
			description: "演示异步日志记录功能",
			fn:          demoAsyncLogging,
		},
		{
			name:        "链路追踪",
			description: "演示日志链路追踪功能",
			fn:          demoTracing,
		},
		{
			name:        "错误处理",
			description: "演示错误日志处理",
			fn:          demoErrorHandling,
		},
	}

	// 执行所有演示
	for _, demo := range demos {
		// 打印分隔线
		logger.Info("========================================")
		logger.Info("开始演示: "+demo.name,
			"description", demo.description,
		)

		// 执行演示函数
		demo.fn()

		// 打印完成信息
		logger.Info("演示完成: " + demo.name)

		// 暂停一会，让输出更清晰
		time.Sleep(200 * time.Millisecond)
	}
}

// demoLogLevels 演示日志级别控制
func demoLogLevels() {
	// 设置全局日志级别
	slog.SetLevelDebug() // 设置为Debug级别
	// 获取当前日志级别
	currentLevel := slog.GetLevel()
	slog.Trace("Current log level", "level", currentLevel)
	slog.Debug("Current log level", "level", currentLevel)
	slog.Info("Current log level", "level", currentLevel)
	slog.Warn("Current log level", "level", currentLevel)
	slog.Error("Current log level", "level", currentLevel)
}

// demoBasicLogging 演示基本日志记录
func demoBasicLogging(logger *slog.Logger) {
	slog.SetLevelTrace()
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
	// 模拟用户登录场景
	logger.Info("用户登录",
		"user_id", 12345,
		"ip", "192.168.1.100",
		"device", "iPhone",
		"location", "Beijing",
		"timestamp", time.Now(),
	)

	// 模拟API请求场景
	logger.Info("API请求处理",
		"method", "POST",
		"path", "/api/v1/orders",
		"duration_ms", 45,
		"status", 200,
		"client_ip", "10.0.0.1",
	)

	// 模拟系统监控场景
	logger.Info("系统状态",
		"cpu_usage", 65.5,
		"memory_used_mb", 1024,
		"disk_free_gb", 128,
		"network_in_mbps", 75.2,
		"network_out_mbps", 45.8,
	)
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
func demoOutputFormat(logger *slog.Logger) {
	// 1. 演示纯文本格式
	slog.EnableTextLogger()
	slog.DisableJsonLogger()
	logger.Info("=== 文本格式日志演示 ===")
	logger.Info("这是一条文本格式的日志",
		"field1", "value1",
		"field2", 123,
	)

	time.Sleep(100 * time.Millisecond)

	// 2. 演示JSON格式
	slog.DisableTextLogger()
	slog.EnableJsonLogger()
	logger.Info("=== JSON格式日志演示 ===")
	logger.Info("这是一条JSON格式的日志",
		"field1", "value1",
		"field2", 123,
	)

	time.Sleep(100 * time.Millisecond)

	// 3. 恢复默认设置
	slog.EnableTextLogger()
	slog.DisableJsonLogger()
}

// demoSensitiveData 演示日志脱敏功能
func demoSensitiveData(logger *slog.Logger) {
	// 启用DLP
	slog.EnableDLPLogger()
	time.Sleep(100 * time.Millisecond)

	// 这里不需要重新注册规则，因为内置规则已经在dlp包初始化时注册
	// 直接使用包中定义好的规则名称即可

	// 注册常用的脱敏规则
	//slog.RegisterDLPStrategy("mobile_phone", func(text string) string {
	//	if matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, text); matched {
	//		return text[:3] + "****" + text[len(text)-4:]
	//	}
	//	return text
	//})

	logger = slog.NewLogger(os.Stdout, false, true)

	// 测试不同类型数据的脱敏
	sensitiveData := []struct {
		name     string
		value    string
		strategy string // 对应内置规则的名称
	}{
		{"手机号", "13800138000", "mobile_phone"},
		{"身份证", "440101199001011234", "id_card"},
		{"银行卡", "6222021234567890123", "bank_card"},
		{"电子邮箱", "test@example.com", "email"},
		{"中文姓名", "张小三", "chinese_name"},
		{"固定电话", "0755-12345678", "landline"},
		{"邮政编码", "518000", "postal_code"},
		{"护照号", "E12345678", "passport"},
		{"驾驶证", "440101199001011234", "device_id"},
		{"IPv4地址", "192.168.1.1", "ipv4"},
		{"MAC地址", "00:0A:95:9D:68:16", "mac"},
		{"车牌号", "粤B12345", "plate"},
		{"信用卡", "4111111111111111", "credit_card"},
		{"统一社会信用代码", "91110000100000589B", "company_id"},
		{"地址", "广东省深圳市南山区科技园", "address"},
		{"密码", "password123", "password"},
		{"JWT令牌", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", "jwt"},
		{"密钥", "sk_live_51Mx9fK2eZvKYlo2CJyxB", "private_key"},
		{"用户名", "admin123", "username"},
		{"设备ID", "A1B2C3D4-E5F6-G7H8-I9J0-K1L2M3N4O5P6", "device_id"},
	}

	// 测试每种类型的脱敏效果
	for _, data := range sensitiveData {
		// 直接使用原始值,让DLP引擎自动处理脱敏
		logger.Info("脱敏测试",
			"数据类型", data.name,
			"规则名称", data.strategy,
			"原始值", data.value, // DLP引擎会自动处理脱敏
		)
	}

	// 测试复合信息脱敏
	logger.Info("用户完整信息",
		"姓名", "张三",
		"身份证", "440101199001011234",
		"手机", "13800138000",
		"邮箱", "zhangsan@example.com",
		"银行卡", "6222021234567890123",
		"家庭住址", "广东省深圳市南山区科技园1号楼101室",
		"车牌号", "粤B12345",
		"工作单位", "某某科技有限公司",
	)

	// 测试 URL 中的敏感信息
	logger.Info("API调用信息",
		"url", "https://api.example.com/users?access_token=12345&api_key=secret",
		"authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"client_ip", "192.168.1.100",
		"user_agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	)

	// 测试JSON格式的复合数据
	logger.Info("JSON数据脱敏测试",
		"data", `{
            "user": {
                "name": "张三",
                "id_card": "440101199001011234",
                "phone": "13800138000",
                "email": "zhangsan@example.com",
                "address": "广东省深圳市南山区科技园"
            },
            "payment": {
                "card_no": "6222021234567890123",
                "card_holder": "张三"
            }
        }`,
	)

	// 显示当前启用的脱敏功能
	if slog.IsDLPEnabled() {
		logger.Info("当前脱敏功能已启用")
	}

	// 禁用脱敏功能进行对比
	slog.DisableDLPLogger()
	logger.Info("禁用脱敏后的日志",
		"姓名", "张三",
		"身份证", "440101199001011234",
		"手机", "13800138000",
		"银行卡", "6222021234567890123",
	)
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

// demoAsyncLogging 演示异步日志记录功能
func demoAsyncLogging() {
	// 获取异步日志通道
	recordChan := slog.GetChanRecord(1000)

	// 启动异步处理
	go func() {
		for record := range recordChan {
			// 处理日志记录
			fmt.Printf("异步处理日志: %v\n", record)
		}
	}()

	// 记录一些测试日志
	logger := slog.Default("async")
	for i := 0; i < 5; i++ {
		logger.Info("异步日志测试",
			"index", i,
			"timestamp", time.Now(),
		)
		time.Sleep(100 * time.Millisecond)
	}
}

// demoTracing 演示日志链路追踪功能
func demoTracing() {
	// 创建带追踪ID的logger
	traceLogger := slog.Default("trace").With(
		"trace_id", "trace-123",
		"span_id", "span-456",
	)

	// 模拟请求处理链路
	traceLogger.Info("开始处理请求")

	// 模拟服务调用
	serviceLogger := traceLogger.WithGroup("service")
	serviceLogger.Info("调用用户服务",
		"service", "user",
		"method", "GetUserInfo",
	)

	// 模拟数据库操作
	dbLogger := traceLogger.WithGroup("database")
	dbLogger.Info("执行数据库查询",
		"sql", "SELECT * FROM users WHERE id = ?",
		"params", []interface{}{123},
	)

	traceLogger.Info("请求处理完成")
}

// demoErrorHandling 演示错误日志处理
func demoErrorHandling() {
	logger := slog.Default("error")

	// 模拟不同类型的错误处理
	err1 := fmt.Errorf("数据库连接失败")
	logger.Error("系统错误",
		"error", err1,
		"component", "database",
	)

	// 模拟带堆栈的错误
	err2 := fmt.Errorf("验证失败: %w", err1)
	logger.Error("业务错误",
		"error", err2,
		"component", "auth",
		"stack", string(debug.Stack()),
	)

	// 使用WithGroup组织错误信息
	logger.WithGroup("error_details").Error("详细错误信息",
		"error", err2,
		"stack", string(debug.Stack()),
		"context", map[string]interface{}{
			"user_id": 123,
			"action":  "login",
			"time":    time.Now(),
		},
	)
}
