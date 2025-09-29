package dlp

import (
	"strings"
	"testing"
	"time"
)

func TestDlpEngine_PluginArchitectureIntegration(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 测试默认传统模式
	if engine.IsPluginArchitectureEnabled() {
		t.Error("Plugin architecture should be disabled by default")
	}

	// 传统模式脱敏测试
	result := engine.DesensitizeText("13812345678")
	if !strings.Contains(result, "*") {
		t.Error("Traditional mode should desensitize phone numbers")
	}

	// 启用插件架构
	engine.EnablePluginArchitecture()
	if !engine.IsPluginArchitectureEnabled() {
		t.Error("Plugin architecture should be enabled")
	}

	// 插件模式脱敏测试
	pluginResult := engine.DesensitizeText("13812345678")
	if pluginResult != "138****5678" {
		t.Errorf("Plugin mode result = %s, want 138****5678", pluginResult)
	}

	// 测试混合文本脱敏
	mixedText := "我是张三，手机号13812345678，邮箱test@example.com"
	mixedResult := engine.DesensitizeText(mixedText)

	// 验证所有敏感信息都被脱敏
	// 注意：中文姓名脱敏器已从默认配置中移除，因为它会误判普通文本
	expectedPatterns := []string{"138****5678", "t**@example.com"}
	for _, pattern := range expectedPatterns {
		if !strings.Contains(mixedResult, pattern) {
			t.Errorf("Mixed text result should contain %s, got: %s", pattern, mixedResult)
		}
	}

	// 禁用插件架构，应回退到传统模式
	engine.DisablePluginArchitecture()
	if engine.IsPluginArchitectureEnabled() {
		t.Error("Plugin architecture should be disabled")
	}
}

func TestDlpEngine_SpecificTypeProcessing(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	tests := []struct {
		dataType string
		input    string
		expected string
	}{
		{"phone", "13812345678", "138****5678"},
		{"email", "test@example.com", "t**@example.com"},
		{"id_card", "123456789012345678", "123456789012345678"}, // 不支持的类型返回原文
		{"bank_card", "1234567890123456", "1234567890123456"},   // 不支持的类型返回原文
		{"ip_address", "192.168.1.1", "192.168.1.1"},            // 不支持的类型返回原文
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			result := engine.DesensitizeSpecificType(tt.input, tt.dataType)
			if result != tt.expected {
				t.Errorf("DesensitizeSpecificType(%s, %s) = %s, want %s",
					tt.input, tt.dataType, result, tt.expected)
			}
		})
	}
}

func TestDlpEngine_CustomDesensitizerRegistration(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 创建自定义脱敏器
	customDesensitizer := NewRegexDesensitizer("custom_qq")
	err := customDesensitizer.AddPattern("qq", `\d{5,11}`, "****")
	if err != nil {
		t.Errorf("AddPattern() error = %v", err)
	}

	// 注册自定义脱敏器
	err = engine.RegisterCustomDesensitizer(customDesensitizer)
	if err != nil {
		t.Errorf("RegisterCustomDesensitizer() error = %v", err)
	}

	// 验证注册成功
	desensitizers := engine.ListRegisteredDesensitizers()
	found := false
	for _, name := range desensitizers {
		if name == "custom_qq" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom desensitizer should be registered")
	}

	// 测试自定义脱敏器功能
	result := engine.DesensitizeSpecificType("我的QQ是12345678", "qq")
	if !strings.Contains(result, "****") {
		t.Errorf("Custom desensitizer should work, got: %s", result)
	}

	// 注销自定义脱敏器
	err = engine.UnregisterDesensitizer("custom_qq")
	if err != nil {
		t.Errorf("UnregisterDesensitizer() error = %v", err)
	}

	// 验证注销成功
	desensitizers = engine.ListRegisteredDesensitizers()
	for _, name := range desensitizers {
		if name == "custom_qq" {
			t.Error("Custom desensitizer should be unregistered")
		}
	}
}

func TestDlpEngine_DesensitizerManagement(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 测试禁用特定脱敏器
	err := engine.DisableDesensitizer("phone")
	if err != nil {
		t.Errorf("DisableDesensitizer() error = %v", err)
	}

	// 禁用后应该不脱敏
	result := engine.DesensitizeSpecificType("13812345678", "phone")
	if result != "13812345678" {
		t.Errorf("Disabled desensitizer should return original, got: %s", result)
	}

	// 测试重新启用
	err = engine.EnableDesensitizer("phone")
	if err != nil {
		t.Errorf("EnableDesensitizer() error = %v", err)
	}

	// 启用后应该正常脱敏
	result = engine.DesensitizeSpecificType("13812345678", "phone")
	if result != "138****5678" {
		t.Errorf("Enabled desensitizer should desensitize, got: %s", result)
	}
}

func TestDlpEngine_Statistics(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 执行一些脱敏操作
	inputs := []string{"13812345678", "test@example.com", "张三"}
	for _, input := range inputs {
		engine.DesensitizeText(input)
	}

	// 获取统计信息
	stats := engine.GetDesensitizerStats()
	if stats.TotalDesensitizers == 0 {
		t.Error("Should have registered desensitizers")
	}

	// 获取支持的类型映射
	typeMapping := engine.GetSupportedTypesWithPlugin()
	if typeMapping == nil {
		t.Error("Plugin architecture should provide type mapping")
	}

	// 验证关键类型存在
	// 注意：chinese_name 已从默认配置中移除
	expectedTypes := []string{"phone", "email", "id_card", "bank_card"}
	for _, dataType := range expectedTypes {
		if _, exists := typeMapping[dataType]; !exists {
			t.Errorf("Type mapping should include %s", dataType)
		}
	}
}

