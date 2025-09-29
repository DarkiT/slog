package dlp

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// EnhancedBankCardDesensitizer 增强的银行卡脱敏器
type EnhancedBankCardDesensitizer struct {
	*BaseDesensitizer
	patterns        []*regexp.Regexp
	validatePattern *regexp.Regexp // 预编译的验证模式
	maxLength       int
	luhnValidator   func(string) bool
}

// NewEnhancedBankCardDesensitizer 创建增强的银行卡脱敏器
func NewEnhancedBankCardDesensitizer() *EnhancedBankCardDesensitizer {
	ebcd := &EnhancedBankCardDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer("bank_card"),
		maxLength:        30,
		luhnValidator:    validateLuhn,
		// 预编译验证模式
		validatePattern: regexp.MustCompile(`\d{13,19}|\d{4}[\s\-]\d{4}[\s\-]\d{4}[\s\-]\d{1,7}`),
	}

	// 多重正则表达式防护
	ebcd.patterns = []*regexp.Regexp{
		// 标准连续格式
		regexp.MustCompile(`\b\d{13,19}\b`),
		// 带空格格式 (4-4-4-4 或类似)
		regexp.MustCompile(`\b\d{4}\s+\d{4}\s+\d{4}\s+\d{1,7}\b`),
		// 带连字符格式
		regexp.MustCompile(`\b\d{4}[\-\.]\d{4}[\-\.]\d{4}[\-\.]\d{1,7}\b`),
		// 全角数字格式
		regexp.MustCompile(`[０-９]{13,19}`),
		// 混合格式
		regexp.MustCompile(`[\d０-９\s\-\.]{15,25}`),
	}

	return ebcd
}

