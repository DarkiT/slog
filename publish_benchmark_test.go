package slog

import (
	"io"
	"testing"
)

// BenchmarkPublishPipelineNoSubscribers 测试无订阅者时发布链路的短路开销。
func BenchmarkPublishPipelineNoSubscribers(b *testing.B) {
	logger := NewLogger(io.Discard, true, false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("publish benchmark", "k", "v")
		}
	})
}

// BenchmarkPublishPipelineWithSubscriber 测试带订阅者时统一发布视图的额外开销。
func BenchmarkPublishPipelineWithSubscriber(b *testing.B) {
	logger := NewLogger(io.Discard, true, false)
	events, cancel := Subscribe(1024)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range events {
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("publish benchmark", "k", "v")
		}
	})
	b.StopTimer()

	cancel()
	<-done
}
