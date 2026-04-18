# modules/webhook

`webhook` 模块已重构为 `Codec + Transport` 架构，旧 `Converter` 路径已彻底移除。

## 配置项

```go
modules.Config{
  "endpoint": "https://example.com/hook",
  "timeout": "10s",
  "level": "info",   // debug/info/warn/error
  "codec": "default",// default/json/自定义
}
```

## 默认行为

- 默认 `Transport`: `HTTPTransport`（POST JSON）
- 默认 `Codec`: `default`

## 扩展点

- 自定义 Codec: `RegisterCodec(codec)`
- 自定义 Transport: 在 `Option.Transport` 注入实现
