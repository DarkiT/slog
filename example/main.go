package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/darkit/slog"
	"github.com/darkit/slog/example/core"
	"github.com/darkit/slog/multi"
)

var ctx = context.Background()

func main() {
	ch := slog.GetChanRecord(100)
	defer close(ch)

	slog.SetLevelTrace()
	slog.NewLogger(os.Stdout, false, false)
	// slog.DisableTextLogger()
	// slog.EnableJsonLogger()

	slog.WithContext(ctx)

	go iChan(ch)
	// XXX:
	fmt.Println("================默认日志记录器==================")
	slog.Error("HTTP请求消息", "code", 403, "status", "server not response", "server", "10.10.121.88")
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

	fmt.Println("================L1分隔符==================")
	l1 := slog.Default("L1", "xxx").WithValue("os", runtime.GOARCH).WithValue("l1", runtime.NumGoroutine())
	l1.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	l1.Infof("lv: %s", l1.GetLevel().String())
	l1.Warn("这是一个警告日志", "aaaa", "bbbb")
	l1.Error("这是一个错误日志", "aaaa", "bbbb")
	l1.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	l1.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	l1.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	l1.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	l1.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	l1.Debug("这是一个调试日志", slog.Group("data",
		slog.Int("width", 4000),
		slog.Int("height", 3000),
		slog.String("format", "jpeg png"),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("time1", time.Duration(333)),
	))

	fmt.Println("================L2分隔符==================")
	l2 := slog.Default("L2").WithValue("os", runtime.GOARCH).WithValue("l2", runtime.NumGoroutine())
	l2.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	l2.Info("Level", "Level", l2.GetLevel().String())
	l2.Warn("这是一个警告日志", "aaaa", "bbbb")
	l2.Error("这是一个错误日志", "aaaa", "bbbb")
	l2.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	l2.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	l2.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	l2.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	l2.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	l2.Debug("这是一个调试日志", slog.Group("data",
		slog.Int("width", 4000),
		slog.Int("height", 3000),
		slog.String("format", "jpeg png"),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("time1", time.Duration(333)),
	))

	fmt.Println("================L3分隔符==================")
	core.Logger.WithValue("os", runtime.GOARCH).WithValue("l3", runtime.NumGoroutine())
	core.Logger.Info("当前日志等级", "Level", core.Logger.GetLevel().String())
	core.Logger.Warn("这是一个警告日志", "aaaa", "bbbb")
	core.Logger.Error("这是一个错误日志", "aaaa", "bbbb")
	core.Logger.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	core.Logger.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	core.Logger.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	core.Logger.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	core.Logger.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	core.Logger.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	core.Logger.WithGroup("main").Debug("这是一个调试日志", slog.Group("data",
		slog.Int("width", 4000),
		slog.Int("height", 3000),
		slog.String("format", "jpeg png"),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("duration", time.Duration(333)),
	))
	time.Sleep(8 * time.Second)
	// goto XXX
}

func iChan(ch chan slog.Record) {
	conn, _ := multi.Dial("tcp", "127.0.0.1:1900")
	mlog := slog.New(
		multi.Fanout(
			slog.NewJSONHandler(conn, slog.NewOptions(nil)),
			slog.NewConsoleHandler(os.Stdout, false, slog.NewOptions(nil)),
			// ...
		),
	)

	nlog := mlog.With("goroutine", runtime.NumGoroutine())

	tk := time.NewTicker(time.Minute * 5)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			return
		case d := <-ch:
			nlog.Handler().Handle(ctx, d)
		default:
			nlog.With("os", runtime.GOARCH).Info("这是Chan测试消息", "time", time.Now().UnixMicro())
			time.Sleep(time.Second)
		}
	}
}
