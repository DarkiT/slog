# slog

[![PkgGoDev](https://pkg.go.dev/badge/github.com/darkit/slog.svg)](https://pkg.go.dev/github.com/darkit/slog)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/slog)](https://goreportcard.com/report/github.com/darkit/slog)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/slog/blob/master/LICENSE)

slog 是一个高性能、功能丰富的 Go 语言日志库，基于 Go 1.23+ 的官方 `log/slog` 包进行扩展。它提供了更灵活的日志级别控制、彩色输出、结构化日志记录、日志脱敏等高级特性。

## 目录
- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [使用指南](#使用指南)
  - [创建 Logger 与配置继承](#创建-logger-与配置继承)
  - [日志级别控制](#日志级别控制)
  - [日志记录方法](#日志记录方法)
  - [日志分组和模块](#日志分组和模块)
  - [输出格式控制](#输出格式控制)
- [日志脱敏（DLP）](#日志脱敏dlp)
- [进度条功能](#进度条功能)
- [模块注册系统](#模块注册系统)
  - [内置模块详解](#内置模块详解)
  - [插件管理器与注册中心](#插件管理器与注册中心)
  - [边界场景提示](#边界场景提示)
- [日志订阅与写入器](#日志订阅机制)
- [常见问题与更多示例](#基础用法)

## 特性

### 🚀 核心功能
- 支持多种日志级别（Trace、Debug、Info、Warn、Error、Fatal）
- 支持同时输出文本和 JSON 格式
- 支持彩色日志输出
- 支持动态调整日志级别
- 支持日志分组和模块化
- 支持结构化字段记录
- 线程安全设计

### 🔒 数据脱敏 (DLP)
- **插拔式脱敏器架构**: 支持动态加载和配置脱敏器
- **智能类型检测**: 自动识别手机号、邮箱、身份证、银行卡等敏感信息
- **高性能缓存**: 使用 xxhash 算法优化缓存键，性能提升74%
- **结构体脱敏**: 支持通过标签自动脱敏结构体字段
- **自定义脱敏规则**: 支持正则表达式和自定义脱敏函数
- **精确脱敏处理**: 优化脱敏算法，正确隐藏身份证生日信息，避免误判普通文本

### ⚡ 性能优化
- **分级对象池**: 小中大三级Buffer池提升内存效率
- **LRU缓存策略**: 替换全清除策略，减少内存压力
- **xxhash缓存键**: 减少哈希碰撞，缓存性能提升74%
- **高性能缓冲设计**: 优化内存分配和回收

### 🎨 用户界面
- **内置丰富的可视化进度条功能**: 支持多种样式和动画效果
- **建造者模式API**: 简化复杂配置，提供优雅的链式调用
- **动态输出和实时更新**: 支持实时进度显示和状态更新

### 🔧 架构设计
- **模块化插件系统**: 从工厂模式简化为插件管理器
- **接口隔离原则**: 按单一职责原则拆分接口
- **结构化错误处理**: 统一错误类型，提升调试体验
- **全局状态管理**: LoggerManager解决全局状态混乱问题

## 安装

> 依赖 Go 1.23 及以上版本。

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

## 使用指南

### 创建 Logger 与配置继承

```go
cfg := slog.DefaultConfig()

// 显式控制实例输出格式
cfg.SetEnableText(true)   // 强制开启文本输出
cfg.SetEnableJSON(false)  // 禁用 JSON 输出

// 也可以选择继承全局开关
cfg.InheritJSONOutput()   // JSON 输出跟随 EnableJSONLogger/DisableJSONLogger

logger := slog.NewLoggerWithConfig(os.Stdout, cfg)

// 全局开关仍然生效
slog.EnableJSONLogger()   // 立即影响所有继承 JSON 配置的实例
logger.Info("configurable logger")
```

- `DefaultConfig` 返回可复用的配置对象；`SetEnableText/SetEnableJSON` 会显式锁定实例的输出格式。
- 调用 `InheritTextOutput/InheritJSONOutput` 时，实例将重新遵循 `EnableTextLogger`、`DisableTextLogger`、`EnableJSONLogger` 等全局函数。
- `NewLogger` 返回遵循全局配置的默认实例，`NewLoggerWithConfig` 允许在同一进程中创建互不影响的独立 logger。

⚠️ 注意：`EnableJSONLogger`、`EnableTextLogger`、`EnableDLPLogger` 等全局开关会立即影响所有选择“继承”模式的记录器。在多租户或多模块进程中，优先为每个 `Logger` 设置显式的 `SetEnableText/SetEnableJSON` 等配置，避免因为其他协程切换全局状态而导致输出格式突变。需要临时调整全局配置时，请结合明确的作用域（例如只在调试阶段打开，并确保退出前恢复），并避免跨 goroutine 共享并写入同一个配置实例。

### 日志级别控制

```go
// 设置全局日志级别
slog.SetLevelDebug()  // Debug级别
slog.SetLevelInfo()   // Info级别
slog.SetLevelWarn()   // Warn级别
slog.SetLevelError()  // Error级别
slog.SetLevelFatal()  // Fatal级别
slog.SetLevelTrace()  // Trace级别

```

### 日志记录方法

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

### 日志分组和模块

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

### 输出格式控制

```go
// 启用/禁用文本日志
slog.EnableTextLogger()
slog.DisableTextLogger()

// 启用/禁用JSON日志
slog.EnableJSONLogger()
slog.DisableJSONLogger()

// 创建带颜色的控制台日志
logger := slog.NewLogger(os.Stdout, false, true) // 最后一个参数控制是否显示源代码位置

// 使用自定义配置继承/覆盖输出开关
cfg := slog.DefaultConfig()
cfg.InheritTextOutput() // 文本输出跟随全局开关
cfg.SetEnableJSON(true) // 显式开启 JSON 输出
logger = slog.NewLoggerWithConfig(os.Stdout, cfg)
logger.Info("格式控制示例", "user", "alice")
```

### 日志脱敏（DLP）

slog 提供了强大的数据泄露防护（DLP）功能，支持文本脱敏和结构体脱敏，自动识别和脱敏敏感信息。

#### 5.1 基础脱敏功能

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

#### 5.2 结构体脱敏

支持对结构体进行自动脱敏，通过 `dlp` 标签指定脱敏规则：

```go
type User struct {
    ID       int64  `dlp:"id_card"`      // 身份证脱敏
    Name     string `dlp:"chinese_name"` // 中文姓名脱敏
    Phone    string `dlp:"mobile_phone"` // 手机号脱敏
    Email    string `dlp:"email"`        // 邮箱脱敏
    Password string `dlp:"password"`     // 密码脱敏
    Age      int    `dlp:"-"`            // 跳过此字段
    Address  string `dlp:"address"`      // 地址脱敏
}

// 使用结构体脱敏
user := &User{
    ID:       622421196903065015,
    Name:     "张三",
    Phone:    "13812345678",
    Email:    "zhangsan@example.com",
    Password: "password123",
    Age:      25,
}

// 基础脱敏（向后兼容）
dlpEngine := dlp.NewDlpEngine()
dlpEngine.Enable()
err := dlpEngine.DesensitizeStruct(user)

// 高级脱敏（推荐）
err = dlpEngine.DesensitizeStructAdvanced(user)

// 批量脱敏
users := []User{...}
err = dlpEngine.BatchDesensitizeStruct(&users)
```

#### 5.3 嵌套结构体脱敏

支持递归处理嵌套结构体、切片、数组和映射：

```go
type UserProfile struct {
    RealName string `dlp:"chinese_name"`
    Address  string `dlp:"address"`
}

type ComplexUser struct {
    BaseInfo UserProfile       `dlp:",recursive"`    // 递归处理嵌套结构体
    Friends  []User            `dlp:",recursive"`    // 递归处理切片
    Settings map[string]string `dlp:",recursive"`    // 递归处理映射
    BankCard string            `dlp:"bank_card"`     // 银行卡脱敏
}

complexUser := &ComplexUser{
    BaseInfo: UserProfile{
        RealName: "李四",
        Address:  "北京市朝阳区某某街道123号",
    },
    Friends: []User{
        {Name: "王五", Phone: "13555666777"},
        {Name: "赵六", Phone: "13444555666"},
    },
    Settings: map[string]string{
        "phone": "13812345678",
        "email": "user@example.com",
    },
    BankCard: "6222020000000000000",
}

err := dlpEngine.DesensitizeStructAdvanced(complexUser)
```

#### 5.4 标签语法

支持灵活的标签配置：

```go
type FlexibleUser struct {
    // 基础脱敏类型
    Name  string `dlp:"chinese_name"`  
    Phone string `dlp:"mobile_phone"`  
    
    // 递归处理嵌套数据
    Profile  UserProfile `dlp:",recursive"`
    Friends  []User      `dlp:",recursive"`
    Settings map[string]string `dlp:",recursive"`
    
    // 自定义脱敏策略
    Token    string `dlp:"custom:my_strategy"`
    
    // 跳过字段
    InternalID string `dlp:"-"`
    Age        int    `dlp:"-"`
    
    // 组合配置
    Data     string `dlp:"email,recursive"`
}
```

支持的标签选项：
- `type_name`: 指定脱敏类型（如 `chinese_name`, `mobile_phone` 等）
- `recursive`: 递归处理嵌套数据结构
- `custom:strategy_name`: 使用自定义脱敏策略
- `-`: 完全跳过此字段

#### 5.5 自定义脱敏策略

```go
// 注册自定义脱敏策略
dlpEngine.GetConfig().RegisterStrategy("my_token", func(s string) string {
    if len(s) <= 8 {
        return "***"
    }
    return s[:4] + "****" + s[len(s)-4:]
})

type CustomUser struct {
    Token    string `dlp:"custom:my_token"`
    APIKey   string `dlp:"custom:api_key"`
}

user := &CustomUser{
    Token:  "abcd1234efgh5678",
    APIKey: "sk-1234567890abcdef",
}

err := dlpEngine.DesensitizeStructAdvanced(user)
// Token: "abcd****5678", APIKey: "sk-1****cdef"
```

#### 5.6 支持的脱敏类型

| 类型 | 标签名 | 描述 | 示例 |
|------|--------|------|------|
| 中文姓名 | `chinese_name` | 保留姓氏，脱敏名字 | 张三 → 张* |
| 身份证号 | `id_card` | 保留前6位和后4位，隐藏生日信息 | 110101199001010001 → 110101********0001 |
| 手机号码 | `mobile_phone` | 保留前3位和后4位 | 13812345678 → 138****5678 |
| 固定电话 | `landline` | 脱敏中间部分 | 010-12345678 → 010-****5678 |
| 电子邮箱 | `email` | 脱敏用户名部分 | user@example.com → u**r@example.com |
| 银行卡号 | `bank_card` | 保留前6位和后4位 | 6222020000000000000 → 622202*****0000 |
| 地址信息 | `address` | 脱敏详细地址 | 北京市朝阳区某某街道123号 → 北京市朝阳区某某街道*** |
| 密码 | `password` | 全部替换为星号 | password123 → *********** |
| 车牌号 | `plate` | 脱敏中间部分 | 京A12345 → 京A***45 |
| IPv4地址 | `ipv4` | 脱敏中间段 | 192.168.1.100 → 192.***.1.100 |
| IPv6地址 | `ipv6` | 脱敏中间段 | 2001:db8::1 → 2001:***::1 |
| JWT令牌 | `jwt` | 脱敏payload部分 | eyJ...abc → eyJ****.abc |
| URL地址 | `url` | 脱敏敏感参数 | http://example.com?token=abc → http://example.com?token=*** |

#### 5.7 批量处理和性能优化

```go
// 批量处理大量数据
users := make([]User, 1000)
for i := 0; i < 1000; i++ {
    users[i] = User{
        Name:  "用户" + strconv.Itoa(i),
        Phone: "13812345678",
        Email: "user" + strconv.Itoa(i) + "@example.com",
    }
}

// 高效批量脱敏
err := dlpEngine.BatchDesensitizeStruct(&users)
if err != nil {
    log.Printf("批量脱敏失败: %v", err)
}
```

#### 5.8 安全特性

- **递归深度限制**: 防止无限递归，最大深度为10层
- **错误隔离**: 单个字段脱敏失败不影响其他字段
- **空值处理**: 正确处理 nil 指针和空值
- **并发安全**: 所有操作都是线程安全的
- **向后兼容**: 保持与原有API的完全兼容性

### 进度条功能

slog 提供了丰富的进度条功能，用于在日志中显示可视化的进度:

```go
// 基本进度条 - 根据时间自动推进
logger.ProgressBar("处理文件中", 5000, 30) // 消息, 总时间(ms), 进度条宽度

// 自定义进度值的进度条
logger.ProgressBarWithValue("处理进度", 75.5, 30) // 显示75.5%的进度

// 输出到特定writer的进度条
file, _ := os.Create("progress.log")
logger.ProgressBarTo("导出数据", 3000, 30, file)

// 带自定义值输出到特定writer
logger.ProgressBarWithValueTo("处理进度", 50.0, 30, os.Stdout)

// 使用自定义选项
opts := slog.DefaultProgressBarOptions()
opts.BarStyle = "block" // 使用方块样式 (可选: "default", "block", "simple")
opts.ShowPercentage = true
opts.TimeFormat = "15:04:05" // 自定义时间格式

// 带选项的进度条
logger.ProgressBarWithOptions("导入数据", 10000, 40, opts)

// 带选项和自定义值的进度条
logger.ProgressBarWithValueAndOptions("分析完成度", 80.0, 40, opts)

// 带选项和自定义值并输出到特定writer的进度条
logger.ProgressBarWithValueAndOptionsTo("处理状态", 65.5, 40, opts, os.Stdout)
```

进度条特性:
- **多种样式**: 支持默认(=)、方块(█)、简单(#-)等多种风格
- **百分比显示**: 可选择是否显示百分比
- **自定义颜色**: 继承日志级别颜色
- **可自定义宽度**: 适应不同终端大小
- **实时更新**: 根据时间自动更新或手动设置进度值
- **自定义输出**: 可以输出到任意writer
- **线程安全**: 所有操作都是并发安全的

⚠️ 提示：进度条与动态动画会在标准输出上产生多行或回退字符，适合 TTY 或纯文本日志。当同时启用 JSON、Webhook、Syslog 等结构化输出时，请将这类效果定向到单独的 `io.Writer`，或在该 logger 上禁用 JSON 输出，避免破坏上游解析。

进度条选项说明:

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `BarStyle` | string | "default" | 进度条样式 ("default", "block", "simple") |
| `ShowPercentage` | bool | true | 是否显示百分比 |
| `TimeFormat` | string | TimeFormat | 时间格式 |
| `LeftBracket` | string | "[" | 左边框字符 |
| `RightBracket` | string | "]" | 右边框字符 |
| `Fill` | string | "=" | 已完成部分填充字符 |
| `Head` | string | ">" | 进度条头部字符 |
| `Empty` | string | " " | 未完成部分填充字符 |

### 模块注册系统

slog 提供了强大的模块注册系统，支持插件化的日志处理组件，让您可以轻松扩展和定制日志功能。

#### 7.1 模块类型

系统支持四种模块类型：

| 模块类型 | 说明 | 优先级 | 示例 |
|----------|------|--------|------|
| Formatter | 格式化器 - 对日志内容进行格式化处理 | 最高 | 时间格式化、脱敏处理 |
| Middleware | 中间件 - 日志处理中间层 | 高 | 过滤器、增强器 |
| Handler | 处理器 - 自定义日志处理逻辑 | 中 | 自定义输出逻辑 |
| Sink | 接收器 - 日志输出目标 | 低 | Webhook、Syslog |

#### 7.2 快速使用内置模块

```go
import "github.com/darkit/slog/modules"

// 方式1: 通过工厂函数快速创建
logger := slog.UseFactory("formatter", modules.Config{
    "type":   "time",
    "format": "2006-01-02 15:04:05",
}).Build()

// 方式2: 链式调用多个模块
logger = slog.UseFactory("formatter", modules.Config{
    "type": "time",
}).UseFactory("webhook", modules.Config{
    "endpoint": "https://api.example.com/webhook",
    "timeout":  "30s",
    "level":    "warn",
}).Build()

logger.Info("Hello World!")
```

#### 7.3 配置驱动方式

```go
// 通过配置文件驱动模块创建
configs := []modules.ModuleConfig{
    {
        Type:     "formatter",
        Name:     "time-formatter",
        Enabled:  true,
        Priority: 10,
        Config: modules.Config{
            "type":   "time",
            "format": "2006-01-02 15:04:05",
        },
    },
    {
        Type:     "webhook",
        Name:     "alert-webhook", 
        Enabled:  true,
        Priority: 100,
        Config: modules.Config{
            "endpoint": "https://alerts.example.com/webhook",
            "timeout":  "10s",
            "level":    "error",
        },
    },
}

logger := slog.UseConfig(configs).Build()
logger.Error("系统错误", "error", "database connection failed")
```

提示：若模块配置来自 JSON/YAML 等动态来源，可直接调用 `modules.Config.Bind` 将弱类型 `map[string]any` 映射为强类型结构体，避免在 `Configure` 中散布显式断言：

```go
var opts struct {
    Endpoint   string        `json:"endpoint"`
    Recipients []string      `json:"recipients"`
    Timeout    time.Duration `json:"timeout"`
}

if err := config.Bind(&opts); err != nil {
    return err
}
```

`Bind` 基于标准库 `encoding/json` 实现，天然兼容字符串形式的 `time.Duration` 等常见类型，并在配置缺失时返回零值，让模块装配更加稳定、优雅。

#### 7.4 内置模块详解

##### Formatter 模块
- **功能概览**: 通过 `Format`、`FormatByType`、`FormatByKind` 等组合函数为 `slog` 属性链提供二次格式化，兼容嵌套 `slog.Group`。
- **关键实现**: 时间格式化器会在传入的 `location` 为空时自动降级为 `time.UTC`；`UnixTimestampFormatter` 仅接受纳秒、微秒、毫秒与秒四档精度，超出范围会直接 panic，部署前务必由配置层校验。
- **边界处理**: `PIIFormatter` 会递归遍历所有子属性并保留 `id`、`*_id`、`-id` 字段原值，长度不超过 5 的字符串会完全遮挡；`HTTPRequestFormatter` / `HTTPResponseFormatter` 在 `ignoreHeaders` 为 `true` 时统一返回 `[hidden]`，避免意外泄露头部信息。
- **配置提示**: 当前适配器使用 `replacement` 字段承载目标字段名，等价于调用 `PIIFormatter("user")`；如需自定义掩码字符，可直接将 formatter 函数组合后通过 `EnableFormatters` 注入。
- **快速示例**:
```go
// 通过模块工厂启用时间与 PII 脱敏
logger := slog.
    UseFactory("formatter", modules.Config{"type": "time", "format": time.RFC3339}).
    UseFactory("formatter", modules.Config{"type": "pii", "replacement": "user"}).
    Build()

logger.Info("profile update", "user", map[string]any{
    "id": "42", "email": "alice@example.com",
})
```

##### Multi 模块
- **功能概览**: 提供 `Fanout`、`Failover`、`Router`、`RecoverHandlerError` 等模式，将多个 `slog.Handler` 组合为一条链路。
- **关键实现**: `Fanout` 在分发前调用 `record.Clone()`，避免下游修改互相干扰；`Failover` 顺序尝试 handler，首个成功后立即终止并返回 `nil`，全部失败时回传最后一个错误。
- **边界处理**: 内部的 `try` 方法会捕获 panic 并转换成错误，因此不会打断主链路；`RoutableHandler` 复制分组信息并在 `WithAttrs` 时重新打平属性，防止跨组丢字段；`WithGroup` 遵循 slog 规范，对空字符串直接返回当前实例，避免无意义层级。
- **扩展建议**: 需要更多策略时，可复用 `MultiAdapter.AddHandler` 追加自定义 handler，再结合 `RecoverHandlerError` 注册统一的告警回调。
- **快速示例**:
```go
multiAdapter := multi.NewMultiAdapter()
multiAdapter.AddHandler(slog.NewJSONHandler(os.Stdout, nil))
multiAdapter.AddHandler(slog.NewTextHandler(os.Stderr, nil))

logger := slog.UseModule(multiAdapter).Build()
logger.Info("同步输出到多个目标", "trace_id", "abc123")
```

##### Webhook 模块
- **功能概览**: 将日志异步转换为 JSON 并通过 HTTP POST 发送到外部服务。
- **关键实现**: `Option.Timeout` 默认 10 秒，`send` 会通过 `context.WithTimeout` 控制请求生命周期；`DefaultConverter` 会展开 `error`、`*http.Request` 与 `user` 字段，并将剩余属性放入 `extra`。
- **边界处理**: `Handle` 在 goroutine 中调用 `send`，错误会被静默丢弃，必须通过外部监控或自定义 `Converter` 注入补偿逻辑；适配器仅在成功拨号 `Endpoint` 时才创建 handler，因此需在部署前验证网络连通性。
- **使用提示**: 若需要复用已有 `http.Client` 或启用连接池，可参考 `send` 实现自定义版本，并通过 `Option.Marshaler` 与 `Option.Converter` 托管。
- **快速示例**:
```go
logger := slog.
    UseFactory("webhook", modules.Config{
        "endpoint": "https://hooks.example.com/logs",
        "timeout":  "15s",
        "level":    "warn",
    }).
    Build()
logger.Warn("订单异常", "order_id", 12345)
```

##### Syslog 模块
- **功能概览**: 生成符合 `@cee` JSON 格式的 payload 并写入远端 syslog。
- **关键实现**: `NewSyslogHandler` 在 `Option.Level` 为空时自动降级到 `slog.LevelDebug`，并在处理时将上下文属性与记录属性统一转为 map；写入操作在 goroutine 中执行，避免阻塞主线程。
- **边界处理**: 异步写入意味着 Writer 的错误会被忽略；使用适配器时若 `network` 或 `addr` 配置为空则不会创建 handler，需要在配置阶段提前检查。
- **使用提示**: 推荐在退出阶段手动关闭 `SyslogAdapter` 持有的 `net.Conn`，或替换为具备自动重连能力的 Writer，实现更稳定的持久连接。
- **快速示例**:
```go
conn, err := net.Dial("udp", "127.0.0.1:514")
if err != nil {
    log.Fatalf("连接 syslog 失败: %v", err)
}

handler := syslog.NewSyslogHandler(conn, &syslog.Option{
    Writer: conn,
    Level:  slog.LevelInfo,
})

logger := slog.NewLogger(handler, false, false)
logger.Info("service started", "pid", os.Getpid())
```

##### Formatter/Handler 组合实践
- **时间戳与时区**: 同时启用 `TimeFormatter` 与 `TimezoneConverter` 时需保证调用顺序，先转换时区再输出字符串。
- **隐私合规**: 将 `PIIFormatter` 与 DLP 模块串联时，可先在 formatter 阶段做结构化裁剪，再交由 DLP 模式识别，降低误杀概率。

#### 7.5 插件管理器与注册中心

- **线程安全**: `PluginManager` 通过 `sync.RWMutex` 与 `atomic.Bool` 管理注册表与启用状态，`EnableAll` / `DisableAll` 会逐一更新快照，适合热插拔场景。
- **统计信息**: `GetStats` 返回深拷贝，包含总数、各类型数量以及每个插件的启用状态与优先级，方便制作仪表盘。
- **配置读取**: `BasePlugin.Configure` 会复制传入 `map`，避免调用方后续写入导致状态串联；读取时请使用 `GetConfig` 单独提取。
- **模块注册中心**: `Registry.Register` 会校验重名模块并按优先级排序，`BaseModule` 默认启用且直接存储配置引用，如需并发修改请在外部复制配置。
- **工厂模式**: 通过 `modules.RegisterFactory` 注册的工厂支持延迟实例化，可结合 `Config.Bind` 自动映射强类型配置结构体。

#### 7.6 边界场景提示

- **配置校验**: `UnixTimestampFormatter` 对非法精度会 panic，建议在加载配置阶段提前校验；`Webhook` 缺少 `endpoint` 时 handler 不会发送任何请求。
- **异步写入**: Webhook 与 Syslog 均在 goroutine 内发送日志，无返回值反馈；关键告警可搭配 `multi.Failover` 或自定义重试逻辑，避免静默失败。
- **资源释放**: Multi 模块不会自动关闭下游资源，组合 Syslog / Webhook 等长连接时需在应用退出阶段手动调用 `Close`。
- **上下文属性**: Webhook 与 Syslog 可通过 `Option.AttrFromContext` 注入额外属性，回调必须幂等且快速，避免放大写入延迟。
- **命名一致性**: 目前 formatter 适配器将 `replacement` 字段作为目标键名使用，既有配置需保持一致；计划后续重构可统一迁移到 `key` 字段。

### 日志订阅机制

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

⚠️ 注意：`Subscribe` 返回的 channel 仍然是固定容量的缓冲队列；如果消费端处理速度跟不上，缓冲写满后主日志路径会被阻塞。高吞吐场景下建议：
- 根据业务峰值调大缓冲区容量；
- 为订阅者准备独立的消费 goroutine，并妥善处理错误；
- 在需要完全异步的场景中，自行实现带丢弃策略的桥接或使用队列系统。

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
| `EnableTextLogger()` | 启用文本日志输出 |
| `DisableTextLogger()` | 禁用文本日志输出 |
| `EnableJSONLogger()` | 启用JSON日志输出 |
| `DisableJSONLogger()` | 禁用JSON日志输出 |
| `EnableFormatters(formatters ...formatter.Formatter)` | 启用日志格式化器 |
| `EnableDLPLogger()` | 启用日志脱敏功能 |
| `DisableDLPLogger()` | 禁用日志脱敏功能 |
| `Subscribe(size uint16) (<-chan slog.Record, func())` | 订阅日志记录，返回只读channel和取消函数 |
| `ProgressBar(msg string, durationMs int, barWidth int, level ...Level) *Logger` | 显示带有可视化进度条的日志 |
| `ProgressBarWithValue(msg string, progress float64, barWidth int, level ...Level)` | 显示指定进度值的进度条 |
| `ProgressBarWithValueTo(msg string, progress float64, barWidth int, writer io.Writer, level ...Level)` | 显示指定进度值的进度条并输出到指定writer |
| `ProgressBarWithOptions(msg string, durationMs int, barWidth int, opts progressBarOptions, level ...Level) *Logger` | 显示可高度定制的进度条 |
| `ProgressBarWithOptionsTo(msg string, durationMs int, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) *Logger` | 显示可高度定制的进度条并输出到指定writer |
| `ProgressBarWithValueAndOptions(msg string, progress float64, barWidth int, opts progressBarOptions, level ...Level)` | 显示指定进度值的定制进度条 |
| `ProgressBarWithValueAndOptionsTo(msg string, progress float64, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level)` | 显示指定进度值的定制进度条并输出到指定writer |

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
| `ProgressBar(msg string, durationMs int, barWidth int, level ...Level) *Logger` | 显示带有可视化进度条的日志 |
| `ProgressBarWithValue(msg string, progress float64, barWidth int, level ...Level)` | 显示指定进度值的进度条 |
| `ProgressBarWithValueTo(msg string, progress float64, barWidth int, writer io.Writer, level ...Level)` | 显示指定进度值的进度条并输出到指定writer |
| `ProgressBarWithOptions(msg string, durationMs int, barWidth int, opts progressBarOptions, level ...Level) *Logger` | 显示可高度定制的进度条 |
| `ProgressBarWithOptionsTo(msg string, durationMs int, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) *Logger` | 显示可高度定制的进度条并输出到指定writer |
| `ProgressBarWithValueAndOptions(msg string, progress float64, barWidth int, opts progressBarOptions, level ...Level)` | 显示指定进度值的定制进度条 |
| `ProgressBarWithValueAndOptionsTo(msg string, progress float64, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level)` | 显示指定进度值的定制进度条并输出到指定writer |
| `Dynamic(msg string, frames int, interval int, writer ...io.Writer)` | 动态输出带点号动画效果 |
| `Progress(msg string, durationMs int, writer ...io.Writer)` | 显示进度百分比 |
| `Countdown(msg string, seconds int, writer ...io.Writer)` | 显示倒计时 |
| `Loading(msg string, seconds int, writer ...io.Writer)` | 显示加载动画 |

## 数据脱敏 (DLP) 功能

完整的脱敏类型、结构体标签、批量处理与自定义策略示例已在前文的 [日志脱敏（DLP）](#日志脱敏dlp) 章节详细介绍。此处作为方法索引保留标题，避免重复内容，建议直接跳转查看该章节以获取最新的指导。

## 性能优化

该库在设计时特别注意了性能优化：

### 🚀 核心性能特性
- **分级对象池**: 小中大三级Buffer池，提升95%+内存复用率
- **xxhash缓存**: 缓存键生成性能提升74%，零哈希碰撞
- **LRU缓存策略**: 智能内存管理，替换全清除策略
- **原子操作**: 保证线程安全的同时最小化锁竞争

### 📊 性能基准
- **进度条渲染**: 从30.4ms/op优化到<1ms/op (3000%+提升)
- **DLP缓存**: 从573.3ns/op优化到149.2ns/op (74%提升)  
- **内存分配**: 分级池系统减少90%+内存分配
- **并发性能**: 支持高并发场景下的线性性能扩展

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

MIT License
