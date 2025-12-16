package common

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestTieredPools_BufferSizeSelection 测试buffer大小选择逻辑
func TestTieredPools_BufferSizeSelection(t *testing.T) {
	tp := NewTieredPools()

	testCases := []struct {
		expectedSize int
		expectedTier BufferSize
		description  string
	}{
		{100, SmallBuffer, "小数据应该选择小buffer"},
		{1024, SmallBuffer, "1KB数据选择小buffer"},
		{2048, SmallBuffer, "2KB数据选择小buffer"},
		{3000, MediumBuffer, "3KB数据选择中buffer"},
		{8000, MediumBuffer, "8KB数据选择中buffer"},
		{10000, LargeBuffer, "10KB数据选择大buffer"},
		{50000, LargeBuffer, "50KB数据选择大buffer"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			buffer := tp.GetBuffer(tc.expectedSize)
			if buffer.Size() != tc.expectedTier {
				t.Errorf("期望buffer大小: %v, 实际: %v", tc.expectedTier, buffer.Size())
			}
			tp.PutBuffer(buffer)
		})
	}
}

// TestTieredPools_StringBuilderSizeSelection 测试字符串构建器大小选择逻辑
func TestTieredPools_StringBuilderSizeSelection(t *testing.T) {
	tp := NewTieredPools()

	testCases := []struct {
		expectedCapacity int
		expectedPool     string
		description      string
	}{
		{100, "small", "小容量选择小池"},
		{256, "small", "256容量选择小池"},
		{500, "medium", "500容量选择中池"},
		{1024, "medium", "1KB容量选择中池"},
		{2000, "large", "2KB容量选择大池"},
		{5000, "large", "5KB容量选择大池"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			builder := tp.GetStringBuilder(tc.expectedCapacity)

			// 验证容量是否合适
			if builder.Cap() < tc.expectedCapacity {
				t.Errorf("构建器容量 %d 小于期望容量 %d", builder.Cap(), tc.expectedCapacity)
			}

			tp.PutStringBuilder(builder, tc.expectedCapacity)
		})
	}
}

// TestTieredBuffer_Operations 测试分级buffer的基本操作
func TestTieredBuffer_Operations(t *testing.T) {
	tp := NewTieredPools()

	t.Run("写入操作", func(t *testing.T) {
		buffer := tp.GetBuffer(1024)
		defer tp.PutBuffer(buffer)

		// 测试字节写入
		data := []byte("Hello World")
		n, err := buffer.Write(data)
		if err != nil {
			t.Errorf("Write失败: %v", err)
		}
		if n != len(data) {
			t.Errorf("写入长度不匹配: 期望%d, 实际%d", len(data), n)
		}

		// 测试字符串写入
		str := " from TieredBuffer"
		n, err = buffer.WriteString(str)
		if err != nil {
			t.Errorf("WriteString失败: %v", err)
		}
		if n != len(str) {
			t.Errorf("字符串写入长度不匹配: 期望%d, 实际%d", len(str), n)
		}

		// 验证内容
		expected := "Hello World from TieredBuffer"
		if buffer.String() != expected {
			t.Errorf("内容不匹配: 期望%q, 实际%q", expected, buffer.String())
		}
	})

	t.Run("容量和长度", func(t *testing.T) {
		buffer := tp.GetBuffer(2048)
		defer tp.PutBuffer(buffer)

		if buffer.Len() != 0 {
			t.Errorf("新buffer长度应该为0, 实际: %d", buffer.Len())
		}

		// 检查buffer容量是否合理（至少应该有基本容量）
		if buffer.Cap() < 256 {
			t.Errorf("buffer容量应该至少为256, 实际: %d", buffer.Cap())
		}

		// 写入数据后检查长度
		buffer.WriteString("test data")
		if buffer.Len() != 9 {
			t.Errorf("写入后长度应该为9, 实际: %d", buffer.Len())
		}
	})
}

