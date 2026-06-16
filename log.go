package slog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gelfmod "github.com/darkit/slog/modules/output/gelf"
	logfmtmod "github.com/darkit/slog/modules/output/logfmt"
)

var (
	// 创建扩展配置
	ext = &extensions{
		prefixKeys: []string{"$module"},
	}
	dlpEnabled         atomic.Bool
	subscribers        sync.Map // 存储所有订阅者 - 实际类型为 map[int64]*subscriber
	subscriberCount    atomic.Int64
	subscriberSeq      atomic.Int64
	subscriberEvicted  atomic.Uint64
	textEnabled        atomic.Bool
	jsonEnabled        atomic.Bool
	levelVar           = slog.LevelVar{}
	attrFormatterOrder atomic.Value
	globalRateLimiter  = newRateLimiter(0, 0)
)

// subscriberState 订阅者状态
type subscriberState int32

const (
	stateActive subscriberState = iota
	stateClosing
	stateClosed
)

// SubscriptionBackpressurePolicy 定义订阅通道在高压场景下的背压策略。
type SubscriptionBackpressurePolicy string

const (
	// SubscriptionDropOldest 丢弃最旧消息，优先保留最新数据（默认）。
	SubscriptionDropOldest SubscriptionBackpressurePolicy = "drop_oldest"
	// SubscriptionDropNewest 丢弃最新消息，优先保留已入队数据。
	SubscriptionDropNewest SubscriptionBackpressurePolicy = "drop_newest"
	// SubscriptionBlockWithTimeout 在超时时间内阻塞等待可写，超时后丢弃。
	SubscriptionBlockWithTimeout SubscriptionBackpressurePolicy = "block_with_timeout"
)

const defaultSubscriberBlockTimeout = 5 * time.Millisecond

// SubscribeOptions 订阅选项。
type SubscribeOptions struct {
	// BufferSize 订阅缓冲区大小。
	BufferSize uint16
	// Backpressure 背压策略；空值时默认 drop_oldest。
	Backpressure SubscriptionBackpressurePolicy
	// BlockTimeout 仅在 block_with_timeout 模式下生效。
	BlockTimeout time.Duration
}

// SubscriptionEvent 描述一次统一发布后的订阅事件。
// 其中 Record 保留结构化视图，Rendered 则严格跟随当前激活的主输出格式。
type SubscriptionEvent struct {
	// Record 是已应用前缀、formatter、DLP 与 context 字段后的结构化日志。
	Record Record
	// Rendered 是与当前激活主输出一致的最终语义化内容；若未启用任何输出则为空。
	Rendered string
	// Format 表示 Rendered 采用的格式，取值为 text、json 或空字符串。
	Format string
}

func (o SubscribeOptions) normalized() SubscribeOptions {
	if o.Backpressure == "" {
		o.Backpressure = SubscriptionDropOldest
	}
	if o.Backpressure != SubscriptionBlockWithTimeout {
		o.BlockTimeout = 0
		return o
	}
	if o.BlockTimeout <= 0 {
		o.BlockTimeout = defaultSubscriberBlockTimeout
	}
	return o
}

type sendResult int

const (
	sendResultDelivered sendResult = iota
	sendResultDropped
	sendResultInactive
	sendResultClosed
)

type subscriberMetrics struct {
	published     atomic.Uint64
	delivered     atomic.Uint64
	dropped       atomic.Uint64
	droppedOldest atomic.Uint64
	droppedNewest atomic.Uint64
	droppedTimed  atomic.Uint64
	highWatermark atomic.Uint64
}

// SubscriberStats 描述单个订阅者运行状态与背压统计。
type SubscriberStats struct {
	ID            int64                          `json:"id"`
	State         string                         `json:"state"`
	BufferSize    int                            `json:"buffer_size"`
	QueueLen      int                            `json:"queue_len"`
	Backpressure  SubscriptionBackpressurePolicy `json:"backpressure"`
	BlockTimeout  time.Duration                  `json:"block_timeout"`
	CreatedAt     time.Time                      `json:"created_at"`
	Published     uint64                         `json:"published"`
	Delivered     uint64                         `json:"delivered"`
	Dropped       uint64                         `json:"dropped"`
	DroppedOldest uint64                         `json:"dropped_oldest"`
	DroppedNewest uint64                         `json:"dropped_newest"`
	DroppedTimed  uint64                         `json:"dropped_timed_out"`
	HighWatermark uint64                         `json:"high_watermark"`
}

