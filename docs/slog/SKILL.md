# slog - 高性能 Go 结构化日志库

## 技能描述

slog 是一个基于 Go 官方 log/slog 扩展的高性能结构化日志库，专为生产环境设计。提供 DLP 数据脱敏、模块化架构、高性能内存管理、运行时动态控制等企业级特性。

## 适用场景

- 需要结构化日志（JSON/Text）的 Go 服务
- 生产环境需要数据脱敏（PII 保护）
- 高性能日志场景（对象池优化）
- 需要运行时动态调整日志级别
- 多输出目标（文件、控制台、Syslog、Graylog）
- 需要日志订阅和实时处理

## 核心能力

### 1. 基础日志记录

```go
import "github.com/darkit/slog"

// 快速开始
logger := slog.Default()
logger.Info("服务启动成功", "port", 8080)

// 格式化日志
logger.Infof("处理耗时: %.2f ms", 123.45)

// 链式操作
logger.With("request_id", "abc123").
    WithGroup("http").
    Info("请求处理完成", "status", 200)
```

### 2. 日志级别控制

```go
// 6 级日志级别
slog.SetLevelTrace()  // -8 最详细
slog.SetLevelDebug()  // -4
slog.SetLevelInfo()   //  0 (默认)
slog.SetLevelWarn()   //  4
slog.SetLevelError()  //  8
slog.SetLevelFatal()  // 12 记录后退出

// 动态设置（支持多种格式）
slog.SetLevel("debug")     // 字符串
slog.SetLevel(-4)          // 数字
slog.SetLevel(slog.LevelDebug) // Level 类型
```

### 3. 输出格式切换

```go
// 文本格式（带颜色，适合开发）
slog.EnableTextLogger()

// JSON 格式（适合生产）
slog.EnableJSONLogger()

// 同时使用两种格式
slog.EnableTextLogger()
slog.EnableJSONLogger()

// 禁用
slog.DisableTextLogger()
slog.DisableJSONLogger()
```

### 4. 上下文追踪

```go
// 设置上下文传播器
slog.SetContextPropagator(func(ctx context.Context) []slog.Attr {
    attrs := make([]slog.Attr, 0, 2)
    if traceID, ok := ctx.Value("trace_id").(string); ok {
        attrs = append(attrs, slog.String("trace_id", traceID))
    }
    if userID, ok := ctx.Value("user_id").(string); ok {
        attrs = append(attrs, slog.String("user_id", userID))
    }
    return attrs
})

// 使用上下文日志
ctx := context.WithValue(context.Background(), "trace_id", "trace-xxx")
ctxLogger := logger.WithContext(ctx)
ctxLogger.Info("API请求完成")  // 自动注入 trace_id
```

### 5. DLP 数据脱敏

```go
import "github.com/darkit/slog/dlp"

// 启用 DLP
slog.EnableDLPLogger()

// 手动脱敏
dlpEngine := dlp.NewDlpEngine()
dlpEngine.Enable()

// 文本脱敏
masked := dlpEngine.DesensitizeText("手机号：13812345678")
// 输出: 手机号：138****5678

// 结构体标签脱敏
type UserInfo struct {
    Name     string `dlp:"chinese_name"`
    Phone    string `dlp:"mobile_phone"`
    Email    string `dlp:"email"`
    BankCard string `dlp:"bank_card"`
    IDCard   string `dlp:"id_card"`
}

user := UserInfo{
    Name:     "张三",
    Phone:    "13812345678",
    Email:    "zhangsan@example.com",
    BankCard: "6222021234567890123",
    IDCard:   "110101199001011234",
}
dlpEngine.DesensitizeStruct(&user)
```

**支持的脱敏类型**:
- `chinese_name` - 中文姓名（张*）
- `mobile_phone` - 手机号（138****5678）
- `id_card` - 身份证号（110101********1234）
- `bank_card` - 银行卡号（622202*********0123）
- `email` - 邮箱（zhan***@example.com）
- `ipv4` / `ipv6` - IP 地址
- `license_plate` - 车牌号

### 6. 日志订阅

```go
// 基础订阅
records, cancel := slog.Subscribe(1000) // 缓冲区大小
defer cancel()

go func() {
    for event := range records {
        fmt.Println(event.Rendered)
        // 需要结构化数据时使用 event.Record
        // event.Rendered 始终跟随当前激活输出格式
        // 发送到外部系统（如 ES、Kafka）
    }
}()

// 高级订阅（带背压控制）
ch, cancel := slog.SubscribeWithOptions(slog.SubscribeOptions{
    BufferSize:   1000,
    Backpressure: slog.SubscriptionDropOldest, // 或 DropNewest, BlockWithTimeout
    BlockTimeout: 5 * time.Millisecond,
})
```

### 7. 模块化扩展

```go
// 使用 Formatter 模块
formatter, _ := modules.CreateModule("formatter", modules.Config{
    "type": "time",
    "format": "2006-01-02 15:04:05",
})
logger := slog.Default().Use(formatter)

// 使用 Writer 模块（文件输出）
fileWriter, _ := modules.CreateModule("file", modules.Config{
    "path": "/var/log/app.log",
    "max_size": 100,  // MB
    "max_backups": 10,
    "max_age": 30,    // 天
})
logger.Use(fileWriter)

// 使用 Multi Writer（多输出）
multiWriter, _ := modules.CreateModule("multi", modules.Config{
    "writers": []string{"console", "file", "syslog"},
})
logger.Use(multiWriter)
```

