package dlp

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// 个人身份信息
	ChineseNamePattern    = "[\u4e00-\u9fa5]{2,6}"                                                                // 中文姓名
	ChineseIDCardPattern  = "[1-9]\\d{5}(?:18|19|20)\\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]" // 身份证
	PassportPattern       = "[a-zA-Z][0-9]{9}"                                                                    // 护照号
	SocialSecurityPattern = "[1-9]\\d{17}[\\dXx]"                                                                 // 社会保障号
	DriversLicensePattern = "[1-9]\\d{5}[a-zA-Z]\\d{6}"                                                           // 驾驶证号

	// 联系方式
	MobilePhonePattern = "(?:(?:\\+|00)86)?1(?:(?:3[\\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\\d])|(?:9[189]))\\d{8}" // 手机号
	FixedPhonePattern  = "(?:[\\d]{3,4}-)?[\\d]{7,8}(?:-[\\d]{1,4})?"                                                                   // 固定电话
	EmailPattern       = "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"                                                              // 电子邮箱

	// 地址信息
	AddressPattern    = "[\u4e00-\u9fa5]{2,}(?:省|自治区|市|特别行政区|自治州)?[\u4e00-\u9fa5]{2,}(?:市|区|县|镇|村|街道|路|号楼|栋|单元|室)" // 详细地址
	PostalCodePattern = `[1-9]\d{5}([^\d]|$)`                                                                      // 邮政编码

	// 金融信息
	BankCardPattern   = `(?:(?:4\d{12}(?:\d{3})?)|(?:5[1-5]\d{14})|(?:6(?:011|5\d{2})\d{12})|(?:3[47]\d{13})|(?:(?:30[0-5]|36\d|38\d)\d{11}))`                                // 银行卡
	CreditCardPattern = `(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\d{3})\d{11})` // 信用卡

	// 网络标识
	IPv4Pattern = `(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)` // IPv4

	IPv6Pattern = "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))"
	//IPv6Pattern = `(?:(?:[0-9A-Fa-f]{1,4}:){7}[0-9A-Fa-f]{1,4}|(?=(?:[0-9A-Fa-f]{0,4}:){0,7}[0-9A-Fa-f]{0,4}$)(([0-9A-Fa-f]{1,4}:){1,7}|:)((:[0-9A-Fa-f]{1,4}){1,7}|:)|(?:[0-9A-Fa-f]{1,4}:){7}:|:(:[0-9A-Fa-f]{1,4}){7})` // IPv6
	MACPattern  = `(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}` // MAC地址
	IMEIPattern = `\d{15,17}`                               // IMEI号

	// 车辆信息
	CarLicensePattern = `[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼使领][A-Z][A-HJ-NP-Z0-9]{4,5}[A-HJ-NP-Z0-9挂学警港澳]` // 车牌号
	VINPattern        = `[A-HJ-NPR-Z0-9]{17}`                                                            // 车架号

	// 密钥和令牌
	APIKeyPattern      = `[a-zA-Z0-9]{32,}`                                         // API密钥
	JWTPattern         = `eyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*` // JWT令牌
	AccessTokenPattern = `[a-zA-Z0-9]{40,}`                                         // 访问令牌

	// 设备标识
	DeviceIDPattern = `[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}`                // 设备ID
	UUIDPattern     = `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}` // UUID

	// 加密哈希
	MD5Pattern    = `[a-fA-F0-9]{32}` // MD5哈希
	SHA1Pattern   = `[a-fA-F0-9]{40}` // SHA1哈希
	SHA256Pattern = `[a-fA-F0-9]{64}` // SHA256哈希

	// 其他标识
	Base64Pattern = `(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?`                         // Base64编码
	LatLngPattern = `[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)` // 经纬度

	// URL和域名
	//URLPattern = "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\\b([-a-zA-Z0-9()@:%_\\+.~#?&//=]*)"
	URLPattern    = `(?:(?:https?|ftp)://)?[\w/\-?=%.]+\.[\w/\-&?=%.]+`                          // URL
	DomainPattern = `(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]` // 域名

	// 敏感内容
	PasswordPattern = `^[a-zA-Z]\w{5,17}$`  // 密码（以字母开头，长度在6~18之间，只能包含字母、数字和下划线）
	UsernamePattern = `[a-zA-Z0-9_-]{3,16}` // 用户名

	// 证件号码
	MedicalIDPattern = `[1-9]\d{7}`  // 医保卡号
	CompanyIDPattern = `[1-9]\d{14}` // 统一社会信用代码

	// 金融相关
	IBANPattern  = `[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z\d]?){0,16}` // IBAN号码
	SwiftPattern = `[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?`             // SWIFT代码

	// 代码相关
	GitRepoPattern    = `(?:git|ssh|https?|git@[\w\.]+)(?::(\/\/)?)([\w\.@\:/\-~]+)(\.git)(\/)?` // Git仓库
	CommitHashPattern = `[0-9a-f]{7,40}`                                                         // Git提交哈希

	// 特殊格式
	DatePattern   = `\d{4}[-/](0?[1-9]|1[012])[-/](0?[1-9]|[12][0-9]|3[01])`                    // 日期
	TimePattern   = `(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d`                                         // 时间
	IPPortPattern = `(?:6553[0-5]|655[0-2]\d|65[0-4]\d{2}|6[0-4]\d{3}|[1-5]\d{4}|[1-9]\d{0,3})` // 端口号

	// 修复 IPv6Pattern，使其更加准确
	//IPv6Pattern = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`

	// 修复 URLPattern，使其更加准确
	//URLPattern = `https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`

	// 修复 PasswordPattern，增加特殊字符要求
	//PasswordPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[@$!%*#?&])[A-Za-z\d@$!%*#?&]{8,}$`

)

