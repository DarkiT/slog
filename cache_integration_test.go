package slog

import (
	"testing"
)

// TestLRUFormatCacheIntegration 测试LRU格式缓存集成
func TestLRUFormatCacheIntegration(t *testing.T) {
	// 清理缓存以开始测试
	formatCache.Clear()

	// 初始应该为空
	if formatCache.Size() != 0 {
		t.Errorf("初始缓存大小应该为0，实际: %d", formatCache.Size())
	}

	// 测试多个不同的格式字符串
	testCases := []struct {
		msg      string
		expected bool
	}{
		{"hello world", false},               // 无格式说明符
		{"hello %s world", true},             // 有格式说明符
		{"value: %d", true},                  // 数字格式
		{"name: %s, age: %d", true},          // 多个格式说明符
		{"plain text without format", false}, // 纯文本
		{"percent sign %% only", false},      // 转义的百分号
	}

	for _, tc := range testCases {
		result := formatLog(tc.msg, "test")
		if result != tc.expected {
			t.Errorf("格式检测失败: msg=%q, expected=%v, got=%v", tc.msg, tc.expected, result)
		}
	}

	// 验证缓存已被填充
	if formatCache.Size() == 0 {
		t.Error("缓存应该被填充")
	}

	expectedSize := len(testCases)
	if formatCache.Size() != expectedSize {
		t.Errorf("缓存大小不匹配: expected=%d, got=%d", expectedSize, formatCache.Size())
	}

	// 测试缓存命中 - 重复调用同一个字符串
	testMsg := "test %s format"
	formatLog(testMsg, "arg")

	// 验证缓存键的存在
	keys := formatCache.GetStringKeys()
	found := false
	for _, key := range keys {
		if key == testMsg {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("缓存中应该包含键: %q", testMsg)
	}

	// 测试缓存容量限制
	originalCapacity := formatCache.Capacity()
	if originalCapacity != int(maxFormatCacheSize) {
		t.Errorf("缓存容量不匹配: expected=%d, got=%d", maxFormatCacheSize, originalCapacity)
	}

	// 清理缓存测试
	formatCache.Clear()
	if formatCache.Size() != 0 {
		t.Errorf("清理后缓存大小应该为0，实际: %d", formatCache.Size())
	}
}

// TestLRUFormatCacheStats 测试LRU格式缓存统计
func TestLRUFormatCacheStats(t *testing.T) {
	formatCache.Clear()

	// 添加一些测试数据
	testMessages := []string{
		"hello %s",
		"world %d",
		"test %v",
	}

	for _, msg := range testMessages {
		formatLog(msg, "test")
	}

	stats := formatCache.GetStats()

	if stats.Size != len(testMessages) {
		t.Errorf("统计大小不匹配: expected=%d, got=%d", len(testMessages), stats.Size)
	}

	if stats.Capacity != int(maxFormatCacheSize) {
		t.Errorf("统计容量不匹配: expected=%d, got=%d", maxFormatCacheSize, stats.Capacity)
	}

	// 测试命中统计 - 重复访问同一个key
	formatLog("hello %s", "test")

	newStats := formatCache.GetStats()
	if newStats.Hits <= stats.Hits {
		t.Error("重复访问应该增加命中统计")
	}
}
