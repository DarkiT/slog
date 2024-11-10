# slog

[![PkgGoDev](https://pkg.go.dev/badge/github.com/darkit/slog.svg)](https://pkg.go.dev/github.com/darkit/slog)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/slog)](https://goreportcard.com/report/github.com/darkit/slog)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/slog/blob/master/LICENSE)

slog 是一个高性能、功能丰富的 Go 语言日志库，基于 Go 1.21+ 的官方 `log/slog` 包进行扩展。它提供了更灵活的日志级别控制、彩色输出、结构化日志记录、日志脱敏等高级特性。

## 特性

- 支持多种日志级别（Trace、Debug、Info、Warn、Error、Fatal）
- 支持同时输出文本和 JSON 格式
- 支持彩色日志输出
- 支持动态调整日志级别
- 支持日志分组和模块化
- 支持结构化字段记录
- 支持日志脱敏处理
- 高性能缓冲设计
- 线程安全
- 支持自定义格式化

## 安装

```bash
go get github.com/darkit/slog
```

## 快速开始

```go
package main

import (
	"os"
	"github.com/darkit/slog"
)

func main() {
	// 创建默认logger
	logger := slog.NewLogger(os.Stdout, false, false)

	// 获取原始的 slog.Logger
	slogLogger := logger.GetSlogLogger()
	// 现在可以直接使用原始的 slog.Logger
	slogLogger.Info("使用原始slog记录日志")

	// 基础日志记录
	logger.Info("Hello Slog!")

	// 带结构化字段的日志
	logger.Info("User logged in",
		"user_id", 123,
		"action", "login",
	)
}
```

## 核心功能使用

### 1. 日志级别控制

```go
// 设置全局日志级别
slog.SetLevelDebug()  // Debug级别
slog.SetLevelInfo()   // Info级别
slog.SetLevelWarn()   // Warn级别
slog.SetLevelError()  // Error级别
slog.SetLevelFatal()  // Fatal级别
slog.SetLevelTrace()  // Trace级别

// 动态更新日志级别
slog.UpdateLogLevel("debug")           // 使用字符串
slog.UpdateLogLevel(slog.LevelDebug)   // 使用Level类型
slog.UpdateLogLevel(-4)                // 使用数字

// 监听日志级别变化
slog.WatchLevel("observer1", func(level slog.Level) {
fmt.Printf("Log level changed to: %v\n", level)
})

// 取消监听
slog.UnwatchLevel("observer1")
```

### 2. 日志记录方法

```go
// 不同级别的日志记录
logger.Trace("Trace message")
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")
logger.Fatal("Fatal message") // 会导致程序退出

// 格式化日志
logger.Debugf("User %s logged in from %s", username, ip)
logger.Infof("Process took %d ms", time)
logger.Warnf("High CPU usage: %.2f%%", cpuUsage)
logger.Errorf("Failed to connect: %v", err)
logger.Fatalf("Critical error: %v", err)

// 带结构化字段的日志
logger.Info("Database connection established",
"host", "localhost",
"port", 5432,
"user", "admin",
)
```

### 3. 日志分组和模块

```go
// 创建模块化日志记录器
userLogger := slog.Default("user")
authLogger := slog.Default("auth")

// 使用分组
logger := slog.WithGroup("api")
logger.Info("Request received",
"method", "GET",
"path", "/users",
)

// 链式调用
logger.With("request_id", "123").
WithGroup("auth").
Info("User authenticated")
```

### 4. 输出格式控制

```go
// 启用/禁用文本日志
slog.EnableTextLogger()
slog.DisableTextLogger()

// 启用/禁用JSON日志
slog.EnableJsonLogger()
slog.DisableJsonLogger()

// 创建带颜色的控制台日志
logger := slog.NewLogger(os.Stdout, false, true) // 最后一个参数控制是否显示源代码位置
```

### 5. 日志脱敏

```go
// 启用日志脱敏
slog.EnableDLPLogger()

// 禁用日志脱敏功能
slog.DisableDLPLogger()

// 使用脱敏日志
logger.Info("User data",
"credit_card", "1234-5678-9012-3456", // 将被自动脱敏
"phone", "13800138000",               // 将被自动脱敏
)
```

### 6. 日志订阅机制

```go
// 订阅日志记录
ch, cancel := slog.Subscribe(1000) // 指定缓冲大小
defer cancel() // 确保取消订阅

// 处理订阅的日志
go func() {
    for record := range ch {
        fmt.Printf("收到日志: %s [%s] %s\n",
            record.Time.Format(slog.TimeFormat),
            record.Level,
            record.Message,
        )
    }
}()

// 多订阅者模式
ch1, cancel1 := slog.Subscribe(500)
defer cancel1()

ch2, cancel2 := slog.Subscribe(1000)
defer cancel2()

// 不同订阅者可以独立处理日志
go func() {
    for record := range ch1 {
        // 处理所有日志
        fmt.Printf("订阅者1: %v\n", record)
    }
}()

go func() {
    for record := range ch2 {
        // 只处理错误日志
        if record.Level >= slog.LevelError {
            fmt.Printf("订阅者2 - 错误: %v\n", record)
        }
    }
}()
```

订阅特性：
- 支持多个订阅者
- 独立的缓冲区大小控制
- 自动资源清理
- 无阻塞设计
- 支持选择性处理

## 日志文件管理

slog 提供了日志文件管理功能，支持日志文件的自动轮转、压缩和清理。

### 基础用法

```go
// 创建日志文件写入器
writer := slog.NewWriter("logs/app.log")

// 创建日志记录器
logger := slog.NewLogger(writer, false, false)

// 开始记录日志
logger.Info("Application started")
```

### 文件写入器配置

```go
writer := slog.NewWriter("logs/app.log").
SetMaxSize(100).      // 设置单个文件最大为100MB
SetMaxAge(7).         // 保留7天的日志
SetMaxBackups(10).    // 最多保留10个备份
SetLocalTime(true).   // 使用本地时间
SetCompress(true)     // 压缩旧日志文件
```

### 文件写入器特性

- **自动轮转**: 当日志文件达到指定大小时自动创建新文件
- **时间戳备份**: 备份文件名格式为 `原文件名-时间戳.扩展名`
- **自动压缩**: 可选择自动压缩旧的日志文件
- **自动清理**: 支持按时间和数量清理旧日志
- **目录管理**: 自动创建日志目录结构
- **错误处理**: 优雅处理文件操作错误

### 配置选项

| 方法 | 默认值 | 描述 |
|------|--------|------|
| `SetMaxSize(size int)` | 100 | 单个日志文件的最大大小（MB） |
| `SetMaxAge(days int)` | 30 | 日志文件保留的最大天数 |
| `SetMaxBackups(count int)` | 30 | 保留的最大备份文件数 |
| `SetLocalTime(local bool)` | true | 是否使用本地时间 |
| `SetCompress(compress bool)` | true | 是否压缩旧日志文件 |

### 文件命名规则

- **当前日志文件**: `app.log`
- **备份文件**: `app-2024-03-20T15-04-05.log`
- **压缩文件**: `app-2024-03-20T15-04-05.log.gz`

### 目录结构示例

```
logs/
  ├── app.log                           # 当前日志文件
  ├── app-2024-03-20T15-04-05.log       # 未压缩的备份
  └── app-2024-03-19T15-04-05.log.gz    # 压缩的备份
```

## 方法列表

### 全局方法

| 方法 | 描述 |
|------|------|
| `NewLogger(w io.Writer, noColor, addSource bool) Logger` | 创建新的日志记录器 |
| `Default(modules ...string) *Logger` | 获取带模块名的默认日志记录器 |
| `SetLevel{Level}()` | 设置全局日志级别（Level可以是Trace/Debug/Info/Warn/Error/Fatal） |
| `UpdateLogLevel(level interface{}) error` | 动态更新日志级别 |
| `WatchLevel(name string, callback func(Level))` | 监听日志级别变化 |
| `UnwatchLevel(name string)` | 取消日志级别监听 |
| `EnableTextLogger()` | 启用文本日志输出 |
| `DisableTextLogger()` | 禁用文本日志输出 |
| `EnableJsonLogger()` | 启用JSON日志输出 |
| `DisableJsonLogger()` | 禁用JSON日志输出 |
| `EnableFormatters(formatters ...formatter.Formatter)` | 启用日志格式化器 |
| `Subscribe(size uint16) (<-chan slog.Record, func())` | 订阅日志记录，返回只读channel和取消函数 |

### Logger方法

| 方法 | 描述 |
|------|------|
| `Debug/Info/Warn/Error/Fatal/Trace(msg string, args ...any)` | 记录不同级别的日志 |
| `Debugf/Infof/Warnf/Errorf/Fatalf/Tracef(format string, args ...any)` | 记录格式化的日志 |
| `With(args ...any) *Logger` | 创建带有额外字段的日志记录器 |
| `WithGroup(name string) *Logger` | 创建带有分组的日志记录器 |
| `GetLevel() Level` | 获取当前日志级别 |
| `SetLevel(level Level) *Logger` | 设置当前记录器的日志级别 |
| `GetSlogLogger() *slog.Logger` | 获取原始的slog.Logger |

## 性能优化

该库在设计时特别注意了性能优化：

- 使用 buffer pool 减少内存分配
- 支持小对象优化
- 异步日志记录支持
- 高效的级别过滤
- 原子操作保证线程安全

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

MIT License