// SubscriptionStats 汇总所有订阅者统计。
type SubscriptionStats struct {
	Subscribers        int    `json:"subscribers"`
	ActiveSubscribers  int    `json:"active_subscribers"`
	ClosingSubscribers int    `json:"closing_subscribers"`
	ClosedSubscribers  int    `json:"closed_subscribers"`
	Published          uint64 `json:"published"`
	Delivered          uint64 `json:"delivered"`
	Dropped            uint64 `json:"dropped"`
	DroppedOldest      uint64 `json:"dropped_oldest"`
	DroppedNewest      uint64 `json:"dropped_newest"`
	DroppedTimed       uint64 `json:"dropped_timed_out"`
	Evicted            uint64 `json:"evicted"`
}

// subscriber 订阅者结构（升级为原子状态管理）
type subscriber struct {
	id     int64
	ch     chan SubscriptionEvent
	cancel context.CancelFunc
	done   <-chan struct{}
	opts   SubscribeOptions
	mu     sync.RWMutex // 保护订阅通道关闭与并发投递，避免 send/close 竞争。
	state  atomic.Int32 // 原子状态管理
	once   sync.Once    // 确保只关闭一次
	stats  subscriberMetrics
	bornAt time.Time
}

// isActive 检查订阅者是否活跃
func (s *subscriber) isActive() bool {
	return subscriberState(s.state.Load()) == stateActive
}

// close 安全地关闭订阅者
func (s *subscriber) close() {
	s.once.Do(func() {
		s.state.Store(int32(stateClosing))
		s.cancel()
		s.mu.Lock()
		defer s.mu.Unlock()
		close(s.ch)
		s.state.Store(int32(stateClosed))
	})
}

