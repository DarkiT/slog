package dlp

import (
	"regexp"
	"strings"
	"unicode"
)

// EnhancedPhoneDesensitizer 增强的手机号脱敏器 - 防绕过版本
type EnhancedPhoneDesensitizer struct {
	*BaseDesensitizer
	patterns        []*regexp.Regexp
	validatePattern *regexp.Regexp // 预编译的验证模式
	maxLength       int
}

// NewEnhancedPhoneDesensitizer 创建增强的手机号脱敏器
func NewEnhancedPhoneDesensitizer() *EnhancedPhoneDesensitizer {
	epd := &EnhancedPhoneDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer("phone"),
		maxLength:        50, // 防止DoS攻击
		// 预编译验证模式
		validatePattern: regexp.MustCompile(`1[3-9]\d{9}`),
	}

	// 多重正则表达式防护
	epd.patterns = []*regexp.Regexp{
		// 标准格式
		regexp.MustCompile(`1[3-9]\d{9}`),
		// 带分隔符格式
		regexp.MustCompile(`1[3-9]\d\s*\d{4}\s*\d{4}`),
		regexp.MustCompile(`1[3-9]\d[\s\-\.]*\d{4}[\s\-\.]*\d{4}`),
		// 全角数字格式
		regexp.MustCompile(`１[３-９][０-９]{9}`),
		// 混合格式
		regexp.MustCompile(`1[３-９][０-９\d\s\-\.]{8,12}`),
	}

	return epd
}

// preprocessText 预处理文本，规范化各种绕过手段
func (epd *EnhancedPhoneDesensitizer) preprocessText(input string) string {
	if len(input) > epd.maxLength {
		input = input[:epd.maxLength] // 防止DoS
	}

	// 1. 保持原始文本
	normalized := input

	// 2. 移除零宽字符和不可见字符
	var result strings.Builder
	for _, r := range normalized {
		if !isInvisibleChar(r) {
			// 3. 全角数字转半角
			if r >= '０' && r <= '９' {
				result.WriteRune('0' + (r - '０'))
			} else {
				result.WriteRune(r)
			}
		}
	}

	// 4. 规范化分隔符
	text := result.String()
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "-", "")
	text = strings.ReplaceAll(text, ".", "")

	return text
}

// isInvisibleChar 检查是否为不可见字符（零宽字符等）
func isInvisibleChar(r rune) bool {
	return r == '\u200B' || // 零宽空格
		r == '\u200C' || // 零宽非连字符
		r == '\u200D' || // 零宽连字符
		r == '\uFEFF' || // 零宽非断空格
		r == '\u2060' || // 词连接符
		unicode.IsControl(r)
}

// Supports 检查是否支持指定的数据类型
func (epd *EnhancedPhoneDesensitizer) Supports(dataType string) bool {
	return dataType == "phone" || dataType == "mobile" || dataType == "mobile_phone"
}

// Desensitize 执行增强的手机号脱敏
func (epd *EnhancedPhoneDesensitizer) Desensitize(data string) (string, error) {
	if !epd.Enabled() {
		return data, nil
	}

	return epd.desensitizeWithCache(data, func(input string) (string, error) {
		// 预处理文本
		processed := epd.preprocessText(input)
		result := input // 保持原始格式进行替换

		// 多重检测和替换
		for _, pattern := range epd.patterns {
			result = pattern.ReplaceAllStringFunc(result, func(match string) string {
				// 验证是否为有效手机号
				cleaned := epd.preprocessText(match)
				if epd.isValidPhone(cleaned) {
					return epd.maskPhone(match)
				}
				return match
			})
		}

		// 二次检测：如果预处理后的文本中仍有疑似手机号，进行激进脱敏
		if epd.containsSuspiciousPhone(processed) {
			return epd.aggressiveDesensitize(result), nil
		}

		return result, nil
	})
}

// isValidPhone 验证是否为有效的中国手机号
func (epd *EnhancedPhoneDesensitizer) isValidPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}

	// 验证首位和第二位
	if phone[0] != '1' {
		return false
	}

	second := phone[1]
	validSecond := []byte{'3', '4', '5', '6', '7', '8', '9'}
	valid := false
	for _, v := range validSecond {
		if second == v {
			valid = true
			break
		}
	}

	if !valid {
		return false
	}

	// 检查是否全部为数字
	for _, r := range phone {
		if r < '0' || r > '9' {
			return false
		}
	}

	// 避免明显的假号码（如全相同数字）
	if strings.Count(phone, string(phone[0])) == 11 {
		return false
	}

	return true
}

// maskPhone 对手机号进行脱敏
func (epd *EnhancedPhoneDesensitizer) maskPhone(phone string) string {
	// 保留格式，只脱敏数字部分
	runes := []rune(phone)
	digitCount := 0

	for i, r := range runes {
		if r >= '0' && r <= '9' || r >= '０' && r <= '９' {
			digitCount++
			// 脱敏第4-7位数字
			if digitCount >= 4 && digitCount <= 7 {
				runes[i] = '*'
			}
		}
	}

	return string(runes)
}

