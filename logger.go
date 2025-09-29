package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/darkit/slog/common"
)

const (
	LevelTrace Level = -8 // 跟踪级别，最详细的日志记录
	LevelDebug Level = -4 // 调试级别，用于开发调试
	LevelInfo  Level = 0  // 信息级别，普通日志信息
	LevelWarn  Level = 4  // 警告级别，潜在的问题
	LevelError Level = 8  // 错误级别，需要注意的错误
	LevelFatal Level = 12 // 致命级别，会导致程序退出的错误
)

var (
	logger     *Logger                     // 全局日志记录器实例
	TimeFormat = "2006/01/02 15:04.05.000" // 默认时间格式

	// 字符串构建器池，用于优化字符串拼接性能
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// 日志级别对应的TXT名称映射
	levelTextNames = map[slog.Leveler]string{
		LevelInfo:  "I",
		LevelDebug: "D",
		LevelWarn:  "W",
		LevelError: "E",
		LevelTrace: "T",
		LevelFatal: "F",
	}
	// 日志级别对应的JSON名称映射
	levelJSONNames = map[Level]string{
		LevelInfo:  "Info",
		LevelDebug: "Debug",
		LevelWarn:  "Warn",
		LevelError: "Error",
		LevelTrace: "Trace",
		LevelFatal: "Fatal",
	}

	// 日志格式字符串缓存，存储常用的格式字符串检测结果
	// 键是格式字符串，值是布尔结果(是否包含格式说明符)
	formatCache        *common.LRUStringCache
	maxFormatCacheSize int64 = 1000 // 最大缓存条目数

	// 格式化动词查找表，用于O(1)时间查找
	// 使用数组而非map，利用ASCII码直接索引提高性能
	formatVerbTable = [128]bool{}

	// 标志位查找表
	formatFlagTable = [128]bool{}
)

// init 初始化格式化查找表
func init() {
	// 初始化格式动词查找表
	for _, verb := range []byte("vTdefFgGboxXsqptcUw") {
		formatVerbTable[verb] = true
	}

	// 初始化标志位查找表
	for _, flag := range []byte(" #+-.0123456789") {
		formatFlagTable[flag] = true
	}

	// 初始化LRU格式缓存
	formatCache = common.NewLRUStringCache(int(maxFormatCacheSize))
}

// Config 日志配置结构体
type Config struct {
	// 缓存配置
	MaxFormatCacheSize int64 // 最大格式缓存大小

	// 性能配置
	StringBuilderPoolSize int // 字符串构建器池大小

	// 错误处理配置
	LogInternalErrors bool // 是否记录内部错误

	// 输出配置
	EnableText *bool // 启用文本输出（nil 表示继承全局设置）
	EnableJSON *bool // 启用JSON输出（nil 表示继承全局设置）
	NoColor    bool  // 禁用颜色
	AddSource  bool  // 添加源代码位置

	// 时间配置
	TimeFormat string // 时间格式
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	c := &Config{
		MaxFormatCacheSize:    1000,
		StringBuilderPoolSize: 10,
		LogInternalErrors:     true,
		NoColor:               false,
		AddSource:             false,
		TimeFormat:            "2006/01/02 15:04.05.000",
	}
	c.SetEnableText(true)
	return c
}

func boolPtr(v bool) *bool {
	return &v
}

// SetEnableText 显式设置文本输出开关
func (c *Config) SetEnableText(enabled bool) {
	if c == nil {
		return
	}
	c.EnableText = boolPtr(enabled)
}

// SetEnableJSON 显式设置 JSON 输出开关
func (c *Config) SetEnableJSON(enabled bool) {
	if c == nil {
		return
	}
	c.EnableJSON = boolPtr(enabled)
}

// InheritTextOutput 使实例文本输出沿用全局设置
func (c *Config) InheritTextOutput() {
	if c == nil {
		return
	}
	c.EnableText = nil
}

// InheritJSONOutput 使实例 JSON 输出沿用全局设置
func (c *Config) InheritJSONOutput() {
	if c == nil {
		return
	}
	c.EnableJSON = nil
}

