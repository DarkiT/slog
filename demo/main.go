package main

import (
	"context"
	"os"
	"runtime"
	"time"

	log "github.com/darkit/slog"
)

var ctx = context.Background()

func main() {
	opts := log.NewOptions(nil)
	opts.AddSource = false
	opts.ReplaceAttr = func(groups []string, a log.Attr) log.Attr {
		if a.Key == log.LevelKey {
			level := a.Value.Any().(log.Level)
			levelLabel, exists := log.LevelTextNames[level]
			if !exists {
				a.Value = log.StringValue(level.String())
			} else {
				a.Value = log.StringValue(levelLabel)
			}
		}
		return a
	}
	slog := log.New(log.NewConsoleHandler(os.Stdout, opts))
	//log.SetLevelDebug()

	slog.Info("测试", log.String("abc", "def"))
	slog.Debug("xxxx", "xxxx", "xxxx")

	ch := log.GetChannel()
	defer close(ch)

	log.Printf("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())

	log.SetTextLogger(os.Stdout, false)
	//log.SetLevelTrace()

	ctx = log.WithValue(ctx, "context", "value")
	ctx = log.WithValue(ctx, "user", "zishuo")
	log.Prefix("USER")
	log.Info("这是一个信息日志")
	log.Debug("这是一个调试日志", log.Group("data",
		log.Int("width", 4000),
		log.Int("height", 3000),
		log.String("format", "jpeg png"),
		log.Bool("status", true),
		log.Time("time", time.Now()),
		log.Duration("time1", time.Duration(333)),
	))

	//log.SetJsonLogger(os.Stdout, true)
	log.Prefix("SDWAN")
	log.Warn("这是一个警告日志", "aaaa", "bbbb")
	log.Error("这是一个错误日志", "aaaa", "bbbb")
	log.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	log.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	log.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	log.Prefix("")
	log.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	log.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	//log.Panic("这是一个致命错误日志", "age", 18, "name", "foo")

	log.Context(context.Background())
	log.Printf("这是一个格式化打印消息: %s -> %d", "sss", 88888)
	log.Trace("这是一条路由消息: %s -> %d", "sss", 88888)
	log.Info("这是一个测试消息", "aaa", "demo", "bbb", "ddd")
	log.Debug("这是一个调试日志", log.Group("data",
		log.Int("width", 4000),
		log.Int("height", 3000),
		log.String("format", "jpeg png"),
		log.Bool("status", true),
		log.Time("time", time.Now()),
	))

	tk := time.NewTicker(time.Minute * 5)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			return
		case d := <-ch:
			log.WithValue(ctx, "time", time.Now())
			slog.Handler().Enabled(ctx, log.LevelDebug)
			slog.Handler().Handle(ctx, d)
		default:
			log.Warn("这是一个测试消息->: %d", time.Now().UnixMicro())
			time.Sleep(time.Second)
		}
	}
}
