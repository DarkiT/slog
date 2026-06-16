package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/darkit/slog/internal/common"
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
	TimeFormat = "2006/01/02 15:04.05.000" // 默认时间格式

	callerSkipPrefixesMu    sync.RWMutex
	callerSkipPrefixes      []string
	callerSkipPrefixesValue atomic.Value

	// 字符串构建器池，用于优化字符串拼接性能。
	//nolint:unused // 仅供测试/基准代码复用。
	stringBuilderPool = sync.Pool{
		New: func() any {
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
	RegisterDefaultCallerSkipPrefixes()
}

// DefaultCallerSkipPrefixes 返回默认建议跳过的调用栈前缀，供上层按需复用。
// 仅包含 slog 自己稳定暴露的 wrapper 入口，避免对具体仓库结构或源码路径产生耦合。
func DefaultCallerSkipPrefixes() []string {
	return []string{
		"github.com/darkit/slog.(*Logger)",
		"github.com/darkit/slog.Debug",
		"github.com/darkit/slog.Info",
		"github.com/darkit/slog.Warn",
		"github.com/darkit/slog.Error",
		"github.com/darkit/slog.Trace",
		"github.com/darkit/slog.Fatal",
		"github.com/darkit/slog.Debugf",
		"github.com/darkit/slog.Infof",
		"github.com/darkit/slog.Warnf",
		"github.com/darkit/slog.Errorf",
		"github.com/darkit/slog.Tracef",
		"github.com/darkit/slog.Fatalf",
		"github.com/darkit/slog.Printf",
		"github.com/darkit/slog.Println",
		"github.com/darkit/slog.DebugContext",
		"github.com/darkit/slog.InfoContext",
		"github.com/darkit/slog.WarnContext",
		"github.com/darkit/slog.ErrorContext",
		"github.com/darkit/slog.TraceContext",
		"github.com/darkit/slog.DebugfContext",
		"github.com/darkit/slog.InfofContext",
		"github.com/darkit/slog.WarnfContext",
		"github.com/darkit/slog.ErrorfContext",
		"github.com/darkit/slog.TracefContext",
	}
}

// RegisterDefaultCallerSkipPrefixes 注册 slog 默认 wrapper 前缀。
func RegisterDefaultCallerSkipPrefixes() {
	for _, prefix := range DefaultCallerSkipPrefixes() {
		RegisterCallerSkipPrefix(prefix)
	}
}

// ResetCallerSkipPrefixes 重置调用栈跳过前缀，便于测试或上层完全自定义。
func ResetCallerSkipPrefixes(prefixes ...string) {
	callerSkipPrefixesMu.Lock()
	defer callerSkipPrefixesMu.Unlock()
	callerSkipPrefixes = append([]string(nil), prefixes...)
	storeCallerSkipPrefixesLocked()
}

// RegisterCallerSkipPrefix 注册需要跳过的调用栈前缀，供外部 wrapper 透传真实业务 source。
func RegisterCallerSkipPrefix(prefix string) {
	if prefix == "" {
		return
	}
	callerSkipPrefixesMu.Lock()
	defer callerSkipPrefixesMu.Unlock()
	if slices.Contains(callerSkipPrefixes, prefix) {
		return
	}
	callerSkipPrefixes = append(callerSkipPrefixes, prefix)
	storeCallerSkipPrefixesLocked()
}

func storeCallerSkipPrefixesLocked() {
	callerSkipPrefixesValue.Store(append([]string(nil), callerSkipPrefixes...))
}

func currentCallerSkipPrefixes() []string {
	if v := callerSkipPrefixesValue.Load(); v != nil {
		return v.([]string)
	}
	return nil
}

func shouldSkipByRegisteredPrefix(function string) bool {
	for _, prefix := range currentCallerSkipPrefixes() {
		if functionMatchesSkipPrefix(function, prefix) {
			return true
		}
	}
	return false
}

func functionMatchesSkipPrefix(function string, prefix string) bool {
	if !strings.HasPrefix(function, prefix) {
		return false
	}
	if len(function) == len(prefix) {
		return true
	}
	next := function[len(prefix)]
	return next == '.' || next == '(' || next == '/'
}

func isTestingFrame(frame runtime.Frame) bool {
	return strings.HasPrefix(frame.Function, "testing.") || strings.Contains(frame.File, "/testing/")
}

func isStdSlogFrame(frame runtime.Frame) bool {
	return strings.Contains(frame.File, "/log/slog/") ||
		strings.HasPrefix(frame.Function, "log/slog.")
}

func isRuntimeFrame(frame runtime.Frame) bool {
	return strings.HasPrefix(frame.Function, "runtime.") ||
		strings.Contains(frame.File, "/runtime/")
}

func shouldSkipCallerFrame(frame runtime.Frame) bool {
	if frame.PC == 0 {
		return true
	}
	if isRuntimeFrame(frame) || isTestingFrame(frame) || isStdSlogFrame(frame) {
		return true
	}
	return shouldSkipByRegisteredPrefix(frame.Function)
}

func frameForPC(pc uintptr) (runtime.Frame, bool) {
	if pc == 0 {
		return runtime.Frame{}, false
	}
	pcs := [1]uintptr{pc}
	frames := runtime.CallersFrames(pcs[:])
	frame, _ := frames.Next()
	if frame.PC == 0 {
		return runtime.Frame{}, false
	}
	return frame, true
}

func fallbackCallerPC(pcs []uintptr) uintptr {
	for _, pc := range pcs {
		frame, ok := frameForPC(pc)
		if !ok {
			continue
		}
		if !shouldSkipCallerFrame(frame) {
			return pc
		}
	}
	if len(pcs) == 0 {
		return 0
	}
	return pcs[0]
}

// Config 日志配置结构体

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
	return &Config{
		MaxFormatCacheSize:    1000,
		StringBuilderPoolSize: 10,
		LogInternalErrors:     true,
		NoColor:               false,
		AddSource:             false,
		TimeFormat:            "2006/01/02 15:04.05.000",
	}
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

type outputRenderConfig struct {
	addSource   bool
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// Logger 结构体定义，实现日志记录功能
type Logger struct {
	w            io.Writer
	text         *slog.Logger       // 文本格式日志记录器
	json         *slog.Logger       // JSON格式日志记录器
	ctx          context.Context    // 上下文信息
	boundAttrs   []slog.Attr        // 绑定到 Logger 实例的固定属性
	noColor      bool               // 是否禁用颜色输出
	level        Level              // 日志级别
	mu           sync.Mutex         // 添加互斥锁，用于处理并发
	config       *Config            // 配置信息
	renderConfig outputRenderConfig // 渲染订阅语义化内容所需的配置快照
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

func (l *Logger) outputEnabled() (textOn, jsonOn bool) {
	textOn = isGlobalTextEnabled()
	jsonOn = isGlobalJSONEnabled()
	if l == nil || l.config == nil {
		return textOn, jsonOn
	}
	if l.config.EnableText != nil {
		textOn = *l.config.EnableText
	}
	if l.config.EnableJSON != nil {
		jsonOn = *l.config.EnableJSON
	}
	return textOn, jsonOn
}

// GetSlogLogger 方法
func (l *Logger) GetSlogLogger() *SlogLogger {
	if l == nil {
		return nil
	}
	textEnabledForInstance, jsonEnabledForInstance := l.outputEnabled()

	if jsonEnabledForInstance && !textEnabledForInstance && l.json != nil {
		return l.materializedSlogLogger(l.json)
	}
	if l.text != nil {
		return l.materializedSlogLogger(l.text)
	}
	if l.json != nil {
		return l.materializedSlogLogger(l.json)
	}
	return nil
}

// Handler 返回当前 Logger 采用的底层标准 slog.Handler。
func (l *Logger) Handler() Handler {
	if logger := l.GetSlogLogger(); logger != nil {
		return logger.Handler()
	}
	return DiscardHandler
}

// Enabled 判断当前 Logger 在给定上下文和级别下是否会输出日志。
func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler := l.Handler(); handler != nil {
		return handler.Enabled(ctx, level)
	}
	return false
}

// Diagnostics 返回模块健康与指标快照。
func (l *Logger) Diagnostics() []ModuleDiagnostics {
	return CollectModuleDiagnostics()
}

// logWithLevel 使用指定级别记录日志
// 非格式化日志的内部实现
func (l *Logger) logWithLevel(level Level, msg string, args ...any) {
	l.logRecord(level, l.ctx, msg, false, args...)
}

// logfWithLevel 使用指定级别记录格式化日志
// 格式化日志的内部实现
func (l *Logger) logfWithLevel(level Level, format string, args ...any) {
	l.logRecord(level, l.ctx, fmt.Sprintf(format, args...), true, args...)
}

// logRecord 日志记录的核心实现
// 处理所有类型的日志记录请求
func (l *Logger) logRecord(level Level, ctx context.Context, msg string, sprintf bool, args ...any) {
	if l == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if globalRateLimiter != nil && !globalRateLimiter.Allow() {
		if l.config == nil || l.config.LogInternalErrors {
			fmt.Fprintf(os.Stderr, "slog: record dropped by limiter level=%s msg=%q\n", level, msg)
		}
		return
	}

	textEnabledForInstance, jsonEnabledForInstance := l.outputEnabled()
	recordPC := uintptr(0)
	if l.needsCallerPC(textEnabledForInstance, jsonEnabledForInstance) {
		recordPC = resolveCallerPC()
	}

	var r slog.Record
	if sprintf {
		r = newRecordWithPC(level, recordPC, msg)
		appendBoundAttrs(&r, l.boundAttrs)
	} else if formatLog(msg, args...) {
		r = newRecordWithPC(level, recordPC, msg, args...)
		appendBoundAttrs(&r, l.boundAttrs)
	} else {
		r = newRecordWithPC(level, recordPC, msg)
		appendBoundAttrs(&r, l.boundAttrs)
		r.Add(args...)
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
	if subscriberCount.Load() == 0 {
		return
	}

	event := l.subscriptionEvent(ctx, r)
	var toDelete []any

	subscribers.Range(func(key, value any) bool {
		sub := value.(*subscriber)

		// 发布到订阅者：高压下按策略丢弃，不阻塞主链路。
		result := sub.trySend(event)
		if result == sendResultInactive || result == sendResultClosed {
			toDelete = append(toDelete, key)
		}

		return true
	})

	// 清理失活订阅者
	for _, key := range toDelete {
		if value, ok := subscribers.LoadAndDelete(key); ok {
			if sub, ok := value.(*subscriber); ok {
				sub.close()
				subscriberCount.Add(-1)
				subscriberEvicted.Add(1)
			}
		}
	}
}

func (l *Logger) needsCallerPC(textOn, jsonOn bool) bool {
	if l == nil || !l.renderConfig.addSource {
		return false
	}
	if subscriberCount.Load() > 0 {
		return true
	}
	return (textOn && l.text != nil) || (jsonOn && l.json != nil)
}

// With 创建一个带有额外字段的新日志记录器
func (l *Logger) With(args ...any) *Logger {
	if l == nil {
		return nil
	}
	if len(args) == 0 {
		return l
	}

	newLogger := l.clone()
	attrs := argsToAttrs(args)
	newLogger.boundAttrs = append(newLogger.boundAttrs, attrs...)

	return newLogger
}

// WithGroup 在当前日志记录器基础上创建一个新的日志组
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func (l *Logger) WithGroup(name string) *Logger {
	if l == nil {
		return nil
	}
	// 如果组名为空则返回当前logger
	if name == "" {
		return l
	}

	// 创建新的logger
	newLogger := l.clone()

	// 处理text logger
	if l.text != nil {
		newLogger.text = slog.New(l.text.Handler().WithGroup(name))
	}

	// 处理json logger
	if l.json != nil {
		newLogger.json = slog.New(l.json.Handler().WithGroup(name))
	}

	return newLogger
}

const badWithKey = "!BADKEY"

func argsToAttrs(args []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, (len(args)+1)/2)
	for len(args) > 0 {
		var attr slog.Attr
		switch x := args[0].(type) {
		case string:
			if len(args) == 1 {
				attr = slog.String(badWithKey, x)
				args = nil
			} else {
				attr = slog.Any(x, args[1])
				args = args[2:]
			}
		case slog.Attr:
			attr = x
			args = args[1:]
		default:
			attr = slog.Any(badWithKey, x)
			args = args[1:]
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

// Debug 记录Debug级别的日志。
func (l *Logger) Debug(msg string, args ...any) {
	l.logWithLevel(LevelDebug, msg, args...)
}

// DebugContext 记录 Debug 级别日志，附带上下文传播。
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelDebug, ctx, msg, false, args...)
}

// Info 记录信息级别的日志
func (l *Logger) Info(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// InfoContext 记录信息级别日志，附带上下文传播。
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelInfo, ctx, msg, false, args...)
}

// Warn 记录警告级别的日志
func (l *Logger) Warn(msg string, args ...any) {
	l.logWithLevel(LevelWarn, msg, args...)
}

// WarnContext 记录警告级别日志，附带上下文传播。
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelWarn, ctx, msg, false, args...)
}