// Logger 结构体定义，实现日志记录功能
type Logger struct {
	w       io.Writer
	text    *slog.Logger    // 文本格式日志记录器
	json    *slog.Logger    // JSON格式日志记录器
	ctx     context.Context // 上下文信息
	noColor bool            // 是否禁用颜色输出
	level   Level           // 日志级别
	mu      sync.Mutex      // 添加互斥锁，用于处理并发
	config  *Config         // 配置信息
}

// GetLevel 获取当前日志级别
// 优先返回原子存储的级别，否则返回有效级别
func (l *Logger) GetLevel() Level {
	return levelVar.Level()
}

// SetLevel 设置日志级别
// 同时更新普通存储和原子存储
func (l *Logger) SetLevel(level any) *Logger {
	if err := SetLevel(level); err != nil {
		l.Error("SetLogLevel", "error", err.Error())
	}
	l.level = l.GetLevel()
	return l
}

// GetSlogLogger 方法
func (l *Logger) GetSlogLogger() *slog.Logger {
	textEnabledForInstance := textEnabled
	jsonEnabledForInstance := jsonEnabled
	if l.config != nil {
		if l.config.EnableText != nil {
			textEnabledForInstance = textEnabled && *l.config.EnableText
		}
		if l.config.EnableJSON != nil {
			jsonEnabledForInstance = *l.config.EnableJSON
		}
	}

	if jsonEnabledForInstance && !textEnabledForInstance {
		return l.json
	}
	return l.text
}

// logWithLevel 使用指定级别记录日志
// 非格式化日志的内部实现
func (l *Logger) logWithLevel(level Level, msg string, args ...any) {
	l.logRecord(level, msg, false, args...)
}

// logfWithLevel 使用指定级别记录格式化日志
// 格式化日志的内部实现
func (l *Logger) logfWithLevel(level Level, format string, args ...any) {
	l.logRecord(level, fmt.Sprintf(format, args...), true, args...)
}

// logRecord 日志记录的核心实现
// 处理所有类型的日志记录请求
func (l *Logger) logRecord(level Level, msg string, sprintf bool, args ...any) {
	// 使用 logger 的 context 而不是空 context
	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var r slog.Record
	if !sprintf && formatLog(msg, args...) {
		r = newRecord(level, msg, args...)
	} else if !sprintf {
		r = newRecord(level, msg)
		r.Add(args...)
	} else {
		r = newRecord(level, msg)
	}

	// 处理现有的日志输出
	// 检查是否启用文本输出（优先使用实例配置，否则使用全局配置）
	textEnabledForInstance := textEnabled
	jsonEnabledForInstance := jsonEnabled
	if l.config != nil {
		if l.config.EnableText != nil {
			textEnabledForInstance = textEnabled && *l.config.EnableText
		}
		if l.config.EnableJSON != nil {
			jsonEnabledForInstance = *l.config.EnableJSON
		}
	}

	if textEnabledForInstance && l.text != nil && l.text.Enabled(ctx, level) {
		if err := l.text.Handler().Handle(ctx, r); err != nil {
			// 记录内部错误到stderr，但不阻塞日志记录
			if l.config == nil || l.config.LogInternalErrors {
				fmt.Fprintf(os.Stderr, "slog: text handler error: %v\n", err)
			}
		}
	}
	if jsonEnabledForInstance && l.json != nil && l.json.Enabled(ctx, level) {
		if err := l.json.Handler().Handle(ctx, r); err != nil {
			// 记录内部错误到stderr，但不阻塞日志记录
			if l.config == nil || l.config.LogInternalErrors {
				fmt.Fprintf(os.Stderr, "slog: json handler error: %v\n", err)
			}
		}
	}

	// 向所有订阅者发送日志记录（使用原子状态管理）
	var toDelete []interface{}

	subscribers.Range(func(key, value interface{}) bool {
		sub := value.(*subscriber)

		// 使用原子状态管理，优雅地处理发送
		if !sub.trySend(r) {
			// 发送失败，标记为待删除
			toDelete = append(toDelete, key)
		}

		return true
	})

	// 清理无效的订阅者
	for _, key := range toDelete {
		if value, ok := subscribers.LoadAndDelete(key); ok {
			if sub, ok := value.(*subscriber); ok {
				sub.close()
			}
		}
	}
}

// With 创建一个带有额外字段的新日志记录器
func (l *Logger) With(args ...any) *Logger {
	if len(args) == 0 {
		return l
	}

	newLogger := l.clone()

	// 更新text logger
	if l.text != nil {
		newLogger.text = l.text.With(args...)
	}

	// 更新json logger
	if l.json != nil {
		newLogger.json = l.json.With(args...)
	}

	return newLogger
}

