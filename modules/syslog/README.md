# slog: Syslog 处理器

[![tag](https://img.shields.io/github/tag/samber/slog-syslog.svg)](https://github.com/samber/slog-syslog/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
[![GoDoc](https://godoc.org/github.com/samber/slog-syslog?status.svg)](https://pkg.go.dev/github.com/samber/slog-syslog)
![Build Status](https://github.com/samber/slog-syslog/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/samber/slog-syslog)](https://goreportcard.com/report/github.com/samber/slog-syslog)
[![Coverage](https://img.shields.io/codecov/c/github/samber/slog-syslog)](https://codecov.io/gh/samber/slog-syslog)
[![Contributors](https://img.shields.io/github/contributors/samber/slog-syslog)](https://github.com/samber/slog-syslog/graphs/contributors)
[![License](https://img.shields.io/github/license/samber/slog-syslog)](./LICENSE)

一个为 [slog](https://pkg.go.dev/log/slog) Go 库提供的 Syslog 处理器。

## 🚀 安装

```sh
go get github.com/samber/slog-syslog/v2
```

**兼容性**: go >= 1.23

在 v3.0.0 之前，不会对导出的 API 进行破坏性更改。

## 💡 使用方法

GoDoc: [https://pkg.go.dev/github.com/samber/slog-syslog/v2](https://pkg.go.dev/github.com/samber/slog-syslog/v2)

### 处理器选项

```go
type Option struct {
	// 日志级别 (默认: debug)
	Level slog.Leveler

	// 连接到 syslog 服务器
	Writer *syslog.Writer

	// 可选: 自定义 json 负载构建器
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

属性将被注入到日志负载中。

其他全局参数:

```go
slogsyslog.SourceKey = "source"
slogsyslog.ContextKey = "extra"
slogsyslog.ErrorKeys = []string{"error", "err"}
```

### 示例

```go
import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	slogsyslog "github.com/samber/slog-syslog/v2"
)

func main() {
	// ncat -u -l 9999 -k
	writer, err := net.Dial("udp", "localhost:9999")
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slogsyslog.Option{Level: slog.LevelDebug, Writer: writer}.NewSyslogHandler())
	logger = logger.
		With("environment", "dev").
		With("release", "v1.0.0")

	// 记录错误
	logger.
		With("category", "sql").
		With("query.statement", "SELECT COUNT(*) FROM users;").
		With("query.duration", 1*time.Second).
		With("error", fmt.Errorf("could not count users")).
		Error("caramba!")

	// 记录用户注册
	logger.
		With(
			slog.Group("user",
				slog.String("id", "user-123"),
				slog.Time("created_at", time.Now()),
			),
		).
		Info("user registration")
}
```

输出:

```json
@cee: {"timestamp":"2023-04-10T14:00:0.000000", "level":"ERROR", "message":"caramba!", "error":{ "error":"could not count users", "kind":"*errors.errorString", "stack":null }, "extra":{ "environment":"dev", "release":"v1.0.0", "category":"sql", "query.statement":"SELECT COUNT(*) FROM users;", "query.duration": "1s" }}

@cee: {"timestamp":"2023-04-10T14:00:0.000000", "level":"INFO", "message":"user registration", "error":null, "extra":{ "environment":"dev", "release":"v1.0.0", "user":{ "id":"user-123", "created_at":"2023-04-10T14:00:0.000000+00:00"}}}
```

### 链路追踪

导入 samber/slog-otel 库。

```go
import (
	slogsyslog "github.com/samber/slog-syslog"
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
		slogsyslog.Option{
			// ...
			AttrFromContext: []func(ctx context.Context) []slog.Attr{
				slogotel.ExtractOtelAttrFromContext([]string{"tracing"}, "trace_id", "span_id"),
			},
		}.NewSyslogHandler(),
	)

	logger.ErrorContext(ctx, "a message")
}
```

## 🤝 贡献

- 在 Twitter 上 ping 我 [@samuelberthe](https://twitter.com/samuelberthe) (私信、提及，随便什么 :))
- Fork 这个[项目](https://github.com/samber/slog-syslog)
- 修复[开放问题](https://github.com/samber/slog-syslog/issues)或请求新功能

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

![贡献者](https://contrib.rocks/image?repo=samber/slog-syslog)

## 💫 表达你的支持

如果这个项目对你有帮助，请给一个 ⭐️！

[![GitHub Sponsors](https://img.shields.io/github/sponsors/samber?style=for-the-badge)](https://github.com/sponsors/samber)

## 📝 许可证

版权所有 © 2023 [Samuel Berthe](https://github.com/samber)。

本项目采用 [MIT](./LICENSE) 许可证。