// preprocessBankCard 预处理银行卡文本
func (ebcd *EnhancedBankCardDesensitizer) preprocessBankCard(input string) string {
	if len(input) > ebcd.maxLength {
		input = input[:ebcd.maxLength]
	}

	// 保持原始文本
	normalized := input

	// 移除零宽字符并转换全角数字
	var result strings.Builder
	for _, r := range normalized {
		if !isInvisibleChar(r) {
			// 全角数字转半角
			if r >= '０' && r <= '９' {
				result.WriteRune('0' + (r - '０'))
			} else if !unicode.IsSpace(r) && r != '-' && r != '.' {
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

// Supports 检查是否支持指定的数据类型
func (ebcd *EnhancedBankCardDesensitizer) Supports(dataType string) bool {
	return dataType == "bank_card" || dataType == "credit_card" || dataType == "debit_card" || dataType == "card_number"
}

// Desensitize 执行增强的银行卡脱敏
func (ebcd *EnhancedBankCardDesensitizer) Desensitize(data string) (string, error) {
	if !ebcd.Enabled() {
		return data, nil
	}

	return ebcd.desensitizeWithCache(data, func(input string) (string, error) {
		result := input

		// 多重检测和替换
		for _, pattern := range ebcd.patterns {
			result = pattern.ReplaceAllStringFunc(result, func(match string) string {
				// 验证是否为有效银行卡号
				cleaned := ebcd.preprocessBankCard(match)
				if ebcd.isValidBankCard(cleaned) {
					return ebcd.maskBankCard(match)
				}
				return match
			})
		}

		// 二次安全检查：检查预处理后是否还有疑似银行卡号
		processed := ebcd.preprocessBankCard(result)
		if ebcd.containsSuspiciousBankCard(processed) {
			return ebcd.aggressiveDesensitize(result), nil
		}

		return result, nil
	})
}

// isValidBankCard 验证是否为有效的银行卡号
func (ebcd *EnhancedBankCardDesensitizer) isValidBankCard(cardNumber string) bool {
	// 长度检查
	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	// 检查是否全部为数字
	for _, r := range cardNumber {
		if r < '0' || r > '9' {
			return false
		}
	}

	// 避免明显的假号码
	if isObviousFakeCard(cardNumber) {
		return false
	}

	// Luhn算法验证
	return ebcd.luhnValidator(cardNumber)
}

// isObviousFakeCard 检查是否为明显的假卡号
func isObviousFakeCard(cardNumber string) bool {
	// 全相同数字
	first := cardNumber[0]
	allSame := true
	for i := 1; i < len(cardNumber); i++ {
		if cardNumber[i] != first {
			allSame = false
			break
		}
	}
	if allSame {
		return true
	}

	// 连续数字
	consecutive := true
	for i := 1; i < len(cardNumber); i++ {
		if cardNumber[i] != cardNumber[i-1]+1 {
			consecutive = false
			break
		}
	}
	if consecutive {
		return true
	}

	// 常见测试号码（完整匹配）
	testPatterns := []string{
		"4111111111111111", // Visa测试号码
		"5555555555554444", // MasterCard测试号码
		"378282246310005",  // American Express测试号码
		"4000000000000000", // 通用测试号码
		"4000000000000002", // 通用测试号码
	}

	for _, pattern := range testPatterns {
		if cardNumber == pattern {
			return true
		}
	}

	return false
}

// validateLuhn Luhn算法验证
func validateLuhn(cardNumber string) bool {
	sum := 0
	alternate := false

	// 从右到左处理
	for i := len(cardNumber) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(string(cardNumber[i]))
		if err != nil {
			return false
		}

		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// maskBankCard 对银行卡进行脱敏
func (ebcd *EnhancedBankCardDesensitizer) maskBankCard(cardNumber string) string {
	// 保持原始格式，只替换数字
	runes := []rune(cardNumber)
	digitPositions := []int{}

	// 找出所有数字位置
	for i, r := range runes {
		if (r >= '0' && r <= '9') || (r >= '０' && r <= '９') {
			digitPositions = append(digitPositions, i)
		}
	}

	if len(digitPositions) < 8 {
		return cardNumber // 太短，不处理
	}

	// 脱敏策略：保留前4位和后4位，中间用*替换
	for i := 4; i < len(digitPositions)-4; i++ {
		pos := digitPositions[i]
		runes[pos] = '*'
	}

	return string(runes)
}

// containsSuspiciousBankCard 检测是否包含疑似银行卡号
func (ebcd *EnhancedBankCardDesensitizer) containsSuspiciousBankCard(text string) bool {
	// 提取所有连续数字序列
	digitSeqs := regexp.MustCompile(`\d{13,19}`).FindAllString(text, -1)

	for _, seq := range digitSeqs {
		if ebcd.isValidBankCard(seq) {
			return true
		}
	}

	return false
}

// aggressiveDesensitize 激进脱敏方法
func (ebcd *EnhancedBankCardDesensitizer) aggressiveDesensitize(text string) string {
	// 对所有13-19位数字序列进行脱敏
	pattern := regexp.MustCompile(`\d{13,19}`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 8 {
			return match[:4] + strings.Repeat("*", len(match)-8) + match[len(match)-4:]
		}
		return strings.Repeat("*", len(match))
	})
}

// GetSupportedTypes 获取支持的类型
func (ebcd *EnhancedBankCardDesensitizer) GetSupportedTypes() []string {
	return []string{"bank_card", "credit_card", "debit_card", "card_number"}
}

// ValidateType 增强的类型验证
func (ebcd *EnhancedBankCardDesensitizer) ValidateType(data string, dataType string) bool {
	if !ebcd.Supports(dataType) {
		return false
	}

	// 使用预编译的验证模式
	return ebcd.validatePattern.MatchString(data)
}

// GetTypePattern 获取类型的正则表达式模式
func (ebcd *EnhancedBankCardDesensitizer) GetTypePattern(dataType string) string {
	if ebcd.Supports(dataType) {
		return `\d{13,19}|\d{4}[\s\-]\d{4}[\s\-]\d{4}[\s\-]\d{1,7}`
	}
	return ""
}