// trySend 尝试发送订阅事件，如果失败则返回false
func (s *subscriber) trySend(event SubscriptionEvent) (result sendResult) {
	if !s.isActive() {
		return sendResultInactive
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.isActive() {
		return sendResultInactive
	}
	s.stats.published.Add(1)

	defer func() {
		if recover() != nil {
			result = sendResultClosed
		}
	}()
	s.updateHighWatermark()

	switch s.opts.Backpressure {
	case SubscriptionDropNewest:
		select {
		case s.ch <- event:
			s.stats.delivered.Add(1)
			s.updateHighWatermark()
			return sendResultDelivered
		default:
			s.stats.dropped.Add(1)
			s.stats.droppedNewest.Add(1)
			return sendResultDropped
		}
	case SubscriptionBlockWithTimeout:
		timeout := s.opts.BlockTimeout
		if timeout <= 0 {
			timeout = defaultSubscriberBlockTimeout
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case s.ch <- event:
			s.stats.delivered.Add(1)
			s.updateHighWatermark()
			return sendResultDelivered
		case <-timer.C:
			s.stats.dropped.Add(1)
			s.stats.droppedTimed.Add(1)
			return sendResultDropped
		case <-s.done:
			return sendResultClosed
		}
	default:
		// drop_oldest（默认）：优先保留最新数据。
		select {
		case s.ch <- event:
			s.stats.delivered.Add(1)
			s.updateHighWatermark()
			return sendResultDelivered
		default:
			// 队列满，丢弃最旧消息再重试。
			select {
			case <-s.ch:
				s.stats.dropped.Add(1)
				s.stats.droppedOldest.Add(1)
			default:
			}
			select {
			case s.ch <- event:
				s.stats.delivered.Add(1)
				s.updateHighWatermark()
				return sendResultDelivered
			default:
				// 极端竞争场景，保守降级为丢弃最新。
				s.stats.dropped.Add(1)
				s.stats.droppedNewest.Add(1)
				return sendResultDropped
			}
		}
	}
}

func (s *subscriber) updateHighWatermark() {
	current := uint64(len(s.ch))
	for {
		prev := s.stats.highWatermark.Load()
		if current <= prev || s.stats.highWatermark.CompareAndSwap(prev, current) {
			return
		}
	}
}

func (s *subscriber) snapshot() SubscriberStats {
	return SubscriberStats{
		ID:            s.id,
		State:         subscriberStateString(subscriberState(s.state.Load())),
		BufferSize:    cap(s.ch),
		QueueLen:      len(s.ch),
		Backpressure:  s.opts.Backpressure,
		BlockTimeout:  s.opts.BlockTimeout,
		CreatedAt:     s.bornAt,
		Published:     s.stats.published.Load(),
		Delivered:     s.stats.delivered.Load(),
		Dropped:       s.stats.dropped.Load(),
		DroppedOldest: s.stats.droppedOldest.Load(),
		DroppedNewest: s.stats.droppedNewest.Load(),
		DroppedTimed:  s.stats.droppedTimed.Load(),
		HighWatermark: s.stats.highWatermark.Load(),
	}
}

func subscriberStateString(state subscriberState) string {
	switch state {
	case stateActive:
		return "active"
	case stateClosing:
		return "closing"
	case stateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

func init() {
	levelVar.Set(slog.LevelInfo)
	setGlobalTextEnabled(true)
	setGlobalJSONEnabled(false)
	// 使用LoggerManager而不是直接创建全局logger
	// 这确保了更好的状态管理和实例隔离
	config := &GlobalConfig{
		DefaultWriter:  os.Stdout,
		DefaultLevel:   LevelInfo,
		DefaultNoColor: false,
		DefaultSource:  false,
		EnableText:     true,
		EnableJSON:     false,
	}
	if err := globalManager.Configure(config); err != nil {
		panic(fmt.Sprintf("slog: configure default logger: %v", err))
	}
	attrFormatterOrder.Store(defaultAttrFormatterOrder)
}

func isGlobalTextEnabled() bool {
	return textEnabled.Load()
}

func isGlobalJSONEnabled() bool {
	return jsonEnabled.Load()
}

func setGlobalTextEnabled(enabled bool) {
	textEnabled.Store(enabled)
}

func setGlobalJSONEnabled(enabled bool) {
	jsonEnabled.Store(enabled)
}

// ConfigureRecordLimiter 设置全局日志速率限制（ratePerSecond<=0 关闭限制）。
func ConfigureRecordLimiter(ratePerSecond, burst int) {
	if globalRateLimiter == nil {
		globalRateLimiter = newRateLimiter(ratePerSecond, burst)
		return
	}
	globalRateLimiter.configure(ratePerSecond, burst, ratePerSecond > 0)
}

func New(handler Handler) *SlogLogger {
	return slog.New(handler)
}

// NewLogger 创建一个包含文本和JSON格式的日志记录器
// 现在使用LoggerManager来管理实例，确保更好的状态隔离
func NewLogger(w io.Writer, noColor, addSource bool) *Logger {
	// 如果使用默认参数，直接返回管理器的默认实例
	if w == os.Stdout && !noColor && !addSource {
		return globalManager.GetDefault()
	}

	// 为自定义参数创建新实例
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug

	// 如果需要DLP,则初始化
	if dlpEnabled.Load() {
		ext.enableDLP()
	}

	if w == nil {
		w = NewWriter()
	}

	newLogger := &Logger{
		w:            w,
		noColor:      noColor,
		level:        levelVar.Level(),
		ctx:          context.Background(),
		config:       DefaultConfig(), // 使用实例级别的配置
		renderConfig: newOutputRenderConfig(options),
		text:         slog.New(newAddonsHandler(NewConsoleHandler(w, noColor, options), ext)),
		json:         slog.New(newAddonsHandler(NewJSONHandler(w, options), ext)),
	}

	return newLogger
}

// NewLogfmtLogger 使用 logfmt handler 创建 Logger，便于直接接入 Loki/Vector。
func NewLogfmtLogger(w io.Writer, opts *HandlerOptions) *Logger {
	if w == nil {
		w = NewWriter()
	}
	if opts == nil {
		opts = NewOptions(nil)
	}
	logger := globalManager.GetDefault().clone()
	handler := logfmtmod.New(logfmtmod.Option{
		Writer:      w,
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		TimeFormat:  TimeFormat,
		ReplaceAttr: opts.ReplaceAttr,
	})
	logger.text = slog.New(newAddonsHandler(handler, ext))
	logger.json = nil
	logger.w = w
	logger.renderConfig = newOutputRenderConfig(opts)
	return logger
}

// NewGELFLogger 使用 GELF handler 创建 Logger，面向 Graylog/Logstash。
func NewGELFLogger(w io.Writer, opts *HandlerOptions, gopts *gelfmod.Options) *Logger {
	if w == nil {
		w = NewWriter()
	}
	if opts == nil {
		opts = NewOptions(nil)
	}
	logger := globalManager.GetDefault().clone()
	opt := gelfmod.Options{
		Writer:      nil,
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: opts.ReplaceAttr,
	}
	if gopts != nil {
		opt = *gopts
	}
	if opt.Writer == nil {
		opt.Writer = w
	}
	handler := gelfmod.New(opt)
	logger.json = slog.New(newAddonsHandler(handler, ext))
	logger.text = nil
	logger.w = w
	logger.renderConfig = newOutputRenderConfig(opts)
	return logger
}

/*
// NewLoggerWithText 创建一个文本格式的日志记录器
func NewLoggerWithText(writer io.Writer, noColor, addSource bool) Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug
	logger = Logger{
		noColor: noColor,
		ctx:     context.Background(),
		text:    slog.New(newAddonsHandler(NewConsoleHandler(writer, noColor, options), slogPfx)),
	}

	return logger
}

// NewLoggerWithJSON 创建一个JSON格式的日志记录器
func NewLoggerWithJSON(writer io.Writer, addSource bool) Logger {
	options := NewOptions(nil)
	options.AddSource = addSource || levelVar.Level() < LevelDebug
	logger = Logger{
		ctx:  context.Background(),
		json: slog.New(newAddonsHandler(slog.NewJSONHandler(writer, options), slogPfx)),
	}
	return logger
}
*/

type AttrFormatterRule int

const (
	AttrFormatterRuleSource AttrFormatterRule = iota
	AttrFormatterRuleLevel
	AttrFormatterRuleTime
)

var defaultAttrFormatterOrder = []AttrFormatterRule{
	AttrFormatterRuleSource,
	AttrFormatterRuleLevel,
	AttrFormatterRuleTime,
}

// SetAttrFormatterOrder 允许自定义内置属性格式化规则顺序，传入空列表时会恢复默认顺序。
func SetAttrFormatterOrder(order ...AttrFormatterRule) {
	if len(order) == 0 {
		order = defaultAttrFormatterOrder
	}
	copyOrder := append([]AttrFormatterRule(nil), order...)
	attrFormatterOrder.Store(copyOrder)
}

// NewOptions 创建新的处理程序选项。
func NewOptions(options *HandlerOptions) *HandlerOptions {
	var opts slog.HandlerOptions
	if options != nil {
		opts = *options
	}

	if opts.Level == nil {
		opts.Level = &levelVar
	}

	if levelVar.Level() < LevelDebug {
		opts.AddSource = true
	}

	normalizer := newAttrFormatter(TimeFormat)
	opts.ReplaceAttr = chainReplaceAttr(opts.ReplaceAttr, normalizer.replace)

	return &opts
}

// attrFormatter 负责根据项目约定格式化特殊字段，支持递归 group 处理。
type attrFormatter struct {
	timeFormat string
	order      []AttrFormatterRule
}

func newAttrFormatter(timeFormat string) attrFormatter {
	return attrFormatter{
		timeFormat: timeFormat,
		order:      getAttrFormatterOrder(),
	}
}

func getAttrFormatterOrder() []AttrFormatterRule {
	if v := attrFormatterOrder.Load(); v != nil {
		stored := v.([]AttrFormatterRule)
		return append([]AttrFormatterRule(nil), stored...)
	}
	return append([]AttrFormatterRule(nil), defaultAttrFormatterOrder...)
}

func (f attrFormatter) replace(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case TimeKey, LevelKey, SourceKey:
		return f.applyBuiltinRule(a)
	default:
		kind := a.Value.Kind()
		if kind != slog.KindGroup && kind != slog.KindLogValuer {
			return a
		}
		return f.walk(groups, a)
	}
}

func (f attrFormatter) walk(groups []string, a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return a
		}
		nextGroups := groups
		if a.Key != "" {
			nextGroups = append(nextGroups, a.Key)
		}
		newAttrs := make([]slog.Attr, len(attrs))
		for i, child := range attrs {
			newAttrs[i] = f.walk(nextGroups, child)
		}
		a.Value = slog.GroupValue(newAttrs...)
		return a
	}
	return f.applyBuiltinRule(a)
}