// containsSuspiciousPhone 检测是否包含疑似手机号模式
func (epd *EnhancedPhoneDesensitizer) containsSuspiciousPhone(text string) bool {
	// 检查连续数字模式
	digitSeq := ""
	for _, r := range text {
		if r >= '0' && r <= '9' {
			digitSeq += string(r)
		} else {
			if len(digitSeq) == 11 && strings.HasPrefix(digitSeq, "1") {
				return true
			}
			digitSeq = ""
		}
	}

	// 检查最后一个序列
	if len(digitSeq) == 11 && strings.HasPrefix(digitSeq, "1") {
		return true
	}

	return false
}

// aggressiveDesensitize 激进脱敏方法
func (epd *EnhancedPhoneDesensitizer) aggressiveDesensitize(text string) string {
	// 对所有数字序列进行脱敏
	return regexp.MustCompile(`\d{11}`).ReplaceAllStringFunc(text, func(match string) string {
		if strings.HasPrefix(match, "1") {
			return match[:3] + "****" + match[7:]
		}
		return match
	})
}

// GetSupportedTypes 获取支持的类型
func (epd *EnhancedPhoneDesensitizer) GetSupportedTypes() []string {
	return []string{"phone", "mobile", "mobile_phone"}
}

// ValidateType 增强的类型验证
func (epd *EnhancedPhoneDesensitizer) ValidateType(data string, dataType string) bool {
	if !epd.Supports(dataType) {
		return false
	}

	// 使用预编译的验证模式
	return epd.validatePattern.MatchString(data)
}

// GetTypePattern 获取类型的正则表达式模式
func (epd *EnhancedPhoneDesensitizer) GetTypePattern(dataType string) string {
	if epd.Supports(dataType) {
		return `1[3-9]\d{9}`
	}
	return ""
}

// EnhancedEmailDesensitizer 增强的邮箱脱敏器 - 防绕过版本
type EnhancedEmailDesensitizer struct {
	*BaseDesensitizer
	patterns        []*regexp.Regexp
	punycode        *regexp.Regexp
	validatePattern *regexp.Regexp // 预编译的验证模式
	maxLength       int
}

// NewEnhancedEmailDesensitizer 创建增强的邮箱脱敏器
func NewEnhancedEmailDesensitizer() *EnhancedEmailDesensitizer {
	eed := &EnhancedEmailDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer("email"),
		maxLength:        100,
		// 预编译验证模式
		validatePattern: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	}

	// 多重正则表达式防护
	eed.patterns = []*regexp.Regexp{
		// 标准邮箱格式
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		// Punycode域名
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@xn--[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		// 一般邮箱模式（包含更多字符）
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	}

	eed.punycode = regexp.MustCompile(`xn--[a-zA-Z0-9-]+`)

	return eed
}

// preprocessEmail 预处理邮箱文本
func (eed *EnhancedEmailDesensitizer) preprocessEmail(input string) string {
	if len(input) > eed.maxLength {
		input = input[:eed.maxLength]
	}

	// 保持原始文本
	normalized := input

	// 移除零宽字符
	var result strings.Builder
	for _, r := range normalized {
		if !isInvisibleChar(r) {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Supports 检查是否支持指定的数据类型
func (eed *EnhancedEmailDesensitizer) Supports(dataType string) bool {
	return dataType == "email" || dataType == "mail" || dataType == "email_address"
}

// Desensitize 执行增强的邮箱脱敏
func (eed *EnhancedEmailDesensitizer) Desensitize(data string) (string, error) {
	if !eed.Enabled() {
		return data, nil
	}

	return eed.desensitizeWithCache(data, func(input string) (string, error) {
		_ = eed.preprocessEmail(input) // 预处理但不使用结果，保留原始格式
		result := input

		for _, pattern := range eed.patterns {
			result = pattern.ReplaceAllStringFunc(result, func(email string) string {
				return eed.maskEmail(email)
			})
		}

		return result, nil
	})
}

// maskEmail 对邮箱进行脱敏
func (eed *EnhancedEmailDesensitizer) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	// 用户名脱敏：保留首尾各1位，中间用*替换
	if len(username) <= 2 {
		return "*@" + domain
	} else if len(username) <= 4 {
		return username[:1] + "*" + username[len(username)-1:] + "@" + domain
	} else {
		return username[:2] + strings.Repeat("*", len(username)-4) + username[len(username)-2:] + "@" + domain
	}
}

// GetSupportedTypes 获取支持的类型
func (eed *EnhancedEmailDesensitizer) GetSupportedTypes() []string {
	return []string{"email", "mail", "email_address"}
}

// ValidateType 增强的邮箱验证
func (eed *EnhancedEmailDesensitizer) ValidateType(data string, dataType string) bool {
	if !eed.Supports(dataType) {
		return false
	}

	// 使用预编译的验证模式
	return eed.validatePattern.MatchString(data)
}

// GetTypePattern 获取类型的正则表达式模式
func (eed *EnhancedEmailDesensitizer) GetTypePattern(dataType string) string {
	if eed.Supports(dataType) {
		return `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	}
	return ""
}
