# slog 模块注册系统

slog 模块注册系统为日志库提供了强大的插件化架构，让开发者可以轻松地扩展和定制日志功能。

## 🚀 快速开始

### 使用现有模块

```go
import "github.com/darkit/slog/modules"

// 快速启用单个模块
logger := slog.UseFactory("formatter", modules.Config{
    "type":   "time",
    "format": "2006-01-02 15:04:05",
}).Build()

// 链式使用多个模块
logger = slog.UseFactory("formatter", modules.Config{
    "type": "time",
}).UseFactory("webhook", modules.Config{
    "endpoint": "https://api.example.com/webhook",
    "timeout":  "30s",
}).Build()
```

### 配置驱动方式

```go
configs := []modules.ModuleConfig{
    {
        Type:     "formatter",
        Name:     "time-fmt",
        Enabled:  true,
        Priority: 10,
        Config:   modules.Config{"type": "time"},
    },
    {
        Type:     "webhook", 
        Name:     "alert-hook",
        Enabled:  true,
        Priority: 100,
        Config:   modules.Config{
            "endpoint": "https://alerts.example.com",
            "level":    "error",
        },
    },
}

logger := slog.UseConfig(configs).Build()
```

## 📦 模块架构

### 模块类型

| 类型 | 说明 | 优先级 | 用途 |
|------|------|--------|------|
| **Formatter** | 格式化器 | 1-50 | 对日志内容进行格式化处理 |
| **Middleware** | 中间件 | 51-100 | 日志处理的中间层逻辑 |
| **Handler** | 处理器 | 101-150 | 自定义日志处理逻辑 |
| **Sink** | 接收器 | 151-200 | 日志的最终输出目标 |

### 核心接口

```go
// Module 模块接口
type Module interface {
    Name() string                     // 模块名称
    Type() ModuleType                 // 模块类型
    Configure(config Config) error   // 配置模块
    Handler() slog.Handler           // 获取处理器
    Priority() int                   // 执行优先级
    Enabled() bool                   // 是否启用
}

// ModuleFactory 模块工厂函数
type ModuleFactory func(config Config) (Module, error)
```

## 🛠 创建自定义模块

### 步骤1：定义模块结构

```go
package email

import (
    "log/slog"
    "github.com/darkit/slog/modules"
)

// EmailModule 邮件通知模块
type EmailModule struct {
    *modules.BaseModule
    smtpHost     string
    smtpPort     int
    username     string
    password     string
    recipients   []string
    minLevel     slog.Level
}

func NewEmailModule() *EmailModule {
    return &EmailModule{
        BaseModule: modules.NewBaseModule("email", modules.TypeSink, 150),
        minLevel:   slog.LevelWarn, // 默认只发送警告及以上级别
    }
}
```

### 步骤2：实现配置方法

```go
func (e *EmailModule) Configure(config modules.Config) error {
    if err := e.BaseModule.Configure(config); err != nil {
        return err
    }

    // 解析配置
    if host, ok := config["smtp_host"].(string); ok {
        e.smtpHost = host
    }
    
    if port, ok := config["smtp_port"].(int); ok {
        e.smtpPort = port
    }
    
    if username, ok := config["username"].(string); ok {
        e.username = username
    }
    
    if password, ok := config["password"].(string); ok {
        e.password = password
    }
    
    if recipients, ok := config["recipients"].([]string); ok {
        e.recipients = recipients
    }
    
    if level, ok := config["min_level"].(string); ok {
        switch level {
        case "debug":
            e.minLevel = slog.LevelDebug
        case "info":
            e.minLevel = slog.LevelInfo
        case "warn":
            e.minLevel = slog.LevelWarn
        case "error":
            e.minLevel = slog.LevelError
        }
    }

    // 创建处理器
    e.SetHandler(e.createEmailHandler())
    return nil
}
```

### 步骤3：实现处理器逻辑