func (f attrFormatter) applyBuiltinRule(a slog.Attr) slog.Attr {
	for _, rule := range f.order {
		switch rule {
		case AttrFormatterRuleSource:
			if a.Key == slog.SourceKey {
				return f.normalizeSource(a)
			}
		case AttrFormatterRuleLevel:
			if a.Key == LevelKey {
				return f.normalizeLevel(a)
			}
		case AttrFormatterRuleTime:
			if a.Key == TimeKey {
				return f.normalizeTime(a)
			}
		}
	}
	return a
}

func (f attrFormatter) normalizeSource(a slog.Attr) slog.Attr {
	if a.Key != slog.SourceKey {
		return a
	}
	if src := sourceFromValue(a.Value); src != nil {
		copy := *src
		copy.File = filepath.Base(copy.File)
		a.Value = slog.AnyValue(&copy)
	}
	return a
}

func (f attrFormatter) normalizeLevel(a slog.Attr) slog.Attr {
	if a.Key != LevelKey {
		return a
	}
	if level, ok := levelFromValue(a.Value); ok {
		if name, exists := levelJSONName(level); exists {
			a.Value = slog.StringValue(name)
		}
	}
	return a
}

func (f attrFormatter) normalizeTime(a slog.Attr) slog.Attr {
	if a.Key != TimeKey {
		return a
	}
	if formatted, ok := f.formatTime(a.Value); ok {
		a.Value = slog.StringValue(formatted)
	}
	return a
}