// TestTieredPools_Statistics 测试统计信息
func TestTieredPools_Statistics(t *testing.T) {
	tp := NewTieredPools()

	// 获取一些buffer
	buffer1 := tp.GetBuffer(500)   // 小buffer
	buffer2 := tp.GetBuffer(3000)  // 中buffer
	buffer3 := tp.GetBuffer(15000) // 大buffer

	// 获取一些字符串构建器
	builder1 := tp.GetStringBuilder(200)  // 小
	builder2 := tp.GetStringBuilder(800)  // 中
	builder3 := tp.GetStringBuilder(3000) // 大

	// 放回一些对象
	tp.PutBuffer(buffer1)
	tp.PutStringBuilder(builder1, 200)

	stats := tp.GetStats()

	// 验证小buffer统计
	if stats[SmallBuffer].Gets < 2 { // buffer + string builder
		t.Errorf("小池获取次数应该至少为2, 实际: %d", stats[SmallBuffer].Gets)
	}

	// 验证中buffer统计
	if stats[MediumBuffer].Gets < 2 {
		t.Errorf("中池获取次数应该至少为2, 实际: %d", stats[MediumBuffer].Gets)
	}

	// 验证大buffer统计
	if stats[LargeBuffer].Gets < 2 {
		t.Errorf("大池获取次数应该至少为2, 实际: %d", stats[LargeBuffer].Gets)
	}

	// 清理
	tp.PutBuffer(buffer2)
	tp.PutBuffer(buffer3)
	tp.PutStringBuilder(builder2, 800)
	tp.PutStringBuilder(builder3, 3000)
}

// TestTieredPools_BufferReuse 测试buffer复用
func TestTieredPools_BufferReuse(t *testing.T) {
	tp := NewTieredPools()

	// 第一次获取
	buffer1 := tp.GetBuffer(1024)
	buffer1.WriteString("test data")
	originalPtr := fmt.Sprintf("%p", buffer1)
	tp.PutBuffer(buffer1)

	// 第二次获取，应该复用同一个buffer
	buffer2 := tp.GetBuffer(1024)
	newPtr := fmt.Sprintf("%p", buffer2)

	// 验证是否复用了同一个对象
	if originalPtr != newPtr {
		t.Logf("Buffer可能没有复用 (这是正常的): 原始=%s, 新的=%s", originalPtr, newPtr)
	}

	// 验证buffer已经被重置
	if buffer2.Len() != 0 {
		t.Errorf("复用的buffer应该被重置, 长度应该为0, 实际: %d", buffer2.Len())
	}

	tp.PutBuffer(buffer2)
}

// TestTieredPools_Concurrency 测试并发安全
func TestTieredPools_Concurrency(t *testing.T) {
	tp := NewTieredPools()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 并发获取和放回buffer
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 随机选择不同大小的buffer
				size := (id*numOperations + j) % 3
				var buffer *TieredBuffer

				switch size {
				case 0:
					buffer = tp.GetBuffer(512)
				case 1:
					buffer = tp.GetBuffer(4096)
				case 2:
					buffer = tp.GetBuffer(16384)
				}

				// 写入一些数据
				buffer.WriteString(fmt.Sprintf("goroutine-%d-op-%d", id, j))

				// 放回池中
				tp.PutBuffer(buffer)
			}
		}(i)
	}

	wg.Wait()

	// 验证统计信息
	stats := tp.GetStats()
	totalOps := int64(numGoroutines * numOperations)

	if stats[SmallBuffer].Gets == 0 && stats[MediumBuffer].Gets == 0 && stats[LargeBuffer].Gets == 0 {
		t.Error("并发操作后应该有统计数据")
	}

	t.Logf("并发测试完成，总操作数: %d", totalOps)
	t.Logf("小池统计: Gets=%d, Puts=%d", stats[SmallBuffer].Gets, stats[SmallBuffer].Puts)
	t.Logf("中池统计: Gets=%d, Puts=%d", stats[MediumBuffer].Gets, stats[MediumBuffer].Puts)
	t.Logf("大池统计: Gets=%d, Puts=%d", stats[LargeBuffer].Gets, stats[LargeBuffer].Puts)
}

// TestGlobalTieredPools 测试全局便捷函数
func TestGlobalTieredPools(t *testing.T) {
	// 测试全局便捷函数
	smallBuffer := GetSmallBuffer()
	mediumBuffer := GetMediumBuffer()
	largeBuffer := GetLargeBuffer()

	if smallBuffer.Size() != SmallBuffer {
		t.Errorf("GetSmallBuffer应该返回小buffer")
	}
	if mediumBuffer.Size() != MediumBuffer {
		t.Errorf("GetMediumBuffer应该返回中buffer")
	}
	if largeBuffer.Size() != LargeBuffer {
		t.Errorf("GetLargeBuffer应该返回大buffer")
	}

	// 测试字符串构建器全局函数
	smallBuilder := GetSmallStringBuilder()
	mediumBuilder := GetMediumStringBuilder()
	largeBuilder := GetLargeStringBuilder()

	// 验证容量
	if smallBuilder.Cap() < smallStringCapacity {
		t.Errorf("小字符串构建器容量不足: %d < %d", smallBuilder.Cap(), smallStringCapacity)
	}
	if mediumBuilder.Cap() < mediumStringCapacity {
		t.Errorf("中字符串构建器容量不足: %d < %d", mediumBuilder.Cap(), mediumStringCapacity)
	}
	if largeBuilder.Cap() < largeStringCapacity {
		t.Errorf("大字符串构建器容量不足: %d < %d", largeBuilder.Cap(), largeStringCapacity)
	}

	// 放回全局池
	PutBuffer(smallBuffer)
	PutBuffer(mediumBuffer)
	PutBuffer(largeBuffer)
	PutStringBuilder(smallBuilder)
	PutStringBuilder(mediumBuilder)
	PutStringBuilder(largeBuilder)
}

