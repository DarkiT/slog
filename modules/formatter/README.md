# slog: 属性格式化

[![tag](https://img.shields.io/github/tag/samber/slog-formatter.svg)](https://github.com/samber/slog-formatter/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-%23007d9c)
[![GoDoc](https://godoc.org/github.com/samber/slog-formatter?status.svg)](https://pkg.go.dev/github.com/samber/slog-formatter)
![Build Status](https://github.com/samber/slog-formatter/actions/workflows/test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/samber/slog-formatter)](https://goreportcard.com/report/github.com/samber/slog-formatter)
[![Coverage](https://img.shields.io/codecov/c/github/samber/slog-formatter)](https://codecov.io/gh/samber/slog-formatter)
[![Contributors](https://img.shields.io/github/contributors/samber/slog-formatter)](https://github.com/samber/slog-formatter/graphs/contributors)
[![License](https://img.shields.io/github/license/samber/slog-formatter)](./LICENSE)

为 [slog](https://pkg.go.dev/log/slog) 库提供通用格式化器 + 构建自定义格式化器的助手。

**处理器:**
- [NewFormatterHandler](#NewFormatterHandler): 主处理器

**通用格式化器:**
- [TimeFormatter](#TimeFormatter): 将 `time.Time` 转换为可读字符串
- [UnixTimestampFormatter](#UnixTimestampFormatter): 将 `time.Time` 转换为 unix 时间戳
- [TimezoneConverter](#TimezoneConverter): 将 `time.Time` 设置为不同的时区
- [ErrorFormatter](#ErrorFormatter): 将 go error 转换为可读错误
- [HTTPRequestFormatter](#HTTPRequestFormatter-和-HTTPResponseFormatter): 将 *http.Request 转换为可读对象
- [HTTPResponseFormatter](#HTTPRequestFormatter-和-HTTPResponseFormatter): 将 *http.Response 转换为可读对象
- [PIIFormatter](#PIIFormatter): 隐藏私人个人身份信息 (PII)
- [IPAddressFormatter](#IPAddressFormatter): 从日志中隐藏 IP 地址
- [FlattenFormatterMiddleware](#FlattenFormatterMiddleware): 返回递归展平属性的格式化器中间件

**自定义格式化器:**
- [Format](#Format): 将任何属性传递到格式化器
- [FormatByKind](#FormatByKind): 将匹配 `slog.Kind` 的属性传递到格式化器
- [FormatByType](#FormatByType): 将匹配泛型类型的属性传递到格式化器
- [FormatByKey](#FormatByKey): 将匹配键的属性传递到格式化器
- [FormatByFieldType](#FormatByFieldType): 将同时匹配键和泛型类型的属性传递到格式化器
- [FormatByGroup](#FormatByGroup): 将组下的属性传递到格式化器
- [FormatByGroupKey](#FormatByGroupKey): 将组下匹配键的属性传递到格式化器
- [FormatByGroupKeyType](#FormatByGroupKeyType): 将组下匹配键且匹配泛型类型的属性传递到格式化器

## 🚀 安装

```sh
go get github.com/samber/slog-formatter
```

**兼容性**: go >= 1.23

在 v2.0.0 之前，不会对导出的 API 进行破坏性更改。

⚠️ 警告:
- 在某些情况下，你应该考虑实现 `slog.LogValuer` 而不是使用此库。
- 请谨慎使用此库，日志处理可能成本很高 (!)

## 🚀 快速开始

以下示例有 3 个格式化器，用于匿名化数据、格式化错误和格式化用户。👇

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

formatter1 := slogformatter.FormatByKey("very_private_data", func(v slog.Value) slog.Value {
    return slog.StringValue("***********")
})
formatter2 := slogformatter.ErrorFormatter("error")
formatter3 := slogformatter.FormatByType(func(u User) slog.Value {
	return slog.StringValue(fmt.Sprintf("%s %s", u.firstname, u.lastname))
})

logger := slog.New(
    slogformatter.NewFormatterHandler(formatter1, formatter2, formatter3)(
        slog.NewTextHandler(os.Stdout, nil),
    ),
)

err := fmt.Errorf("an error")
logger.Error("a message",
    slog.Any("very_private_data", "abcd"),
    slog.Any("user", user),
    slog.Any("err", err))

// 输出:
// time=2023-04-10T14:00:0.000000+00:00 level=ERROR msg="a message" error.message="an error" error.type="*errors.errorString" user="John doe" very_private_data="********"
```

## 💡 规范

GoDoc: [https://pkg.go.dev/github.com/samber/slog-formatter](https://pkg.go.dev/github.com/samber/slog-formatter)

### NewFormatterHandler

返回一个应用格式化器的 slog.Handler。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

type User struct {
	email     string
	firstname string
	lastname  string
}

formatter1 := slogformatter.FormatByKey("very_private_data", func(v slog.Value) slog.Value {
    return slog.StringValue("***********")
})
formatter2 := slogformatter.ErrorFormatter("error")
formatter3 := slogformatter.FormatByType(func(u User) slog.Value {
	return slog.StringValue(fmt.Sprintf("%s %s", u.firstname, u.lastname))
})

logger := slog.New(
    slogformatter.NewFormatterHandler(formatter1, formatter2, formatter3)(
        slog.NewTextHandler(os.StdErr, nil),
    ),
)

err := fmt.Errorf("an error")
logger.Error("a message",
    slog.Any("very_private_data", "abcd"),
    slog.Any("user", user),
    slog.Any("err", err))

// 输出:
// time=2023-04-10T14:00:0.000000+00:00 level=ERROR msg="a message" error.message="an error" error.type="*errors.errorString" user="John doe" very_private_data="********"
```

### TimeFormatter

将 `time.Time` 转换为可读字符串。

```go
slogformatter.NewFormatterHandler(
    slogformatter.TimeFormatter(time.DateTime, time.UTC),
)
```

### UnixTimestampFormatter

将 `time.Time` 转换为 unix 时间戳。

```go
slogformatter.NewFormatterHandler(
    slogformatter.UnixTimestampFormatter(time.Millisecond),
)
```

### TimezoneConverter

将 `time.Time` 设置为不同的时区。

```go
slogformatter.NewFormatterHandler(
    slogformatter.TimezoneConverter(time.UTC),
)
```

### ErrorFormatter

将 Go error 转换为可读错误。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

logger := slog.New(
    slogformatter.NewFormatterHandler(
        slogformatter.ErrorFormatter("error"),
    )(
        slog.NewTextHandler(os.Stdout, nil),
    ),
)

err := fmt.Errorf("an error")
logger.Error("a message", slog.Any("error", err))

// 输出:
// {
//   "time":"2023-04-10T14:00:0.000000+00:00",
//   "level": "ERROR",
//   "msg": "a message",
//   "error": {
//     "message": "an error",
//     "type": "*errors.errorString"
//     "stacktrace": "main.main()\n\t/Users/samber/src/github.com/samber/slog-formatter/example/example.go:108 +0x1c\n"
//   }
// }
```

### HTTPRequestFormatter 和 HTTPResponseFormatter

将 *http.Request 和 *http.Response 转换为可读对象。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

logger := slog.New(
    slogformatter.NewFormatterHandler(
        slogformatter.HTTPRequestFormatter(false),
        slogformatter.HTTPResponseFormatter(false),
    )(
        slog.NewJSONHandler(os.Stdout, nil),
    ),
)

req, _ := http.NewRequest(http.MethodGet, "https://api.screeb.app", nil)
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-TOKEN", "1234567890")

res, _ := http.DefaultClient.Do(req)

logger.Error("a message",
    slog.Any("request", req),
    slog.Any("response", res))
```

### PIIFormatter

隐藏私人个人身份信息 (PII)。

ID 保持原样。长度超过 5 个字符的值有明文前缀。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

logger := slog.New(
    slogformatter.NewFormatterHandler(
        slogformatter.PIIFormatter("user"),
    )(
        slog.NewTextHandler(os.Stdout, nil),
    ),
)

logger.
    With(
        slog.Group(
            "user",
            slog.String("id", "bd57ffbd-8858-4cc4-a93b-426cef16de61"),
            slog.String("email", "foobar@example.com"),
            slog.Group(
                "address",
                slog.String("street", "1st street"),
                slog.String("city", "New York"),
                slog.String("country", "USA"),
                slog.Int("zip", 12345),
            ),
        ),
    ).
    Error("an error")

// 输出:
// {
//   "time":"2023-04-10T14:00:0.000000+00:00",
//   "level": "ERROR",
//   "msg": "an error",
//   "user": {
//     "id": "bd57ffbd-8858-4cc4-a93b-426cef16de61",
//     "email": "foob*******",
//     "address": {
//       "street": "1st *******",
//       "city": "New *******",
//       "country": "*******",
//       "zip": "*******"
//     }
//   }
// }
```

### IPAddressFormatter

将 IP 地址转换为 "********"。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	"log/slog"
)

logger := slog.New(
    slogformatter.NewFormatterHandler(
        slogformatter.IPAddressFormatter("ip_address"),
    )(
        slog.NewTextHandler(os.Stdout, nil),
    ),
)

logger.
    With("ip_address", "1.2.3.4").
    Error("an error")

// 输出:
// {
//   "time":"2023-04-10T14:00:0.000000+00:00",
//   "level": "ERROR",
//   "msg": "an error",
//   "ip_address": "*******",
// }
```

### FlattenFormatterMiddleware

递归展平属性的格式化器中间件。

```go
import (
	slogformatter "github.com/samber/slog-formatter"
	slogmulti "github.com/samber/slog-multi"
	"log/slog"
)

logger := slog.New(
    slogmulti.
        Pipe(slogformatter.FlattenFormatterMiddlewareOptions{Separator: ".", Prefix: "attrs", IgnorePath: false}.NewFlattenFormatterMiddlewareOptions()).
        Handler(slog.NewJSONHandler(os.Stdout, nil)),
)

logger.
    With("email", "samuel@acme.org").
    With("environment", "dev").
    WithGroup("group1").
    With("hello", "world").
    WithGroup("group2").
    With("hello", "world").
    Error("A message", "foo", "bar")

// 输出:
// {
//   "time": "2023-05-20T22:14:55.857065+02:00",
//   "level": "ERROR",
//   "msg": "A message",
//   "attrs.email": "samuel@acme.org",
//   "attrs.environment": "dev",
//   "attrs.group1.hello": "world",
//   "attrs.group1.group2.hello": "world",
//   "foo": "bar"
// }
```

### Format

将每个属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.Format(func(groups []string, key string, value slog.Value) slog.Value {
        // 隐藏 "user" 组下的所有内容
        if lo.Contains(groups, "user") {
            return slog.StringValue("****")
        }

        return value
    }),
)
```

### FormatByKind

将匹配 `slog.Kind` 的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByKind(slog.KindDuration, func(value slog.Value) slog.Value {
        return ...
    }),
)
```

### FormatByType

将匹配泛型类型的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    // 格式化自定义错误类型
    slogformatter.FormatByType[*customError](func(err *customError) slog.Value {
        return slog.GroupValue(
            slog.Int("code", err.code),
            slog.String("message", err.msg),
        )
    }),
    // 格式化其他错误
    slogformatter.FormatByType[error](func(err error) slog.Value {
        return slog.GroupValue(
            slog.Int("code", err.Error()),
            slog.String("type", reflect.TypeOf(err).String()),
        )
    }),
)
```

⚠️ 在可能的情况下考虑实现 `slog.LogValuer`:

```go
type customError struct {
    ...
}

func (customError) Error() string {
    ...
}

// 实现 slog.LogValuer
func (customError) LogValue() slog.Value {
	return slog.StringValue(...)
}
```

### FormatByKey

将匹配键的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByKey("abcd", func(value slog.Value) slog.Value {
        return ...
    }),
)
```

### FormatByFieldType

将同时匹配键和泛型类型的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByFieldType[User]("user", func(u User) slog.Value {
        return ...
    }),
)
```

### FormatByGroup

将组下的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByGroup([]{"user", "address"}, func(attr []slog.Attr) slog.Value {
        return ...
    }),
)
```

### FormatByGroupKey

将组下匹配键的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByGroupKey([]{"user", "address"}, "country", func(value slog.Value) slog.Value {
        return ...
    }),
)
```

### FormatByGroupKeyType

将组下匹配键且匹配泛型类型的属性传递到格式化器。

```go
slogformatter.NewFormatterHandler(
    slogformatter.FormatByGroupKeyType[string]([]{"user", "address"}, "country", func(value string) slog.Value {
        return ...
    }),
)
```

## 🤝 贡献

- 在 Twitter 上 ping 我 [@samuelberthe](https://twitter.com/samuelberthe) (私信、提及，随便什么 :))
- Fork 这个[项目](https://github.com/samber/slog-formatter)
- 修复[开放问题](https://github.com/samber/slog-formatter/issues)或请求新功能

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

![贡献者](https://contrib.rocks/image?repo=samber/slog-formatter)

## 💫 表达你的支持

如果这个项目对你有帮助，请给一个 ⭐️！

[![GitHub Sponsors](https://img.shields.io/github/sponsors/samber?style=for-the-badge)](https://github.com/sponsors/samber)

## 📝 许可证

版权所有 © 2023 [Samuel Berthe](https://github.com/samber)。

本项目采用 [MIT](./LICENSE) 许可证。 