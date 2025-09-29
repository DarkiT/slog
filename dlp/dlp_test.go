package dlp

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewRegexSearcher(t *testing.T) {
	searcher := NewRegexSearcher()
	if searcher == nil {
		t.Error("NewRegexSearcher should return a non-nil searcher")
	}
	// 处理手机号
	text := "联系方式：13812345678"
	result := searcher.ReplaceParallel(text, MobilePhone)
	// 输出: 联系方式：138****5678
	t.Log(result)
	// 处理多种类型
	text = "邮箱：test@example.com，手机：13812345678"
	matches := searcher.SearchSensitiveByType(text, Email)
	for _, match := range matches {
		t.Logf("找到邮箱：%s", match.Content)
	}
	// 验证默认匹配器是否正确注册
	types := searcher.GetAllSupportedTypes()

	expectedTypes := []string{
		"chinese_name",
		"mobile_phone",
		"email",
		"id_card",
		"bank_card",
		"address",
		"url",
		"password",
		"ipv4",
		"ipv6",
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, actualType := range types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected matcher type %s not found", expectedType)
		}
	}
}

func TestRegisterMatcher(t *testing.T) {
	searcher := NewRegexSearcher()

	// 测试注册新的匹配器
	newMatcher := &Matcher{
		Name:     "test_matcher",
		Pattern:  "test\\d+",
		Priority: 50,
		Validator: func(s string) bool {
			return len(s) > 4
		},
		Transformer: func(s string) string {
			return "***" + s[len(s)-2:]
		},
	}

	regex, err := regexp.Compile(newMatcher.Pattern)
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}
	newMatcher.Regex = regex

	searcher.AddMatcher(newMatcher)

	// 验证是否成功注册
	results := searcher.SearchSensitiveByType("test123", "test_matcher")
	if len(results) != 1 {
		t.Error("Expected 1 match for test_matcher")
	}
}

func TestSearchSensitiveByType(t *testing.T) {
	searcher := NewRegexSearcher()

	tests := []struct {
		name      string
		text      string
		matchType string
		expected  int
	}{
		{
			name:      "Mobile Phone",
			text:      "手机号码：13812345678 和 13987654321",
			matchType: "mobile_phone",
			expected:  2,
		},
		{
			name:      "Email",
			text:      "邮箱：test@example.com, another@test.com",
			matchType: "email",
			expected:  2,
		},
		{
			name:      "Chinese Name",
			text:      "姓名：张三李四",
			matchType: "chinese_name",
			expected:  2,
		},
		{
			name:      "ID Card",
			text:      "身份证：622421196903065015",
			matchType: "id_card",
			expected:  1,
		},
		{
			name:      "Non-existent Type",
			text:      "Some text",
			matchType: "non_existent",
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searcher.SearchSensitiveByType(tt.text, tt.matchType)

			if tt.name == "Chinese Name" {
				t.Logf("Testing text: %s", tt.text)
				for i, r := range results {
					t.Logf("Match %d: Type=%s, Content=%s, Position=%v", i, r.Type, r.Content, r.Position)
				}
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d matches, got %d for type %s",
					tt.expected, len(results), tt.matchType)
			}
		})
	}
}

func TestReplaceParallel(t *testing.T) {
	searcher := NewRegexSearcher()

	tests := []struct {
		name      string
		text      string
		matchType string
		expected  string
		checkFunc func(string) bool
	}{
		{
			name:      "Mobile Phone Replacement",
			text:      "联系方式：13812345678",
			matchType: "mobile_phone",
			checkFunc: func(result string) bool {
				return strings.Contains(result, "****") &&
					!strings.Contains(result, "13812345678")
			},
		},
		{
			name:      "Email Replacement",
			text:      "邮箱：test@example.com",
			matchType: "email",
			checkFunc: func(result string) bool {
				return strings.Contains(result, "**") &&
					strings.Contains(result, "@example.com")
			},
		},
		{
			name:      "Multiple Mobile Phones",
			text:      "手机号码：13812345678，13987654321",
			matchType: "mobile_phone",
			checkFunc: func(result string) bool {
				return strings.Count(result, "****") == 2
			},
		},
		{
			name:      "Long Text Parallel Processing",
			text:      generateLongText(),
			matchType: "mobile_phone",
			checkFunc: func(result string) bool {
				return !strings.Contains(result, "13812345678")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := searcher.ReplaceParallel(tt.text, tt.matchType)
			if !tt.checkFunc(result) {
				t.Errorf("Replacement failed for test %s", tt.name)
			}
		})
	}
}

