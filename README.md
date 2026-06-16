# slog

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/slog.svg)](https://pkg.go.dev/github.com/darkit/slog)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/slog)](https://goreportcard.com/report/github.com/darkit/slog)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/slog/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-00ADD8.svg)](https://go.dev/doc/devel/release)

基于 Go 1.23+ 官方 `log/slog` 扩展的高性能结构化日志库。内置 DLP 数据脱敏、分级对象池、日志订阅、模块化扩展，专为生产环境设计。

## 安装

```bash
go get github.com/darkit/slog@latest
```

> 要求 Go 1.23+

## 快速开始

```go
package main

import (
    "os"
    "github.com/darkit/slog"
)

func main() {
    // 使用默认 Logger（文本彩色输出）
    logger := slog.Default()

    // 键值对结构化日志
    logger.Info("服务启动", "port", 8080, "env", "production")

    // 格式化日志
    logger.Infof("处理耗时 %d ms", 150)

    // 带上下文的日志（自动注入 trace_id 等）
    ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
    logger.WithContext(ctx).Info("请求完成")
}
```

## 核心特性

| 特性 | 说明 |
|------|------|
| **六级别日志** | Trace(-8)、Debug(-4)、Info(0)、Warn(4)、Error(8)、Fatal(12) |
| **双格式输出** | 文本 + JSON 可同时启用，独立控制开关 |
| **彩色终端** | 自动检测 TTY，可关闭 |
| **DLP 数据脱敏** | 36 种敏感信息类型，可按需启用/禁用 matcher |
| **模块化架构** | Formatter / Webhook / Syslog / GELF / Logfmt / Net / Multi 模块 |
| **高性能** | 分级对象池、LRU 缓存、xxhash64 缓存键、原子操作 |
| **日志订阅** | 支持 3 种背压策略（丢旧、丢新、阻塞超时） |
| **运行时控制** | 动态调整级别、格式开关、DLP 开关 |
| **上下文传播** | 自定义 ContextPropagator，自动注入 trace / user 等字段 |
| **动态渲染** | 内置进度条、倒计时、加载动画 |
| **日志限流** | 令牌桶算法，防止日志风暴 |

## 日志级别

```go
slog.SetLevelTrace()          // 最详细
slog.SetLevelDebug()
slog.SetLevelInfo()           // 默认
slog.SetLevelWarn()
slog.SetLevelError()
slog.SetLevelFatal()          // 记录后 os.Exit(1)

// 动态设置（支持 int / string / Level）
slog.SetLevel("debug")
slog.SetLevel(-4)
slog.SetLevel(slog.LevelDebug)
slog.SetLevel(slog.LevelWarn) // 注意这里是指 slog 包中的 Level
```

## 创建 Logger

### 基础创建

```go
// 默认 stdout
logger := slog.NewLogger(os.Stdout, false, false)

// 带配置
cfg := slog.DefaultConfig()
cfg.SetEnableText(true)
cfg.SetEnableJSON(true)
cfg.NoColor = true
logger := slog.NewLoggerWithConfig(os.Stdout, cfg)
```

### Builder 模式（推荐）

```go
logger := slog.NewLoggerBuilder().
    WithWriter(os.Stdout).
    WithModule("order-service").
    WithGroup("http").
    WithAttrs(slog.String("req_id", "r-1"), slog.Int("version", 2)).
    EnableText(true).
    EnableJSON(true).
    EnableDLP(true).
    Build()
```

### 多格式 Builder

```go
// Logfmt（接入 Loki / Vector）
logger := slog.NewLoggerBuilder().UseLogfmt().Build()

// GELF（接入 Graylog / Logstash）
logger := slog.NewLoggerBuilder().UseGELF(nil).Build()

// 网络输出（TCP/UDP）
logger := slog.NewLoggerBuilder().UseNetOutput(&outputnet.SenderOption{
    Network: "tcp",
    Address: "logs.example.com:514",
}).Build()
```

### 模块化 Logger

```go
userLogger := slog.Default("user-service")
authLogger := slog.Default("auth-service")

// WithGroup 分组
logger := slog.WithGroup("api")
logger.Info("请求处理", "method", "GET", "path", "/users")
```

## 数据脱敏 (DLP)

### 启用

```go
slog.EnableDLPLogger()

// 或通过 Builder
logger := slog.NewLoggerBuilder().EnableDLP(true).Build()
```

### 自动脱敏

启用 DLP 后，日志文本中的敏感信息会被自动脱敏：

```go
logger.Info("用户登录",
    "phone", "13812345678",       // → 138****5678
    "email", "user@example.com",   // → use***@example.com
)
```

### 结构体标签脱敏

```go
type UserInfo struct {
    Name     string `dlp:"chinese_name"`
    Phone    string `dlp:"mobile_phone"`
    Email    string `dlp:"email"`
    IDCard   string `dlp:"id_card"`
    BankCard string `dlp:"bank_card"`
}
```

支持高级用法：`dlp:"type,recursive"`（递归脱敏嵌套结构体）、`dlp:"custom:strategy_name"`（自定义策略）。

### Engine 级直接调用

```go
import "github.com/darkit/slog/dlp"

engine := dlp.NewDlpEngine()
engine.Enable()

// 文本脱敏
masked := engine.DesensitizeText("手机号：13812345678，邮箱：user@example.com")

// 结构体脱敏
engine.DesensitizeStructAdvanced(&userInfo)

// 仅脱敏属性值（消息原文不动）
msg, attrs := engine.DesensitizeAttrsOnly("原始消息", map[string]string{
    "phone": "13812345678",
    "role":  "admin",
})
```

### Matcher 管理

```go
engine := dlp.NewDlpEngine()
engine.Enable()

// 禁用指定 matcher（variadic，更简洁）
engine.DisableMatchers("ipv4", "ipv6")

// 重新启用
engine.EnableMatchers("ipv4")

// 单个精确控制
engine.SetMatcherEnabled("email", false)

// 查询状态
engine.IsMatcherDisabled("email")        // bool
engine.DisabledMatchers()                  // []string
engine.EnabledMatchers()                   // []string
engine.GetSupportedTypes()                 // []string
```

### 支持的 36 种敏感信息类型

| 类别 | 类型 |
|------|------|
| 个人身份 | `chinese_name` `id_card` `passport` `social_security` `license_number` |
| 联系方式 | `mobile_phone` `landline` `email` `address` `postal_code` |
| 金融信息 | `bank_card` `credit_card` `iban` `swift` |
| 网络标识 | `ipv4` `ipv6` `mac` `url` `domain` |
| 设备标识 | `imei` `plate` `vin` `device_id` `uuid` |
| 密钥令牌 | `api_key` `jwt` `access_token` `password` `username` |
| 加密哈希 | `md5` `sha1` `sha256` |
| 其他 | `lat_lng` `medical_id` `company_id` `git_repo` |

> `username` `api_key` `access_token` `password` 因误报率高，默认不参与自由文本扫描，但结构体标签 `dlp:"username"` 等仍可显式使用。

## 输出格式控制

```go
// 独立控制文本 / JSON 开关
slog.EnableTextLogger()
slog.EnableJSONLogger()
slog.DisableTextLogger()
slog.DisableJSONLogger()

// 两者可同时启用
slog.EnableTextLogger()
slog.EnableJSONLogger()
```

## 文件日志

```go
writer := slog.NewWriter("logs/app.log").
    SetMaxSize(100).      // 单文件最大 100MB
    SetMaxAge(7).         // 保留 7 天
    SetMaxBackups(10).    // 最多 10 个备份
    SetCompress(true)     // gzip 压缩旧文件

logger := slog.NewLogger(writer, true, false)
```

## 运行时控制

```go
// 获取状态快照
snapshot := slog.GetRuntimeSnapshot()
// snapshot.Level / .TextEnabled / .JSONEnabled / .DLPEnabled / .DLPVersion

// 动态调整
slog.ApplyRuntimeOption("level", "warn")
slog.ApplyRuntimeOption("json", "on")
slog.ApplyRuntimeOption("text", "off")
slog.ApplyRuntimeOption("dlp", "on")
```

## 日志订阅

```go
// 基础订阅
ch, cancel := slog.Subscribe(1000)
defer cancel()

// 高级订阅（背压控制）
ch, cancel := slog.SubscribeWithOptions(slog.SubscribeOptions{
    BufferSize:   1000,
    Backpressure: slog.SubscriptionDropOldest, // DropOldest / DropNewest / BlockWithTimeout
    BlockTimeout: 5 * time.Millisecond,
})

go func() {
    for event := range ch {
        event.Record    // 结构化视图（已应用 formatter / DLP / context 字段）
        event.Rendered  // 当前激活输出格式的最终渲染结果
        event.Format    // "text" / "json" / ""
    }
}()

// 订阅统计
stats := slog.GetSubscriptionStats()       // 汇总
detail := slog.GetSubscriberStats(id)       // 单个
all    := slog.ListSubscriberStats()        // 全部
```

## 上下文传播

```go
// 注册传播器
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

// 使用
ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
logger.WithContext(ctx).Info("请求完成")  // 自动注入 trace_id
```

## 动态渲染

```go
// 进度条（0% → 100%，持续 3 秒）
slog.Progress("部署中", 3000)

// 倒计时
slog.Countdown("服务关闭", 10)

// 加载动画
slog.Loading("正在加载数据", 5)
```

## 日志限流

```go
// 令牌桶：1000 条/秒，突发上限 100
slog.ConfigureRecordLimiter(1000, 100)

// 关闭限流
slog.ConfigureRecordLimiter(0, 0)
```

## 性能优化配置

```go
cfg := &slog.Config{
    MaxFormatCacheSize:    2000,   // 格式字符串缓存上限
    StringBuilderPoolSize: 200,    // 对象池大小
    LogInternalErrors:     false,  // 生产环境关闭内部错误日志
    NoColor:               true,   // 生产环境关闭颜色
    AddSource:             false,  // 生产环境关闭源码位置
    TimeFormat:            time.RFC3339,
}
logger := slog.NewLoggerWithConfig(os.Stdout, cfg)
```

**性能指标**：DLP 缓存命中 ~46ns/op（无缓存 ~2790ns/op），缓存键生成 ~314ns/op（xxhash64），内存复用率 95%+。

## 模块系统

| 模块 | 说明 |
|------|------|
| `formatter` | 时间格式化、错误格式化、HTTP 请求格式化 |
| `multi` | Fanout / Failover / Router 多输出模式 |
| `webhook` | HTTP 日志推送（支持 Slack / Discord / 自定义 Webhook） |
| `syslog` | RFC5424 Syslog 协议输出 |
| `gelf` | Graylog 扩展日志格式 |
| `logfmt` | 键值对格式（Loki / Vector 友好） |
| `output/net` | TCP/UDP 网络日志输出 |

```go
logger := slog.NewLoggerBuilder().
    UseLogfmt().   // 或 UseGELF(nil) 或 UseNetOutput(...)
    Build()
```

## 并发安全

所有 Logger 方法均为并发安全，可安全跨 goroutine 共享。全局配置变更为原子操作。

## 开发 & 构建

```bash
make build          # 编译
make test           # 运行测试
make test-race      # 带 race 检测的测试
make test-coverage  # 测试 + 覆盖率报告
make lint           # golangci-lint
make fmt            # gofmt + goimports
make tidy           # go mod tidy + verify
make clean          # 清理构建产物
make help           # 查看所有目标
```

## 文档

- [API 参考](https://pkg.go.dev/github.com/darkit/slog)
- [贡献指南](./CONTRIBUTING.md)
- [安全策略](./SECURITY.md)
- [更新日志](./CHANGELOG.md)

## 许可证

[MIT](./LICENSE)
## 致谢

基于 Go 官方 [`log/slog`](https://pkg.go.dev/log/slog) 包扩展开发。