func (f attrFormatter) formatTime(val slog.Value) (string, bool) {
	switch val.Kind() {
	case slog.KindTime:
		return val.Time().Format(f.timeFormat), true
	case slog.KindAny:
		if t, ok := val.Any().(time.Time); ok {
			return t.Format(f.timeFormat), true
		}
	}
	return "", false
}

func sourceFromValue(val slog.Value) *slog.Source {
	if val.Kind() != slog.KindAny {
		return nil
	}
	src, ok := val.Any().(*slog.Source)
	if !ok || src == nil {
		return nil
	}
	return src
}

func levelFromValue(val slog.Value) (Level, bool) {
	if val.Kind() != slog.KindAny {
		return 0, false
	}
	level, ok := val.Any().(Level)
	return level, ok
}

func levelJSONName(level Level) (string, bool) {
	switch level {
	case LevelInfo:
		return "Info", true
	case LevelDebug:
		return "Debug", true
	case LevelWarn:
		return "Warn", true
	case LevelError:
		return "Error", true
	case LevelTrace:
		return "Trace", true
	case LevelFatal:
		return "Fatal", true
	default:
		return "", false
	}
}

func chainReplaceAttr(first, second func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	switch {
	case first == nil:
		return second
	case second == nil:
		return first
	default:
		return func(groups []string, a slog.Attr) slog.Attr {
			a = first(groups, a)
			if a.Equal(slog.Attr{}) {
				return a
			}
			return second(groups, a)
		}
	}
}

// Default 返回一个新的带前缀的日志记录器
func Default(modules ...string) *Logger {
	if len(modules) == 0 {
		// 使用LoggerManager获取默认logger而不是全局变量
		return globalManager.GetDefault()
	}

	// 构建模块标识符
	module := strings.Join(modules, ".")

	// 创建新的带模块前缀的logger
	newLogger := globalManager.GetDefault().clone()

	// 创建新的上下文
	newLogger.ctx = context.Background() // 确保每个模块有独立的上下文
	if newLogger.text != nil {
		newHandler := newAddonsHandler(newLogger.text.Handler(), ext)
		newHandler.prefixes[0] = slog.StringValue(module)
		newLogger.text = slog.New(newHandler)
	}
	if newLogger.json != nil {
		jsonHandler := newAddonsHandler(newLogger.json.Handler(), ext)
		jsonHandler.prefixes[0] = slog.StringValue(module)
		newLogger.json = slog.New(jsonHandler)
	}

	return newLogger
}

// GetSlogLogger 返回原始log/slog的日志记录器
func GetSlogLogger() *SlogLogger {
	return globalManager.GetDefault().GetSlogLogger()
}

// ModuleDiagnostics 快速获取当前已注册模块的健康状态

// GetLevel 获取全局日志级别。
func GetLevel() Level { return levelVar.Level() }

// Debug 记录全局Debug级别的日志。
func Debug(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelDebug, msg, args...)
}

// Info 记录全局Info级别的日志。
func Info(msg string, args ...any) { globalManager.GetDefault().logWithLevel(LevelInfo, msg, args...) }

