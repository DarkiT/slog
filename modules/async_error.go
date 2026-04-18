package modules

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

type asyncErrorWriter struct {
	io.Writer
}

type asyncTask struct {
	component string
	task      func() error
}

const (
	defaultAsyncWorkerCount = 4
	defaultAsyncQueueSize   = 256
)

// AsyncExecutorOptions controls module async task execution behavior.
// Values <= 0 fallback to built-in defaults.
type AsyncExecutorOptions struct {
	WorkerCount int
	QueueSize   int
}

type asyncExecutorState struct {
	queue   chan asyncTask
	options AsyncExecutorOptions
}

var (
	asyncErrorSink      atomic.Value // stores io.Writer
	asyncErrorLogEnable atomic.Bool
	asyncErrorMu        sync.Mutex
	asyncExecutorMu     sync.RWMutex
	asyncExecutor       *asyncExecutorState
	asyncDropCount      atomic.Uint64
)

func init() {
	asyncErrorSink.Store(asyncErrorWriter{Writer: os.Stderr})
	asyncErrorLogEnable.Store(true)
	SetAsyncExecutorOptions(AsyncExecutorOptions{})
}

func normalizeAsyncExecutorOptions(options AsyncExecutorOptions) AsyncExecutorOptions {
	if options.WorkerCount <= 0 {
		options.WorkerCount = defaultAsyncWorkerCount
	}
	if options.QueueSize <= 0 {
		options.QueueSize = defaultAsyncQueueSize
	}
	return options
}

func startAsyncExecutorState(options AsyncExecutorOptions) *asyncExecutorState {
	state := &asyncExecutorState{
		queue:   make(chan asyncTask, options.QueueSize),
		options: options,
	}
	for i := 0; i < options.WorkerCount; i++ {
		go func(ch <-chan asyncTask) {
			for task := range ch {
				runAsyncTask(task.component, task.task)
			}
		}(state.queue)
	}
	return state
}

// SetAsyncExecutorOptions updates async executor workers/queue at runtime.
// Existing queued tasks are drained by old workers before old executor exits.
func SetAsyncExecutorOptions(options AsyncExecutorOptions) {
	options = normalizeAsyncExecutorOptions(options)
	newState := startAsyncExecutorState(options)

	asyncExecutorMu.Lock()
	oldState := asyncExecutor
	asyncExecutor = newState
	if oldState != nil {
		close(oldState.queue)
	}
	asyncExecutorMu.Unlock()
}

// GetAsyncExecutorOptions returns current async executor settings.
func GetAsyncExecutorOptions() AsyncExecutorOptions {
	asyncExecutorMu.RLock()
	defer asyncExecutorMu.RUnlock()
	if asyncExecutor == nil {
		return normalizeAsyncExecutorOptions(AsyncExecutorOptions{})
	}
	return asyncExecutor.options
}

func runAsyncTask(component string, task func() error) {
	defer func() {
		if rec := recover(); rec != nil {
			ReportAsyncError(component, fmt.Errorf("panic: %v\n%s", rec, string(debug.Stack())))
		}
	}()
	if err := task(); err != nil {
		ReportAsyncError(component, err)
	}
}

// SetAsyncErrorWriter 设置模块异步任务错误输出目标；传入 nil 会恢复为 stderr。
func SetAsyncErrorWriter(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	asyncErrorSink.Store(asyncErrorWriter{Writer: w})
}

// EnableAsyncErrorLogging 开关模块异步任务错误日志输出。
func EnableAsyncErrorLogging(enabled bool) {
	asyncErrorLogEnable.Store(enabled)
}

// ReportAsyncError 统一上报模块中的异步任务错误。
func ReportAsyncError(component string, err error) {
	if err == nil || !asyncErrorLogEnable.Load() {
		return
	}
	sink, _ := asyncErrorSink.Load().(asyncErrorWriter)
	writer := sink.Writer
	if writer == nil {
		writer = os.Stderr
	}
	asyncErrorMu.Lock()
	defer asyncErrorMu.Unlock()
	_, _ = fmt.Fprintf(writer, "slog/modules: async %s error: %v\n", component, err)
}

// RunAsync 统一执行模块异步任务并处理 panic/error 上报。
func RunAsync(component string, task func() error) {
	if task == nil {
		return
	}
	asyncExecutorMu.RLock()
	current := asyncExecutor
	asyncExecutorMu.RUnlock()
	if current == nil {
		return
	}
	select {
	case current.queue <- asyncTask{component: component, task: task}:
	default:
		dropped := asyncDropCount.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			ReportAsyncError(component, fmt.Errorf("task dropped: async queue full (dropped=%d)", dropped))
		}
	}
}
