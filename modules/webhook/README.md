# slog: Webhook 处理器

[![tag](https://img.shields.io/github/tag/samber/slog-webhook.svg)](https://github.com/samber/slog-webhook/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
[![GoDoc](https://godoc.org/github.com/samber/slog-webhook?status.svg)](https://pkg.go.dev/github.com/samber/slog-webhook)
![Build Status](https://github.com/samber/slog-webhook/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/samber/slog-webhook)](https://goreportcard.com/report/github.com/samber/slog-webhook)
[![Coverage](https://img.shields.io/codecov/c/github/samber/slog-webhook)](https://codecov.io/gh/samber/slog-webhook)
[![Contributors](https://img.shields.io/github/contributors/samber/slog-webhook)](https://github.com/samber/slog-webhook/graphs/contributors)
[![License](https://img.shields.io/github/license/samber/slog-webhook)](./LICENSE)

为 [slog](https://pkg.go.dev/log/slog) 库提供通用格式化器 + 构建自定义格式化器的助手。

## 🚀 安装

```sh
go get github.com/samber/slog-webhook/v2
```

**兼容性**: go >= 1.23

在 v3.0.0 之前，不会对导出的 API 进行破坏性更改。

## 💡 使用方法

GoDoc: [https://pkg.go.dev/github.com/samber/slog-webhook/v2](https://pkg.go.dev/github.com/samber/slog-webhook/v2)

### 处理器选项

```go
type Option struct {
  // 日志级别 (默认: debug)
  Level     slog.Leveler

  // URL
  Endpoint string
  Timeout  time.Duration // 默认: 10s

  // 可选: 自定义 webhook 事件构建器
  Converter Converter
  // 可选: 自定义序列化器
  Marshaler func(v any) ([]byte, error)
  // 可选: 从上下文获取属性
  AttrFromContext []func(ctx context.Context) []slog.Attr

  // 可选: 参见 slog.HandlerOptions
  AddSource   bool
  ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}
```

其他全局参数:

```go
slogwebhook.SourceKey = "source"
slogwebhook.ContextKey = "extra"
slogwebhook.ErrorKeys = []string{"error", "err"}
slogwebhook.RequestIgnoreHeaders = false
```

### 支持的属性

`slogwebhook.DefaultConverter` 解释以下属性:

| 属性名称    | `slog.Kind`       | 底层类型 |
| ---------------- | ----------------- | --------------- |
| "user"           | group (见下文) |                 |
| "error"          | any               | `error`         |
| "request"        | any               | `*http.Request` |
| 其他属性 | *                 |                 |

其他属性将被注入到 `extra` 字段中。

用户必须是 `slog.Group` 类型。例如:

```go
slog.Group("user",
  slog.String("id", "user-123"),
  slog.String("username", "samber"),
  slog.Time("created_at", time.Now()),
)
```

### 示例

```go
import (
	"fmt"
	"net/http"
	"time"

	slogwebhook "github.com/samber/slog-webhook/v2"

	"log/slog"
)

func main() {
  url := "https://webhook.site/xxxxxx"

  logger := slog.New(slogwebhook.Option{Level: slog.LevelDebug, Endpoint: url}.NewWebhookHandler())
  logger = logger.With("release", "v1.0.0")

  req, _ := http.NewRequest(http.MethodGet, "https://api.screeb.app", nil)
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("X-TOKEN", "1234567890")

  logger.
    With(
      slog.Group("user",
        slog.String("id", "user-123"),
        slog.Time("created_at", time.Now()),
      ),
    ).
    With("request", req).
    With("error", fmt.Errorf("an error")).
    Error("a message")
}
```

输出:

```json
{
  "error": {
    "error": "an error",
    "kind": "*errors.errorString",
    "stack": null
  },
  "extra": {
	"release": "v1.0.0"
  },
  "level": "ERROR",
  "logger": "samber/slog-webhook",
  "message": "a message",
  "request": {
    "headers": {
      "Content-Type": "application/json",
      "X-Token": "1234567890"
    },
    "host": "api.screeb.app",
    "method": "GET",
    "url": {
      "fragment": "",
      "host": "api.screeb.app",
      "path": "",
      "query": {},
      "raw_query": "",
      "scheme": "https",
      "url": "https://api.screeb.app"
    }
  },
  "timestamp": "2023-04-10T14:00:0.000000",
  "user": {
	"id": "user-123",
    "created_at": "2023-04-10T14:00:0.000000"
  }
}
```

### 链路追踪

导入 samber/slog-otel 库。

```go
import (
	slogwebhook "github.com/samber/slog-webhook"
	slogotel "github.com/samber/slog-otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	tracer := tp.Tracer("hello/world")

	ctx, span := tracer.Start(context.Background(), "foo")
	defer span.End()

	span.AddEvent("bar")

	logger := slog.New(
		slogwebhook.Option{
			// ...
			AttrFromContext: []func(ctx context.Context) []slog.Attr{
				slogotel.ExtractOtelAttrFromContext([]string{"tracing"}, "trace_id", "span_id"),
			},
		}.NewWebhookHandler(),
	)

	logger.ErrorContext(ctx, "a message")
}
```

## 🤝 贡献

- 在 Twitter 上 ping 我 [@samuelberthe](https://twitter.com/samuelberthe) (私信、提及，随便什么 :))
- Fork 这个[项目](https://github.com/samber/slog-webhook)
- 修复[开放问题](https://github.com/samber/slog-webhook/issues)或请求新功能

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

![贡献者](https://contrib.rocks/image?repo=samber/slog-webhook)

## 💫 表达你的支持

如果这个项目对你有帮助，请给一个 ⭐️！

[![GitHub Sponsors](https://img.shields.io/github/sponsors/samber?style=for-the-badge)](https://github.com/sponsors/samber)

## 📝 许可证

版权所有 © 2023 [Samuel Berthe](https://github.com/samber)。

本项目采用 [MIT](./LICENSE) 许可证。
