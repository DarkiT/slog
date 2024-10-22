# slog

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
slog.EnableFormatters(formatter.SensitiveFormatter)

// 使用脱敏日志
logger.Info("User data",
"credit_card", "1234-5678-9012-3456", // 将被自动脱敏
"phone", "13800138000",               // 将被自动脱敏
)
```

### 6. 高级功能

```go
// 获取日志记录通道
recordChan := slog.GetChanRecord(1000) // 指定缓冲大小

// 自定义属性
logger.With(
slog.String("app", "myapp"),
slog.Int("version", 1),
slog.Time("start_time", time.Now()),
).Info("Application started")

// 获取原始slog.Logger
slogLogger := logger.GetSlogLogger()
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

## 版本

当前版本：v0.0.18