// TestTieredPools_MemoryManagement 测试内存管理
func TestTieredPools_MemoryManagement(t *testing.T) {
	tp := NewTieredPools()

	// 测试大buffer丢弃逻辑
	t.Run("过大buffer丢弃", func(t *testing.T) {
		buffer := tp.GetBuffer(1024)

		// 人为扩大buffer容量（模拟过度使用）
		largeData := make([]byte, 100*1024) // 100KB
		buffer.Write(largeData)

		initialStats := tp.GetStats()
		tp.PutBuffer(buffer)
		finalStats := tp.GetStats()

		// 应该增加丢弃计数
		expectedDiscards := initialStats[SmallBuffer].Discards + 1
		if finalStats[SmallBuffer].Discards != expectedDiscards {
			t.Errorf("应该丢弃过大的buffer: 期望丢弃数=%d, 实际=%d",
				expectedDiscards, finalStats[SmallBuffer].Discards)
		}
	})

	// 测试字符串构建器容量控制
	t.Run("字符串构建器容量控制", func(t *testing.T) {
		builder := tp.GetStringBuilder(256)

		// 人为扩大容量
		largeString := strings.Repeat("x", 10000)
		builder.WriteString(largeString)

		initialStats := tp.GetStats()
		tp.PutStringBuilder(builder, 256)
		finalStats := tp.GetStats()

		// 可能会丢弃过大的构建器
		if finalStats[SmallBuffer].Discards > initialStats[SmallBuffer].Discards {
			t.Logf("正确丢弃了过大的字符串构建器")
		}
	})
}

// BenchmarkTieredPools_BufferOperations 基准测试buffer操作
func BenchmarkTieredPools_BufferOperations(b *testing.B) {
	tp := NewTieredPools()

	b.Run("SmallBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := tp.GetBuffer(512)
			buffer.WriteString("Hello World")
			tp.PutBuffer(buffer)
		}
	})

	b.Run("MediumBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := tp.GetBuffer(4096)
			buffer.WriteString(strings.Repeat("data ", 100))
			tp.PutBuffer(buffer)
		}
	})

	b.Run("LargeBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := tp.GetBuffer(16384)
			buffer.WriteString(strings.Repeat("large data ", 500))
			tp.PutBuffer(buffer)
		}
	})
}

// BenchmarkTieredPools_StringBuilders 基准测试字符串构建器
func BenchmarkTieredPools_StringBuilders(b *testing.B) {
	tp := NewTieredPools()

	b.Run("SmallStringBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := tp.GetStringBuilder(256)
			builder.WriteString("Small content")
			tp.PutStringBuilder(builder, 256)
		}
	})

	b.Run("MediumStringBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := tp.GetStringBuilder(1024)
			for j := 0; j < 20; j++ {
				builder.WriteString("Medium content ")
			}
			tp.PutStringBuilder(builder, 1024)
		}
	})

	b.Run("LargeStringBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := tp.GetStringBuilder(4096)
			for j := 0; j < 100; j++ {
				builder.WriteString("Large content block ")
			}
			tp.PutStringBuilder(builder, 4096)
		}
	})
}

// BenchmarkTieredPools_vs_SyncPool 对比测试
func BenchmarkTieredPools_vs_SyncPool(b *testing.B) {
	// 分级池
	tp := NewTieredPools()

	// 传统sync.Pool
	var traditional sync.Pool
	traditional.New = func() interface{} {
		return &strings.Builder{}
	}

	b.Run("TieredPools", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := tp.GetStringBuilder(1024)
			builder.WriteString("test data")
			tp.PutStringBuilder(builder, 1024)
		}
	})

	b.Run("SyncPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := traditional.Get().(*strings.Builder)
			builder.Reset()
			builder.WriteString("test data")
			traditional.Put(builder)
		}
	})
}