func TestValidateChineseIDCard(t *testing.T) {
	tests := []struct {
		name     string
		idCard   string
		expected bool
	}{
		{
			name:     "Valid ID Card",
			idCard:   "440101199001011234", // 示例ID，实际使用时需要真实的校验码
			expected: false,                // 因为示例ID不是真实的
		},
		{
			name:     "Invalid Length",
			idCard:   "4401011990010",
			expected: false,
		},
		{
			name:     "Invalid Date",
			idCard:   "440101199013011234",
			expected: false,
		},
		{
			name:     "Invalid Year",
			idCard:   "440101180001011234",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChineseIDCardDesensitize(tt.idCard)
			if result != tt.expected {
				t.Errorf("Expected %v for ID card %s, got %v",
					tt.expected, tt.idCard, result)
			}
		})
	}
}

func TestGetAllSupportedTypes(t *testing.T) {
	searcher := NewRegexSearcher()
	types := searcher.GetAllSupportedTypes()

	// 验证返回的类型列表
	if len(types) == 0 {
		t.Error("Expected non-empty type list")
	}

	// 验证是否包含必要的类型
	requiredTypes := map[string]bool{
		"mobile_phone": false,
		"email":        false,
		"id_card":      false,
	}

	for _, t := range types {
		if _, exists := requiredTypes[t]; exists {
			requiredTypes[t] = true
		}
	}

	for typ, found := range requiredTypes {
		if !found {
			t.Errorf("Required type %s not found in supported types", typ)
		}
	}
}

func TestMatcherValidation(t *testing.T) {
	searcher := NewRegexSearcher()

	tests := []struct {
		name        string
		text        string
		matchType   string
		shouldMatch bool
	}{
		{
			name:        "Valid Mobile",
			text:        "13812345678",
			matchType:   "mobile_phone",
			shouldMatch: true,
		},
		{
			name:        "Invalid Mobile",
			text:        "1381234567", // 少一位
			matchType:   "mobile_phone",
			shouldMatch: false,
		},
		{
			name:        "Valid Email",
			text:        "test@example.com",
			matchType:   "email",
			shouldMatch: true,
		},
		{
			name:        "Invalid Email",
			text:        "test@", // 无效邮箱
			matchType:   "email",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searcher.SearchSensitiveByType(tt.text, tt.matchType)
			hasMatch := len(results) > 0
			if hasMatch != tt.shouldMatch {
				t.Errorf("Expected match=%v for %s, got %v",
					tt.shouldMatch, tt.text, hasMatch)
			}
		})
	}
}

// 性能测试
func BenchmarkReplaceParallel(b *testing.B) {
	searcher := NewRegexSearcher()
	text := generateLongText()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.ReplaceParallel(text, "mobile_phone")
	}
}

// generateLongText 生成包含大量手机号的长文本用于测试
func generateLongText() string {
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString(fmt.Sprintf("手机号码%d：138%08d\n", i, i))
	}
	return builder.String()
}

// TestMatcherPriority 测试匹配器优先级
func TestMatcherPriority(t *testing.T) {
	searcher := NewRegexSearcher()

	// 注册两个可能冲突的匹配器
	highPriorityMatcher := &Matcher{
		Name:     "high_priority",
		Pattern:  "\\d{11}",
		Priority: 100,
		Transformer: func(s string) string {
			return "HIGH_PRIORITY"
		},
	}

	lowPriorityMatcher := &Matcher{
		Name:     "low_priority",
		Pattern:  "\\d{11}",
		Priority: 50,
		Transformer: func(s string) string {
			return "LOW_PRIORITY"
		},
	}

	regex, _ := regexp.Compile(highPriorityMatcher.Pattern)
	highPriorityMatcher.Regex = regex
	lowPriorityMatcher.Regex = regex

	searcher.AddMatcher(highPriorityMatcher)
	searcher.AddMatcher(lowPriorityMatcher)

	// 测试优先级处理
	text := "13812345678"
	result := searcher.ReplaceParallel(text, "high_priority")
	if result != "HIGH_PRIORITY" {
		t.Error("High priority matcher should be applied")
	}
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	searcher := NewRegexSearcher()
	text := "测试文本 13812345678 test@example.com"
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			searcher.SearchSensitiveByType(text, "mobile_phone")
			searcher.SearchSensitiveByType(text, "email")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	searcher := NewRegexSearcher()

	tests := []struct {
		name      string
		text      string
		matchType string
	}{
		{
			name:      "Empty Text",
			text:      "",
			matchType: "mobile_phone",
		},
		{
			name:      "Very Long Text",
			text:      strings.Repeat("a", 1000000),
			matchType: "email",
		},
		{
			name:      "Special Characters",
			text:      "!@#$%^&*()",
			matchType: "chinese_name",
		},
		{
			name:      "Unicode Characters",
			text:      "测试😊👍",
			matchType: "address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保不会panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic occurred: %v", r)
				}
			}()

			searcher.ReplaceParallel(tt.text, tt.matchType)
		})
	}
}

