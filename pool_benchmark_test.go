package slog

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/darkit/slog/internal/common"
)

// BenchmarkPoolComparison 对比分级池与传统sync.Pool的性能
func BenchmarkPoolComparison(b *testing.B) {
	// 传统单一sync.Pool
	var traditionalPool sync.Pool
	traditionalPool.New = func() interface{} {
		return &strings.Builder{}
	}

	// 分级池
	tieredPools := common.NewTieredPools()

	b.Run("Traditional_SmallData", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := traditionalPool.Get().(*strings.Builder)
			builder.Reset()
			builder.WriteString("小数据测试")
			_ = builder.String()
			traditionalPool.Put(builder)
		}
	})

	b.Run("Tiered_SmallData", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := tieredPools.GetStringBuilder(256)
			builder.WriteString("小数据测试")
			_ = builder.String()
			tieredPools.PutStringBuilder(builder, 256)
		}
	})

	b.Run("Traditional_MediumData", func(b *testing.B) {
		mediumData := strings.Repeat("中等数据 ", 50)
		for i := 0; i < b.N; i++ {
			builder := traditionalPool.Get().(*strings.Builder)
			builder.Reset()
			builder.WriteString(mediumData)
			_ = builder.String()
			traditionalPool.Put(builder)
		}
	})

	b.Run("Tiered_MediumData", func(b *testing.B) {
		mediumData := strings.Repeat("中等数据 ", 50)
		for i := 0; i < b.N; i++ {
			builder := tieredPools.GetStringBuilder(1024)
			builder.WriteString(mediumData)
			_ = builder.String()
			tieredPools.PutStringBuilder(builder, 1024)
		}
	})

	b.Run("Traditional_LargeData", func(b *testing.B) {
		largeData := strings.Repeat("大量数据内容 ", 200)
		for i := 0; i < b.N; i++ {
			builder := traditionalPool.Get().(*strings.Builder)
			builder.Reset()
			builder.WriteString(largeData)
			_ = builder.String()
			traditionalPool.Put(builder)
		}
	})

	b.Run("Tiered_LargeData", func(b *testing.B) {
		largeData := strings.Repeat("大量数据内容 ", 200)
		for i := 0; i < b.N; i++ {
			builder := tieredPools.GetStringBuilder(4096)
			builder.WriteString(largeData)
			_ = builder.String()
			tieredPools.PutStringBuilder(builder, 4096)
		}
	})
}

// BenchmarkMemoryEfficiency 测试内存使用效率
func BenchmarkMemoryEfficiency(b *testing.B) {
	b.Run("Traditional_MixedSizes", func(b *testing.B) {
		var pool sync.Pool
		pool.New = func() interface{} {
			return &strings.Builder{}
		}

		for i := 0; i < b.N; i++ {
			builder := pool.Get().(*strings.Builder)
			builder.Reset()

			// 混合大小的数据
			switch i % 3 {
			case 0:
				builder.WriteString("小")
			case 1:
				builder.WriteString(strings.Repeat("中 ", 50))
			case 2:
				builder.WriteString(strings.Repeat("大 ", 200))
			}

			_ = builder.String()
			pool.Put(builder)
		}
	})

	b.Run("Tiered_MixedSizes", func(b *testing.B) {
		tieredPools := common.NewTieredPools()

		for i := 0; i < b.N; i++ {
			var builder *strings.Builder
			var expectedCap int

			// 根据数据大小选择合适的池
			switch i % 3 {
			case 0:
				builder = tieredPools.GetStringBuilder(256)
				builder.WriteString("小")
				expectedCap = 256
			case 1:
				builder = tieredPools.GetStringBuilder(1024)
				builder.WriteString(strings.Repeat("中 ", 50))
				expectedCap = 1024
			case 2:
				builder = tieredPools.GetStringBuilder(4096)
				builder.WriteString(strings.Repeat("大 ", 200))
				expectedCap = 4096
			}

			_ = builder.String()
			tieredPools.PutStringBuilder(builder, expectedCap)
		}
	})
}

// BenchmarkConcurrentAccess 并发访问性能测试
func BenchmarkConcurrentAccess(b *testing.B) {
	b.Run("Traditional_Concurrent", func(b *testing.B) {
		var pool sync.Pool
		pool.New = func() interface{} {
			return &strings.Builder{}
		}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				builder := pool.Get().(*strings.Builder)
				builder.Reset()
				builder.WriteString(fmt.Sprintf("并发测试 %d", i))
				_ = builder.String()
				pool.Put(builder)
				i++
			}
		})
	})

	b.Run("Tiered_Concurrent", func(b *testing.B) {
		tieredPools := common.NewTieredPools()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				builder := tieredPools.GetStringBuilder(512)
				builder.WriteString(fmt.Sprintf("并发测试 %d", i))
				_ = builder.String()
				tieredPools.PutStringBuilder(builder, 512)
				i++
			}
		})
	})
}

