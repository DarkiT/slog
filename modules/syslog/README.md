# modules/syslog

`syslog` 模块已重构为 `Codec + Transport(Writer)` 架构，旧 `Converter/Marshaler` 路径已彻底移除。

## 配置项

```go
modules.Config{
  "network": "udp",          // tcp / udp
  "addr": "127.0.0.1:514",
  "level": "info",           // debug/info/warn/error
  "codec": "default",        // default/json/自定义
}
```

## 默认行为

- 使用 `output.net` 的 `Sender` 建连与重连
- 输出格式为 `@cee: <payload>`
- 默认 `Codec`: `default`

## 扩展点

- 自定义 Codec: `RegisterCodec(codec)`
- 自定义 Writer: 直接通过 `Option.Writer` 注入
