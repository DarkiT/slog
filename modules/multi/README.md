# slog: 处理器链、扇出、路由、故障转移、负载均衡...

[![tag](https://img.shields.io/github/tag/samber/slog-multi.svg)](https://github.com/samber/slog-multi/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
[![GoDoc](https://godoc.org/github.com/samber/slog-multi?status.svg)](https://pkg.go.dev/github.com/samber/slog-multi)
![Build Status](https://github.com/samber/slog-multi/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/samber/slog-multi)](https://goreportcard.com/report/github.com/samber/slog-multi)
[![Coverage](https://img.shields.io/codecov/c/github/samber/slog-multi)](https://codecov.io/gh/samber/slog-multi)
[![Contributors](https://img.shields.io/github/contributors/samber/slog-multi)](https://github.com/samber/slog-multi/graphs/contributors)
[![License](https://img.shields.io/github/license/samber/slog-multi)](./LICENSE)

为 [slog](https://pkg.go.dev/log/slog) 库提供通用格式化器 + 构建自定义格式化器的助手。

## 🚀 安装

```sh
go get github.com/samber/slog-multi
```

**兼容性**: go >= 1.23

在 v2.0.0 之前，不会对导出的 API 进行破坏性更改。

> [!WARNING]
> 请谨慎使用此库，日志处理可能成本很高 (!)

## 💡 使用方法

GoDoc: [https://pkg.go.dev/github.com/samber/slog-multi](https://pkg.go.dev/github.com/samber/slog-multi)

### 广播: `slogmulti.Fanout()`

并行将日志分发到多个 `slog.Handler`。

```go
import (
    slogmulti "github.com/samber/slog-multi"
    "log/slog"
)

func main() {
    logstash, _ := slogmulti.Dial("tcp", "logstash.acme:4242")    // 使用 github.com/netbrain/goautosocket 进行自动重连
    stderr := os.Stderr

    logger := slog.New(
        slogmulti.Fanout(
            slog.NewJSONHandler(logstash, &slog.HandlerOptions{}),  // 传递给第一个处理器: 通过 tcp 传递给 logstash
            slog.NewTextHandler(stderr, &slog.HandlerOptions{}),    // 然后传递给第二个处理器: stderr
            // ...
        ),
    )

    logger.
        With(
            slog.Group("user",
                slog.String("id", "user-123"),
                slog.Time("created_at", time.Now()),
            ),
        ).
        With("environment", "dev").
        With("error", fmt.Errorf("an error")).
        Error("A message")
}
```

Stderr 输出:

```
time=2023-04-10T14:00:0.000000+00:00 level=ERROR msg="A message" user.id=user-123 user.created_at=2023-04-10T14:00:0.000000+00:00 environment=dev error="an error"
```

Netcat 输出:

```json
{
	"time":"2023-04-10T14:00:0.000000+00:00",
	"level":"ERROR",
	"msg":"A message",
	"user":{
		"id":"user-123",
		"created_at":"2023-04-10T14:00:0.000000+00:00"
	},
	"environment":"dev",
	"error":"an error"
}
```

### 路由: `slogmulti.Router()`

并行将日志分发到所有匹配的 `slog.Handler`。

```go
import (
    slogmulti "github.com/samber/slog-multi"
    slogslack "github.com/samber/slog-slack"
    "log/slog"
)

func main() {
    slackChannelUS := slogslack.Option{Level: slog.LevelError, WebhookURL: "xxx", Channel: "supervision-us"}.NewSlackHandler()
    slackChannelEU := slogslack.Option{Level: slog.LevelError, WebhookURL: "xxx", Channel: "supervision-eu"}.NewSlackHandler()
    slackChannelAPAC := slogslack.Option{Level: slog.LevelError, WebhookURL: "xxx", Channel: "supervision-apac"}.NewSlackHandler()

    logger := slog.New(
        slogmulti.Router().
            Add(slackChannelUS, recordMatchRegion("us")).
            Add(slackChannelEU, recordMatchRegion("eu")).
            Add(slackChannelAPAC, recordMatchRegion("apac")).
            Handler(),
    )

    logger.
        With("region", "us").
        With("pool", "us-east-1").
        Error("Server desynchronized")
}

func recordMatchRegion(region string) func(ctx context.Context, r slog.Record) bool {
    return func(ctx context.Context, r slog.Record) bool {
        ok := false

        r.Attrs(func(attr slog.Attr) bool {
            if attr.Key == "region" && attr.Value.Kind() == slog.KindString && attr.Value.String() == region {
                ok = true
                return false
            }

            return true
        })

        return ok
    }
}
```

### 故障转移: `slogmulti.Failover()`

为 `slog.Record` 列出多个目标，而不是在同一个不可用的日志管理系统上重试。

```go
import (
    "net"
    slogmulti "github.com/samber/slog-multi"
    "log/slog"
)


func main() {
    // ncat -l 1000 -k
    // ncat -l 1001 -k
    // ncat -l 1002 -k

    // 列出可用区
    // 使用 github.com/netbrain/goautosocket 进行自动重连
    logstash1, _ := net.Dial("tcp", "logstash.eu-west-3a.internal:1000")
    logstash2, _ := net.Dial("tcp", "logstash.eu-west-3b.internal:1000")
    logstash3, _ := net.Dial("tcp", "logstash.eu-west-3c.internal:1000")

    logger := slog.New(
        slogmulti.Failover()(
            slog.HandlerOptions{}.NewJSONHandler(logstash1, nil),    // 首先发送到此实例
            slog.HandlerOptions{}.NewJSONHandler(logstash2, nil),    // 然后在失败时发送到此实例
            slog.HandlerOptions{}.NewJSONHandler(logstash3, nil),    // 最后在双重失败时发送到此实例
        ),
    )

    logger.
        With(
            slog.Group("user",
                slog.String("id", "user-123"),
                slog.Time("created_at", time.Now()),
            ),
        ).
        With("environment", "dev").
        With("error", fmt.Errorf("an error")).
        Error("A message")
}
```

### 负载均衡: `slogmulti.Pool()`

通过将 `log.Record` 发送到 `slog.Handler` 池来增加日志带宽。

```go
import (
    "net"
    slogmulti "github.com/samber/slog-multi"
    "log/slog"
)

func main() {
    // ncat -l 1000 -k
    // ncat -l 1001 -k
    // ncat -l 1002 -k

    // 列出可用区
    // 使用 github.com/netbrain/goautosocket 进行自动重连
    logstash1, _ := net.Dial("tcp", "logstash.eu-west-3a.internal:1000")
    logstash2, _ := net.Dial("tcp", "logstash.eu-west-3b.internal:1000")
    logstash3, _ := net.Dial("tcp", "logstash.eu-west-3c.internal:1000")

    logger := slog.New(
        slogmulti.Pool()(
            // 将随机选择一个处理器
            slog.HandlerOptions{}.NewJSONHandler(logstash1, nil),
            slog.HandlerOptions{}.NewJSONHandler(logstash2, nil),
            slog.HandlerOptions{}.NewJSONHandler(logstash3, nil),
        ),
    )

    logger.
        With(
            slog.Group("user",
                slog.String("id", "user-123"),
                slog.Time("created_at", time.Now()),
            ),
        ).
        With("environment", "dev").
        With("error", fmt.Errorf("an error")).
        Error("A message")
}
```

### 恢复错误: `slog.RecoverHandlerError()`

返回一个从处理器链的恐慌或错误中恢复的 `slog.Handler`。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	slogmulti "github.com/samber/slog-multi"
	"log/slog"
)

recovery := slogmulti.RecoverHandlerError(
    func(ctx context.Context, record slog.Record, err error) {
        // 只有在后续处理器失败或返回错误时才会被调用
        log.Println(err.Error())
    },
)
sink := NewSinkHandler(...)

logger := slog.New(
    slogmulti.
        Pipe(recovery).
        Handler(sink),
)

err := fmt.Errorf("an error")
logger.Error("a message",
    slog.Any("very_private_data", "abcd"),
    slog.Any("user", user),
    slog.Any("err", err))

// 输出:
// time=2023-04-10T14:00:0.000000+00:00 level=ERROR msg="a message" error.message="an error" error.type="*errors.errorString" user="John doe" very_private_data="********"
```

### 链接: `slogmulti.Pipe()`

实时重写 `log.Record`（例如：出于隐私原因）。

```go
func main() {
    // 第一个中间件: 将 go `error` 类型格式化为对象 {error: "*myCustomErrorType", message: "could not reach https://a.b/c"}
    errorFormattingMiddleware := slogmulti.NewHandleInlineMiddleware(errorFormattingMiddleware)

    // 第二个中间件: 移除 PII
    gdprMiddleware := NewGDPRMiddleware()

    // 最终处理器
    sink := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{})

    logger := slog.New(
        slogmulti.
            Pipe(errorFormattingMiddleware).
            Pipe(gdprMiddleware).
            // ...
            Handler(sink),
    )

    logger.
        With(
            slog.Group("user",
                slog.String("id", "user-123"),
                slog.String("email", "user-123"),
                slog.Time("created_at", time.Now()),
            ),
        ).
        With("environment", "dev").
        Error("A message",
            slog.String("foo", "bar"),
            slog.Any("error", fmt.Errorf("an error")),
        )
}
```

Stderr 输出:

```json
{
    "time":"2023-04-10T14:00:0.000000+00:00",
    "level":"ERROR",
    "msg":"A message",
    "user":{
        "id":"*******",
        "email":"*******",
        "created_at":"*******"
    },
    "environment":"dev",
    "foo":"bar",
    "error":{
        "type":"*myCustomErrorType",
        "message":"an error"
    }
}
```

#### 自定义中间件

中间件必须匹配以下原型:

```go
type Middleware func(slog.Handler) slog.Handler
```

上面的示例使用了:
- 自定义中间件，可参考当前仓库中的 `modules/async_error.go`
- 内联中间件，可参考当前仓库中的 `errors.go`

注意: 自定义中间件的 `WithAttrs` 和 `WithGroup` 方法必须返回新实例，而不是 `this`。

#### 内联处理器

"内联处理器"（又名 lambda）是实现 `slog.Handler` 的快捷方式，它钩住单个方法并代理其他方法。

```go
mdw := slogmulti.NewHandleInlineHandler(
    // 模拟 "Handle()"
    func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
        // [...]
        return nil
    },
)
```

```go
mdw := slogmulti.NewInlineHandler(
    // 模拟 "Enabled()"
    func(ctx context.Context, groups []string, attrs []slog.Attr, level slog.Level) bool {
        // [...]
        return true
    },
    // 模拟 "Handle()"
    func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
        // [...]
        return nil
    },
)
```

#### 内联中间件

"内联中间件"（又名 lambda）是实现中间件的快捷方式，它钩住单个方法并代理其他方法。

```go
// 钩住 `logger.Enabled` 方法
mdw := slogmulti.NewEnabledInlineMiddleware(func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool{
    // [...]
    return next(ctx, level)
})
```

```go
// 钩住 `logger.Handle` 方法
mdw := slogmulti.NewHandleInlineMiddleware(func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
    // [...]
    return next(ctx, record)
})
```

```go
// 钩住 `logger.WithAttrs` 方法
mdw := slogmulti.NewWithAttrsInlineMiddleware(func(attrs []slog.Attr, next func([]slog.Attr) slog.Handler) slog.Handler{
    // [...]
    return next(attrs)
})
```

```go
// 钩住 `logger.WithGroup` 方法
mdw := slogmulti.NewWithGroupInlineMiddleware(func(name string, next func(string) slog.Handler) slog.Handler{
    // [...]
    return next(name)
})
```

钩住所有方法的超级内联中间件。

> 警告: 你最好实现自己的中间件。

```go
mdw := slogmulti.NewInlineMiddleware(
    func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool{
        // [...]
        return next(ctx, level)
    },
    func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error{
        // [...]
        return next(ctx, record)
    },
    func(attrs []slog.Attr, next func([]slog.Attr) slog.Handler) slog.Handler{
        // [...]
        return next(attrs)
    },
    func(name string, next func(string) slog.Handler) slog.Handler{
        // [...]
        return next(name)
    },
)
```

## 🤝 贡献

- 在 Twitter 上 ping 我 [@samuelberthe](https://twitter.com/samuelberthe) (私信、提及，随便什么 :))
- Fork 这个[项目](https://github.com/samber/slog-multi)
- 修复[开放问题](https://github.com/samber/slog-multi/issues)或请求新功能

不要犹豫 ;)

```bash
# 安装一些开发依赖
make tools

# 运行测试
make test
# 或
make watch-test
```

## 👤 贡献者

![贡献者](https://contrib.rocks/image?repo=samber/slog-multi)

## 💫 表达你的支持

如果这个项目对你有帮助，请给一个 ⭐️！

[![GitHub Sponsors](https://img.shields.io/github/sponsors/samber?style=for-the-badge)](https://github.com/sponsors/samber)

## 📝 许可证

版权所有 © 2023 [Samuel Berthe](https://github.com/samber)。

本项目采用 [MIT](../../LICENSE) 许可证。 