```go
func (e *EmailModule) createEmailHandler() slog.Handler {
    return &EmailHandler{
        module: e,
    }
}

type EmailHandler struct {
    module *EmailModule
}

func (h *EmailHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.module.Enabled() && level >= h.module.minLevel
}

func (h *EmailHandler) Handle(ctx context.Context, record slog.Record) error {
    if !h.Enabled(ctx, record.Level) {
        return nil
    }
    
    // 构建邮件内容
    subject := fmt.Sprintf("[%s] %s", record.Level, record.Message)
    body := h.formatLogRecord(record)
    
    // 发送邮件
    return h.sendEmail(subject, body)
}

func (h *EmailHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    // 处理属性
    return h
}

func (h *EmailHandler) WithGroup(name string) slog.Handler {
    // 处理分组
    return h
}

func (h *EmailHandler) sendEmail(subject, body string) error {
    // 实现邮件发送逻辑
    // 使用 net/smtp 或第三方邮件库
    return nil
}

func (h *EmailHandler) formatLogRecord(record slog.Record) string {
    // 格式化日志记录为邮件正文
    return fmt.Sprintf("时间: %s\n级别: %s\n消息: %s\n", 
        record.Time.Format("2006-01-02 15:04:05"),
        record.Level,
        record.Message)
}
```

### 步骤4：注册模块工厂

```go
// init 函数中注册模块工厂
func init() {
    modules.RegisterFactory("email", func(config modules.Config) (modules.Module, error) {
        module := NewEmailModule()
        return module, module.Configure(config)
    })
}
```

### 步骤5：使用自定义模块

```go
// 使用自定义邮件模块
logger := slog.UseFactory("email", modules.Config{
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "username":  "your-email@gmail.com", 
    "password":  "your-app-password",
    "recipients": []string{"admin@company.com", "alerts@company.com"},
    "min_level": "error",
}).Build()

logger.Error("系统发生严重错误", "component", "database", "error", "connection timeout")
```

## 📚 现有模块参考

### Formatter 模块

位置：`modules/formatter/`

支持的格式化器类型：
- `time` - 时间格式化
- `error` - 错误信息脱敏  
- `pii` - 个人身份信息脱敏
- `http` - HTTP请求/响应格式化

使用示例：
```go
slog.UseFactory("formatter", modules.Config{
    "type":   "time",
    "format": "2006-01-02 15:04:05",
})
```

### Webhook 模块

位置：`modules/webhook/`

将日志发送到HTTP端点，支持：
- 自定义端点URL
- 超时控制
- 级别过滤
- 重试机制

使用示例：
```go
slog.UseFactory("webhook", modules.Config{
    "endpoint": "https://api.slack.com/webhooks/...",
    "timeout":  "30s",
    "level":    "warn",
})
```

### Syslog 模块

位置：`modules/syslog/`

发送日志到系统日志服务，支持：
- TCP/UDP协议
- 远程syslog服务器
- 标准syslog格式

使用示例：
```go
slog.UseFactory("syslog", modules.Config{
    "network": "udp",
    "addr":    "localhost:514",
    "level":   "info",
})
```

### Multi 模块

位置：`modules/multi/`

Multi模块提供了强大的日志分发、路由、故障转移和负载均衡功能，是构建高可用、高性能日志系统的核心组件。

#### 核心功能

| 功能 | 说明 | 使用场景 |
|------|------|----------|
| **Fanout** | 扇出分发 | 同时发送到多个目标（如文件+网络） |
| **Router** | 条件路由 | 基于日志内容智能路由（如按区域分发） |
| **Failover** | 故障转移 | 高可用性，主备切换 |
| **Pool** | 负载均衡 | 高性能，分散负载 |
| **Pipe** | 中间件链 | 链式处理，数据转换 |
| **TCP** | 自动重连 | 网络日志传输 |
| **Recover** | 错误恢复 | 异常处理和容错 |

#### 1. Fanout（扇出分发）

并行将日志发送到多个处理器，适用于需要同时输出到多个目标的场景。

```go
// 同时输出到文件、控制台和远程服务器
logger := slog.UseFactory("multi", modules.Config{
    "type": "fanout",
    "handlers": []modules.Config{
        {
            "type": "file",
            "config": modules.Config{
                "path": "/var/log/app.log",
                "format": "json",
            },
        },
        {
            "type": "console", 
            "config": modules.Config{
                "format": "text",
                "color": true,
            },
        },
        {
            "type": "tcp",
            "config": modules.Config{
                "endpoint": "logstash.company.com:5044",
                "format": "json",
            },
        },
    },
}).Build()

logger.Info("用户登录", "user_id", "12345", "ip", "192.168.1.100")
// 这条日志会同时写入：文件、控制台显示、发送到logstash
```

