package dlp

import (
	"regexp"
	"strconv"
	"strings"
)

// EnhancedIDCardDesensitizer 增强的身份证脱敏器
type EnhancedIDCardDesensitizer struct {
	*BaseDesensitizer
	patterns        []*regexp.Regexp
	validatePattern *regexp.Regexp // 预编译的验证模式
	maxLength       int
}

// NewEnhancedIDCardDesensitizer 创建增强的身份证脱敏器
func NewEnhancedIDCardDesensitizer() *EnhancedIDCardDesensitizer {
	eicd := &EnhancedIDCardDesensitizer{
		BaseDesensitizer: NewBaseDesensitizer("id_card"),
		maxLength:        50,
		// 预编译验证模式
		validatePattern: regexp.MustCompile(`\d{17}[\dXx]|\d{15}`),
	}

	// 多重正则表达式防护
	eicd.patterns = []*regexp.Regexp{
		// 18位身份证
		regexp.MustCompile(`\d{17}[\dXx]`),
		// 15位老身份证
		regexp.MustCompile(`\d{15}`),
		// 带空格的身份证
		regexp.MustCompile(`\d{6}\s+\d{8}\s+\d{3}[\dXx]`),
		// 带连字符的身份证
		regexp.MustCompile(`\d{6}[\-]\d{8}[\-]\d{3}[\dXx]`),
	}

	return eicd
}

// Supports 检查是否支持指定的数据类型
func (eicd *EnhancedIDCardDesensitizer) Supports(dataType string) bool {
	return dataType == "id_card" || dataType == "identity" || dataType == "citizen_id"
}

// Desensitize 执行增强的身份证脱敏
func (eicd *EnhancedIDCardDesensitizer) Desensitize(data string) (string, error) {
	if !eicd.Enabled() {
		return data, nil
	}

	return eicd.desensitizeWithCache(data, func(input string) (string, error) {
		result := input

		// 多重检测和替换
		for _, pattern := range eicd.patterns {
			result = pattern.ReplaceAllStringFunc(result, func(match string) string {
				// 验证是否为有效身份证号
				cleaned := eicd.cleanIDCard(match)
				if eicd.isValidIDCard(cleaned) {
					return eicd.maskIDCard(match)
				}
				return match
			})
		}

		return result, nil
	})
}

// cleanIDCard 清理身份证号
func (eicd *EnhancedIDCardDesensitizer) cleanIDCard(idCard string) string {
	// 移除空格和连字符
	cleaned := strings.ReplaceAll(idCard, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	// 将小写x转为大写X
	cleaned = strings.ToUpper(cleaned)
	return cleaned
}

// isValidIDCard 验证是否为有效的身份证号
func (eicd *EnhancedIDCardDesensitizer) isValidIDCard(idCard string) bool {
	// 基本长度检查
	if len(idCard) != 15 && len(idCard) != 18 {
		return false
	}

	// 15位身份证验证
	if len(idCard) == 15 {
		// 检查是否全为数字
		for _, r := range idCard {
			if r < '0' || r > '9' {
				return false
			}
		}
		// 验证出生日期（第7-12位，格式YYMMDD）
		year, _ := strconv.Atoi("19" + idCard[6:8])
		month, _ := strconv.Atoi(idCard[8:10])
		day, _ := strconv.Atoi(idCard[10:12])
		if !isValidDate(year, month, day) {
			return false
		}
		return true
	}

	// 18位身份证验证
	if len(idCard) == 18 {
		// 前17位必须是数字
		for i := 0; i < 17; i++ {
			if idCard[i] < '0' || idCard[i] > '9' {
				return false
			}
		}
		// 第18位可以是数字或X
		if idCard[17] != 'X' && (idCard[17] < '0' || idCard[17] > '9') {
			return false
		}

		// 验证出生日期（第7-14位，格式YYYYMMDD）
		year, _ := strconv.Atoi(idCard[6:10])
		month, _ := strconv.Atoi(idCard[10:12])
		day, _ := strconv.Atoi(idCard[12:14])
		if !isValidDate(year, month, day) {
			return false
		}

		// 验证校验码
		return eicd.validateChecksum(idCard)
	}

	return false
}

// isValidDate 验证日期是否有效
func isValidDate(year, month, day int) bool {
	// 年份范围检查（1900-当前年份）
	if year < 1900 || year > 2100 {
		return false
	}

	// 月份检查
	if month < 1 || month > 12 {
		return false
	}

	// 日期检查
	daysInMonth := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// 闰年2月有29天
	if month == 2 && isLeapYear(year) {
		daysInMonth[1] = 29
	}

	if day < 1 || day > daysInMonth[month-1] {
		return false
	}

	return true
}

// isLeapYear 判断是否为闰年
func isLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || (year%400 == 0)
}