var (
	ChineseNameRegex    *regexp.Regexp
	ChineseIDCardRegex  *regexp.Regexp
	PassportRegex       *regexp.Regexp
	SocialSecurityRegex *regexp.Regexp
	DriversLicenseRegex *regexp.Regexp
	MobilePhoneRegex    *regexp.Regexp
	FixedPhoneRegex     *regexp.Regexp
	EmailRegex          *regexp.Regexp
	AddressRegex        *regexp.Regexp
	PostalCodeRegex     *regexp.Regexp
	BankCardRegex       *regexp.Regexp
	CreditCardRegex     *regexp.Regexp
	IPv4Regex           *regexp.Regexp
	IPv6Regex           *regexp.Regexp
	MACRegex            *regexp.Regexp
	IMEIRegex           *regexp.Regexp
	CarLicenseRegex     *regexp.Regexp
	VINRegex            *regexp.Regexp
	APIKeyRegex         *regexp.Regexp
	JWTRegex            *regexp.Regexp
	AccessTokenRegex    *regexp.Regexp
	DeviceIDRegex       *regexp.Regexp
	UUIDRegex           *regexp.Regexp
	MD5Regex            *regexp.Regexp
	SHA1Regex           *regexp.Regexp
	SHA256Regex         *regexp.Regexp
	Base64Regex         *regexp.Regexp
	LatLngRegex         *regexp.Regexp
	URLRegex            *regexp.Regexp
	DomainRegex         *regexp.Regexp
	PasswordRegex       *regexp.Regexp
	UsernameRegex       *regexp.Regexp
	MedicalIDRegex      *regexp.Regexp
	CompanyIDRegex      *regexp.Regexp
	IBANRegex           *regexp.Regexp
	SwiftRegex          *regexp.Regexp
	GitRepoRegex        *regexp.Regexp
	CommitHashRegex     *regexp.Regexp
	DateRegex           *regexp.Regexp
	TimeRegex           *regexp.Regexp
	IPPortRegex         *regexp.Regexp
)

