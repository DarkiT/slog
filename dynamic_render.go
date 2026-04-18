package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"
)

type dynamicRenderContextKey struct{}

type dynamicRenderState struct {
	final bool
}

func withDynamicRender(ctx context.Context, state dynamicRenderState) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, dynamicRenderContextKey{}, state)
}

func dynamicRenderFromContext(ctx context.Context) (dynamicRenderState, bool) {
	if ctx == nil {
		return dynamicRenderState{}, false
	}
	state, ok := ctx.Value(dynamicRenderContextKey{}).(dynamicRenderState)
	return state, ok
}

func cloneConfig(config *Config) *Config {
	if config == nil {
		return DefaultConfig()
	}
	cloned := *config
	if config.EnableText != nil {
		cloned.EnableText = boolPtr(*config.EnableText)
	}
	if config.EnableJSON != nil {
		cloned.EnableJSON = boolPtr(*config.EnableJSON)
	}
	return &cloned
}

func (l *Logger) dynamicTarget(writer ...io.Writer) *Logger {
	if len(writer) == 0 || writer[0] == nil {
		return l
	}

	config := cloneConfig(l.config)
	config.NoColor = l.noColor

	target := NewLoggerWithConfig(writer[0], config)
	target.ctx = l.ctx
	target.level = l.GetLevel()
	return target
}

func (l *Logger) supportsInlineDynamic() bool {
	if l == nil {
		return false
	}

	textOn, jsonOn := l.outputEnabled()
	if !textOn || jsonOn || l.text == nil {
		return false
	}

	return handlerSupportsDynamicInline(l.text.Handler())
}

func handlerSupportsDynamicInline(h slog.Handler) bool {
	switch current := h.(type) {
	case *eHandler:
		return handlerSupportsDynamicInline(current.handler)
	case *handler:
		return current.canRenderDynamicInline()
	default:
		return false
	}
}

func (h *handler) canRenderDynamicInline() bool {
	return h != nil && h.dynamicTTY
}

func (l *Logger) emitDynamicRecord(msg string, final bool) {
	l.logRecord(LevelInfo, withDynamicRender(l.ctx, dynamicRenderState{final: final}), msg, false)
}

func (l *Logger) emitInfo(msg string) {
	l.logRecord(LevelInfo, l.ctx, msg, false)
}

func (l *Logger) sleepBetweenMarks(total time.Duration, prevPercent, currentPercent int) {
	if total <= 0 || currentPercent <= prevPercent {
		return
	}
	wait := total * time.Duration(currentPercent-prevPercent) / 100
	if wait > 0 {
		time.Sleep(wait)
	}
}

func countdownMarks(seconds int) []int {
	if seconds <= 0 {
		return []int{0}
	}
	uniq := map[int]struct{}{
		seconds: {},
		0:       {},
	}
	if seconds <= 10 {
		for remaining := seconds; remaining >= 0; remaining-- {
			uniq[remaining] = struct{}{}
		}
	} else {
		for remaining := seconds - 1; remaining > 5; remaining-- {
			if remaining%10 == 0 {
				uniq[remaining] = struct{}{}
			}
		}
		for remaining := 5; remaining >= 0; remaining-- {
			uniq[remaining] = struct{}{}
		}
	}

	marks := make([]int, 0, len(uniq))
	for remaining := range uniq {
		marks = append(marks, remaining)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(marks)))
	return marks
}

// Progress 显示进度百分比。
func (l *Logger) Progress(msg string, durationMs int, writer ...io.Writer) {
	target := l.dynamicTarget(writer...)
	total := time.Duration(durationMs) * time.Millisecond

	l.mu.Lock()
	defer l.mu.Unlock()

	if target.supportsInlineDynamic() {
		if total <= 0 {
			target.emitDynamicRecord(fmt.Sprintf("%s %.1f%%", msg, 100.0), true)
			return
		}

		start := time.Now()
		step := total / 100
		if step <= 0 {
			step = time.Millisecond
		}

		for {
			progress := float64(time.Since(start)) / float64(total) * 100
			if progress > 100 {
				progress = 100
			}
			final := progress >= 100
			target.emitDynamicRecord(fmt.Sprintf("%s %.1f%%", msg, progress), final)
			if final {
				return
			}
			time.Sleep(step)
		}
	}

	if total <= 0 {
		target.emitInfo(fmt.Sprintf("%s %.1f%%", msg, 100.0))
		return
	}

	marks := []int{0, 25, 50, 75, 100}
	prev := 0
	for i, mark := range marks {
		if i > 0 {
			target.sleepBetweenMarks(total, prev, mark)
		}
		target.emitInfo(fmt.Sprintf("%s %.1f%%", msg, float64(mark)))
		prev = mark
	}
}

// Countdown 显示倒计时。
func (l *Logger) Countdown(msg string, seconds int, writer ...io.Writer) {
	target := l.dynamicTarget(writer...)

	l.mu.Lock()
	defer l.mu.Unlock()

	if target.supportsInlineDynamic() {
		if seconds <= 0 {
			target.emitDynamicRecord(fmt.Sprintf("%s %ds", msg, 0), true)
			return
		}
		for remaining := seconds; remaining >= 0; remaining-- {
			final := remaining == 0
			target.emitDynamicRecord(fmt.Sprintf("%s %ds", msg, remaining), final)
			if final {
				return
			}
			time.Sleep(time.Second)
		}
		return
	}

	marks := countdownMarks(seconds)
	for i, remaining := range marks {
		if i > 0 {
			sleep := time.Duration(marks[i-1]-remaining) * time.Second
			if sleep > 0 {
				time.Sleep(sleep)
			}
		}
		target.emitInfo(fmt.Sprintf("%s %ds", msg, remaining))
	}
}

// Loading 显示加载动画。
func (l *Logger) Loading(msg string, seconds int, writer ...io.Writer) {
	target := l.dynamicTarget(writer...)
	total := time.Duration(seconds) * time.Second

	l.mu.Lock()
	defer l.mu.Unlock()

	if target.supportsInlineDynamic() {
		spinner := []string{"|", "/", "-", "\\"}
		if total <= 0 {
			target.emitDynamicRecord(msg+" done", true)
			return
		}
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.Now().Add(total)
		frame := 0

		for {
			final := time.Now().After(deadline)
			if final {
				target.emitDynamicRecord(msg+" done", true)
				return
			}

			target.emitDynamicRecord(fmt.Sprintf("%s %s", spinner[frame%len(spinner)], msg), false)
			frame++
			<-ticker.C
		}
	}

	target.emitInfo(msg + " started")
	if total > 0 {
		time.Sleep(total)
	}
	target.emitInfo(msg + " done")
}
