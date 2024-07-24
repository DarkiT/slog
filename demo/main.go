package main

import (
	"context"
	"os"
	"time"

	log "github.com/darkit/slog"
)

func main() {

	ch := log.GetChannel()
	defer close(ch)

	log.SetTextLogger(os.Stdout, true)
	//log.SetJsonLogger(os.Stdout, true)
	log.SetLevelDebug()

	log.WithValue("context", "value")
	log.WithValue("user", "zishuo")
	log.Info("这是一个信息日志")
	log.Debug("这是一个调试日志", log.Group("data",
		log.Int("width", 4000),
		log.Int("height", 3000),
		log.String("format", "jpeg png"),
		log.Bool("status", true),
		log.Time("time", time.Now()),
		log.Duration("time1", time.Duration(333)),
	))

	log.Warn("这是一个警告日志", "aaaa", "bbbb")
	log.Error("这是一个错误日志", "aaaa", "bbbb")
	log.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	log.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	log.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	log.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	log.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	//log.Panic("这是一个致命错误日志", "age", 18, "name", "foo")

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

	tk := time.NewTicker(time.Minute)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			return
		case d := <-ch:
			log.WithValue("time", time.Now())
			log.GetHandler().Handle(context.Background(), d)
		default:
			log.Warn("这是一个测试消息->: %d", time.Now().UnixMicro())
			time.Sleep(time.Second)
		}
	}
}
