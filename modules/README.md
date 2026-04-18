# modules

`modules` 子系统已完成去历史化，当前仅保留可维护的核心能力。

## 当前稳定能力

- `formatter`：属性格式化链
- `multi`：`Fanout` 多路分发
- `output/net`：通用 TCP/UDP 输出（codec 可扩展）
- `syslog`：CEE 前缀输出（codec 可扩展）
- `webhook`：HTTP 输出（codec + transport）

## 注册与使用

模块统一通过 `modules.RegisterFactory` 注册，通过 `modules.CreateModule` 创建，最终通过：

```go
logger := slog.Default().Use(module)
```

## 设计约束

- 不再保留旧 `Converter` 兼容路径
- 不再保留反射式 formatter 适配分支
- 新扩展统一走强类型接口与可测试的最小抽象

## Async 执行器配置

模块内的 `RunAsync` 使用有界队列 + worker 池，默认配置为：

- `WorkerCount=4`
- `QueueSize=256`

可在运行时调整：

```go
modules.SetAsyncExecutorOptions(modules.AsyncExecutorOptions{
    WorkerCount: 8,
    QueueSize:   1024,
})
```

读取当前配置：

```go
opts := modules.GetAsyncExecutorOptions()
```

队列满时采用 non-blocking 丢弃策略，不阻塞日志主链路。
