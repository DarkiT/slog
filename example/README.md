# Slog 日志库示例

本目录包含了 `github.com/darkit/slog` 日志库的综合功能演示，展示了所有核心特性和最佳实践。

## 快速开始

```bash
# 运行完整演示
go run main.go

# 或者在项目根目录运行
go run example/main.go
```

## 功能特性

### 🎯 基础日志功能

- **多级别日志**：Trace, Debug, Info, Warn, Error 五个级别
- **格式化日志**：支持 `Printf` 风格的格式化输出
- **动态效果**：进度条、倒计时、加载动画等可视化效果

```go
slog.Info("用户登录成功", "user_id", 12345, "ip", "192.168.1.100")
slog.Infof("用户 %s (ID: %d) 在 %s 登录成功", username, userID, time.Now())
slog.Dynamic("系统初始化", 8, 200)  // 动态效果
```

### 🏗️ 结构化日志

- **键值对字段**：灵活的结构化数据记录
- **业务指标**：支持复杂业务数据的结构化记录
- **上下文传递**：链路追踪和请求 ID 传递

```go
logger.Info("API请求处理",
    "method", "POST",
    "endpoint", "/api/orders",
    "status_code", 200,
    "response_time_ms", 245,
)
```

### 🎚️ 动态级别控制

- **运行时切换**：生产环境无需重启即可调整日志级别
- **全局同步**：所有 Logger 实例自动同步级别变化
- **HTTP/信号控制**：支持多种控制方式

```go
// 生产模式 - 只记录错误
slog.SetLevel("error")

// 故障排查 - 开启详细日志
slog.SetLevel("debug")

// 恢复生产模式
slog.SetLevel("error")
```

### 🔒 DLP 数据脱敏

- **自动脱敏**：手机号、身份证、银行卡等敏感信息自动脱敏
- **结构体脱敏**：支持结构体字段的标签式脱敏
- **多种规则**：内置中国常用敏感信息脱敏规则

```go
// 文本脱敏
dlpEngine := dlp.NewDlpEngine()
dlpEngine.Enable()
masked := dlpEngine.DesensitizeText("手机号：13812345678")
// 输出: 手机号：138****5678

// 结构体脱敏
type User struct {
    Phone string `dlp:"mobile_phone"`
}
dlpEngine.DesensitizeStruct(&user)
```

### 🔧 模块注册系统

- **插件架构**：支持动态加载和配置各种日志模块
- **工厂模式**：统一的模块创建和管理接口
- **链式调用**：简洁优雅的 API 设计

```go
// 快速启用模块
logger := slog.UseFactory("formatter", modules.Config{
    "type": "time",
}).Build()

// 配置驱动方式
configs := []modules.ModuleConfig{...}
logger := slog.UseConfig(configs).Build()
```

### ⚡ 异步日志处理

- **订阅者模式**：支持多个消费者并行处理日志
- **缓冲控制**：可配置的缓冲区大小
- **优雅关闭**：确保日志不丢失

```go
records, cancel := slog.Subscribe(1000)
defer cancel()

go func() {
    for record := range records {
        // 处理日志记录
        processLogRecord(record)
    }
}()
```

### 📊 性能基准测试

- **基础性能**：单线程日志记录性能测试
- **并发性能**：多协程并发日志记录测试
- **内存使用**：内存分配和使用情况分析

### 🏭 生产环境场景

- **Web 服务**：HTTP 请求处理日志记录
- **数据库事务**：事务操作的完整生命周期记录
- **微服务通信**：服务间调用的链路追踪
- **定时任务**：计划任务的执行状态记录

## 演示内容

运行示例程序将依次展示以下功能：

1. **基础日志功能** - 所有日志级别和基本功能
2. **结构化日志** - 结构化字段和格式化日志
3. **动态级别控制** - 生产环境动态级别切换
4. **DLP 数据脱敏** - 敏感信息脱敏功能
5. **模块注册系统** - 模块注册中心和各种使用方式
6. **异步日志处理** - 异步日志和订阅者模式
7. **性能基准测试** - 各种场景下的性能表现
8. **上下文和追踪** - 上下文传递和链路追踪
9. **错误处理** - 错误日志和异常处理
10. **生产环境场景** - 真实生产环境使用场景

## 最佳实践

### 1. 生产环境级别控制

```go
// 启动时设置为生产级别
slog.SetLevel("error")

// 发现问题时临时开启调试
slog.SetLevel("debug")
// ... 排查问题
// 问题解决后恢复
slog.SetLevel("error")
```

### 2. 结构化日志记录

```go
// 好的实践：使用结构化字段
logger.Info("用户操作",
    "user_id", userID,
    "action", "create_order",
    "order_id", orderID,
    "amount", amount,
)

// 避免：字符串拼接
logger.Infof("用户%d创建订单%s，金额%f", userID, orderID, amount)
```

### 3. 错误处理

```go
// 记录错误时包含丰富的上下文
logger.Error("数据库操作失败",
    "error", err.Error(),
    "operation", "INSERT",
    "table", "orders",
    "duration_ms", duration,
    "retry_count", retries,
)
```

### 4. 性能考虑

```go
// 在高频场景使用级别判断
if logger.GetLevel() <= slog.LevelDebug {
    logger.Debug("详细调试信息", "data", expensiveOperation())
}
```

## 技术亮点

- ✅ **零分配优化**：在关键路径上实现零内存分配
- ✅ **并发安全**：全局状态管理完全并发安全
- ✅ **模块化设计**：插件式架构，易于扩展
- ✅ **性能优异**：高吞吐量，低延迟
- ✅ **生产就绪**：丰富的生产环境特性支持

## 系统要求

- Go 1.23+
- 支持所有主流操作系统

---

💡 **提示**: 运行示例时建议在终端中查看，可以看到完整的颜色输出和动态效果。