#### 2. Router（智能路由）

基于日志内容的条件路由，将不同类型的日志发送到不同的目标。

```go
// 按服务和级别路由日志
logger := slog.UseFactory("multi", modules.Config{
    "type": "router",
    "routes": []modules.Config{
        {
            "name": "error-alerts",
            "condition": modules.Config{
                "level": "error",
                "service": "payment",
            },
            "handler": modules.Config{
                "type": "webhook",
                "endpoint": "https://alerts.company.com/payment",
            },
        },
        {
            "name": "user-events", 
            "condition": modules.Config{
                "component": "auth",
            },
            "handler": modules.Config{
                "type": "file",
                "path": "/var/log/auth.log",
            },
        },
        {
            "name": "performance-metrics",
            "condition": modules.Config{
                "type": "metric",
            },
            "handler": modules.Config{
                "type": "tcp",
                "endpoint": "metrics.company.com:8086",
            },
        },
    },
}).Build()

// 这些日志会被路由到不同目标
logger.Error("支付失败", "service", "payment", "error", "timeout")  // -> 发送告警
logger.Info("用户登录", "component", "auth", "user", "alice")        // -> auth.log
logger.Info("响应耗时", "type", "metric", "duration", "120ms")       // -> 指标收集器
```

#### 3. Failover（故障转移）

提供高可用性，当主要日志目标不可用时自动切换到备用目标。

```go
// 多级故障转移：主数据中心 -> 备数据中心 -> 本地文件
logger := slog.UseFactory("multi", modules.Config{
    "type": "failover",
    "handlers": []modules.Config{
        {
            "name": "primary",
            "type": "tcp", 
            "config": modules.Config{
                "endpoint": "logs-primary.company.com:5044",
                "timeout": "5s",
            },
        },
        {
            "name": "secondary",
            "type": "tcp",
            "config": modules.Config{
                "endpoint": "logs-backup.company.com:5044", 
                "timeout": "10s",
            },
        },
        {
            "name": "local",
            "type": "file",
            "config": modules.Config{
                "path": "/var/log/failover.log",
                "max_size": "100MB",
            },
        },
    },
}).Build()

logger.Error("数据库连接失败")
// 首先尝试发送到主服务器，失败后尝试备用服务器，都失败则写入本地文件
```

#### 4. Pool（负载均衡）

通过处理器池分散负载，提高日志处理性能和吞吐量。

```go
// 负载均衡到多个Elasticsearch节点
logger := slog.UseFactory("multi", modules.Config{
    "type": "pool",
    "strategy": "round_robin", // 或 "random", "hash"
    "handlers": []modules.Config{
        {
            "type": "elasticsearch",
            "config": modules.Config{
                "endpoint": "http://es-node1.company.com:9200",
                "index": "logs-2024",
            },
        },
        {
            "type": "elasticsearch", 
            "config": modules.Config{
                "endpoint": "http://es-node2.company.com:9200",
                "index": "logs-2024",
            },
        },
        {
            "type": "elasticsearch",
            "config": modules.Config{
                "endpoint": "http://es-node3.company.com:9200", 
                "index": "logs-2024",
            },
        },
    },
}).Build()

// 高并发场景下日志会被均匀分发到三个ES节点
for i := 0; i < 10000; i++ {
    logger.Info("订单处理", "order_id", i, "status", "completed")
}
```

#### 5. Pipe（中间件管道）

链式处理日志，支持数据转换、过滤、增强等操作。