// Warn 记录全局Warn级别的日志。
func Warn(msg string, args ...any) { globalManager.GetDefault().logWithLevel(LevelWarn, msg, args...) }

// Error 记录全局Error级别的日志。
func Error(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelError, msg, args...)
}

// Trace 记录全局Trace级别的日志。
func Trace(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelTrace, msg, args...)
}

// Fatal 记录全局Fatal级别的日志，并退出程序。
func Fatal(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelFatal, msg, args...)
	os.Exit(1)
}

// Debugf 记录格式化的全局Debug级别的日志。
func Debugf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelDebug, format, args...)
}

// Infof 记录格式化的全局Info级别的日志。
func Infof(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelInfo, format, args...)
}

// Warnf 记录格式化的全局Warn级别的日志。
func Warnf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelWarn, format, args...)
}

// Errorf 记录格式化的全局Error级别的日志。
func Errorf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelError, format, args...)
}

// Tracef 记录格式化的全局Trace级别的日志。
func Tracef(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelTrace, format, args...)
}

// InfoContext 记录全局 Info 日志并传播上下文。
func InfoContext(ctx context.Context, msg string, args ...any) {
	globalManager.GetDefault().logRecord(LevelInfo, ctx, msg, false, args...)
}

// ErrorContext 记录全局 Error 日志并传播上下文。
func ErrorContext(ctx context.Context, msg string, args ...any) {
	globalManager.GetDefault().logRecord(LevelError, ctx, msg, false, args...)
}

// WarnContext 记录全局 Warn 日志并传播上下文。
func WarnContext(ctx context.Context, msg string, args ...any) {
	globalManager.GetDefault().logRecord(LevelWarn, ctx, msg, false, args...)
}

// DebugContext 记录全局 Debug 日志并传播上下文。
func DebugContext(ctx context.Context, msg string, args ...any) {
	globalManager.GetDefault().logRecord(LevelDebug, ctx, msg, false, args...)
}

// TraceContext 记录全局 Trace 日志并传播上下文。
func TraceContext(ctx context.Context, msg string, args ...any) {
	globalManager.GetDefault().logRecord(LevelTrace, ctx, msg, false, args...)
}

// InfofContext 记录格式化 Info 日志并传播上下文。
func InfofContext(ctx context.Context, format string, args ...any) {
	globalManager.GetDefault().logRecord(LevelInfo, ctx, fmt.Sprintf(format, args...), true, args...)
}

// ErrorfContext 记录格式化 Error 日志并传播上下文。
func ErrorfContext(ctx context.Context, format string, args ...any) {
	globalManager.GetDefault().logRecord(LevelError, ctx, fmt.Sprintf(format, args...), true, args...)
}

// WarnfContext 记录格式化 Warn 日志并传播上下文。
func WarnfContext(ctx context.Context, format string, args ...any) {
	globalManager.GetDefault().logRecord(LevelWarn, ctx, fmt.Sprintf(format, args...), true, args...)
}

// DebugfContext 记录格式化 Debug 日志并传播上下文。
func DebugfContext(ctx context.Context, format string, args ...any) {
	globalManager.GetDefault().logRecord(LevelDebug, ctx, fmt.Sprintf(format, args...), true, args...)
}

// TracefContext 记录格式化 Trace 日志并传播上下文。
func TracefContext(ctx context.Context, format string, args ...any) {
	globalManager.GetDefault().logRecord(LevelTrace, ctx, fmt.Sprintf(format, args...), true, args...)
}

// Fatalf 记录格式化的全局Fatal级别的日志，并退出程序。
func Fatalf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelFatal, format, args...)
	os.Exit(1)
}

// Println 记录信息级别的日志。
func Println(msg string, args ...any) {
	globalManager.GetDefault().logWithLevel(LevelInfo, msg, args...)
}

// Printf 记录信息级别的格式化日志。
func Printf(format string, args ...any) {
	globalManager.GetDefault().logfWithLevel(LevelInfo, format, args...)
}

// 辅助便捷方法

// Progress 全局进度显示
//
//   - msg: 要显示的消息内容
//   - durationMs: 从0%到100%的总持续时间(毫秒)
func Progress(msg string, durationMs int) {
	globalManager.GetDefault().Progress(msg, durationMs)
}