func init() {
	// 初始化所有正则表达式
	ChineseNameRegex = regexp.MustCompile(ChineseNamePattern)
	ChineseIDCardRegex = regexp.MustCompile(ChineseIDCardPattern)
	PassportRegex = regexp.MustCompile(PassportPattern)
	SocialSecurityRegex = regexp.MustCompile(SocialSecurityPattern)
	DriversLicenseRegex = regexp.MustCompile(DriversLicensePattern)
	MobilePhoneRegex = regexp.MustCompile(MobilePhonePattern)
	FixedPhoneRegex = regexp.MustCompile(FixedPhonePattern)
	EmailRegex = regexp.MustCompile(EmailPattern)
	AddressRegex = regexp.MustCompile(AddressPattern)
	PostalCodeRegex = regexp.MustCompile(PostalCodePattern)
	BankCardRegex = regexp.MustCompile(BankCardPattern)
	CreditCardRegex = regexp.MustCompile(CreditCardPattern)
	IPv4Regex = regexp.MustCompile(IPv4Pattern)
	IPv6Regex = regexp.MustCompile(IPv6Pattern)
	MACRegex = regexp.MustCompile(MACPattern)
	IMEIRegex = regexp.MustCompile(IMEIPattern)
	CarLicenseRegex = regexp.MustCompile(CarLicensePattern)
	VINRegex = regexp.MustCompile(VINPattern)
	APIKeyRegex = regexp.MustCompile(APIKeyPattern)
	JWTRegex = regexp.MustCompile(JWTPattern)
	AccessTokenRegex = regexp.MustCompile(AccessTokenPattern)
	DeviceIDRegex = regexp.MustCompile(DeviceIDPattern)
	UUIDRegex = regexp.MustCompile(UUIDPattern)
	MD5Regex = regexp.MustCompile(MD5Pattern)
	SHA1Regex = regexp.MustCompile(SHA1Pattern)
	SHA256Regex = regexp.MustCompile(SHA256Pattern)
	Base64Regex = regexp.MustCompile(Base64Pattern)
	LatLngRegex = regexp.MustCompile(LatLngPattern)
	URLRegex = regexp.MustCompile(URLPattern)
	DomainRegex = regexp.MustCompile(DomainPattern)
	PasswordRegex = regexp.MustCompile(PasswordPattern)
	UsernameRegex = regexp.MustCompile(UsernamePattern)
	MedicalIDRegex = regexp.MustCompile(MedicalIDPattern)
	CompanyIDRegex = regexp.MustCompile(CompanyIDPattern)
	IBANRegex = regexp.MustCompile(IBANPattern)
	SwiftRegex = regexp.MustCompile(SwiftPattern)
	GitRepoRegex = regexp.MustCompile(GitRepoPattern)
	CommitHashRegex = regexp.MustCompile(CommitHashPattern)
	DateRegex = regexp.MustCompile(DatePattern)
	TimeRegex = regexp.MustCompile(TimePattern)
	IPPortRegex = regexp.MustCompile(IPPortPattern)
}

// Match 正则匹配函数
func Match(text string, regex *regexp.Regexp) bool {
	return regex.MatchString(text)
}

// FindAll 查找所有匹配项
func FindAll(text string, regex *regexp.Regexp) []string {
	return regex.FindAllString(text, -1)
}

// ReplaceAll 替换所有匹配项
func ReplaceAll(text string, regex *regexp.Regexp, replacement string) string {
	return regex.ReplaceAllString(text, replacement)
}

// Matcher 匹配器定义，用于定义特定类型的匹配规则
type Matcher struct {
	Name        string              // 匹配器名称
	Pattern     string              // 正则表达式模式
	Regex       *regexp.Regexp      // 编译后的正则表达式
	Validator   func(string) bool   // 额外的验证函数
	Transformer func(string) string // 转换函数
}

// 全局匹配器注册表
var (
	matcherRegistry = make(map[string]*Matcher)
	registryMutex   sync.RWMutex
)

// 注册所有内置匹配器
func init() {
	registerDefaultMatchers()
}

// 注册默认的匹配器
func registerDefaultMatchers() {
	matchers := []*Matcher{
		{
			Name:    "ChineseName",
			Pattern: ChineseNamePattern,
			Validator: func(s string) bool {
				return len([]rune(s)) >= 2 && len([]rune(s)) <= 6
			},
			Transformer: func(s string) string {
				return ChineseNameDesensitize(s)
			},
		},
		{
			Name:    "ChineseIDCard",
			Pattern: ChineseIDCardPattern,
			Validator: func(s string) bool {
				return validateChineseIDCard(s)
			},
			Transformer: func(s string) string {
				return IdCardDesensitize(s)
			},
		},
		{
			Name:    "MobilePhone",
			Pattern: MobilePhonePattern,
			Validator: func(s string) bool {
				return len(s) == 11
			},
			Transformer: func(s string) string {
				return MobilePhoneDesensitize(s)
			},
		},
		// ... 其他匹配器注册
	}

	for _, m := range matchers {
		if regex, err := regexp.Compile(m.Pattern); err == nil {
			m.Regex = regex
			registerMatcher(m)
		}
	}
}

// 注册新的匹配器
func registerMatcher(matcher *Matcher) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	matcherRegistry[matcher.Name] = matcher
}

// 获取匹配器
func getMatcher(name string) (*Matcher, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	matcher, ok := matcherRegistry[name]
	return matcher, ok
}

// 正则匹配相关的实用函数