// WithGroup 在当前日志记录器基础上创建一个新的日志组
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func (l *Logger) WithGroup(name string) *Logger {
	// 如果组名为空则返回当前logger
	if name == "" {
		return l
	}

	// 创建新的logger
	newLogger := l.clone()

	// 处理text logger
	if l.text != nil {
		newLogger.text = l.text.WithGroup(name)
	}

	// 处理json logger
	if l.json != nil {
		newLogger.json = l.json.WithGroup(name)
	}

	return newLogger
}

// Debug 记录Debug级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.logWithLevel(LevelDebug, msg, args...)
}

// Info 记录信息级别的日志
func (l *Logger) Info(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// Warn 记录警告级别的日志
func (l *Logger) Warn(msg string, args ...any) {
	l.logWithLevel(LevelWarn, msg, args...)
}

// Error 记录错误级别的日志
func (l *Logger) Error(msg string, args ...any) {
	l.logWithLevel(LevelError, msg, args...)
}

// Fatal 记录致命错误并终止程序
func (l *Logger) Fatal(msg string, args ...any) {
	l.logWithLevel(LevelFatal, msg, args...)
	os.Exit(1)
}

// Trace 记录跟踪级别的日志
func (l *Logger) Trace(msg string, args ...any) {
	l.logWithLevel(LevelTrace, msg, args...)
}

// Debugf 记录格式化的调试级别日志
func (l *Logger) Debugf(format string, args ...any) {
	l.logfWithLevel(LevelDebug, format, args...)
}

// Infof 记录格式化的信息级别日志
func (l *Logger) Infof(format string, args ...any) {
	l.logfWithLevel(LevelInfo, format, args...)
}

// Warnf 记录格式化的警告级别日志
func (l *Logger) Warnf(format string, args ...any) {
	l.logfWithLevel(LevelWarn, format, args...)
}

// Errorf 记录格式化的错误级别日志
func (l *Logger) Errorf(format string, args ...any) {
	l.logfWithLevel(LevelError, format, args...)
}

// Fatalf 记录格式化的致命错误并终止程序
func (l *Logger) Fatalf(format string, args ...any) {
	l.logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Tracef 记录格式化的跟踪级别日志
func (l *Logger) Tracef(format string, args ...any) {
	l.logfWithLevel(LevelTrace, format, args...)
}

// Printf 兼容标准库的格式化日志方法
func (l *Logger) Printf(format string, args ...any) {
	l.logWithLevel(LevelInfo, format, args...)
}

// Println 兼容标准库的普通日志方法
func (l *Logger) Println(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// formatOptions 定义日志格式化选项
type formatOptions struct {
	TextEnabled bool   // 是否启用文本格式
	NoColor     bool   // 是否禁用颜色
	TimeFormat  string // 时间格式
}

// formatDynamicLogLine 格式化动态日志行
// 将日志格式化逻辑统一提取，确保与项目中其他日志格式一致
func formatDynamicLogLine(t time.Time, level Level, content string, opts formatOptions) string {
	if opts.TextEnabled {
		// 文本格式输出 - 应与项目中其他日志格式保持一致
		levelStr := formatLevelString(level, opts.NoColor)
		return fmt.Sprintf("%s %s %s",
			t.Format(opts.TimeFormat),
			levelStr,
			content)
	} else {
		// JSON格式输出 - 应与项目中其他JSON日志格式保持一致
		return fmt.Sprintf("{\"time\":\"%s\",\"level\":\"%s\",\"msg\":\"%s\"}",
			t.Format(opts.TimeFormat),
			levelJSONNames[level],
			content)
	}
}

// formatLevelString 格式化日志级别字符串
// 提取级别格式化逻辑，确保一致性
func formatLevelString(level Level, noColor bool) string {
	if !textEnabled {
		return fmt.Sprintf("[%s]", levelJSONNames[level])
	}

	// 获取对应的颜色代码
	color, ok := levelColorMap[level]
	if !ok {
		color = ansiBrightRed
	}

	// 如果没有禁用颜色，则添加颜色代码
	if !noColor {
		return fmt.Sprintf("%s[%s]%s", color, levelTextNames[level], ansiReset)
	}

	return fmt.Sprintf("[%s]", levelTextNames[level])
}

// ProgressBar 显示带有可视化进度条的日志
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBar(msg string, durationMs int, barWidth int, level ...Level) *Logger {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用默认进度条选项
	opts := progressBarOptions{
		BarStyle:       "default",
		ShowPercentage: true,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用默认的l.w
	w := l.w

	// 动态输出内容
	for i := 0; ; i++ {
		if i > steps {
			break
		}

		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}

		// 获取当前时间
		now := time.Now()

		// 格式化进度条内容
		content := formatProgressBar(msg, progress, barWidth, opts)

		// 使用统一的格式化逻辑
		logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  TimeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(w, "\r"+logLine)

		// 如果已经达到100%，退出循环
		if progress >= 100 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(w)

	return l
}

// ProgressBarTo 显示带有可视化进度条的日志，并可指定输出目标
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarTo(msg string, durationMs int, barWidth int, writer io.Writer, level ...Level) *Logger {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用默认进度条选项
	opts := progressBarOptions{
		BarStyle:       "default",
		ShowPercentage: true,
	}

	// 修改logWithDynamic实现，使用指定的writer
	l.mu.Lock()
	defer l.mu.Unlock()

	// 动态输出内容
	for i := 0; ; i++ {
		if i > steps {
			break
		}

		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}

		// 获取当前时间
		now := time.Now()

		// 格式化进度条内容
		content := formatProgressBar(msg, progress, barWidth, opts)

		// 使用统一的格式化逻辑
		logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  TimeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(writer, "\r"+logLine)

		// 如果已经达到100%，退出循环
		if progress >= 100 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(writer)

	return l
}

// ProgressBarWithValue 显示指定进度值的进度条
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithValue(msg string, progress float64, barWidth int, level ...Level) {
	// 确保进度值在0-100之间
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用默认选项
	opts := progressBarOptions{
		BarStyle:       "default",
		ShowPercentage: true,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用默认的writer
	w := l.w

	// 获取当前时间
	now := time.Now()

	// 格式化进度条内容
	content := formatProgressBar(msg, progress, barWidth, opts)

	// 使用统一的格式化逻辑
	logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
		TextEnabled: textEnabled,
		NoColor:     l.noColor,
		TimeFormat:  TimeFormat,
	})

	// 输出到writer，添加换行符
	fmt.Fprintln(w, logLine)
}

// ProgressBarWithOptions 显示可高度定制的进度条
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithOptions(msg string, durationMs int, barWidth int, opts progressBarOptions, level ...Level) *Logger {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用自定义时间格式（如果指定）
	timeFormat := TimeFormat
	if opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用默认的l.w
	w := l.w

	// 动态输出内容
	for i := 0; ; i++ {
		if i > steps {
			break
		}

		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}

		// 获取当前时间
		now := time.Now()

		// 格式化进度条内容
		content := formatProgressBar(msg, progress, barWidth, opts)

		// 使用统一的格式化逻辑，但尊重自定义时间格式
		logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  timeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(w, "\r"+logLine)

		// 如果已经达到100%，退出循环
		if progress >= 100 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(w)

	return l
}

// ProgressBarWithOptionsTo 显示可高度定制的进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithOptionsTo(msg string, durationMs int, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) *Logger {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用自定义时间格式（如果指定）
	timeFormat := TimeFormat
	if opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 动态输出内容
	for i := 0; ; i++ {
		if i > steps {
			break
		}

		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}

		// 获取当前时间
		now := time.Now()

		// 格式化进度条内容
		content := formatProgressBar(msg, progress, barWidth, opts)

		// 使用统一的格式化逻辑，但尊重自定义时间格式
		logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  timeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(writer, "\r"+logLine)

		// 如果已经达到100%，退出循环
		if progress >= 100 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(writer)

	return l
}

// ProgressBarWithValueAndOptions 显示指定进度值的定制进度条
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithValueAndOptions(msg string, progress float64, barWidth int, opts progressBarOptions, level ...Level) {
	// 确保进度值在0-100之间
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用自定义时间格式
	timeFormat := TimeFormat
	if opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用指定的writer
	w := l.w

	// 获取当前时间
	now := time.Now()

	// 格式化进度条内容
	content := formatProgressBar(msg, progress, barWidth, opts)

	// 使用统一的格式化逻辑，但尊重自定义时间格式
	logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
		TextEnabled: textEnabled,
		NoColor:     l.noColor,
		TimeFormat:  timeFormat,
	})

	// 输出到writer，添加换行符
	fmt.Fprintln(w, logLine)
}

