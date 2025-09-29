package dlp

import (
	"strings"
	"testing"
)

func TestCacheKeyOptimizer_GenerateKey(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()

	tests := []struct {
		name     string
		prefix   string
		data     string
		expected bool // 是否应该使用哈希优化
	}{
		{"短数据", "phone", "13812345678", false},                         // 11字符，应该保持原样
		{"中等数据", "email", strings.Repeat("test@example.com", 3), true}, // 超过32字符，应该使用哈希
		{"长数据", "text", strings.Repeat("long text content", 10), true}, // 超过32字符，应该使用哈希
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := optimizer.GenerateKey(tt.prefix, tt.data)

			// 验证键包含前缀
			if !strings.HasPrefix(key, tt.prefix+":") {
				t.Errorf("生成的键应该包含前缀: %s", key)
			}

			// 验证哈希优化逻辑
			if tt.expected {
				// 长数据应该使用哈希
				if !strings.Contains(key, ":h") {
					t.Errorf("长数据应该使用哈希优化: %s", key)
				}
			} else {
				// 短数据应该保持原样
				if strings.Contains(key, ":h") {
					t.Errorf("短数据不应该使用哈希优化: %s", key)
				}
			}
		})
	}
}

func TestCacheKeyOptimizer_GenerateKeyWithContext(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()

	tests := []struct {
		desensitizer string
		dataType     string
		data         string
	}{
		{"phone", "mobile", "13812345678"},
		{"email", "email_address", "user@example.com"},
		{"id_card", "identity", "123456789012345678"},
	}

	for _, tt := range tests {
		t.Run(tt.desensitizer+"_"+tt.dataType, func(t *testing.T) {
			key1 := optimizer.GenerateKeyWithContext(tt.desensitizer, tt.dataType, tt.data)
			key2 := optimizer.GenerateKeyWithContext(tt.desensitizer, tt.dataType, tt.data)

			// 相同输入应该产生相同的键
			if key1 != key2 {
				t.Errorf("相同输入应该产生相同键: %s != %s", key1, key2)
			}

			// 不同的脱敏器应该产生不同的键
			key3 := optimizer.GenerateKeyWithContext("different", tt.dataType, tt.data)
			if key1 == key3 {
				t.Errorf("不同脱敏器应该产生不同键: %s == %s", key1, key3)
			}
		})
	}
}

func TestCacheKeyOptimizer_HashCollision(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()

	// 测试哈希碰撞概率
	keys := make(map[string]bool)
	collisions := 0

	// 生成大量不同的输入
	for i := 0; i < 10000; i++ {
		data := strings.Repeat("data", i%100) + string(rune(i))
		key := optimizer.GenerateHashKey(data)

		if keys[key] {
			collisions++
		} else {
			keys[key] = true
		}
	}

	// 碰撞率应该很低（小于1%）
	collisionRate := float64(collisions) / 10000.0
	if collisionRate > 0.01 {
		t.Errorf("哈希碰撞率过高: %.4f%%", collisionRate*100)
	}

	t.Logf("哈希碰撞率: %.4f%%, 总计: %d 碰撞", collisionRate*100, collisions)
}

func TestCacheKeyOptimizer_Performance(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()
	longText := strings.Repeat("This is a long text for performance testing. ", 100)

	// 测试不同方法的性能特征
	methods := map[string]func(string) string{
		"GenerateHashKey": optimizer.GenerateHashKey,
		"GenerateFastKey": optimizer.GenerateFastKey,
	}

	for name, method := range methods {
		t.Run(name, func(t *testing.T) {
			// 预热
			for i := 0; i < 100; i++ {
				method(longText)
			}

			// 实际测试
			key1 := method(longText)
			key2 := method(longText)

			if key1 != key2 {
				t.Errorf("相同输入应该产生相同输出: %s != %s", key1, key2)
			}

			t.Logf("%s 生成的键: %s", name, key1)
		})
	}
}

func TestCacheKeyOptimizer_LayeredKey(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()

	tests := []struct {
		name   string
		layers []string
	}{
		{"单层", []string{"data"}},
		{"双层", []string{"type", "data"}},
		{"多层", []string{"desensitizer", "type", "subtype", "data"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := optimizer.GenerateLayeredKey(tt.layers...)

			if key == "" && len(tt.layers) > 0 {
				t.Error("非空输入不应该产生空键")
			}

			// 测试一致性
			key2 := optimizer.GenerateLayeredKey(tt.layers...)
			if key != key2 {
				t.Errorf("相同输入应该产生相同键: %s != %s", key, key2)
			}
		})
	}
}

func TestCacheKeyOptimizer_XXHashToggle(t *testing.T) {
	optimizer := NewCacheKeyOptimizer()
	longData := strings.Repeat("test data", 50)

	// 启用xxhash
	optimizer.SetXXHashEnabled(true)
	key1 := optimizer.GenerateKey("test", longData)

	// 禁用xxhash
	optimizer.SetXXHashEnabled(false)
	key2 := optimizer.GenerateKey("test", longData)

	// 两种模式应该产生不同的键
	if key1 == key2 {
		t.Error("启用和禁用xxhash应该产生不同的键")
	}

	// 验证状态
	if optimizer.IsXXHashEnabled() {
		t.Error("xxhash应该被禁用")
	}

	// 重新启用
	optimizer.SetXXHashEnabled(true)
	if !optimizer.IsXXHashEnabled() {
		t.Error("xxhash应该被启用")
	}
}

func BenchmarkCacheKeyOptimizer_ShortData(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := "13812345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateKey("phone", data)
	}
}

func BenchmarkCacheKeyOptimizer_LongData(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := strings.Repeat("This is a long text for benchmarking. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateKey("text", data)
	}
}

func BenchmarkCacheKeyOptimizer_WithContext(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := strings.Repeat("test@example.com", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateKeyWithContext("email", "email_address", data)
	}
}

func BenchmarkCacheKeyOptimizer_HashKey(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := strings.Repeat("benchmark data", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateHashKey(data)
	}
}

func BenchmarkCacheKeyOptimizer_FastKey(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := strings.Repeat("fast key benchmark", 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateFastKey(data)
	}
}

func BenchmarkCacheKeyOptimizer_LayeredKey(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	layers := []string{"desensitizer", "type", "subtype", strings.Repeat("data", 30)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateLayeredKey(layers...)
	}
}

// 对比基准测试：传统字符串拼接 vs xxhash优化
func BenchmarkCacheKey_Traditional(b *testing.B) {
	data := strings.Repeat("traditional cache key test", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = "prefix:" + data
	}
}

func BenchmarkCacheKey_XXHash(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	data := strings.Repeat("xxhash cache key test", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateKey("prefix", data)
	}
}

func BenchmarkCacheKey_XXHashDisabled(b *testing.B) {
	optimizer := NewCacheKeyOptimizer()
	optimizer.SetXXHashEnabled(false)
	data := strings.Repeat("xxhash disabled test", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.GenerateKey("prefix", data)
	}
}