// DetectSensitiveInfo 检测文本中的所有敏感信息
func DetectSensitiveInfo(text string) map[string][]string {
	result := make(map[string][]string)

	registryMutex.RLock()
	defer registryMutex.RUnlock()

	for name, matcher := range matcherRegistry {
		if matches := matcher.Regex.FindAllString(text, -1); len(matches) > 0 {
			var validMatches []string
			for _, match := range matches {
				if matcher.Validator == nil || matcher.Validator(match) {
					validMatches = append(validMatches, match)
				}
			}
			if len(validMatches) > 0 {
				result[name] = validMatches
			}
		}
	}

	return result
}

// DesensitizeText 脱敏文本中的所有敏感信息
func DesensitizeText(text string) string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	for _, matcher := range matcherRegistry {
		text = matcher.Regex.ReplaceAllStringFunc(text, func(match string) string {
			if matcher.Validator == nil || matcher.Validator(match) {
				if matcher.Transformer != nil {
					return matcher.Transformer(match)
				}
				return strings.Repeat("*", utf8.RuneCountInString(match))
			}
			return match
		})
	}

	return text
}

// 验证中国身份证号的辅助函数
func validateChineseIDCard(id string) bool {
	if len(id) != 18 {
		return false
	}

	// 验证生日
	year, _ := strconv.Atoi(id[6:10])
	month, _ := strconv.Atoi(id[10:12])
	day, _ := strconv.Atoi(id[12:14])

	if year < 1900 || year > time.Now().Year() || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}

	// 检查日期是否有效
	_, err := time.Parse("20060102", id[6:14])
	if err != nil {
		return false
	}

	// 验证校验码
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	validChecksum := "10X98765432"
	sum := 0

	for i := 0; i < 17; i++ {
		n, _ := strconv.Atoi(string(id[i]))
		sum += n * weights[i]
	}

	checksum := validChecksum[sum%11]
	return string(id[17]) == string(checksum)
}

// RegexSearcher 正则搜索器，用于优化大文本的搜索性能
type RegexSearcher struct {
	matchers []*Matcher
	pool     sync.Pool
}

// NewRegexSearcher 创建新的正则搜索器
func NewRegexSearcher() *RegexSearcher {
	searcher := &RegexSearcher{
		pool: sync.Pool{
			New: func() interface{} {
				return new(strings.Builder)
			},
		},
	}

	registryMutex.RLock()
	defer registryMutex.RUnlock()

	for _, matcher := range matcherRegistry {
		searcher.matchers = append(searcher.matchers, matcher)
	}

	return searcher
}

// SearchParallel 并行搜索大文本中的敏感信息
func (s *RegexSearcher) SearchParallel(text string) map[string][]string {
	result := make(map[string][]string)
	resultMutex := sync.Mutex{}

	const chunkSize = 10000 // 每个协程处理的文本大小
	textLen := len(text)
	numGoroutines := (textLen + chunkSize - 1) / chunkSize

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > textLen {
			end = textLen
		}

		go func(chunk string) {
			defer wg.Done()

			matches := make(map[string][]string)
			for _, matcher := range s.matchers {
				if found := matcher.Regex.FindAllString(chunk, -1); len(found) > 0 {
					matches[matcher.Name] = append(matches[matcher.Name], found...)
				}
			}

			if len(matches) > 0 {
				resultMutex.Lock()
				for k, v := range matches {
					result[k] = append(result[k], v...)
				}
				resultMutex.Unlock()
			}
		}(text[start:end])
	}

	wg.Wait()
	return result
}

// ReplaceParallel 批量替换文本中的敏感信息
func (s *RegexSearcher) ReplaceParallel(text string) string {
	builder := s.pool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		s.pool.Put(builder)
	}()

	const chunkSize = 10000
	textLen := len(text)
	results := make(chan string, (textLen+chunkSize-1)/chunkSize)

	var wg sync.WaitGroup

	for start := 0; start < textLen; start += chunkSize {
		wg.Add(1)
		go func(begin int) {
			defer wg.Done()
			end := begin + chunkSize
			if end > textLen {
				end = textLen
			}

			chunk := text[begin:end]
			for _, matcher := range s.matchers {
				chunk = matcher.Regex.ReplaceAllStringFunc(chunk, func(match string) string {
					if matcher.Transformer != nil {
						return matcher.Transformer(match)
					}
					return strings.Repeat("*", utf8.RuneCountInString(match))
				})
			}

			results <- chunk
		}(start)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for chunk := range results {
		builder.WriteString(chunk)
	}

	return builder.String()
}
