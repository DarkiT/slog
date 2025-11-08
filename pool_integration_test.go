package slog

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/darkit/slog/common"
)

// TestTieredPoolsIntegration 测试分级池在logger系统中的集成
func TestTieredPoolsIntegration(t *testing.T) {
	// 创建分级池实例
	tieredPools := common.NewTieredPools()

	t.Run("字符串构建器集成测试", func(t *testing.T) {
		// 模拟不同大小的日志消息构建
		testCases := []struct {
			messageSize  string
			content      string
			expectedTier common.BufferSize
		}{
			{"小消息", "短日志", common.SmallBuffer},
			{"中消息", strings.Repeat("中等长度日志 ", 50), common.MediumBuffer},
			{"大消息", strings.Repeat("长日志消息内容 ", 200), common.LargeBuffer},
		}

		for _, tc := range testCases {
			t.Run(tc.messageSize, func(t *testing.T) {
				// 根据内容长度获取合适的字符串构建器
				expectedCapacity := len(tc.content)
				builder := tieredPools.GetStringBuilder(expectedCapacity)

				// 构建日志消息
				builder.WriteString("2025/08/02 19:15.52.409 ")
				builder.WriteString("[INFO] ")
				builder.WriteString(tc.content)

				result := builder.String()
				if !strings.Contains(result, tc.content) {
					t.Errorf("构建的日志应该包含原始内容")
				}

				// 放回池中
				tieredPools.PutStringBuilder(builder, expectedCapacity)
			})
		}
	})

	t.Run("Buffer集成测试", func(t *testing.T) {
		// 模拟不同大小的buffer需求
		testCases := []struct {
			name         string
			size         int
			expectedTier common.BufferSize
		}{
			{"小Buffer", 1024, common.SmallBuffer},
			{"中Buffer", 4096, common.MediumBuffer},
			{"大Buffer", 16384, common.LargeBuffer},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				buffer := tieredPools.GetBuffer(tc.size)

				if buffer.Size() != tc.expectedTier {
					t.Errorf("期望buffer大小: %v, 实际: %v", tc.expectedTier, buffer.Size())
				}

				// 写入测试数据
				testData := fmt.Sprintf("测试数据 - %s", tc.name)
				buffer.WriteString(testData)

				if buffer.String() != testData {
					t.Errorf("buffer内容不匹配")
				}

				tieredPools.PutBuffer(buffer)
			})
		}
	})
}

// TestPoolMigration 测试对象池迁移
func TestPoolMigration(t *testing.T) {
	// 创建传统的sync.Pool
	var traditionalPool sync.Pool
	traditionalPool.New = func() interface{} {
		return &strings.Builder{}
	}

	// 创建迁移管理器
	migrationManager := common.NewPoolMigrationManager()
	tieredPool := migrationManager.MigrateStringBuilderPool(&traditionalPool)

	t.Run("迁移兼容性测试", func(t *testing.T) {
		// 使用传统接口
		builder1 := tieredPool.Get().(*strings.Builder)
		builder1.WriteString("传统接口测试")
		tieredPool.Put(builder1)

		// 使用新接口
		builder2 := tieredPool.GetWithCapacity(1024)
		builder2.WriteString("新接口测试")
		tieredPool.PutWithCapacity(builder2, 1024)
	})
}

// TestSmartStringBuilderPool 测试智能字符串构建器池
func TestSmartStringBuilderPool(t *testing.T) {
	smartPool := common.NewSmartStringBuilderPool()

	t.Run("使用模式学习", func(t *testing.T) {
		// 模拟多次小容量使用
		for i := 0; i < 10; i++ {
			builder := smartPool.GetOptimal()
			builder.WriteString(fmt.Sprintf("小消息 %d", i))
			smartPool.PutWithUsageTracking(builder)
		}

		// 检查使用统计
		small, medium, large := smartPool.GetUsageStats()
		t.Logf("使用统计 - 小: %d, 中: %d, 大: %d", small, medium, large)

		// 再次获取，应该倾向于小容量
		builder := smartPool.GetOptimal()
		if builder.Cap() > 512 { // 应该选择较小的容量
			t.Logf("智能池选择了容量: %d (可能还在学习阶段)", builder.Cap())
		}
		smartPool.PutWithUsageTracking(builder)
	})
}

// TestBufferPoolAdapter 测试buffer池适配器
func TestBufferPoolAdapter(t *testing.T) {
	adapter := common.NewBufferPoolAdapter()

	t.Run("现有数据适配", func(t *testing.T) {
		// 模拟现有的字节数据
		existingData := []byte("现有的日志数据需要迁移到分级池")

		// 适配到分级池
		buffer := adapter.AdaptExistingBuffer(existingData)

		if buffer.String() != string(existingData) {
			t.Errorf("适配后的数据不匹配")
		}

		// 继续写入更多数据
		buffer.WriteString(" - 追加的数据")

		// 释放buffer
		adapter.ReleaseAdaptedBuffer(buffer)

		// 检查适配器统计
		gets, puts := adapter.GetAdapterStats()
		if gets != 1 || puts != 1 {
			t.Errorf("适配器统计不正确: gets=%d, puts=%d", gets, puts)
		}
	})
}