func TestDlpEngine_CacheIntegration(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	input := "13812345678"

	// 第一次处理
	start1 := time.Now()
	result1 := engine.DesensitizeText(input)
	duration1 := time.Since(start1)

	// 第二次处理（应使用缓存）
	start2 := time.Now()
	result2 := engine.DesensitizeText(input)
	duration2 := time.Since(start2)

	if result1 != result2 {
		t.Error("Cache results should be identical")
	}

	// 第二次应该更快（缓存效果）
	if duration2 > duration1 {
		t.Log("Second call might not be significantly faster, but results are correct")
	}

	// 测试缓存统计
	hits, misses := engine.GetCacheStats()
	if hits == 0 && misses == 0 {
		t.Error("Cache stats should be updated")
	}

	// 清除缓存
	engine.ClearDesensitizerCaches()

	// 验证缓存被清除
	hitsAfter, missesAfter := engine.GetCacheStats()
	if hitsAfter != 0 || missesAfter != 0 {
		t.Error("Cache should be cleared")
	}
}

func TestDlpEngine_BackwardCompatibility(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 测试传统模式的结构体脱敏
	type TestStruct struct {
		Phone string `dlp:"phone"`
		Email string `dlp:"email"`
		Name  string
	}

	data := &TestStruct{
		Phone: "13812345678",
		Email: "test@example.com",
		Name:  "张三",
	}

	// 传统模式脱敏
	err := engine.DesensitizeStruct(data)
	if err != nil {
		t.Errorf("DesensitizeStruct() error = %v", err)
	}

	// 验证脱敏效果（传统模式可能有不同的脱敏结果）
	if data.Phone == "13812345678" && data.Email == "test@example.com" {
		t.Error("Traditional mode should desensitize tagged fields")
	}

	// 测试检测功能
	detectionResult := engine.DetectSensitiveInfo("包含手机号13812345678和邮箱test@example.com的文本")
	if detectionResult == nil || len(detectionResult) == 0 {
		t.Error("Should detect sensitive information")
	}
}

func TestDlpEngine_LongTextHandling(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 创建超长文本（超过5000字符）
	longText := strings.Repeat("这是一段很长的文本，", 200) + "包含手机号13812345678和邮箱test@example.com"

	// 长文本不应使用缓存，但应正常处理
	result := engine.DesensitizeText(longText)

	// 验证脱敏效果
	if !strings.Contains(result, "138****5678") {
		t.Error("Long text should be desensitized")
	}
	if !strings.Contains(result, "t**@example.com") {
		t.Error("Long text should desensitize email")
	}

	// 测试特定类型长文本处理
	longPhoneText := strings.Repeat("文本", 1000) + "13812345678"
	phoneResult := engine.DesensitizeSpecificType(longPhoneText, "phone")
	if !strings.Contains(phoneResult, "138****5678") {
		t.Error("Long text specific type processing should work")
	}
}

func TestDlpEngine_ErrorRecovery(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 测试禁用管理器时的降级处理
	manager := engine.GetDesensitizerManager()
	manager.Disable()

	// 应该降级到传统模式
	result := engine.DesensitizeText("13812345678")
	if result == "13812345678" {
		t.Error("Should fallback to traditional mode when plugin architecture fails")
	}

	// 重新启用管理器
	manager.Enable()

	// 应该恢复正常功能
	result = engine.DesensitizeText("13812345678")
	if result != "138****5678" {
		t.Error("Should work normally after re-enabling manager")
	}
}

func TestDlpEngine_MultithreadSafety(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 并发测试
	numRoutines := 50
	results := make(chan string, numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			input := "13812345678"
			result := engine.DesensitizeText(input)
			results <- result
		}(i)
	}

	// 验证所有结果
	for i := 0; i < numRoutines; i++ {
		result := <-results
		if result != "138****5678" {
			t.Errorf("Concurrent result = %s, want 138****5678", result)
		}
	}

	// 验证统计信息的一致性
	stats := engine.GetDesensitizerStats()
	if stats.TotalDesensitizers == 0 {
		t.Error("Stats should be maintained during concurrent access")
	}
}

// 性能基准测试

func BenchmarkDlpEngine_TraditionalMode(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()
	// 保持传统模式

	input := "我是张三，手机号13812345678，邮箱test@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.DesensitizeText(input)
	}
}

func BenchmarkDlpEngine_PluginMode(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	input := "我是张三，手机号13812345678，邮箱test@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.DesensitizeText(input)
	}
}

func BenchmarkDlpEngine_SpecificType(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	input := "13812345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.DesensitizeSpecificType(input, "phone")
	}
}

func BenchmarkDlpEngine_WithCache(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	input := "13812345678"

	// 预热缓存
	_ = engine.DesensitizeText(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.DesensitizeText(input)
	}
}

func BenchmarkDlpEngine_LongText(b *testing.B) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 超长文本（不使用缓存）
	longInput := strings.Repeat("这是一段很长的文本，", 200) + "13812345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.DesensitizeText(longInput)
	}
}

func BenchmarkDlpEngine_ModeComparison(b *testing.B) {
	input := "我是张三，手机号13812345678，邮箱test@example.com"

	b.Run("Traditional", func(b *testing.B) {
		engine := NewDlpEngine()
		engine.Enable()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.DesensitizeText(input)
		}
	})

	b.Run("Plugin", func(b *testing.B) {
		engine := NewDlpEngine()
		engine.Enable()
		engine.EnablePluginArchitecture()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.DesensitizeText(input)
		}
	})
}