// Countdown 全局倒计时显示
//
//   - msg: 要显示的消息内容
//   - seconds: 倒计时的秒数
func Countdown(msg string, seconds int) {
	globalManager.GetDefault().Countdown(msg, seconds)
}

// Loading 全局加载动画
//
//   - msg: 要显示的消息内容
//   - seconds: 动画持续的秒数
func Loading(msg string, seconds int) {
	globalManager.GetDefault().Loading(msg, seconds)
}

// With 创建一个新的日志记录器，带有指定的属性。
func With(args ...any) *Logger {
	return globalManager.GetDefault().With(args...)
}

// WithGroup 创建一个带有指定组名的全局日志记录器
// 这是一个包级别的便捷方法
// 参数:
//   - name: 日志组的名称
//
// 返回:
//   - 带有指定组名的新日志记录器实例
func WithGroup(name string) *Logger { return globalManager.GetDefault().WithGroup(name) }

// WithValue 在全局上下文中添加键值对并返回新的 Logger
func WithValue(key string, val any) *Logger {
	// 获取现有的全局Logger并调用其WithValue方法
	return globalManager.GetDefault().WithValue(key, val)
}

// SetLevelTrace 设置全局日志级别为Trace。
func SetLevelTrace() { levelVar.Set(LevelTrace) }

// SetLevelDebug 设置全局日志级别为Debug。
func SetLevelDebug() { levelVar.Set(LevelDebug) }

// SetLevelInfo 设置全局日志级别为Info。
func SetLevelInfo() { levelVar.Set(LevelInfo) }

// SetLevelWarn 设置全局日志级别为Warn。
func SetLevelWarn() { levelVar.Set(LevelWarn) }

// SetLevelError 设置全局日志级别为Error。
func SetLevelError() { levelVar.Set(LevelError) }

// SetLevelFatal 设置全局日志级别为Fatal。
func SetLevelFatal() { levelVar.Set(LevelFatal) }

// SetLevel 动态更新日志级别
// level 可以是数字(-8, -4, 0, 4, 8, 12)或字符串(trace, debug, info, warn, error, fatal)
func SetLevel(level any) error {
	var newLevel Level

	switch v := level.(type) {
	case Level:
		newLevel = v
	case int:
		newLevel = Level(v)
	case string:
		// 将字符串转换为Level
		switch strings.ToLower(v) {
		case "trace":
			newLevel = LevelTrace
		case "debug":
			newLevel = LevelDebug
		case "info":
			newLevel = LevelInfo
		case "warn":
			newLevel = LevelWarn
		case "error":
			newLevel = LevelError
		case "fatal":
			newLevel = LevelFatal
		default:
			return errors.New("invalid log level string")
		}
	default:
		return errors.New("unsupported level type")
	}

	// 验证级别是否有效
	if !isValidLevel(newLevel) {
		return errors.New("invalid log level value")
	}

	levelVar.Set(newLevel)

	return nil
}

// SetTimeFormat 全局方法：设置日志时间格式
//
//   - format: 时间格式字符串，例如 "2006-01-02 15:04:05.000"
func SetTimeFormat(format string) {
	if format != "" {
		TimeFormat = format
	}
}

// ResetGlobalLogger 重置全局logger实例
// 这在某些情况下很有用，比如需要更改全局logger的输出目标
func ResetGlobalLogger(w io.Writer, noColor, addSource bool) *Logger {
	if w == nil {
		w = os.Stdout
	}
	config := &GlobalConfig{
		DefaultWriter:  w,
		DefaultLevel:   levelVar.Level(),
		DefaultNoColor: noColor,
		DefaultSource:  addSource,
		EnableText:     isGlobalTextEnabled(),
		EnableJSON:     isGlobalJSONEnabled(),
	}
	_ = globalManager.Configure(config)
	globalManager.Reset()

	return globalManager.GetDefault()
}

// GetGlobalLogger 返回全局 logger 实例。
func GetGlobalLogger() *Logger {
	return globalManager.GetDefault()
}

// Subscribe 订阅日志记录
// 创建一个新的日志订阅，返回接收日志记录的通道和取消订阅的函数
//
// 参数:
//   - size: 通道缓冲区大小，决定可以在不阻塞的情况下缓存多少日志记录
//
// 返回值:
//   - <-chan SubscriptionEvent: 只读的订阅事件通道，包含结构化视图和当前激活输出对应的最终渲染结果
//   - context.CancelFunc: 取消订阅的函数，调用后会停止接收日志并清理资源
func Subscribe(size uint16) (<-chan SubscriptionEvent, context.CancelFunc) {
	return SubscribeWithOptions(SubscribeOptions{BufferSize: size})
}

