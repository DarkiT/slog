package dlp

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SecurityEnhancedManager 安全增强的脱敏管理器
type SecurityEnhancedManager struct {
	*DefaultDesensitizerManager

	// 安全特性
	bypassDetection   atomic.Bool     // 绕过检测功能
	rateLimiter       *RateLimiter    // 速率限制器
	suspiciousCounter int64           // 可疑活动计数
	alertThreshold    int64           // 告警阈值
	securityLog       []SecurityEvent // 安全事件日志
	securityLogMu     sync.RWMutex    // 安全日志锁
}

// SecurityEvent 安全事件
type SecurityEvent struct {
	Timestamp   time.Time
	EventType   string
	Data        string
	ThreatLevel string
	Details     string
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	requests    map[string]*RequestInfo
	mu          sync.RWMutex
	maxRequests int
	window      time.Duration
}

// RequestInfo 请求信息
type RequestInfo struct {
	Count     int
	FirstTime time.Time
	LastTime  time.Time
}

// NewSecurityEnhancedManager 创建安全增强管理器
func NewSecurityEnhancedManager() *SecurityEnhancedManager {
	// 创建基础管理器
	baseManager := &DefaultDesensitizerManager{
		desensitizers: make(map[string]Desensitizer),
		typeMapping:   make(map[string][]string),
	}
	baseManager.enabled.Store(true)
	baseManager.initializeStats()

	sem := &SecurityEnhancedManager{
		DefaultDesensitizerManager: baseManager,
		rateLimiter:                NewRateLimiter(100, time.Minute), // 每分钟100次请求
		alertThreshold:             10,                               // 10次可疑活动触发告警
		securityLog:                make([]SecurityEvent, 0),
	}

	// 启用绕过检测
	sem.bypassDetection.Store(true)

	// 注册增强版脱敏器
	sem.registerEnhancedDesensitizers()

	return sem
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string]*RequestInfo),
		maxRequests: maxRequests,
		window:      window,
	}
}

// registerEnhancedDesensitizers 注册增强版脱敏器
func (sem *SecurityEnhancedManager) registerEnhancedDesensitizers() {
	// 替换默认脱敏器为增强版本
	// 注意：不默认注册中文姓名脱敏器，因为它容易误判普通文本
	enhancedDesensitizers := []Desensitizer{
		NewEnhancedPhoneDesensitizer(),
		NewEnhancedEmailDesensitizer(),
		NewEnhancedBankCardDesensitizer(),
		NewEnhancedIDCardDesensitizer(), // 增强版身份证脱敏器，正确隐藏生日信息
		// NewChineseNameDesensitizer(), // 中文姓名脱敏器容易误判，需要用户显式注册
	}

	for _, desensitizer := range enhancedDesensitizers {
		if err := sem.RegisterDesensitizer(desensitizer); err != nil {
			// 记录注册失败事件
			sem.logSecurityEvent("REGISTER_FAILED", "", "LOW",
				fmt.Sprintf("Failed to register enhanced desensitizer: %v", err))
		}
	}
}

