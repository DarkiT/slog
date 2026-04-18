# modules/output/net

`output.net` 是统一网络输出模块，用于把日志发送到任意 `tcp` / `udp` 服务端，不要求对端是 syslog 协议。

## 核心能力

- 通用网络传输：`network + addr`
- 延迟建连：首次写入时建立连接
- 断线重连：写失败后自动重连一次
- Backoff：避免目标不可达时频繁重拨
- 异步发送：不阻塞主日志路径

## 模块配置

通过 `modules.Config` 传入：

```go
modules.Config{
  "network": "tcp",          // tcp / udp
  "addr": "127.0.0.1:9000",  // 目标地址
  "level": "info",           // debug/info/warn/error
  "codec": "raw",            // raw / json / 自定义注册名
  "dial_timeout": "3s",
  "write_timeout": "3s",
  "delimiter": "\n",         // 每条日志末尾分隔符
}
```

## Builder 用法

```go
logger := slog.NewLoggerBuilder().
  UseNetOutput(&outputnet.SenderOption{
    Network:   "udp",
    Addr:      "127.0.0.1:9000",
    Delimiter: []byte("\n"),
  }).
  Build()
```

默认编码是行文本：`level=<LEVEL> msg=<MESSAGE> key=value...`。

可选内置 codec：

- `raw`：行文本
- `json`：JSON payload
