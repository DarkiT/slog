# slog

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/slog.svg)](https://pkg.go.dev/github.com/darkit/slog)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/slog)](https://goreportcard.com/report/github.com/darkit/slog)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/slog/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://go.dev/doc/devel/release)

高性能、功能丰富的 Go 结构化日志库，基于 Go 1.23+ 官方 [`log/slog`](https://pkg.go.dev/log/slog) 扩展。

## 安装

```bash
go get github.com/darkit/slog@latest
```

**要求**: Go 1.23 或更高版本

## 快速开始

```go
package main

import (
    "os"
    "github.com/darkit/slog"
)

func main() {
    // 使用默认 logger
    logger := slog.Default()

    // 基础日志
    logger.Info("Hello slog!")

    // 结构化字段
    logger.Info("User logged in",
        "user_id", 123,
        "action", "login",
    )

    // 格式化日志
    logger.Infof("Process took %d ms", 100)
}
```

## 核心特性

- **多级别日志**: Trace、Debug、Info、Warn、Error、Fatal
- **双格式输出**: 同时输出文本和 JSON 格式
- **彩色终端**: 自动检测 TTY，支持彩色输出
- **数据脱敏 (DLP)**: 内置敏感信息脱敏系统
- **模块化架构**: Formatter/Middleware/Handler/Sink 插件系统
- **高性能**: 分级对象池、LRU 缓存、原子操作
- **运行时控制**: 动态调整日志级别和输出
- **订阅机制**: 支持日志订阅和背压控制

## 日志级别

```go
// 全局级别设置
slog.SetLevelTrace()  // 最详细
slog.SetLevelDebug()
slog.SetLevelInfo()   // 默认
slog.SetLevelWarn()
slog.SetLevelError()

// 动态设置
slog.SetLevel("debug")
slog.SetLevel(slog.LevelDebug)
```

## 创建 Logger

### 基础创建

```go
// 默认输出到 stdout
logger := slog.NewLogger(os.Stdout, false, false)

// 自定义配置
cfg := slog.DefaultConfig()
cfg.SetEnableText(true)
cfg.SetEnableJSON(true)
logger := slog.NewLoggerWithConfig(os.Stdout, cfg)
```

### Builder 模式

```go
logger := slog.NewLoggerBuilder().
    WithModule("order").
    WithGroup("api").
    WithAttrs(slog.String("req_id", "r-1")).
    EnableJSON(true).
    Build()
```

### 模块化日志

```go
// 创建带模块名的 logger
userLogger := slog.Default("user")
authLogger := slog.Default("auth")

// 使用分组
logger := slog.WithGroup("api")
logger.Info("Request", "method", "GET", "path", "/users")
```

## 数据脱敏 (DLP)

```go
// 启用 DLP
slog.EnableDLPLogger()

// 结构体标签脱敏
type User struct {
    Name     string `dlp:"chinese_name"`
    Phone    string `dlp:"mobile_phone"`
    Email    string `dlp:"email"`
    Password string `dlp:"password"`
    IDCard   string `dlp:"id_card"`
}

user := &User{
    Name:  "张三",
    Phone: "13812345678",
    Email: "zhangsan@example.com",
}

// 自动脱敏输出
logger.Info("user", "data", user)
// 输出: 张三 → 张*, 13812345678 → 138****5678
```

支持的脱敏类型：中文姓名、身份证、手机号、邮箱、银行卡、地址、密码、车牌、IP、JWT、URL

## 文件日志

```go
writer := slog.NewWriter("logs/app.log").
    SetMaxSize(100).      // 单文件最大 100MB
    SetMaxAge(7).         // 保留 7 天
    SetMaxBackups(10).    // 最多 10 个备份
    SetCompress(true)     // 压缩旧文件

logger := slog.NewLogger(writer, false, false)
```

## 运行时控制

```go
// 获取当前状态
snapshot := slog.GetRuntimeSnapshot()

// 动态调整
slog.ApplyRuntimeOption("level", "warn")
slog.ApplyRuntimeOption("json", "on")
slog.ApplyRuntimeOption("dlp", "on")
```

## 日志订阅

```go
// 订阅日志
ch, cancel := slog.Subscribe(1000)
defer cancel()

// 异步处理
go func() {
    for event := range ch {
        // event.Record 是结构化视图；event.Rendered 是当前激活输出对应的语义化结果
        fmt.Println(event.Rendered)
    }
}()
```

## 模块系统

```go
import "github.com/darkit/slog/modules"

// 使用工厂创建带模块的 logger
logger := slog.NewLoggerBuilder().
    UseLogfmt().           // logfmt 输出 (Loki/Vector)
    Build()

// GELF 输出 (Graylog/Logstash)
logger := slog.NewLoggerBuilder().
    UseGELF(nil).
    Build()

// 网络输出 (TCP/UDP)
logger := slog.NewLoggerBuilder().
    UseNetOutput(&outputnet.SenderOption{
        Network: "tcp",
        Address: "logs.example.com:514",
    }).
    Build()
```

内置模块：

- **Formatter**: 时间格式化、PII 脱敏
- **Multi**: Fanout、Failover、Router 模式
- **Webhook**: HTTP 日志推送
- **Syslog**: RFC5424 日志输出
- **GELF**: Graylog 扩展日志格式
- **Logfmt**: 键值对格式

## 性能

- **DLP 缓存**: ~46ns/op (有缓存) vs ~2790ns/op (无缓存)
- **缓存键生成**: ~314ns/op (xxhash64)
- **内存复用率**: 95%+ (分级对象池)

## 兼容性

- 支持 Go 1.23+ (使用 `log/slog` 标准库)
- API 设计遵循 `log/slog` 接口约定
- 支持所有 `slog.Handler` 实现

## 文档

- [API 参考](https://pkg.go.dev/github.com/darkit/slog)
- [贡献指南](./CONTRIBUTING.md)
- [安全策略](./SECURITY.md)
- [更新日志](./CHANGELOG.md)

## 许可证

[MIT License](./LICENSE)

## 致谢

基于 Go 官方 [`log/slog`](https://pkg.go.dev/log/slog) 包扩展开发。