// SecureDesensitize 安全增强的脱敏方法
func (sem *SecurityEnhancedManager) SecureDesensitize(data string) (*DesensitizationResult, error) {
	// 1. 速率限制检查
	clientID := "default" // 在实际应用中应该是真实的客户端ID
	if !sem.rateLimiter.Allow(clientID) {
		sem.logSecurityEvent("RATE_LIMIT_EXCEEDED", data, "MEDIUM",
			fmt.Sprintf("Rate limit exceeded for client: %s", clientID))
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// 2. 输入验证
	if err := sem.validateInput(data); err != nil {
		sem.logSecurityEvent("INVALID_INPUT", data, "LOW", err.Error())
		return nil, err
	}

	// 3. 绕过检测
	if sem.bypassDetection.Load() {
		threats := sem.detectBypassAttempts(data)
		if len(threats) > 0 {
			sem.incrementSuspiciousActivity()
			for _, threat := range threats {
				sem.logSecurityEvent("BYPASS_ATTEMPT", data, "HIGH", threat)
			}

			// 使用激进模式脱敏
			return sem.aggressiveDesensitize(data)
		}
	}

	// 4. 正常脱敏处理
	result, err := sem.AutoDetectAndProcess(data)
	if err != nil {
		sem.logSecurityEvent("DESENSITIZE_ERROR", data, "LOW", err.Error())
		return nil, err
	}

	// 5. 结果验证
	if sem.validateResult(data, result) {
		return result, nil
	} else {
		// 验证失败，使用激进模式
		sem.logSecurityEvent("RESULT_VALIDATION_FAILED", data, "HIGH",
			"Desensitization result failed validation")
		return sem.aggressiveDesensitize(data)
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// 清理过期记录
	if info, exists := rl.requests[clientID]; exists {
		if now.Sub(info.FirstTime) > rl.window {
			delete(rl.requests, clientID)
		}
	}

	// 检查请求数
	info, exists := rl.requests[clientID]
	if !exists {
		rl.requests[clientID] = &RequestInfo{
			Count:     1,
			FirstTime: now,
			LastTime:  now,
		}
		return true
	}

	if info.Count >= rl.maxRequests {
		return false
	}

	info.Count++
	info.LastTime = now
	return true
}

// validateInput 输入验证
func (sem *SecurityEnhancedManager) validateInput(data string) error {
	if len(data) == 0 {
		return fmt.Errorf("empty input")
	}

	if len(data) > 10000 { // 防止DoS攻击
		return fmt.Errorf("input too long: %d characters", len(data))
	}

	// 检查是否包含恶意模式
	maliciousPatterns := []string{
		"<script", "</script>", "javascript:", "data:text/html",
		"vbscript:", "onload=", "onerror=",
	}

	for _, pattern := range maliciousPatterns {
		if contains(data, pattern) {
			return fmt.Errorf("malicious pattern detected: %s", pattern)
		}
	}

	return nil
}

// contains 大小写不敏感的字符串包含检查
func contains(text, pattern string) bool {
	return len(text) >= len(pattern) &&
		strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
}

// detectBypassAttempts 检测绕过尝试
func (sem *SecurityEnhancedManager) detectBypassAttempts(data string) []string {
	var threats []string

	// 检测零宽字符
	for _, r := range data {
		if isInvisibleChar(r) {
			threats = append(threats, fmt.Sprintf("Zero-width character detected: U+%04X", r))
			break
		}
	}

	// 检测全角字符
	fullWidthCount := 0
	for _, r := range data {
		if r >= '０' && r <= '９' {
			fullWidthCount++
		}
	}
	if fullWidthCount > 0 {
		threats = append(threats, fmt.Sprintf("Full-width digits detected: %d occurrences", fullWidthCount))
	}

	// 检测Unicode规范化攻击
	if sem.detectUnicodeAttack(data) {
		threats = append(threats, "Unicode normalization attack detected")
	}

	// 检测银行卡号的变体格式（带分隔符）
	// 查找看起来像银行卡号的模式
	bankCardPattern := regexp.MustCompile(`\d{4}[\s\-\.]\d{4}[\s\-\.]\d{4}[\s\-\.]\d{1,7}`)
	if bankCardPattern.MatchString(data) {
		threats = append(threats, "Bank card number with separators detected")
	}

	// 检测过多分隔符（用于其他情况）
	// 排除正常的银行卡和手机号格式
	if !bankCardPattern.MatchString(data) {
		separatorCount := strings.Count(data, " ") + strings.Count(data, "-") + strings.Count(data, ".")
		// 检测像 "test-test-test-test-test" 这样的模式
		// 如果分隔符超过4个，可能是绕过尝试
		if separatorCount >= 4 {
			threats = append(threats, fmt.Sprintf("Excessive separators detected: %d", separatorCount))
		}
	}

	return threats
}

// detectUnicodeAttack 检测Unicode攻击
func (sem *SecurityEnhancedManager) detectUnicodeAttack(data string) bool {
	// 检测是否包含看起来相似但编码不同的字符（混淆字符）
	// 这些是常见的用于绕过的西里尔字母
	suspiciousRunes := []rune{
		'\u0430', // 西里尔字母 а (看起来像拉丁字母 a)
		'\u0435', // 西里尔字母 е (看起来像拉丁字母 e)
		'\u043e', // 西里尔字母 о (看起来像拉丁字母 o)
		'\u0440', // 西里尔字母 р (看起来像拉丁字母 p)
		'\u0441', // 西里尔字母 с (看起来像拉丁字母 c)
		'\u0445', // 西里尔字母 х (看起来像拉丁字母 x)
		'\u0443', // 西里尔字母 у (看起来像拉丁字母 y)
	}

	// 检查数据中是否包含这些混淆字符
	for _, r := range data {
		for _, suspicious := range suspiciousRunes {
			if r == suspicious {
				return true
			}
		}
	}

	return false
}

// aggressiveDesensitize 激进脱敏模式
func (sem *SecurityEnhancedManager) aggressiveDesensitize(data string) (*DesensitizationResult, error) {
	result := data

	// 首先使用所有注册的增强脱敏器进行处理
	// 这些脱敏器已经包含了处理零宽字符、全角数字、分隔符的逻辑
	// 在激进模式下跳过中文姓名脱敏器，因为它可能误判普通文本
	for _, name := range sem.ListDesensitizers() {
		if name == "chinese_name" {
			continue // 跳过中文姓名脱敏器，避免误判
		}
		if desensitizer, exists := sem.GetDesensitizer(name); exists {
			processed, err := desensitizer.Desensitize(result)
			if err == nil && processed != result {
				result = processed
			}
		}
	}

	// 额外的激进模式：处理所有可能遗漏的模式
	// 脱敏11位数字（手机号）
	phonePattern := regexp.MustCompile(`\d{11}`)
	result = phonePattern.ReplaceAllStringFunc(result, func(match string) string {
		if strings.HasPrefix(match, "1") {
			return match[:3] + "****" + match[7:]
		}
		return strings.Repeat("*", len(match))
	})

	// 处理全角数字手机号
	fullWidthPhonePattern := regexp.MustCompile(`[１][３-９][０-９]{9}`)
	result = fullWidthPhonePattern.ReplaceAllStringFunc(result, func(match string) string {
		return string(match[0]) + string(match[1]) + string(match[2]) + "****" + match[len(match)-4:]
	})

	// 处理带分隔符的手机号
	separatedPhonePattern := regexp.MustCompile(`1[3-9]\d[\s\-\.]*\d{4}[\s\-\.]*\d{4}`)
	result = separatedPhonePattern.ReplaceAllStringFunc(result, func(match string) string {
		// 提取数字
		digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
		if len(digits) == 11 {
			// 保留原格式但替换中间4位
			digitCount := 0
			var resultRunes []rune
			for _, r := range match {
				if r >= '0' && r <= '9' {
					digitCount++
					if digitCount >= 4 && digitCount <= 7 {
						resultRunes = append(resultRunes, '*')
					} else {
						resultRunes = append(resultRunes, r)
					}
				} else {
					resultRunes = append(resultRunes, r)
				}
			}
			return string(resultRunes)
		}
		return match
	})

	// 处理带零宽字符的手机号 - 使用更灵活的方法
	// 因为Go正则不直接支持\u转义，我们使用另一种方法
	// 查找所有可能包含零宽字符的手机号模式
	phoneWithZeroWidthPattern := regexp.MustCompile(`1[3-9]\d.{0,5}\d{4}.{0,5}\d{4}`)
	result = phoneWithZeroWidthPattern.ReplaceAllStringFunc(result, func(match string) string {
		// 移除零宽字符并提取数字
		var digits []rune
		for _, r := range match {
			if r >= '0' && r <= '9' {
				digits = append(digits, r)
			}
		}

		// 如果提取出11位数字且以1开头，则脱敏
		if len(digits) == 11 && digits[0] == '1' {
			// 保留原格式但替换中间4位
			digitCount := 0
			var resultRunes []rune
			for _, r := range match {
				if r >= '0' && r <= '9' {
					digitCount++
					if digitCount >= 4 && digitCount <= 7 {
						resultRunes = append(resultRunes, '*')
					} else {
						resultRunes = append(resultRunes, r)
					}
				} else if !isInvisibleChar(r) {
					resultRunes = append(resultRunes, r)
				}
			}
			return string(resultRunes)
		}
		return match
	})

	// 脱敏邮箱模式（包括Unicode变体）
	// 更宽松的邮箱模式，可以捕获包含Unicode字符的域名
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[^\s]+\.[a-zA-Z]{2,}`)
	result = emailPattern.ReplaceAllStringFunc(result, func(email string) string {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			username := parts[0]
			if len(username) > 2 {
				return username[:1] + "**@" + parts[1]
			} else if len(username) > 0 {
				return "*@" + parts[1]
			}
		}
		return email
	})

	// 脱敏银行卡号（包括带分隔符的）
	cardPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\d{13,19}`),
		regexp.MustCompile(`\d{4}[\s\-\.]\d{4}[\s\-\.]\d{4}[\s\-\.]\d{1,7}`),
	}
	for _, pattern := range cardPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 提取数字
			digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
			digitStr := strings.Join(digits, "")
			if len(digitStr) >= 13 && len(digitStr) <= 19 {
				// 保留原格式但脱敏中间部分
				digitCount := 0
				var resultRunes []rune
				for _, r := range match {
					if r >= '0' && r <= '9' {
						digitCount++
						if digitCount > 4 && digitCount <= len(digits)-4 {
							resultRunes = append(resultRunes, '*')
						} else {
							resultRunes = append(resultRunes, r)
						}
					} else {
						resultRunes = append(resultRunes, r)
					}
				}
				return string(resultRunes)
			}
			return match
		})
	}

	// 如果结果没有变化，但检测到了绕过尝试，添加标记
	// 这种情况通常是输入不包含实际敏感信息，但有可疑模式
	if result == data && len(sem.detectBypassAttempts(data)) > 0 {
		// 在末尾添加一个不可见的标记，表示已处理
		result = result + "\u200B" // 零宽空格作为处理标记
	}

	return &DesensitizationResult{
		Original:     data,
		Desensitized: result,
		Desensitizer: "aggressive",
		Error:        nil,
	}, nil
}