// Error 记录错误级别的日志
func (l *Logger) Error(msg string, args ...any) {
	l.logWithLevel(LevelError, msg, args...)
}

// ErrorContext 记录错误级别日志，附带上下文传播。
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelError, ctx, msg, false, args...)
}

// Fatal 记录致命错误并终止程序
func (l *Logger) Fatal(msg string, args ...any) {
	l.logWithLevel(LevelFatal, msg, args...)
	os.Exit(1)
}

// FatalContext 记录致命日志并退出，附带上下文传播。
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelFatal, ctx, msg, false, args...)
	os.Exit(1)
}

// Trace 记录跟踪级别的日志
func (l *Logger) Trace(msg string, args ...any) {
	l.logWithLevel(LevelTrace, msg, args...)
}

// TraceContext 记录跟踪日志，附带上下文传播。
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.logRecord(LevelTrace, ctx, msg, false, args...)
}

// Log 以指定级别记录日志，兼容标准库 slog.Logger.Log。
func (l *Logger) Log(ctx context.Context, level Level, msg string, args ...any) {
	l.logRecord(level, ctx, msg, false, args...)
}

// LogAttrs 以指定级别记录 Attr 列表，兼容标准库 slog.Logger.LogAttrs。
func (l *Logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	attrArgs := make([]any, len(attrs))
	for i, attr := range attrs {
		attrArgs[i] = attr
	}
	l.logRecord(level, ctx, msg, false, attrArgs...)
}

