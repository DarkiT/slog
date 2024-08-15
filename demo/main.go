package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/darkit/slog"
	"github.com/darkit/slog/multi"
)

var (
	ctx    = context.Background()
	l1, l2 *slog.Logger
)

func main() {
	ch := slog.GetChannel()
	defer close(ch)

	slog.SetLevelTrace()

	slog.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	slog.WithValue("Slog", runtime.NumGoroutine())
	slog.Warn("这是一个警告日志", "aaaa", "bbbb")
	slog.Error("这是一个错误日志", "aaaa", "bbbb")
	slog.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	slog.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	slog.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	slog.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	slog.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	slog.Debug("这是一个调试日志", slog.Group("data",
		slog.Int("width", 4000),
		slog.Int("height", 3000),
		slog.String("format", "jpeg png"),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("time1", time.Duration(333)),
	))

	go iChan(ch)
XXX:
	fmt.Println("================分隔符==================")
	l1 = slog.Default("L1")
	l1.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	l1.WithValue("L1", runtime.NumGoroutine())
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

	fmt.Println("================分隔符==================")
	l2 = slog.Default("L2")
	l2.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	l2.WithValue("L2", runtime.NumGoroutine())
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

	fmt.Println("================分隔符==================")

	time.Sleep(5 * time.Second)
	goto XXX
}

func iChan(ch chan slog.Record) {

	conn, _ := multi.Dial("tcp", "127.0.0.1:1900")
	mlog := slog.New(
		multi.Fanout(
			slog.NewJSONHandler(conn, slog.NewOptions(nil)),
			slog.NewConsoleHandler(os.Stdout, slog.NewOptions(nil)),
			// ...
		),
	)

	mlog.With("goroutine", runtime.NumGoroutine())
	mlog.Handler().Enabled(ctx, slog.LevelDebug)

	tk := time.NewTicker(time.Minute * 5)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			return
		case d := <-ch:
			mlog.Handler().Handle(ctx, d)
		default:
			mlog.With("$service", "Mlog").With("os", runtime.GOARCH).Warn("这是Chan测试消息", "time", time.Now().UnixMicro())
			time.Sleep(time.Second)
		}
	}
}