// progressBarOptions 定义进度条的样式选项
type progressBarOptions struct {
	LeftBracket    string // 左括号，默认为 "["
	RightBracket   string // 右括号，默认为 "]"
	Fill           string // 填充字符，默认为 "="
	Head           string // 进度条头部字符，默认为 ">"
	Empty          string // 空白部分字符，默认为 " "
	ShowPercentage bool   // 是否显示百分比，默认为 true
	TimeFormat     string // 时间格式，默认为 TimeFormat
	BarStyle       string // 进度条样式，默认为 "default"
}

// DefaultProgressBarOptions 返回默认的进度条选项
func DefaultProgressBarOptions() progressBarOptions {
	return progressBarOptions{
		LeftBracket:    "[",
		RightBracket:   "]",
		Fill:           "=",
		Head:           ">",
		Empty:          " ",
		ShowPercentage: true,
		TimeFormat:     TimeFormat,
		BarStyle:       "default",
	}
}

// formatProgressBar 格式化进度条的显示内容
//
//   - msg: 要显示的消息
//   - progress: 当前进度（0-100）
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//
// 返回格式化后的进度条字符串
func formatProgressBar(msg string, progress float64, barWidth int, opts progressBarOptions) string {
	// 确保进度在0-100范围内
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	// 计算已完成的宽度和未完成的宽度
	completedWidth := int(float64(barWidth) * progress / 100)
	remainingWidth := barWidth - completedWidth

	// 选择进度条样式
	completed := "="
	remaining := " "
	indicator := ">"
	switch opts.BarStyle {
	case "block":
		completed = "█"
		remaining = "░"
		indicator = ""
	case "simple":
		completed = "#"
		remaining = "-"
		indicator = ">"
	}

	// 构建进度条
	var progressBar strings.Builder
	progressBar.WriteString("[")

	// 绘制已完成部分
	for i := 0; i < completedWidth; i++ {
		progressBar.WriteString(completed)
	}

	// 添加指示器（除非已经100%完成）
	if completedWidth < barWidth && indicator != "" {
		progressBar.WriteString(indicator)
		remainingWidth--
	}

	// 绘制未完成部分
	for i := 0; i < remainingWidth; i++ {
		progressBar.WriteString(remaining)
	}

	progressBar.WriteString("]")

	// 格式化最终输出
	var result string
	if opts.ShowPercentage {
		result = fmt.Sprintf("%s %s %.1f%%", msg, progressBar.String(), progress)
	} else {
		result = fmt.Sprintf("%s %s", msg, progressBar.String())
	}

	return result
}