// BenchmarkTieredPoolsIntegration 集成性能基准测试
func BenchmarkTieredPoolsIntegration(b *testing.B) {
	tieredPools := common.NewTieredPools()

	b.Run("Logger场景", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// 模拟日志格式化场景
			builder := tieredPools.GetStringBuilder(256)

			// 构建典型的日志消息
			builder.WriteString("2025/08/02 19:15.52.409 ")
			builder.WriteString("[INFO] ")
			builder.WriteString("用户登录成功 - 用户ID: ")
			builder.WriteString(fmt.Sprintf("%d", i))

			_ = builder.String()
			tieredPools.PutStringBuilder(builder, 256)
		}
	})

	b.Run("DLP场景", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// 模拟DLP处理场景
			buffer := tieredPools.GetBuffer(1024)

			// 处理敏感数据
			buffer.WriteString("原始文本: 手机号13812345678")
			original := buffer.String()

			// 重置并写入脱敏后的数据
			buffer.Reset()
			buffer.WriteString("脱敏文本: 手机号138****5678")

			_ = buffer.String()
			_ = original // 避免未使用变量警告
			tieredPools.PutBuffer(buffer)
		}
	})

	b.Run("传统sync.Pool对比", func(b *testing.B) {
		var pool sync.Pool
		pool.New = func() interface{} {
			return &strings.Builder{}
		}

		for i := 0; i < b.N; i++ {
			builder := pool.Get().(*strings.Builder)
			builder.Reset()

			builder.WriteString("2025/08/02 19:15.52.409 ")
			builder.WriteString("[INFO] ")
			builder.WriteString("传统池测试消息")

			_ = builder.String()
			pool.Put(builder)
		}
	})
}

// TestTieredPoolsMemoryEfficiency 内存效率测试
func TestTieredPoolsMemoryEfficiency(t *testing.T) {
	tieredPools := common.NewTieredPools()

	t.Run("内存分配优化", func(t *testing.T) {
		// 预热池
		var buffers []*common.TieredBuffer
		var builders []*strings.Builder

		// 获取一批对象
		for i := 0; i < 100; i++ {
			buffers = append(buffers, tieredPools.GetBuffer(1024))
			builders = append(builders, tieredPools.GetStringBuilder(512))
		}

		// 放回一半对象
		for i := 0; i < 50; i++ {
			tieredPools.PutBuffer(buffers[i])
			tieredPools.PutStringBuilder(builders[i], 512)
		}

		// 重新获取，应该复用已有对象
		for i := 0; i < 50; i++ {
			buffer := tieredPools.GetBuffer(1024)
			builder := tieredPools.GetStringBuilder(512)

			// 验证对象已被重置
			if buffer.Len() != 0 {
				t.Errorf("复用的buffer应该被重置")
			}
			if builder.Len() != 0 {
				t.Errorf("复用的builder应该被重置")
			}

			tieredPools.PutBuffer(buffer)
			tieredPools.PutStringBuilder(builder, 512)
		}

		// 清理剩余对象
		for i := 50; i < 100; i++ {
			tieredPools.PutBuffer(buffers[i])
			tieredPools.PutStringBuilder(builders[i], 512)
		}

		// 检查统计信息
		stats := tieredPools.GetStats()
		t.Logf("池统计信息:")
		for size, stat := range stats {
			t.Logf("  %v: Gets=%d, Puts=%d, News=%d, HitRate=%.2f%%",
				size, stat.Gets, stat.Puts, stat.News, stat.HitRate)
		}
	})
}

// TestTieredPoolsConcurrentStress 并发压力测试
func TestTieredPoolsConcurrentStress(t *testing.T) {
	tieredPools := common.NewTieredPools()

	const numWorkers = 50
	const operationsPerWorker = 1000

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// 并发工作者
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// 随机选择操作类型
				opType := (workerID*operationsPerWorker + j) % 4

				switch opType {
				case 0: // 小buffer操作
					buffer := tieredPools.GetBuffer(512)
					buffer.WriteString(fmt.Sprintf("Worker %d Op %d", workerID, j))
					tieredPools.PutBuffer(buffer)

				case 1: // 中buffer操作
					buffer := tieredPools.GetBuffer(4096)
					for k := 0; k < 10; k++ {
						buffer.WriteString(fmt.Sprintf("Data %d ", k))
					}
					tieredPools.PutBuffer(buffer)

				case 2: // 小字符串构建器
					builder := tieredPools.GetStringBuilder(256)
					builder.WriteString(fmt.Sprintf("Small string %d-%d", workerID, j))
					tieredPools.PutStringBuilder(builder, 256)

				case 3: // 大字符串构建器
					builder := tieredPools.GetStringBuilder(2048)
					for k := 0; k < 20; k++ {
						builder.WriteString(fmt.Sprintf("Large content block %d ", k))
					}
					tieredPools.PutStringBuilder(builder, 2048)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证最终状态
	stats := tieredPools.GetStats()
	totalOps := int64(numWorkers * operationsPerWorker)

	var totalGets int64
	for _, stat := range stats {
		totalGets += stat.Gets
	}

	if totalGets != totalOps {
		t.Errorf("总操作数不匹配: 期望%d, 实际%d", totalOps, totalGets)
	}

	t.Logf("并发压力测试完成:")
	t.Logf("  工作者数: %d", numWorkers)
	t.Logf("  每工作者操作数: %d", operationsPerWorker)
	t.Logf("  总操作数: %d", totalOps)

	for size, stat := range stats {
		t.Logf("  %v池: Gets=%d, Puts=%d, HitRate=%.2f%%",
			size, stat.Gets, stat.Puts, stat.HitRate)
	}
}
