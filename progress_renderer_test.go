package slog

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// testSyncBuffer 是一个线程安全的 bytes.Buffer 包装器（用于测试）
type testSyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (tsb *testSyncBuffer) Write(p []byte) (n int, err error) {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.Write(p)
}

func (tsb *testSyncBuffer) Len() int {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.Len()
}

func (tsb *testSyncBuffer) String() string {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.String()
}

func (tsb *testSyncBuffer) Reset() {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	tsb.buf.Reset()
}

var _ io.Writer = (*testSyncBuffer)(nil)

func TestDynamicProgressRenderer_Basic(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	renderer := GetDynamicProgressRenderer()

	// 测试基本功能
	err := renderer.StartDynamic(logger, "测试动态进度", 3, 50*time.Millisecond, &buf)
	if err != nil {
		t.Fatalf("StartDynamic失败: %v", err)
	}

	// 等待渲染完成
	time.Sleep(200 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "测试动态进度") {
		t.Error("输出应该包含动态进度消息")
	}
}

func TestDynamicProgressRenderer_PrecomputeFrames(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	renderer := GetDynamicProgressRenderer()

	// 测试预计算帧
	frames := renderer.precomputeFrames(logger, "预计算测试", 5)

	if len(frames) != 5 {
		t.Errorf("期望5帧，实际: %d", len(frames))
	}

	// 验证帧内容
	for i, frame := range frames {
		if !strings.Contains(frame, "预计算测试") {
			t.Errorf("帧%d应该包含消息", i)
		}
		if !strings.HasPrefix(frame, "\r") {
			t.Errorf("帧%d应该以\\r开头", i)
		}
	}
}

func TestDynamic(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	// 测试优化后的Dynamic方法
	start := time.Now()
	logger.Dynamic("优化测试", 3, 10, &buf)
	elapsed := time.Since(start)

	// 由于是异步的，应该很快返回
	if elapsed > 50*time.Millisecond {
		t.Errorf("Dynamic应该快速返回，实际用时: %v", elapsed)
	}

	// 等待渲染完成
	time.Sleep(100 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "优化测试") {
		t.Error("输出应该包含优化测试消息")
	}
}

func TestFastDynamic(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	// 测试快速动态进度条
	start := time.Now()
	logger.FastDynamic("快速测试", 300, &buf)
	elapsed := time.Since(start)

	// 应该快速返回（异步）
	if elapsed > 50*time.Millisecond {
		t.Errorf("FastDynamic应该快速返回，实际用时: %v", elapsed)
	}

	// 等待渲染完成
	time.Sleep(400 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "快速测试") {
		t.Error("输出应该包含快速测试消息")
	}
}

func TestDynamicProgressRenderer_StopAll(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	renderer := GetDynamicProgressRenderer()

	// 启动多个动态进度条
	renderer.StartDynamic(logger, "进度1", 10, 100*time.Millisecond, &buf)
	renderer.StartDynamic(logger, "进度2", 10, 100*time.Millisecond, &buf)

	// 验证活跃数量
	if renderer.GetActiveCount() < 2 {
		t.Error("应该有至少2个活跃的进度条")
	}

	// 停止所有
	renderer.StopAll()

	// 等待清理
	time.Sleep(50 * time.Millisecond)

	if renderer.GetActiveCount() != 0 {
		t.Errorf("停止后应该没有活跃的进度条，实际: %d", renderer.GetActiveCount())
	}
}

func TestFramePool(t *testing.T) {
	pool := newFramePool()

	// 测试获取和归还
	frames1 := pool.get()
	frames1 = append(frames1, "test1", "test2")

	pool.put(frames1)

	frames2 := pool.get()
	if len(frames2) != 0 {
		t.Error("从池中获取的切片应该长度为0")
	}
	if cap(frames2) < 2 {
		t.Error("从池中获取的切片应该保留容量")
	}
}

func TestDynamicFallback(t *testing.T) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	// 测试同步回退实现
	start := time.Now()
	logger.dynamicFallback("回退测试", 3, 10, &buf)
	elapsed := time.Since(start)

	// 同步版本应该等待完成
	expectedTime := 3 * 10 * time.Millisecond // 3帧 * 10ms间隔
	if elapsed < expectedTime/2 {
		t.Errorf("同步版本应该等待，实际用时: %v", elapsed)
	}

	output := buf.String()
	if !strings.Contains(output, "回退测试") {
		t.Error("输出应该包含回退测试消息")
	}
}

// 性能基准测试

func BenchmarkDynamicOriginal(b *testing.B) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		// 使用较短的时间避免测试过长
		logger.dynamicFallback("基准测试", 3, 1, &buf)
	}
}

func BenchmarkDynamicRenderer(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 为每次迭代创建独立的buffer，避免并发问题
		var buf testSyncBuffer
		logger := NewLogger(&buf, true, false)
		logger.Dynamic("基准测试", 3, 1, &buf)
		// 不等待完成，测试启动性能
	}
}

func BenchmarkPrecomputeFrames(b *testing.B) {
	var buf testSyncBuffer
	logger := NewLogger(&buf, true, false)
	renderer := GetDynamicProgressRenderer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frames := renderer.precomputeFrames(logger, "基准测试", 10)
		renderer.pool.put(frames) // 归还到池中
	}
}

func BenchmarkFramePool(b *testing.B) {
	pool := newFramePool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frames := pool.get()
		frames = append(frames, "test1", "test2", "test3")
		pool.put(frames)
	}
}

// 压力测试

func TestDynamicProgressRenderer_Concurrent(t *testing.T) {
	renderer := GetDynamicProgressRenderer()
	var logBuf testSyncBuffer
	logger := NewLogger(&logBuf, true, false)

	// 清理之前的状态
	renderer.StopAll()
	time.Sleep(50 * time.Millisecond)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// 并发启动多个动态进度条
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			var buf testSyncBuffer
			err := renderer.StartDynamic(logger, "并发测试", 5, 20*time.Millisecond, &buf)
			if err != nil {
				t.Errorf("Goroutine %d: StartDynamic失败: %v", id, err)
			}
			done <- true
		}(i)
	}

	// 等待所有启动完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证并发启动成功
	activeCount := renderer.GetActiveCount()
	if activeCount < numGoroutines {
		t.Errorf("期望至少%d个活跃进度条，实际: %d", numGoroutines, activeCount)
	}

	// 清理
	renderer.StopAll()
	time.Sleep(100 * time.Millisecond)

	if renderer.GetActiveCount() != 0 {
		t.Errorf("清理后应该没有活跃进度条，实际: %d", renderer.GetActiveCount())
	}
}

func TestDynamicStats(t *testing.T) {
	renderer := GetDynamicProgressRenderer()
	var logBuf testSyncBuffer
	logger := NewLogger(&logBuf, true, false)

	// 清理状态
	renderer.StopAll()
	time.Sleep(50 * time.Millisecond)

	// 获取初始统计
	stats := renderer.GetDynamicStats()
	if stats.ActiveCount != 0 {
		t.Errorf("初始活跃数量应该为0，实际: %d", stats.ActiveCount)
	}

	// 启动一些进度条
	var buf testSyncBuffer
	renderer.StartDynamic(logger, "统计测试1", 5, 50*time.Millisecond, &buf)
	renderer.StartDynamic(logger, "统计测试2", 5, 50*time.Millisecond, &buf)

	stats = renderer.GetDynamicStats()
	if stats.ActiveCount < 2 {
		t.Errorf("应该有至少2个活跃进度条，实际: %d", stats.ActiveCount)
	}
}