// Progress 显示进度百分比
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
//   - writer: 可选的输出writer，如果为nil则使用默认的l.w
func (l *Logger) Progress(msg string, durationMs int, writer ...io.Writer) {
	steps := 100
	interval := durationMs / steps // 计算每步的时间间隔
	startTime := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	// 确定使用的writer
	w := l.w
	if len(writer) > 0 && writer[0] != nil {
		w = writer[0]
	}

	// 动态输出内容
	for i := 0; ; i++ {
		if i > steps {
			break
		}

		// 根据实际经过的时间计算进度
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(time.Duration(durationMs)*time.Millisecond) * 100
		if progress > 100 {
			progress = 100
		}

		// 获取当前时间
		now := time.Now()

		// 格式化内容
		content := fmt.Sprintf("%s %.1f%%", msg, progress)

		// 使用统一的格式化逻辑
		logLine := formatDynamicLogLine(now, l.level, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  TimeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(w, "\r"+logLine)

		// 如果已经达到100%，退出循环
		if progress >= 100 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(w)
}

// Countdown 显示倒计时
//
//   - msg: 要显示的消息内容
//   - seconds: 倒计时的秒数
//   - writer: 可选的输出writer，如果为nil则使用默认的l.w
func (l *Logger) Countdown(msg string, seconds int, writer ...io.Writer) {
	interval := 1000 // 1秒更新一次

	l.mu.Lock()
	defer l.mu.Unlock()

	// 确定使用的writer
	w := l.w
	if len(writer) > 0 && writer[0] != nil {
		w = writer[0]
	}

	// 动态输出内容
	for i := 0; i <= seconds; i++ {
		remaining := seconds - i
		if remaining < 0 {
			break
		}

		// 获取当前时间
		now := time.Now()

		// 格式化内容
		content := fmt.Sprintf("%s %ds", msg, remaining)

		// 使用统一的格式化逻辑
		logLine := formatDynamicLogLine(now, l.level, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  TimeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(w, "\r"+logLine)

		if remaining == 0 {
			break
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(w)
}

// Loading 显示加载动画
//
//   - msg: 要显示的消息内容
//   - seconds: 持续时间(秒)
//   - writer: 可选的输出writer，如果为nil则使用默认的l.w
func (l *Logger) Loading(msg string, seconds int, writer ...io.Writer) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	steps := seconds * 10 // 每秒10帧
	interval := 100       // 固定100ms间隔，即10帧/秒

	l.mu.Lock()
	defer l.mu.Unlock()

	// 确定使用的writer
	w := l.w
	if len(writer) > 0 && writer[0] != nil {
		w = writer[0]
	}

	// 动态输出内容
	for i := 0; i < steps; i++ {
		// 获取当前时间
		now := time.Now()

		// 格式化内容
		content := fmt.Sprintf("%s %s", spinner[i%len(spinner)], msg)

		// 使用统一的格式化逻辑
		logLine := formatDynamicLogLine(now, l.level, content, formatOptions{
			TextEnabled: textEnabled,
			NoColor:     l.noColor,
			TimeFormat:  TimeFormat,
		})

		// 输出到writer，保留回车符以便覆盖上一行
		fmt.Fprint(w, "\r"+logLine)

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// 输出完成后换行
	fmt.Fprintln(w)
}

// clone 创建Logger的深度复制
func (l *Logger) clone() *Logger {
	// 创建新的Logger实例，但需要考虑writer的并发安全
	newLogger := &Logger{
		w:       l.w, // 注意：共享writer需要在使用时进行同步
		text:    l.text,
		json:    l.json,
		ctx:     l.ctx,
		noColor: l.noColor,
		level:   l.level,
		mu:      sync.Mutex{}, // 每个logger实例都有独立的互斥锁
		config:  l.config,
	}

	return newLogger
}

// newRecord 创建新的日志记录
// 设置时间戳、级别、消息和调用栈信息
func newRecord(level Level, format string, args ...any) slog.Record {
	t := time.Now()
	var pcs [1]uintptr
	// 跳过runtime.Callers和当前函数调用
	runtime.Callers(5, pcs[:])

	if args == nil {
		return slog.NewRecord(t, level, format, pcs[0])
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pcs[0])
}

// formatLog 检查格式字符串并决定是否使用格式化输出
// 使用缓存优化重复字符串的检测性能
func formatLog(msg string, args ...any) bool {
	// 如果没有参数，直接返回false
	if len(args) == 0 {
		return false
	}

	// 首先尝试从缓存中获取结果
	if val, ok := formatCache.GetString(msg); ok {
		return val == "true"
	}

	// 以下是完整的格式扫描逻辑
	// 因为缓存会存储结果，所以即使这部分代码复杂，也只会对每个唯一的字符串执行一次
	result := scanFormatSpecifiers(msg)

	// 存储结果到缓存
	resultStr := "false"
	if result {
		resultStr = "true"
	}
	formatCache.PutString(msg, resultStr)

	return result
}

// cleanFormatCache 清理格式缓存
func cleanFormatCache() {
	// 清空缓存
	formatCache.Clear()
}

// scanFormatSpecifiers 扫描并检查格式说明符
// 使用手动解析而非正则表达式，以提高性能
func scanFormatSpecifiers(msg string) bool {
	msgBytes := []byte(msg) // 避免在循环中重复字符索引操作
	msgLen := len(msgBytes)

	// 手动解析格式说明符
	for i := 0; i < msgLen; {
		// 查找下一个%字符
		if msgBytes[i] != '%' {
			i++
			continue
		}

		// 处理%%转义情况
		if i+1 < msgLen && msgBytes[i+1] == '%' {
			i += 2
			continue
		}

		// 找到非转义的%，开始解析格式说明符
		pos := i + 1
		if pos >= msgLen {
			// %在字符串末尾，不是有效的格式说明符
			return false
		}

		// 使用查找表快速检查标志位、宽度、精度等
		// 这比多个if条件检查更快
		for pos < msgLen && formatFlagTable[msgBytes[pos]&127] {
			pos++
		}

		// 检查是否到达字符串末尾
		if pos >= msgLen {
			return false
		}

		// 使用查找表检查格式动词(O(1)时间)
		// 仅当ASCII范围内才使用查找表
		if msgBytes[pos] < 128 && formatVerbTable[msgBytes[pos]] {
			return true
		}

		// 移动到下一个位置
		i = pos + 1
	}

	return false
}

// Dynamic 动态输出带点号动画效果
//
//   - msg: 要显示的消息内容
//   - frames: 动画更新的总帧数
//   - interval: 每次更新的时间间隔(毫秒)
//   - writer: 可选的输出writer，如果为nil则使用默认的l.w
func (l *Logger) Dynamic(msg string, frames int, interval int, writer ...io.Writer) {
	// 使用高性能的优化版本
	l.DynamicOptimized(msg, frames, interval, writer...)
}

// ProgressBarWithValueTo 显示指定进度值的进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithValueTo(msg string, progress float64, barWidth int, writer io.Writer, level ...Level) {
	// 确保进度值在0-100之间
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用默认进度条选项
	opts := progressBarOptions{
		BarStyle:       "default",
		ShowPercentage: true,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取当前时间
	now := time.Now()

	// 格式化进度条内容
	content := formatProgressBar(msg, progress, barWidth, opts)

	// 使用统一的格式化逻辑
	logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
		TextEnabled: textEnabled,
		NoColor:     l.noColor,
		TimeFormat:  TimeFormat,
	})

	// 输出到writer，添加换行符
	fmt.Fprintln(writer, logLine)
}

// ProgressBarWithValueAndOptionsTo 显示指定进度值的定制进度条并输出到指定writer
//
//   - msg: 要显示的消息内容
//   - progress: 进度值(0-100之间)
//   - barWidth: 进度条的总宽度（字符数）
//   - opts: 进度条选项，控制显示样式
//   - writer: 指定的输出writer
//   - level: 可选的日志级别，默认使用logger的默认级别
func (l *Logger) ProgressBarWithValueAndOptionsTo(msg string, progress float64, barWidth int, opts progressBarOptions, writer io.Writer, level ...Level) {
	// 确保进度值在0-100之间
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	// 确定使用的日志级别
	logLevel := l.level
	if len(level) > 0 {
		logLevel = level[0]
	}

	// 使用自定义时间格式
	timeFormat := TimeFormat
	if opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取当前时间
	now := time.Now()

	// 格式化进度条内容
	content := formatProgressBar(msg, progress, barWidth, opts)

	// 使用统一的格式化逻辑，但尊重自定义时间格式
	logLine := formatDynamicLogLine(now, logLevel, content, formatOptions{
		TextEnabled: textEnabled,
		NoColor:     l.noColor,
		TimeFormat:  timeFormat,
	})

	// 输出到writer，添加换行符
	fmt.Fprintln(writer, logLine)
}

// NewLoggerWithConfig 使用配置创建新的日志记录器
func NewLoggerWithConfig(w io.Writer, config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	// 不修改全局配置，仅将配置应用到本实例
	// 全局配置通过其他函数管理
	timeFormat := config.TimeFormat
	if timeFormat == "" {
		timeFormat = TimeFormat
	}

	options := NewOptions(nil)
	options.AddSource = config.AddSource || levelVar.Level() < LevelDebug

	// 如果需要DLP,则初始化
	if dlpEnabled.Load() {
		ext.enableDLP()
	}

	if w == nil {
		w = NewWriter()
	}

	newLogger := &Logger{
		w:       w,
		noColor: config.NoColor,
		level:   levelVar.Level(),
		ctx:     context.Background(),
		config:  config,
		text:    slog.New(newAddonsHandler(NewConsoleHandler(w, config.NoColor, options), ext)),
		json:    slog.New(newAddonsHandler(NewJSONHandler(w, options), ext)),
	}

	return newLogger
}
