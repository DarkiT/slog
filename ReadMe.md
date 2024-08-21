### 说明
log 包封装了 slog 包，提供了更简单的接口。并且提供了一个全局的 logger(同时包含了使用[slog.TextHandler]和[slog.JSONHandler]的Logger)，可以直接使用。


# Example

```go
slog.SetLevelInfo()

slog.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
slog.Warn("这是一个警告日志", "aaaa", "bbbb")
slog.Error("这是一个错误日志", "aaaa", "bbbb")
slog.Info("这是一个信息日志: %s -> %d", "Info", 88888)
slog.Debug("这是一个调试日志: %s -> %d", "Debug", 88888)
slog.Warnf("这是一个警告日志: %s -> %d", "Warnf", 88888)
slog.Printf("这是一个信息日志: %s -> %d", "Printf", 88888)
slog.Trace("这是一个路由日志: %s -> %d", "Trace", 88888)
slog.WithGroup("slog").Debug("这是一个调试日志", slog.Group("data",
    slog.Int("width", 4000),
    slog.Int("height", 3000),
    slog.String("format", "jpeg png"),
    slog.Bool("status", true),
    slog.Time("time", time.Now()),
    slog.Duration("duration", time.Duration(333)),
))

l1 := slog.Default("apps", "module").WithValue("os", runtime.GOARCH).WithValue("l1", runtime.NumGoroutine())
l1.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
l1.Infof("lv: %s", l1.GetLevel().String())
l1.Warn("这是一个警告日志", "aaaa", "bbbb")
l1.Error("这是一个错误日志", "aaaa", "bbbb")
l1.Info("这是一个信息日志: %s -> %d", "sss", 88888)
l1.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
l1.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
l1.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
l1.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
l1.SetLevel(slog.LevelDebug)
l1.Debug("这是一个调试日志", slog.Group("data",
    slog.Int("width", 4000),
    slog.Int("height", 3000),
    slog.String("format", "jpeg png"),
    slog.Bool("status", true),
    slog.Time("time", time.Now()),
    slog.Duration("time1", time.Duration(333)),
))
```