// BenchmarkRealWorldScenario 真实世界场景测试
func BenchmarkRealWorldScenario(b *testing.B) {
	b.Run("Logger_Traditional", func(b *testing.B) {
		var pool sync.Pool
		pool.New = func() interface{} {
			return &strings.Builder{}
		}

		for i := 0; i < b.N; i++ {
			builder := pool.Get().(*strings.Builder)
			builder.Reset()

			// 模拟日志格式化
			builder.WriteString("2025/08/02 19:15.52.409 ")
			builder.WriteString("[INFO] ")
			builder.WriteString("用户操作 - ID: ")
			builder.WriteString(fmt.Sprintf("%d", i))
			builder.WriteString(", 操作: 登录")

			_ = builder.String()
			pool.Put(builder)
		}
	})

	b.Run("Logger_Tiered", func(b *testing.B) {
		tieredPools := common.NewTieredPools()

		for i := 0; i < b.N; i++ {
			// 根据日志内容长度选择合适大小
			builder := tieredPools.GetStringBuilder(256)

			// 模拟日志格式化
			builder.WriteString("2025/08/02 19:15.52.409 ")
			builder.WriteString("[INFO] ")
			builder.WriteString("用户操作 - ID: ")
			builder.WriteString(fmt.Sprintf("%d", i))
			builder.WriteString(", 操作: 登录")

			_ = builder.String()
			tieredPools.PutStringBuilder(builder, 256)
		}
	})

	b.Run("DLP_Traditional", func(b *testing.B) {
		var pool sync.Pool
		pool.New = func() interface{} {
			return &strings.Builder{}
		}

		for i := 0; i < b.N; i++ {
			builder := pool.Get().(*strings.Builder)
			builder.Reset()

			// 模拟DLP脱敏处理
			originalText := fmt.Sprintf("用户信息: 手机号13812345678, 邮箱user%d@example.com", i)
			builder.WriteString(originalText)
			original := builder.String()

			builder.Reset()
			// 模拟脱敏结果
			builder.WriteString("用户信息: 手机号138****5678, 邮箱u***@example.com")

			_ = builder.String()
			_ = original // 避免未使用变量警告
			pool.Put(builder)
		}
	})

	b.Run("DLP_Tiered", func(b *testing.B) {
		tieredPools := common.NewTieredPools()

		for i := 0; i < b.N; i++ {
			// DLP处理通常需要中等大小的buffer
			builder := tieredPools.GetStringBuilder(1024)

			// 模拟DLP脱敏处理
			originalText := fmt.Sprintf("用户信息: 手机号13812345678, 邮箱user%d@example.com", i)
			builder.WriteString(originalText)
			original := builder.String()

			builder.Reset()
			// 模拟脱敏结果
			builder.WriteString("用户信息: 手机号138****5678, 邮箱u***@example.com")

			_ = builder.String()
			_ = original // 避免未使用变量警告
			tieredPools.PutStringBuilder(builder, 1024)
		}
	})
}

// TestTieredPoolsMemoryFootprint 内存占用测试
func TestTieredPoolsMemoryFootprint(t *testing.T) {
	tieredPools := common.NewTieredPools()

	// 获取大量对象以测试内存使用
	var builders []*strings.Builder
	var buffers []*common.TieredBuffer

	// 小对象
	for i := 0; i < 100; i++ {
		builders = append(builders, tieredPools.GetStringBuilder(256))
		buffers = append(buffers, tieredPools.GetBuffer(512))
	}

	// 中对象
	for i := 0; i < 50; i++ {
		builders = append(builders, tieredPools.GetStringBuilder(1024))
		buffers = append(buffers, tieredPools.GetBuffer(4096))
	}

	// 大对象
	for i := 0; i < 20; i++ {
		builders = append(builders, tieredPools.GetStringBuilder(4096))
		buffers = append(buffers, tieredPools.GetBuffer(16384))
	}

	// 使用对象
	for i, builder := range builders {
		builder.WriteString(fmt.Sprintf("测试数据 %d", i))
	}

	for i, buffer := range buffers {
		buffer.WriteString(fmt.Sprintf("缓冲区数据 %d", i))
	}

	// 释放对象
	for i, builder := range builders {
		var expectedCap int
		switch {
		case i < 100:
			expectedCap = 256
		case i < 150:
			expectedCap = 1024
		default:
			expectedCap = 4096
		}
		tieredPools.PutStringBuilder(builder, expectedCap)
	}

	for _, buffer := range buffers {
		tieredPools.PutBuffer(buffer)
	}

	// 获取最终统计
	stats := tieredPools.GetStats()
	t.Logf("内存占用测试完成:")
	for size, stat := range stats {
		t.Logf("  %v: Gets=%d, Puts=%d, Hit Rate=%.2f%%",
			size, stat.Gets, stat.Puts, stat.HitRate)
	}
}

// TestTieredPoolsAdaptiveResize 自适应大小调整测试
func TestTieredPoolsAdaptiveResize(t *testing.T) {
	tieredPools := common.NewTieredPools()

	// 模拟逐渐增长的使用模式
	sizes := []int{100, 200, 500, 800, 1200, 2000, 3000, 5000}

	for _, size := range sizes {
		builder := tieredPools.GetStringBuilder(size)

		// 写入期望大小的数据
		data := strings.Repeat("x", size/2)
		builder.WriteString(data)

		actualCap := builder.Cap()
		t.Logf("期望容量: %d, 实际容量: %d, 实际数据: %d",
			size, actualCap, builder.Len())

		if actualCap < size {
			t.Errorf("实际容量 %d 小于期望容量 %d", actualCap, size)
		}

		tieredPools.PutStringBuilder(builder, size)
	}

	// 检查最终统计
	stats := tieredPools.GetStats()
	for size, stat := range stats {
		t.Logf("池 %v 统计: Gets=%d, News=%d, Hit Rate=%.2f%%",
			size, stat.Gets, stat.News, stat.HitRate)
	}
}