// SubscribeWithOptions 使用可配置背压策略订阅日志记录。
// 订阅者拿到的是统一发布视图，而不是原始未处理的内部 record。
func SubscribeWithOptions(options SubscribeOptions) (<-chan SubscriptionEvent, context.CancelFunc) {
	options = options.normalized()
	ch := make(chan SubscriptionEvent, options.BufferSize)
	ctx, cancel := context.WithCancel(context.Background())

	subID := subscriberSeq.Add(1)
	sub := &subscriber{
		id:     subID,
		ch:     ch,
		cancel: cancel,
		done:   ctx.Done(),
		opts:   options,
		bornAt: time.Now(),
	}
	sub.state.Store(int32(stateActive)) // 设置为活跃状态

	subscribers.Store(subID, sub)
	subscriberCount.Add(1)

	// 创建安全的取消函数
	safeCancel := func() {
		if value, ok := subscribers.LoadAndDelete(subID); ok {
			if existing, ok := value.(*subscriber); ok {
				existing.close()
				subscriberCount.Add(-1)
				return
			}
		}
		sub.close()
	}

	// 监听context取消
	go func() {
		<-ctx.Done()
		safeCancel()
	}()

	return ch, safeCancel
}

// GetSubscriptionStats 返回订阅系统汇总统计。
func GetSubscriptionStats() SubscriptionStats {
	var stats SubscriptionStats
	stats.Evicted = subscriberEvicted.Load()

	subscribers.Range(func(_, value any) bool {
		sub := value.(*subscriber)
		s := sub.snapshot()
		stats.Subscribers++
		switch s.State {
		case "active":
			stats.ActiveSubscribers++
		case "closing":
			stats.ClosingSubscribers++
		case "closed":
			stats.ClosedSubscribers++
		}
		stats.Published += s.Published
		stats.Delivered += s.Delivered
		stats.Dropped += s.Dropped
		stats.DroppedOldest += s.DroppedOldest
		stats.DroppedNewest += s.DroppedNewest
		stats.DroppedTimed += s.DroppedTimed
		return true
	})

	return stats
}

// ListSubscriberStats 返回所有订阅者统计快照（按订阅ID升序）。
func ListSubscriberStats() []SubscriberStats {
	all := make([]SubscriberStats, 0, 8)
	subscribers.Range(func(_, value any) bool {
		sub := value.(*subscriber)
		all = append(all, sub.snapshot())
		return true
	})
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})
	return all
}

// GetSubscriberStats 根据订阅ID返回统计快照。
func GetSubscriberStats(id int64) (SubscriberStats, bool) {
	if value, ok := subscribers.Load(id); ok {
		return value.(*subscriber).snapshot(), true
	}
	return SubscriberStats{}, false
}

// EnableTextLogger 启用文本日志记录器。
func EnableTextLogger() {
	setGlobalTextEnabled(true)
}

// EnableJSONLogger 启用 JSON 日志记录器。
func EnableJSONLogger() {
	setGlobalJSONEnabled(true)
}

// DisableTextLogger 禁用文本日志记录器。
func DisableTextLogger() {
	setGlobalTextEnabled(false)
}

// DisableJSONLogger 禁用 JSON 日志记录器。
func DisableJSONLogger() {
	setGlobalJSONEnabled(false)
}

// EnableDLPLogger 启用日志脱敏功能
func EnableDLPLogger() {
	dlpEnabled.Store(true)
	if ext != nil {
		ext.enableDLP()
	}
}

// DisableDLPLogger 禁用日志脱敏功能
func DisableDLPLogger() {
	dlpEnabled.Store(false)
	if ext != nil {
		ext.disableDLP()
	}
}

// IsDLPEnabled 检查DLP是否启用
func IsDLPEnabled() bool {
	return dlpEnabled.Load()
}

// isValidLevel 检查日志级别是否有效
func isValidLevel(level Level) bool {
	validLevels := []Level{
		LevelTrace, // -8
		LevelDebug, // -4
		LevelInfo,  // 0
		LevelWarn,  // 4
		LevelError, // 8
		LevelFatal, // 12
	}

	return slices.Contains(validLevels, level)
}