// 基准测试新的优化版本
func BenchmarkEngine_DesensitizeText(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()

	testText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.DesensitizeText(testText)
	}
}

func BenchmarkEngine_DesensitizeTextLarge(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()

	// 构造大文本
	baseText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1。"
	largeText := ""
	for i := 0; i < 100; i++ {
		largeText += baseText
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.DesensitizeText(largeText)
	}
}

func BenchmarkReplaceAllTypes(b *testing.B) {
	searcher := NewRegexSearcher()
	testText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.ReplaceAllTypes(testText)
	}
}

func BenchmarkDetectAllTypes(b *testing.B) {
	searcher := NewRegexSearcher()
	testText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.DetectAllTypes(testText)
	}
}

// 对比旧方法和新方法
func BenchmarkOldMethod(b *testing.B) {
	searcher := NewRegexSearcher()
	testText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := testText
		types := searcher.GetAllSupportedTypes()
		for _, typeName := range types {
			result = searcher.ReplaceParallel(result, typeName)
		}
	}
}

// 性能对比测试
func TestPerformanceComparison(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	testText := "张三的手机号是13812345678，身份证号是110101199001011237，邮箱是zhangsan@example.com，IP地址是192.168.1.1"

	// 第一次运行（无缓存）
	start := time.Now()
	result1 := engine.DesensitizeText(testText)
	firstRun := time.Since(start)

	// 第二次运行（有缓存）
	start = time.Now()
	result2 := engine.DesensitizeText(testText)
	secondRun := time.Since(start)

	// 验证结果一致性
	if result1 != result2 {
		t.Errorf("Results differ: %s vs %s", result1, result2)
	}

	// 缓存统计
	hits, misses := engine.GetCacheStats()

	t.Logf("第一次运行时间: %v", firstRun)
	t.Logf("第二次运行时间: %v", secondRun)
	t.Logf("性能提升: %.2fx", float64(firstRun)/float64(secondRun))
	t.Logf("缓存统计 - 命中: %d, 未命中: %d", hits, misses)
	t.Logf("脱敏结果: %s", result1)
}

// 内存使用和大批量处理测试
func TestMemoryAndBatchProcessing(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 测试大批量相同文本处理（验证缓存效果）
	testText := "张三的手机号是13812345678，身份证号是110101199001011237"
	batchSize := 10000

	start := time.Now()
	for i := 0; i < batchSize; i++ {
		engine.DesensitizeText(testText)
	}
	duration := time.Since(start)

	hits, misses := engine.GetCacheStats()

	t.Logf("批量处理 %d 次相同文本:", batchSize)
	t.Logf("总用时: %v", duration)
	t.Logf("平均每次: %v", duration/time.Duration(batchSize))
	t.Logf("缓存命中率: %.2f%%", float64(hits)/float64(hits+misses)*100)

	// 测试不同文本的处理
	engine.ClearCache()

	start = time.Now()
	for i := 0; i < 1000; i++ {
		text := fmt.Sprintf("用户%d的手机号是1381234%04d", i, i)
		engine.DesensitizeText(text)
	}
	duration = time.Since(start)

	hits, misses = engine.GetCacheStats()

	t.Logf("\n批量处理 1000 次不同文本:")
	t.Logf("总用时: %v", duration)
	t.Logf("平均每次: %v", duration/1000)
	t.Logf("缓存命中率: %.2f%%", float64(hits)/float64(hits+misses)*100)
}

// 测试资源占用优化
func TestResourceOptimization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 测试各种长度的文本
	texts := []string{
		"短文本",
		"中等长度的文本包含手机号13812345678",
		strings.Repeat("这是一个较长的文本，包含敏感信息手机号13812345678，", 10),
		strings.Repeat("超长文本重复内容", 1000), // 超长文本不会被缓存
	}

	for i, text := range texts {
		start := time.Now()
		result := engine.DesensitizeText(text)
		duration := time.Since(start)

		t.Logf("文本 %d (长度 %d): %v", i+1, len(text), duration)
		if len(result) < 100 {
			t.Logf("  结果: %s", result)
		} else {
			t.Logf("  结果: %s...(truncated)", result[:100])
		}
	}

	hits, misses := engine.GetCacheStats()
	t.Logf("\n总缓存统计 - 命中: %d, 未命中: %d", hits, misses)
}