### 8. 自定义 Formatter

```go
// 注册自定义属性格式化器
formatterID := slog.RegisterFormatter("uppercase", func(groups []string, attr slog.Attr) (slog.Value, bool) {
    if attr.Key == "level" {
        return slog.StringValue(strings.ToUpper(attr.Value.String())), true
    }
    return attr.Value, true
})

// 移除	slog.RemoveFormatter(formatterID)

// 查看已注册
formatters := slog.ListFormatters()
```

### 9. 性能优化配置

```go
// 创建高性能 Logger
cfg := &slog.Config{
    MaxFormatCacheSize:    1000,  // 格式缓存
    StringBuilderPoolSize: 100,   // 字符串构建器池
    LogInternalErrors:     false, // 不记录内部错误
    NoColor:               true,  // 生产环境禁用颜色
    AddSource:             false, // 生产环境禁用源码位置
    TimeFormat:            time.RFC3339,
}

logger := slog.NewLoggerWithConfig(os.Stdout, cfg)

// 日志限流（防止日志风暴）
slog.ConfigureRecordLimiter(1000, 100) // rate=1000/s, burst=100
```

### 10. 与标准库兼容

```go
// 获取标准库 *slog.Logger
stdLogger := slog.GetSlogLogger()

// 使用标准库 Handler
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
logger := slog.New(handler)
```

## 最佳实践

### 开发环境配置

```go
func init() {
    slog.EnableTextLogger()
    slog.SetLevelDebug()
    slog.SetTimeFormat("15:04:05.000")
}
```

### 生产环境配置

```go
func init() {
    slog.EnableJSONLogger()
    slog.SetLevelInfo()
    slog.EnableDLPLogger()  // 启用脱敏
    
    // 禁用颜色和源码位置以提高性能
    logger := slog.NewLogger(os.Stdout, true, false)
    slog.ResetGlobalLogger(os.Stdout, true, false)
}
```

### 结构化日志规范

```go
// ✅ 推荐：使用键值对
logger.Info("用户登录",
    "user_id", user.ID,
    "username", user.Name,
    "ip", clientIP,
    "duration_ms", time.Since(start).Milliseconds(),
)

// ❌ 避免：使用格式化字符串记录结构化数据
logger.Infof("用户 %s (ID: %d) 从 %s 登录", user.Name, user.ID, clientIP)
```

### 错误日志处理

```go
// ✅ 推荐：分离错误和上下文
if err != nil {
    logger.Error("数据库查询失败",
        "error", err,
        "query", query,
        "table", "users",
    )
}

// ❌ 避免：将错误作为格式字符串
logger.Errorf("数据库查询失败: %v", err)
```

### 性能敏感场景

```go
// 使用对象池优化的 Logger
cfg := &slog.Config{
    StringBuilderPoolSize: 200,
    MaxFormatCacheSize:    2000,
}
logger := slog.NewLoggerWithConfig(io.Discard, cfg)

// 避免在热路径创建临时 Logger
// ❌ 不推荐
for i := 0; i < 1000000; i++ {
    slog.With("iteration", i).Info("处理中")
}

// ✅ 推荐
logger := slog.With("batch", batchID)
for i := 0; i < 1000000; i++ {
    logger.Info("处理中", "iteration", i)
}
```

## 完整示例

```go
package main

import (
    "context"
    "os"
    "time"
    
    "github.com/darkit/slog"
    "github.com/darkit/slog/dlp"
)

func main() {
    // 初始化配置
    slog.EnableJSONLogger()
    slog.SetLevelInfo()
    slog.EnableDLPLogger()
    
    // 设置上下文传播
    slog.SetContextPropagator(func(ctx context.Context) []slog.Attr {
        var attrs []slog.Attr
        if traceID, ok := ctx.Value("trace_id").(string); ok {
            attrs = append(attrs, slog.String("trace_id", traceID))
        }
        return attrs
    })
    
    // 创建带上下文的 Logger
    ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
    logger := slog.Default().WithContext(ctx)
    
    // 记录启动日志
    logger.Info("服务启动",
        "version", "1.0.0",
        "port", 8080,
        "env", "production",
    )
    
    // 模拟业务逻辑
    processRequest(logger)
    
    // 记录关闭日志
    logger.Info("服务关闭")
}

func processRequest(logger *slog.Logger) {
    start := time.Now()
    
    // 处理用户数据（自动脱敏）
    userData := struct {
        Name  string `dlp:"chinese_name"`
        Phone string `dlp:"mobile_phone"`
    }{
        Name:  "张三",
        Phone: "13812345678",
    }
    
    logger.Info("处理用户请求",
        "user", userData,
        "action", "update_profile",
    )
    
    logger.Info("请求处理完成",
        "duration_ms", time.Since(start).Milliseconds(),
    )
}
```

## 注意事项

1. **Fatal 级别会退出程序** - 使用 `logger.Fatal()` 会调用 `os.Exit(1)`
2. **DLP 有性能开销** - 敏感数据检测会增加 ~46ns/op（有缓存时）
3. **订阅缓冲区会满** - 消费者必须及时处理，否则可能丢日志
4. **全局设置影响所有 Logger** - `slog.SetLevel*()` 影响全局
5. **并发安全** - 所有方法都是并发安全的，无需额外同步

## 相关链接

- GitHub: https://github.com/darkit/slog
- Go 版本要求: >= 1.23
- 基于: Go 官方 log/slog 扩展