// Debugf 记录格式化的调试级别日志
func (l *Logger) Debugf(format string, args ...any) {
	l.logfWithLevel(LevelDebug, format, args...)
}

// DebugfContext 记录格式化调试日志，附带上下文传播。
func (l *Logger) DebugfContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelDebug, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Infof 记录格式化的信息级别日志
func (l *Logger) Infof(format string, args ...any) {
	l.logfWithLevel(LevelInfo, format, args...)
}

// InfofContext 记录格式化的信息日志，附带上下文传播。
func (l *Logger) InfofContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelInfo, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Warnf 记录格式化的警告级别日志
func (l *Logger) Warnf(format string, args ...any) {
	l.logfWithLevel(LevelWarn, format, args...)
}

// WarnfContext 记录格式化的警告日志，附带上下文传播。
func (l *Logger) WarnfContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelWarn, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Errorf 记录格式化的错误级别日志
func (l *Logger) Errorf(format string, args ...any) {
	l.logfWithLevel(LevelError, format, args...)
}

// ErrorfContext 记录格式化的错误日志，附带上下文传播。
func (l *Logger) ErrorfContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelError, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Fatalf 记录格式化的致命错误并终止程序
func (l *Logger) Fatalf(format string, args ...any) {
	l.logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// FatalfContext 记录格式化致命日志并退出，附带上下文传播。
func (l *Logger) FatalfContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelFatal, ctx, fmt.Sprintf(format, args...), true, args...)
	os.Exit(1)
}