```go
// 中间件链：DLP脱敏 -> 添加追踪ID -> 格式转换 -> 发送
logger := slog.UseFactory("multi", modules.Config{
    "type": "pipe",
    "middlewares": []modules.Config{
        {
            "type": "dlp",
            "config": modules.Config{
                "enabled": true,
                "phone": true,
                "email": true,
                "card": true,
            },
        },
        {
            "type": "tracer",
            "config": modules.Config{
                "add_trace_id": true,
                "add_span_id": true,
            },
        },
        {
            "type": "formatter",
            "config": modules.Config{
                "output_format": "json",
                "timestamp_format": "rfc3339",
            },
        },
    ],
    "final_handler": modules.Config{
        "type": "tcp",
        "endpoint": "logs.company.com:5044",
    },
}).Build()

logger.Info("用户注册", "phone", "13812345678", "email", "user@example.com")
// 输出: {"time":"2024-01-01T12:00:00Z","level":"INFO","msg":"用户注册","phone":"138****5678","email":"u***@***.com","trace_id":"abc123","span_id":"def456"}
```

#### 6. TCP自动重连

提供自动重连的TCP客户端，适用于网络日志传输。

```go
// 自动重连的TCP日志传输
logger := slog.UseFactory("multi", modules.Config{
    "type": "tcp",
    "endpoint": "logstash.company.com:5044",
    "auto_reconnect": true,
    "max_retries": 10,
    "retry_interval": "1s",
    "connection_timeout": "30s",
    "write_timeout": "10s",
}).Build()

// 网络中断时会自动重试连接
logger.Error("网络异常", "component", "database", "error", "connection lost")
```

#### 7. Recover（错误恢复）

处理日志处理过程中的panic和错误，提供容错能力。

```go
// 带错误恢复的日志处理
logger := slog.UseFactory("multi", modules.Config{
    "type": "recover",
    "recovery_handler": modules.Config{
        "type": "file",
        "path": "/var/log/errors.log",
    },
    "main_handler": modules.Config{
        "type": "custom", // 可能有bug的自定义处理器
    },
    "on_error": modules.Config{
        "continue": true,        // 出错后继续处理
        "log_error": true,      // 记录错误信息
        "fallback_enabled": true, // 启用备选处理器
    },
}).Build()

logger.Info("这可能触发处理器错误")
// 即使处理器出现panic，也会被捕获并记录到错误日志中，不会导致程序崩溃
```

#### 复合使用示例

将多种功能组合使用，构建企业级日志系统：

```go
// 企业级日志系统：路由 + 故障转移 + 负载均衡 + 恢复
configs := []modules.ModuleConfig{
    {
        Type:     "multi",
        Name:     "enterprise-logging",
        Enabled:  true,
        Priority: 100,
        Config:   modules.Config{
            "type": "router",
            "routes": []modules.Config{
                {
                    "name": "critical-alerts",
                    "condition": modules.Config{"level": "error"},
                    "handler": modules.Config{
                        "type": "failover",
                        "handlers": []modules.Config{
                            {"type": "webhook", "endpoint": "https://alerts.company.com"},
                            {"type": "file", "path": "/var/log/critical.log"},
                        },
                    },
                },
                {
                    "name": "business-logs",
                    "condition": modules.Config{"service": "api"},
                    "handler": modules.Config{
                        "type": "pool", 
                        "handlers": []modules.Config{
                            {"type": "elasticsearch", "endpoint": "http://es1:9200"},
                            {"type": "elasticsearch", "endpoint": "http://es2:9200"},
                            {"type": "elasticsearch", "endpoint": "http://es3:9200"},
                        },
                    },
                },
                {
                    "name": "default",
                    "handler": modules.Config{
                        "type": "fanout",
                        "handlers": []modules.Config{
                            {"type": "file", "path": "/var/log/app.log"},
                            {"type": "tcp", "endpoint": "logs.company.com:5044"},
                        },
                    },
                },
            },
        },
    },
}

logger := slog.UseConfig(configs).Build()

// 不同类型的日志会被智能路由到相应的处理器
logger.Error("支付异常", "service", "payment")      // -> 告警系统 + 本地文件
logger.Info("API请求", "service", "api")          // -> ES集群（负载均衡）
logger.Debug("调试信息")                         // -> 文件 + TCP（扇出）
```

#### 性能优化配置

