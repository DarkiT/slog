package dlp

import (
	"regexp"
	"strings"
	"testing"
)

// TestEnhancedDesensitizersBypassPrevention 测试增强脱敏器的绕过防护
func TestEnhancedDesensitizersBypassPrevention(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	testCases := []struct {
		name           string
		input          string
		shouldDetect   bool
		expectedMasked bool
	}{
		{
			name:           "正常手机号",
			input:          "联系我：13812345678",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "零宽字符绕过尝试",
			input:          "联系我：138‌1234‌5678",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "全角数字绕过尝试",
			input:          "联系我：１３８１２３４５６７８",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "空格分隔绕过尝试",
			input:          "联系我：138 1234 5678",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "连字符分隔绕过尝试",
			input:          "联系我：138-1234-5678",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "正常邮箱",
			input:          "邮箱：user@example.com",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "Unicode域名绕过尝试",
			input:          "邮箱：user@еxample.com",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "正常银行卡",
			input:          "卡号：6222020000000000000",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "空格分隔银行卡绕过尝试",
			input:          "卡号：6222 0200 0000 0000 000",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "连字符分隔银行卡绕过尝试",
			input:          "卡号：6222-0200-0000-0000-000",
			shouldDetect:   true,
			expectedMasked: true,
		},
		{
			name:           "普通文本",
			input:          "这是一段普通文本",
			shouldDetect:   false,
			expectedMasked: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := manager.SecureDesensitize(tc.input)
			if err != nil {
				t.Fatalf("SecureDesensitize() error = %v", err)
			}

			if result == nil {
				t.Fatal("SecureDesensitize() returned nil result")
			}

			// 检查是否被脱敏
			wasMasked := result.Desensitized != tc.input

			if tc.expectedMasked && !wasMasked {
				t.Errorf("Expected input to be masked, but it wasn't. Input: %s, Output: %s",
					tc.input, result.Desensitized)
			}

			if !tc.expectedMasked && wasMasked {
				t.Errorf("Expected input NOT to be masked, but it was. Input: %s, Output: %s",
					tc.input, result.Desensitized)
			}

			// 检查脱敏结果中是否还包含原始敏感信息
			if wasMasked {
				if containsOriginalSensitiveData(tc.input, result.Desensitized) {
					t.Errorf("Desensitized result still contains original sensitive data. Input: %s, Output: %s",
						tc.input, result.Desensitized)
				}
			}
		})
	}
}

// TestBypassDetection 测试绕过检测功能
func TestBypassDetection(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	bypassAttempts := []string{
		"138‌1234‌5678",            // 零宽字符
		"１３８１２３４５６７８",              // 全角数字
		"user@еxample.com",         // 西里尔字母
		"6222 0200 0000 0000 000",  // 过多空格
		"test-test-test-test-test", // 过多分隔符
	}

	for _, attempt := range bypassAttempts {
		t.Run("bypass_"+attempt, func(t *testing.T) {
			result, err := manager.SecureDesensitize(attempt)
			if err != nil {
				t.Fatalf("SecureDesensitize() error = %v", err)
			}

			// 检查是否记录了安全事件
			events := manager.GetSecurityEvents()
			found := false
			for _, event := range events {
				if event.EventType == "BYPASS_ATTEMPT" && event.Data == attempt {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Bypass attempt not detected for input: %s", attempt)
			}

			// 验证输出被正确脱敏
			if result.Desensitized == attempt {
				t.Errorf("Bypass attempt was not mitigated. Input: %s, Output: %s",
					attempt, result.Desensitized)
			}
		})
	}
}

// TestRateLimit 测试速率限制
func TestRateLimit(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 快速连续请求，应该触发速率限制
	for i := 0; i < 150; i++ { // 超过100次限制
		_, err := manager.SecureDesensitize("13812345678")
		if err != nil && strings.Contains(err.Error(), "rate limit exceeded") {
			// 成功触发速率限制
			return
		}
	}

	t.Error("Rate limit was not triggered after 150 requests")
}

// TestInputValidation 测试输入验证
func TestInputValidation(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	invalidInputs := []struct {
		input       string
		expectedErr string
	}{
		{
			input:       "",
			expectedErr: "empty input",
		},
		{
			input:       strings.Repeat("a", 10001),
			expectedErr: "input too long",
		},
		{
			input:       "<script>alert('xss')</script>",
			expectedErr: "malicious pattern detected",
		},
		{
			input:       "javascript:alert('test')",
			expectedErr: "malicious pattern detected",
		},
	}

	for _, tc := range invalidInputs {
		t.Run("invalid_"+tc.input[:min(10, len(tc.input))], func(t *testing.T) {
			result, err := manager.SecureDesensitize(tc.input)

			if err == nil {
				t.Errorf("Expected error for invalid input: %s", tc.input)
			}

			if result != nil {
				t.Errorf("Expected nil result for invalid input: %s", tc.input)
			}

			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error containing '%s', got: %s", tc.expectedErr, err.Error())
			}
		})
	}
}

// TestSecurityStats 测试安全统计功能
func TestSecurityStats(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 生成一些安全事件
	_, _ = manager.SecureDesensitize("138‌1234‌5678") // 绕过尝试
	_, _ = manager.SecureDesensitize("")              // 输入验证失败

	stats := manager.GetSecurityStats()

	if stats["total_events"].(int) == 0 {
		t.Error("Expected security events to be recorded")
	}

	if stats["bypass_detection_enabled"].(bool) != true {
		t.Error("Expected bypass detection to be enabled")
	}

	eventTypes := stats["event_types"].(map[string]int)
	if eventTypes["BYPASS_ATTEMPT"] == 0 {
		t.Error("Expected bypass attempt events to be recorded")
	}
}

// TestAggressiveDesensitization 测试激进脱敏模式
func TestAggressiveDesensitization(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 包含多种绕过手段的复杂输入
	complexInput := "联系方式：138‌1234‌5678，邮箱：user@еxample.com，卡号：6222-0200-0000-0000-000"

	result, err := manager.SecureDesensitize(complexInput)
	if err != nil {
		t.Fatalf("SecureDesensitize() error = %v", err)
	}

	// 验证所有敏感信息都被脱敏
	sensitivePatterns := []string{
		"13812345678",
		"user@example.com",
		"6222020000000000000",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(result.Desensitized, pattern) {
			t.Errorf("Sensitive pattern '%s' still present in result: %s", pattern, result.Desensitized)
		}
	}

	// 验证脱敏标记存在
	if !strings.Contains(result.Desensitized, "*") {
		t.Errorf("No masking characters found in result: %s", result.Desensitized)
	}
}

// 辅助函数：检查是否包含原始敏感数据
func containsOriginalSensitiveData(original, desensitized string) bool {
	// 提取原始数据中的敏感信息模式
	phoneRegex := regexp.MustCompile(`1[3-9]\d{9}`)
	phones := phoneRegex.FindAllString(original, -1)

	for _, phone := range phones {
		if strings.Contains(desensitized, phone) {
			return true
		}
	}

	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailRegex.FindAllString(original, -1)

	for _, email := range emails {
		if strings.Contains(desensitized, email) {
			return true
		}
	}

	cardRegex := regexp.MustCompile(`\d{13,19}`)
	cards := cardRegex.FindAllString(original, -1)

	for _, card := range cards {
		if strings.Contains(desensitized, card) {
			return true
		}
	}

	return false
}

// 辅助函数：获取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