// Tracef 记录格式化的跟踪级别日志
func (l *Logger) Tracef(format string, args ...any) {
	l.logfWithLevel(LevelTrace, format, args...)
}

// TracefContext 记录格式化的 Trace 日志，附带上下文传播。
func (l *Logger) TracefContext(ctx context.Context, format string, args ...any) {
	l.logRecord(LevelTrace, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Printf 兼容标准库的格式化日志方法
func (l *Logger) Printf(format string, args ...any) {
	l.logWithLevel(LevelInfo, format, args...)
}

// Println 兼容标准库的普通日志方法
func (l *Logger) Println(msg string, args ...any) {
	l.logWithLevel(LevelInfo, msg, args...)
}

// clone 创建Logger的深度复制
func (l *Logger) clone() *Logger {
	// 创建新的Logger实例，但需要考虑writer的并发安全
	newLogger := &Logger{
		w:            l.w, // 注意：共享writer需要在使用时进行同步
		text:         l.text,
		json:         l.json,
		ctx:          l.ctx,
		boundAttrs:   slices.Clone(l.boundAttrs),
		noColor:      l.noColor,
		level:        l.level,
		mu:           sync.Mutex{}, // 每个logger实例都有独立的互斥锁
		config:       l.config,
		renderConfig: l.renderConfig,
	}

	return newLogger
}

// newRecord 创建新的日志记录
// 设置时间戳、级别、消息和调用栈信息
func newRecordWithPC(level Level, pc uintptr, format string, args ...any) slog.Record {
	t := time.Now()
	if args == nil {
		return slog.NewRecord(t, level, format, pc)
	}
	return slog.NewRecord(t, level, fmt.Sprintf(format, args...), pc)
}

func resolveCallerPC() uintptr {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:])
	if n == 0 {
		return 0
	}

	for _, pc := range pcs[:n] {
		frame, ok := frameForPC(pc)
		if !ok {
			continue
		}
		if !shouldSkipCallerFrame(frame) {
			return pc
		}
	}
	return fallbackCallerPC(pcs[:n])
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
	if !strings.Contains(msg, "%") {
		formatCache.PutString(msg, "false")
		return false
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

//nolint:unused // 仅供测试代码显式重置缓存。
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

func appendBoundAttrs(r *slog.Record, attrs []slog.Attr) {
	if r == nil || len(attrs) == 0 {
		return
	}
	r.AddAttrs(attrs...)
}

func (l *Logger) materializedSlogLogger(base *slog.Logger) *slog.Logger {
	if base == nil || len(l.boundAttrs) == 0 {
		return base
	}
	return slog.New(base.Handler().WithAttrs(l.boundAttrs))
}

// NewLoggerWithConfig 使用配置创建新的日志记录器
func NewLoggerWithConfig(w io.Writer, config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
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
		w:            w,
		noColor:      config.NoColor,
		level:        levelVar.Level(),
		ctx:          context.Background(),
		config:       config,
		renderConfig: newOutputRenderConfig(options),
		text:         slog.New(newAddonsHandler(NewConsoleHandler(w, config.NoColor, options), ext)),
		json:         slog.New(newAddonsHandler(NewJSONHandler(w, options), ext)),
	}

	return newLogger
}

func newOutputRenderConfig(options *slog.HandlerOptions) outputRenderConfig {
	if options == nil {
		return outputRenderConfig{}
	}
	return outputRenderConfig{
		addSource:   options.AddSource,
		replaceAttr: options.ReplaceAttr,
	}
}
