package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/darkit/slog"
	"github.com/darkit/slog/dlp/dlpheader"
	"github.com/darkit/slog/example/core"
	"github.com/darkit/slog/formatter"
	"github.com/darkit/slog/multi"
)

var ctx = context.Background()

func main() {
	// ch := slog.GetChanRecord(100)
	// defer close(ch)

	formatter1 := formatter.FormatByKey("server", func(v slog.Value) slog.Value {
		return slog.StringValue(slog.DlpMask(v.String(), dlpheader.IP))
	})
	formatter2 := formatter.FormatByKey("mobile", func(v slog.Value) slog.Value {
		return slog.StringValue(slog.DlpMask(v.String(), dlpheader.CHINAPHONE))
	})
	formatter3 := formatter.FormatByKey("url", func(v slog.Value) slog.Value {
		return slog.StringValue(slog.DlpMask(v.String(), dlpheader.URL))
	})
	slog.EnableFormatters(formatter1, formatter2, formatter3)
	slog.SetLevelTrace()
	slog.NewLogger(os.Stdout, false, false)
	// slog.DisableTextLogger()
	// slog.EnableJsonLogger()

	slog.WithContext(ctx)

	core.Init()
	// go iChan(ch)

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
	slog.With("url", "https://user:123456@www.zishuo.net:8081/live/demo").Infof("这是一个脱敏测试日志")
	slog.With("url", "rtsp://user:123456@192.168.1.123:554/live/demo").Infof("这是一个脱敏测试日志")
	slog.WithValue("mobile", "13800139000").Info("我的Email: abcd@abcd.com 手机号 13800138000 家庭住址：我家住在北京市海淀区北三环西路43号")
	slog.WithGroup("slog").Debug("这是一个调试日志", slog.Group("data",
		slog.String("mobile", "13800138000"),
		slog.Int("width", 4000),
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
	l1.With("url", "https://user:123456@www.zishuo.net/live/demo").Infof("这是一个脱敏测试日志")
	l1.With("url", "rtsp://user:123456@192.168.1.123/live/demo").Infof("这是一个脱敏测试日志")
	l1.Infof("这是一个脱敏测试日志 %s: %s", "用户", "admin")
	l1.Infof("这是一个脱敏测试日志 %s: %s", "密码", "12345678")
	l1.Info("我的Email: abcd@abcd.com 手机号 13800138000", slog.String("mobile", "13800138000"))
	l1.Debug("这是一个调试日志", slog.Group("data",
		slog.String("mobile", "13800138000"),
		slog.Int("width", 4000),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("duration", time.Duration(333)),
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
	l2.Infof("这是一个脱敏测试日志: %s -> %s", "RTSP", "rtsp://192.168.1.123/live/demo")
	l2.Infof("这是一个脱敏测试日志: %s -> %s", "RTSP", "rtsp://admin:password@192.168.1.123/live/demo")
	l2.Infof("这是一个脱敏测试日志: %s -> %s", "HTTP", "https://admin:123456@www.baidu.com/live/demo")
	l2.With("url", "https://user:123456@www.zishuo.net/live/demo").Infof("这是一个脱敏测试日志")
	l2.With("url", "rtsp://user:123456@192.168.1.123/live/demo").Infof("这是一个脱敏测试日志")
	l2.WithValue("mobile", "13800139000").Info("我的Email: abcd@abcd.com 手机号 13800138000")
	l2.Debug("这是一个调试日志", slog.Group("data",
		slog.String("mobile", "13800138000"),
		slog.Int("width", 4000),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("duration", time.Duration(333)),
	))

	fmt.Println("================L3分隔符==================")
	l3 := core.Logger.WithValue("os", runtime.GOARCH).WithValue("l3", runtime.NumGoroutine())
	l3.Info("当前日志等级", "Level", core.Logger.GetLevel().String())
	l3.Warn("这是一个警告日志", "aaaa", "bbbb")
	l3.Error("这是一个错误日志", "aaaa", "bbbb")
	l3.Info("这是一个信息日志: %s -> %d", "sss", 88888)
	l3.Debug("这是一个调试日志: %s -> %d", "sss", 88888)
	l3.Warnf("这是一个警告日志: %s -> %d", "sss", 88888)
	l3.Printf("这是一个信息日志: %s -> %d", "sss", 88888)
	l3.Trace("这是一个路由日志: %s -> %d", "sss", 88888)
	l3.Infof("这是一个脱敏测试日志: %s -> %s", "RTSP", "rtsp://admin:password@192.168.1.123/live/demo")
	l3.Infof("这是一个脱敏测试日志: %s -> %s", "HTTP", "https://admin:123456@www.baidu.com/live/demo")
	l3.Infof("Pid: %d 服务已经初始化完成, %d 个协程被创建.", os.Getpid(), runtime.NumGoroutine())
	l3.WithValue("mobile", "13800138000").Info("我的Email: abcd@abcd.com 手机号 13800138000")
	l3.WithGroup("Demo").Debug("这是一个调试日志", slog.Group("data",
		slog.String("mobile", "13800138000"),
		slog.Int("width", 4000),
		slog.Bool("status", true),
		slog.Time("time", time.Now()),
		slog.Duration("duration", time.Duration(333)),
	))
}

func iChan(ch chan slog.Record) {
	conn, _ := multi.Dial("tcp", "192.168.1.123:1900")
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