```go
// 高性能配置
logger := slog.UseFactory("multi", modules.Config{
    "type": "fanout",
    "async": true,              // 异步处理
    "buffer_size": 10000,       // 缓冲区大小
    "flush_interval": "1s",     // 刷新间隔
    "worker_count": 4,          // 工作协程数
    "handlers": []modules.Config{
        {
            "type": "pool",
            "batch_size": 100,      // 批处理大小
            "handlers": []modules.Config{
                {"type": "elasticsearch", "endpoint": "http://es1:9200"},
                {"type": "elasticsearch", "endpoint": "http://es2:9200"},
            },
        },
    },
}).Build()
```

## 🎯 最佳实践

### 1. 模块命名

- 使用描述性的名称
- 避免与现有模块冲突
- 建议使用小写字母和下划线

### 2. 优先级设置

```go
// 推荐的优先级范围
const (
    PriorityFormatter  = 10-50   // 格式化器
    PriorityMiddleware = 51-100  // 中间件  
    PriorityHandler    = 101-150 // 处理器
    PrioritySink      = 151-200  // 接收器
)
```

### 3. 错误处理

```go
func (m *MyModule) Configure(config modules.Config) error {
    if err := m.BaseModule.Configure(config); err != nil {
        return fmt.Errorf("配置基础模块失败: %w", err)
    }
    
    // 验证必需的配置项
    endpoint, ok := config["endpoint"].(string)
    if !ok || endpoint == "" {
        return fmt.Errorf("endpoint配置项是必需的")
    }
    
    return nil
}
```

### 4. 资源管理

```go
type MyModule struct {
    *modules.BaseModule
    conn    net.Conn
    closeCh chan struct{}
}

func (m *MyModule) Configure(config modules.Config) error {
    // 配置逻辑...
    
    // 创建资源
    conn, err := net.Dial("tcp", endpoint)
    if err != nil {
        return err
    }
    m.conn = conn
    m.closeCh = make(chan struct{})
    
    // 启动后台协程
    go m.backgroundWorker()
    
    return nil
}

func (m *MyModule) Close() error {
    close(m.closeCh)
    if m.conn != nil {
        return m.conn.Close()
    }
    return nil
}
```

### 5. 配置验证

```go
type ModuleConfig struct {
    Endpoint string   `json:"endpoint" validate:"required,url"`
    Timeout  string   `json:"timeout" validate:"required"`
    Level    string   `json:"level" validate:"oneof=debug info warn error"`
    Headers  []string `json:"headers"`
}

func validateConfig(config modules.Config) (*ModuleConfig, error) {
    var cfg ModuleConfig
    
    // 类型转换和验证
    if endpoint, ok := config["endpoint"].(string); ok {
        cfg.Endpoint = endpoint
    } else {
        return nil, fmt.Errorf("endpoint必须是字符串类型")
    }
    
    // 更多验证逻辑...
    
    return &cfg, nil
}
```

## 🔍 调试和测试

### 调试模块

```go
func (m *MyModule) Configure(config modules.Config) error {
    // 启用调试日志
    if debug, ok := config["debug"].(bool); ok && debug {
        m.enableDebugLogging()
    }
    
    return nil
}

func (m *MyModule) debugLog(msg string, args ...any) {
    if m.debugEnabled {
        log.Printf("[%s] %s", m.Name(), fmt.Sprintf(msg, args...))
    }
}
```

### 单元测试

```go
func TestEmailModule(t *testing.T) {
    module := NewEmailModule()
    
    config := modules.Config{
        "smtp_host": "localhost",
        "smtp_port": 25,
        "recipients": []string{"test@example.com"},
    }
    
    err := module.Configure(config)
    assert.NoError(t, err)
    assert.Equal(t, "email", module.Name())
    assert.Equal(t, modules.TypeSink, module.Type())
}
```

## 📝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/my-module`)
3. 在 `modules/` 目录下创建模块目录
4. 实现模块接口
5. 添加单元测试
6. 更新文档
7. 提交 Pull Request

## 📖 参考资料

- [模块接口定义](./registry.go)
- [Formatter模块示例](./formatter/)
- [Webhook模块示例](./webhook/)
- [Syslog模块示例](./syslog/)
- [项目主文档](../README.md) 