// validateChecksum 验证18位身份证的校验码
func (eicd *EnhancedIDCardDesensitizer) validateChecksum(idCard string) bool {
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checksumMap := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := 0; i < 17; i++ {
		digit, _ := strconv.Atoi(string(idCard[i]))
		sum += digit * weights[i]
	}

	expectedChecksum := checksumMap[sum%11]
	return idCard[17] == expectedChecksum
}

// maskIDCard 对身份证进行脱敏
// 身份证号结构：前6位地区码 + 8位出生日期(YYYYMMDD) + 3位顺序码 + 1位校验码
// 脱敏策略：保留前6位（地区码）和后4位，中间的出生日期全部脱敏
func (eicd *EnhancedIDCardDesensitizer) maskIDCard(idCard string) string {
	runes := []rune(idCard)

	// 处理带分隔符的情况
	hasSpace := strings.Contains(idCard, " ")
	hasDash := strings.Contains(idCard, "-")

	// 清理后获取长度
	cleaned := eicd.cleanIDCard(idCard)

	if len(cleaned) == 15 {
		// 15位：保留前6位和后3位，中间6位出生日期脱敏
		if hasSpace || hasDash {
			// 保持原格式
			digitCount := 0
			var result []rune
			for _, r := range runes {
				if r >= '0' && r <= '9' {
					digitCount++
					if digitCount > 6 && digitCount <= 12 {
						result = append(result, '*')
					} else {
						result = append(result, r)
					}
				} else {
					result = append(result, r)
				}
			}
			return string(result)
		}
		// 无分隔符
		return cleaned[:6] + "******" + cleaned[12:]
	}

	if len(cleaned) == 18 {
		// 18位：保留前6位和后4位，中间8位出生日期脱敏
		if hasSpace || hasDash {
			// 保持原格式
			digitCount := 0
			var result []rune
			for _, r := range runes {
				if (r >= '0' && r <= '9') || r == 'X' || r == 'x' {
					digitCount++
					if digitCount > 6 && digitCount <= 14 {
						result = append(result, '*')
					} else {
						result = append(result, r)
					}
				} else {
					result = append(result, r)
				}
			}
			return string(result)
		}
		// 无分隔符
		return cleaned[:6] + "********" + cleaned[14:]
	}

	return idCard
}

// GetSupportedTypes 获取支持的类型
func (eicd *EnhancedIDCardDesensitizer) GetSupportedTypes() []string {
	return []string{"id_card", "identity", "citizen_id"}
}

// ValidateType 增强的类型验证
func (eicd *EnhancedIDCardDesensitizer) ValidateType(data string, dataType string) bool {
	if !eicd.Supports(dataType) {
		return false
	}

	// 使用预编译的验证模式
	return eicd.validatePattern.MatchString(data)
}

// GetTypePattern 获取类型的正则表达式模式
func (eicd *EnhancedIDCardDesensitizer) GetTypePattern(dataType string) string {
	if eicd.Supports(dataType) {
		return `\d{17}[\dXx]|\d{15}`
	}
	return ""
}

// GetConfig 获取配置
func (eicd *EnhancedIDCardDesensitizer) GetConfig(key string) (interface{}, bool) {
	if key == "strict_validation" {
		return true, true
	}
	return eicd.BaseDesensitizer.GetConfig(key)
}