// validateResult 验证脱敏结果
func (sem *SecurityEnhancedManager) validateResult(original string, result *DesensitizationResult) bool {
	if result == nil || result.Error != nil {
		return false
	}

	// 检查是否还包含明显的敏感信息
	sensitivePatterns := []string{
		`1[3-9]\d{9}`,         // 手机号
		`\d{13,19}`,           // 银行卡号
		`\d{17}[\dXx]|\d{15}`, // 身份证号
	}

	for _, pattern := range sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, result.Desensitized); matched {
			return false // 仍包含敏感信息，验证失败
		}
	}

	return true
}

// incrementSuspiciousActivity 增加可疑活动计数
func (sem *SecurityEnhancedManager) incrementSuspiciousActivity() {
	count := atomic.AddInt64(&sem.suspiciousCounter, 1)
	if count >= sem.alertThreshold {
		sem.logSecurityEvent("ALERT_THRESHOLD_REACHED", "", "CRITICAL",
			fmt.Sprintf("Suspicious activity threshold reached: %d", count))

		// 重置计数器
		atomic.StoreInt64(&sem.suspiciousCounter, 0)
	}
}

// logSecurityEvent 记录安全事件
func (sem *SecurityEnhancedManager) logSecurityEvent(eventType, data, threatLevel, details string) {
	sem.securityLogMu.Lock()
	defer sem.securityLogMu.Unlock()

	event := SecurityEvent{
		Timestamp:   time.Now(),
		EventType:   eventType,
		Data:        data,
		ThreatLevel: threatLevel,
		Details:     details,
	}

	sem.securityLog = append(sem.securityLog, event)

	// 保持日志大小在合理范围内
	if len(sem.securityLog) > 1000 {
		sem.securityLog = sem.securityLog[500:] // 保留最近500条
	}
}

// GetSecurityEvents 获取安全事件日志
func (sem *SecurityEnhancedManager) GetSecurityEvents() []SecurityEvent {
	sem.securityLogMu.RLock()
	defer sem.securityLogMu.RUnlock()

	// 返回副本
	events := make([]SecurityEvent, len(sem.securityLog))
	copy(events, sem.securityLog)
	return events
}

// GetSecurityStats 获取安全统计信息
func (sem *SecurityEnhancedManager) GetSecurityStats() map[string]interface{} {
	sem.securityLogMu.RLock()
	defer sem.securityLogMu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_events"] = len(sem.securityLog)
	stats["suspicious_count"] = atomic.LoadInt64(&sem.suspiciousCounter)
	stats["bypass_detection_enabled"] = sem.bypassDetection.Load()

	// 统计事件类型
	eventTypes := make(map[string]int)
	threatLevels := make(map[string]int)

	for _, event := range sem.securityLog {
		eventTypes[event.EventType]++
		threatLevels[event.ThreatLevel]++
	}

	stats["event_types"] = eventTypes
	stats["threat_levels"] = threatLevels

	return stats
}
