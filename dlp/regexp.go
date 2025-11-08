package dlp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	// 个人身份信息
	// 百家姓列表（按拼音排序）
	ChineseSurnames = "(?:" +
		"艾|安|敖|巴|白|班|包|暴|鲍|贝|贲|毕|边|卞|别|邴|伯|薄|卜|蔡|曹|岑|柴|昌|常|晁|车|陈|成|程|池|充|仇|储|楚|褚|淳|从|崔|戴|党|邓|狄|刁|丁|董|窦|杜|端|段|鄂|樊|范|方|房|费|丰|封|冯|凤|伏|扶|符|福|傅|甘|高|郜|戈|盖|葛|耿|龚|宫|勾|苟|辜|古|谷|顾|关|管|桂|郭|国|韩|杭|郝|何|和|贺|赫|衡|洪|侯|胡|扈|花|华|滑|怀|宦|黄|惠|霍|姬|嵇|吉|汲|籍|计|纪|季|贾|简|姜|江|蒋|焦|金|靳|荆|井|景|居|鞠|阚|康|柯|空|孔|寇|蒯|匡|邝|况|赖|蓝|郎|劳|雷|冷|黎|李|利|连|廉|练|梁|廖|林|蔺|凌|令|刘|柳|龙|隆|娄|卢|鲁|陆|路|逯|禄|吕|栾|罗|骆|麻|马|满|毛|茅|梅|蒙|孟|米|宓|闵|明|莫|牟|穆|倪|聂|年|宁|牛|钮|农|潘|庞|裴|彭|皮|平|蒲|濮|浦|戚|祁|齐|钱|强|乔|谯|秦|邱|裘|曲|屈|瞿|全|阙|冉|饶|任|荣|容|阮|芮|桑|沙|山|单|商|上|邵|佘|申|沈|盛|师|施|时|石|史|寿|殳|舒|束|双|水|司|松|宋|苏|宿|孙|索|邰|太|谈|谭|汤|唐|陶|滕|田|通|童|涂|屠|万|汪|王|危|韦|卫|魏|温|文|闻|翁|巫|邬|伍|武|吴|务|西|席|夏|咸|向|项|萧|谢|辛|邢|幸|熊|徐|许|轩|宣|薛|荀|闫|严|言|阎|颜|晏|燕|杨|姚|叶|伊|易|殷|尹|应|庸|雍|尤|游|于|余|虞|元|袁|岳|云|臧|曾|翟|詹|湛|张|章|赵|甄|郑|支|钟|仲|周|朱|诸|祝|庄|卓|子|宗|邹|祖|左" +
		")"
	ChineseNamePattern   = "(?:" + ChineseSurnames + ")[\u4e00-\u9fa5]{1,5}"                                     // 中文姓名：百家姓+名字
	ChineseIDCardPattern = "[1-9]\\d{5}(?:18|19|20)\\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]" // 身份证
	PassportPattern      = "[a-zA-Z][0-9]{9}"                                                                    // 护照号
	// 社会保障号：18位身份证格式，避免误匹配19位银行卡
	SocialSecurityPattern = "[1-9]\\d{5}(?:18|19|20)\\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]" // 社会保障号（18位身份证格式）
	DriversLicensePattern = "[1-9]\\d{5}[a-zA-Z]\\d{6}"                                                           // 驾驶证号

	// 联系方式
	MobilePhonePattern = "(?:(?:\\+|00)86)?1(?:(?:3[\\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\\d])|(?:9[189]))\\d{8}" // 手机号
	// 固定电话：更严格的格式，避免误匹配银行卡号
	FixedPhonePattern = "(?:(?:0\\d{2,3}[-]?)?[1-9]\\d{6,7})|(?:\\d{3,4}[-]\\d{7,8}(?:[-]\\d{1,4})?)" // 固定电话（更严格格式）
	EmailPattern      = `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`                              // 电子邮箱

	// 地址信息
	AddressPattern = "[\u4e00-\u9fa5]{2,}(?:省|自治区|市|特别行政区|自治州)?[\u4e00-\u9fa5]{2,}(?:市|区|县|镇|村|街道|路|号楼|栋|单元|室)" // 详细地址
	// 邮政编码：更严格的格式，避免误匹配银行卡号尾数
	PostalCodePattern = `(?:^|[^\d])[1-9]\d{5}(?:[^\d]|$)` // 邮政编码（独立的6位数字）

	// 金融信息
	// 中国银行卡：19位，以特定BIN码开头（更精确的匹配）
	BankCardPattern = `(?:(?:6(?:2[2-9]\d|3\d{2}|4\d{2}|5\d{2}|6\d{2}|7\d{2}|8\d{2}|9\d{2})\d{13,16})|(?:62\d{16,17}))` // 中国银行卡（19位，以62-69开头）
	// 国际信用卡：16位，标准国际格式
	CreditCardPattern = `(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\d{3})\d{11})` // 国际信用卡（主要16位）

	// 网络标识
	IPv4Pattern = `(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)` // IPv4
	IPv6Pattern = "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))"
	MACPattern  = `(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}` // MAC地址
	IMEIPattern = `\d{15,17}`                               // IMEI号

	// 车辆信息
	LicensePlatePattern = `[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼使领][A-Z][A-HJ-NP-Z0-9]{4,5}[A-HJ-NP-Z0-9挂学警港澳]` // 车牌号
	VINPattern          = `[A-HJ-NPR-Z0-9]{17}`                                                            // 车架号

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
	LatLngPattern = `[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)` // 经纬度

	// URL和域名
	URLPattern    = `\b(([a-zA-Z]{1,6}:\/\/?)([^:@]*:[^:@]+@)?(?:[a-z0-9.\-]+|www|[a-z0-9.\-])[.](?:[^\s()<>]+|\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\))+(?:\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))(:(?:6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3}))?\b`
	DomainPattern = `(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]` // 域名

	// 敏感内容
	PasswordPattern = `^[a-zA-Z]\w{5,17}$` // 密码（以字母开头，长度在6~18之间，只能包含字母、数字和下划线）
	// PasswordPattern，增加特殊字符要求
	// PasswordPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[@$!%*#?&])[A-Za-z\d@$!%*#?&]{8,}$`
	UsernamePattern = `[a-zA-Z0-9_]{3,16}` // 用户名
	// 医保卡号：更严格的格式，避免误匹配银行卡号
	MedicalIDPattern = `[1-5]\\d{7}`  // 医保卡号（8位，1-5开头）
	CompanyIDPattern = `[0-9A-Z]{15}` // 公司编号

	// 金融相关
	IBANPattern  = `[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z\d]?){0,16}` // IBAN号码
	SwiftPattern = `[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?`             // SWIFT代码

	// 代码相关
	GitRepoPattern = `(?:git|ssh|git@[\w\.]+)(?::(\/\/)?)([\w\.@\:/\-~]+)(\.git)(\/)?` // Git仓库

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
	LicensePlateRegex   *regexp.Regexp
	VINRegex            *regexp.Regexp
	APIKeyRegex         *regexp.Regexp
	JWTRegex            *regexp.Regexp
	AccessTokenRegex    *regexp.Regexp
	DeviceIDRegex       *regexp.Regexp
	UUIDRegex           *regexp.Regexp
	MD5Regex            *regexp.Regexp
	SHA1Regex           *regexp.Regexp
	SHA256Regex         *regexp.Regexp
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
)

// Matcher 定义匹配器结构体
type Matcher struct {
	Name        string              // 匹配器名称
	Pattern     string              // 正则表达式模式
	Regex       *regexp.Regexp      // 编译后的正则表达式
	Validator   func(string) bool   // 验证函数
	Transformer func(string) string // 转换函数
	Priority    int                 // 优先级，数字越大优先级越高
	Complexity  int                 // 正则表达式复杂度评分
	FastFilters []string            // 快速包含过滤，全部缺失时跳过匹配
}

// MatchResult 定义匹配结果结构体
type MatchResult struct {
	Type     string // 匹配类型
	Content  string // 匹配内容
	Position [2]int // 匹配位置 [start, end]
}

// RegexSearcher 定义正则搜索器
type RegexSearcher struct {
	matchers []*Matcher                // 按优先级排序的匹配器列表
	pool     sync.Pool                 // 字符串构建器池
	mu       sync.RWMutex              // 读写锁
	cache    map[string]*regexp.Regexp // 正则表达式缓存
	version  int64                     // 版本号，用于缓存失效
}

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
	LicensePlateRegex = regexp.MustCompile(LicensePlatePattern)
	VINRegex = regexp.MustCompile(VINPattern)
	APIKeyRegex = regexp.MustCompile(APIKeyPattern)
	JWTRegex = regexp.MustCompile(JWTPattern)
	AccessTokenRegex = regexp.MustCompile(AccessTokenPattern)
	DeviceIDRegex = regexp.MustCompile(DeviceIDPattern)
	UUIDRegex = regexp.MustCompile(UUIDPattern)
	MD5Regex = regexp.MustCompile(MD5Pattern)
	SHA1Regex = regexp.MustCompile(SHA1Pattern)
	SHA256Regex = regexp.MustCompile(SHA256Pattern)
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
}

// NewRegexSearcher 创建新的正则搜索器
func NewRegexSearcher() *RegexSearcher {
	searcher := &RegexSearcher{
		matchers: make([]*Matcher, 0, 50), // 预分配合适的容量
		cache:    make(map[string]*regexp.Regexp),
		pool: sync.Pool{
			New: func() interface{} {
				return new(strings.Builder)
			},
		},
	}

	if err := searcher.registerDefaultMatchers(); err != nil {
		panic(fmt.Sprintf("Failed to initialize RegexSearcher: %v", err))
	}

	// 根据复杂度和优先级排序匹配器
	searcher.sortMatchers()
	return searcher
}

// sortMatchers 根据复杂度和优先级排序匹配器
func (s *RegexSearcher) sortMatchers() {
	sort.Slice(s.matchers, func(i, j int) bool {
		// 首先比较复杂度
		if s.matchers[i].Complexity != s.matchers[j].Complexity {
			return s.matchers[i].Complexity > s.matchers[j].Complexity
		}
		// 复杂度相同时比较优先级
		return s.matchers[i].Priority > s.matchers[j].Priority
	})
}

// calculateComplexity 计算正则表达式的复杂度
func calculateComplexity(pattern string) int {
	score := 0

	// 特殊字符评分
	specialChars := []string{"\\", "^", "$", "*", "+", "?", "{", "}", "[", "]", "(", ")", "|", "."}
	for _, char := range specialChars {
		score += strings.Count(pattern, char) * 2
	}

	// 字符类评分
	charClasses := []string{"\\d", "\\w", "\\s", "\\b", "\\D", "\\W", "\\S", "\\B"}
	for _, class := range charClasses {
		score += strings.Count(pattern, class) * 3
	}

	// 量词评分
	quantifiers := []string{"{", "+", "*", "?"}
	for _, q := range quantifiers {
		score += strings.Count(pattern, q) * 4
	}

	// 捕获组评分
	score += strings.Count(pattern, "(") * 5

	// 否定和前瞻后顾评分
	if strings.Contains(pattern, "(?!") || strings.Contains(pattern, "(?=") {
		score += 10
	}

	return score
}

// Match 执行匹配操作
func (s *RegexSearcher) Match(text string) []MatchResult {
	if text == "" {
		return nil
	}

	var results []MatchResult
	positions := make(map[[2]int]bool) // 用于跟踪已匹配的位置

	// 按优先级顺序遍历匹配器
	for _, matcher := range s.matchers {
		if len(matcher.FastFilters) > 0 && !containsFastToken(text, matcher.FastFilters) {
			continue
		}
		// 尝试匹配
		matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)
		if matches == nil {
			continue
		}

		for _, match := range matches {
			pos := [2]int{match[0], match[1]}

			// 检查位置是否已被更高优先级的模式匹配
			if positions[pos] {
				continue
			}

			content := text[match[0]:match[1]]

			// 如果有验证器，验证匹配内容
			if matcher.Validator != nil && !matcher.Validator(content) {
				continue
			}

			// 标记该位置已被匹配
			positions[pos] = true

			results = append(results, MatchResult{
				Type:     matcher.Name,
				Content:  content,
				Position: pos,
			})
		}
	}

	// 按位置排序结果
	sort.Slice(results, func(i, j int) bool {
		return results[i].Position[0] < results[j].Position[0]
	})

	return results
}

func containsFastToken(text string, tokens []string) bool {
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if len(token) == 1 {
			if strings.IndexByte(text, token[0]) >= 0 {
				return true
			}
			continue
		}
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

// AddMatcher 添加新的匹配器
func (s *RegexSearcher) AddMatcher(matcher *Matcher) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 编译正则表达式
	regex, err := regexp.Compile(matcher.Pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex pattern for %s: %v", matcher.Name, err)
	}

	// 计算复杂度
	matcher.Complexity = calculateComplexity(matcher.Pattern)
	matcher.Regex = regex

	// 添加到匹配器列表
	s.matchers = append(s.matchers, matcher)

	// 重新排序
	s.sortMatchers()

	// 递增版本号
	s.version++
	return nil
}

// RemoveMatcher 移除匹配器
func (s *RegexSearcher) RemoveMatcher(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, m := range s.matchers {
		if m.Name == name {
			s.matchers = append(s.matchers[:i], s.matchers[i+1:]...)
			break
		}
	}
}

// GetMatcher 获取指定名称的匹配器
func (s *RegexSearcher) GetMatcher(name string) *Matcher {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.matchers {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// UpdateMatcher 更新匹配器
func (s *RegexSearcher) UpdateMatcher(name string, pattern string, validator func(string) bool, transformer func(string) string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.matchers {
		if m.Name == name {
			// 编译新的正则表达式
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("failed to compile regex pattern: %v", err)
			}

			// 更新匹配器
			m.Pattern = pattern
			m.Regex = regex
			m.Validator = validator
			m.Transformer = transformer
			m.Complexity = calculateComplexity(pattern)

			// 重新排序
			s.sortMatchers()
			return nil
		}
	}
	return fmt.Errorf("matcher %s not found", name)
}

// GetAllSupportedTypes 获取所有支持的敏感信息类型
func (s *RegexSearcher) GetAllSupportedTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, len(s.matchers))
	for i, matcher := range s.matchers {
		types[i] = matcher.Name
	}
	return types
}

// SearchSensitiveByType 按类型搜索敏感信息
func (s *RegexSearcher) SearchSensitiveByType(text string, typeName string) []MatchResult {
	if text == "" {
		return nil
	}

	s.mu.RLock()
	matcher := s.GetMatcher(typeName)
	s.mu.RUnlock()

	if matcher == nil {
		return nil
	}

	if len(matcher.FastFilters) > 0 && !containsFastToken(text, matcher.FastFilters) {
		return nil
	}

	var results []MatchResult

	// 中文姓名特殊处理
	if typeName == ChineseName {
		// 普通情况下的中文姓名处理
		namePattern := regexp.MustCompile("[\u4e00-\u9fa5]{2,}")
		matches := namePattern.FindAllStringSubmatchIndex(text, -1)

		// 用于过滤重叠的姓名
		var filteredResults []MatchResult

		for _, match := range matches {
			content := text[match[0]:match[1]]
			chineseChars := []rune(content)

			// 短文本直接处理
			if len(chineseChars) <= 3 {
				if matcher.Regex.MatchString(content) {
					filteredResults = append(filteredResults, MatchResult{
						Type:     matcher.Name,
						Content:  content,
						Position: [2]int{match[0], match[1]},
					})
				}
				continue
			}

			// 对长文本，尝试按分隔符和常见姓氏拆分
			candidateNames := extractChineseNames(chineseChars)
			for _, name := range candidateNames {
				nameStr := string(name.chars)
				if matcher.Regex.MatchString(nameStr) {
					startPos := match[0] + name.startIndex*3 // 每个中文字符约3个字节
					endPos := startPos + len(name.chars)*3
					filteredResults = append(filteredResults, MatchResult{
						Type:     matcher.Name,
						Content:  nameStr,
						Position: [2]int{startPos, endPos},
					})
				}
			}
		}

		// 过滤重叠的结果，只保留最合理的姓名
		results = filterOverlappingNames(filteredResults)
		return results
	}

	// 正常匹配流程
	matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		content := text[match[0]:match[1]]

		// 如果有验证器，验证匹配内容
		if matcher.Validator != nil && !matcher.Validator(content) {
			continue
		}

		results = append(results, MatchResult{
			Type:     matcher.Name,
			Content:  content,
			Position: [2]int{match[0], match[1]},
		})
	}

	return results
}

// nameCandidate 表示候选姓名
type nameCandidate struct {
	chars      []rune
	startIndex int
	score      int // 得分越高越可能是真实姓名
}

// extractChineseNames 从文本中提取可能的中文姓名
func extractChineseNames(chars []rune) []nameCandidate {
	var candidates []nameCandidate

	// 特殊处理"张三李四"模式
	// 检查是否有连续的4个中文字符，可以分解为两个2字姓名
	if len(chars) == 4 {
		firstSurname := string(chars[0])
		secondSurname := string(chars[2])

		// 检查第一个和第三个字符是否都是常见姓氏
		isFirstSurname := strings.Contains(ChineseSurnames, firstSurname)
		isSecondSurname := strings.Contains(ChineseSurnames, secondSurname)

		// 如果两个都是姓氏，很可能是"张三李四"模式
		if isFirstSurname && isSecondSurname {
			candidates = append(candidates, nameCandidate{
				chars:      chars[0:2],
				startIndex: 0,
				score:      100, // 最高优先级
			})

			candidates = append(candidates, nameCandidate{
				chars:      chars[2:4],
				startIndex: 2,
				score:      100, // 最高优先级
			})

			return candidates
		}
	}

	// 1. 先检查常见分隔词
	separators := []rune{'、', '，', '和', '与', '：', ' '}
	separatorIndices := make([]int, 0)

	for i, char := range chars {
		for _, sep := range separators {
			if char == sep {
				separatorIndices = append(separatorIndices, i)
				break
			}
		}
	}

	// 2. 如果有分隔符，提取分隔符两侧的2-3字词
	if len(separatorIndices) > 0 {
		lastEnd := 0
		for _, sepIdx := range separatorIndices {
			// 分隔符前面的内容
			if sepIdx-lastEnd >= 2 && sepIdx-lastEnd <= 3 {
				candidates = append(candidates, nameCandidate{
					chars:      chars[lastEnd:sepIdx],
					startIndex: lastEnd,
					score:      80, // 分隔符两侧的短词很可能是姓名
				})
			}

			lastEnd = sepIdx + 1
		}

		// 最后一个分隔符后面的内容
		if len(chars)-lastEnd >= 2 && len(chars)-lastEnd <= 3 {
			candidates = append(candidates, nameCandidate{
				chars:      chars[lastEnd:],
				startIndex: lastEnd,
				score:      80,
			})
		}
	}

	// 3. 根据常见姓氏匹配可能的姓名
	for i := 0; i < len(chars); i++ {
		surname := string(chars[i])

		// 检查是否是常见姓氏
		isSurname := strings.Contains(ChineseSurnames, surname)

		if isSurname {
			// 姓氏+1个字的名（2字姓名）
			if i+1 < len(chars) {
				candidates = append(candidates, nameCandidate{
					chars:      chars[i : i+2],
					startIndex: i,
					score:      90, // 常见姓氏+1个字，很可能是姓名
				})
			}

			// 姓氏+2个字的名（3字姓名）
			if i+2 < len(chars) {
				candidates = append(candidates, nameCandidate{
					chars:      chars[i : i+3],
					startIndex: i,
					score:      85, // 常见姓氏+2个字，也可能是姓名
				})
			}
		}
	}

	// 4. 添加所有连续2-3个字的组合作为低优先级候选项
	if len(candidates) == 0 {
		for i := 0; i < len(chars)-1; i++ {
			// 2字组合
			candidates = append(candidates, nameCandidate{
				chars:      chars[i : i+2],
				startIndex: i,
				score:      50,
			})

			// 3字组合
			if i+2 < len(chars) {
				candidates = append(candidates, nameCandidate{
					chars:      chars[i : i+3],
					startIndex: i,
					score:      40,
				})
			}
		}
	}

	return candidates
}

// filterOverlappingNames 过滤重叠的姓名，只保留最合理的候选项
func filterOverlappingNames(matches []MatchResult) []MatchResult {
	if len(matches) <= 2 {
		return matches
	}

	// 按位置排序
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Position[0] < matches[j].Position[0]
	})

	var filtered []MatchResult
	var lastEnd int

	// 首先尝试识别连续的姓名模式（如"张三李四"）
	for i := 0; i < len(matches)-1; i++ {
		current := matches[i]
		next := matches[i+1]

		// 检查当前匹配和下一个匹配是否紧邻
		if current.Position[1] == next.Position[0] ||
			next.Position[0]-current.Position[1] <= 3 { // 允许有少量间隔（如标点）

			// 检查两个匹配是否都是合理的姓名长度（2-3个字符）
			currentLen := len([]rune(current.Content))
			nextLen := len([]rune(next.Content))

			if (currentLen == 2 || currentLen == 3) && (nextLen == 2 || nextLen == 3) {
				// 找到连续姓名模式，添加这两个匹配
				filtered = append(filtered, current)
				filtered = append(filtered, next)
				i++ // 跳过下一个匹配，因为已经处理了
				continue
			}
		}

		// 如果当前匹配与之前的不重叠，且是合理的姓名长度
		if current.Position[0] >= lastEnd {
			nameLen := len([]rune(current.Content))
			if nameLen >= 2 && nameLen <= 3 {
				filtered = append(filtered, current)
				lastEnd = current.Position[1]
			}
		}

		// 处理最后一个匹配
		if i == len(matches)-2 && next.Position[0] >= lastEnd {
			nameLen := len([]rune(next.Content))
			if nameLen >= 2 && nameLen <= 3 {
				filtered = append(filtered, next)
			}
		}
	}

	// 如果没有找到任何匹配，返回原始结果中最可能的姓名
	if len(filtered) == 0 && len(matches) > 0 {
		// 按姓名长度和位置优先级排序
		bestMatches := make([]MatchResult, len(matches))
		copy(bestMatches, matches)

		sort.Slice(bestMatches, func(i, j int) bool {
			iLen := len([]rune(bestMatches[i].Content))
			jLen := len([]rune(bestMatches[j].Content))

			// 优先选择2-3个字符的姓名
			if (iLen == 2 || iLen == 3) && (jLen < 2 || jLen > 3) {
				return true
			}
			if (jLen == 2 || jLen == 3) && (iLen < 2 || iLen > 3) {
				return false
			}

			// 其次按位置排序
			return bestMatches[i].Position[0] < bestMatches[j].Position[0]
		})

		// 返回最佳匹配
		if len(bestMatches) > 2 {
			return bestMatches[:2] // 最多返回2个最佳匹配
		}
		return bestMatches
	}

	return filtered
}

// ReplaceParallel 并行处理敏感信息替换
func (s *RegexSearcher) ReplaceParallel(text string, matchType string) string {
	if text == "" {
		return text
	}

	// 跳过规则名称
	if isRuleName(text) {
		return text
	}

	// 跳过已脱敏的内容
	if strings.Contains(text, "****") {
		return text
	}

	builder := s.pool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		s.pool.Put(builder)
	}()

	lastIndex := 0

	// 获取指定类型的匹配器
	var targetMatcher *Matcher
	for _, m := range s.matchers {
		if m.Name == matchType {
			targetMatcher = m
			break
		}
	}

	if targetMatcher == nil {
		// 没有找到对应的匹配器，直接返回原文本
		return text
	}

	// 获取所有匹配结果
	matches := targetMatcher.Regex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	// 处理匹配结果
	for _, match := range matches {
		builder.WriteString(text[lastIndex:match[0]])
		content := text[match[0]:match[1]]

		if targetMatcher.Validator != nil && !targetMatcher.Validator(content) {
			builder.WriteString(content)
		} else {
			builder.WriteString(targetMatcher.Transformer(content))
		}
		lastIndex = match[1]
	}

	builder.WriteString(text[lastIndex:])
	return builder.String()
}

// registerDefaultMatchers 注册默认的匹配器
func (s *RegexSearcher) registerDefaultMatchers() error {
	matchers := []*Matcher{
		{
			Name:     ChineseName,
			Pattern:  ChineseNamePattern,
			Regex:    ChineseNameRegex,
			Priority: 950, // 降低优先级，让金融卡片优先处理
			Transformer: func(s string) string {
				return ChineseNameDesensitize(s)
			},
		},
		{
			Name:     IDCard,
			Pattern:  ChineseIDCardPattern,
			Regex:    ChineseIDCardRegex,
			Priority: 990,
			Validator: func(s string) bool {
				return ChineseIDCardDesensitize(s)
			},
			Transformer: func(s string) string {
				return IDCardDesensitize(s)
			},
		},
		{
			Name:     Passport,
			Pattern:  PassportPattern,
			Regex:    PassportRegex,
			Priority: 980,
			Transformer: func(s string) string {
				return PassportDesensitize(s)
			},
		},
		{
			Name:     SocialSecurity,
			Pattern:  SocialSecurityPattern,
			Regex:    SocialSecurityRegex,
			Priority: 970,
			Transformer: func(s string) string {
				return SocialSecurityDesensitize(s)
			},
		},
		{
			Name:     DriversLicense,
			Pattern:  DriversLicensePattern,
			Regex:    DriversLicenseRegex,
			Priority: 960,
			Transformer: func(s string) string {
				return DriversLicenseDesensitize(s)
			},
		},
		{
			Name:     MobilePhone,
			Pattern:  MobilePhonePattern,
			Regex:    MobilePhoneRegex,
			Priority: 950,
			Transformer: func(s string) string {
				return MobilePhoneDesensitize(s)
			},
		},
		{
			Name:     FixedPhone,
			Pattern:  FixedPhonePattern,
			Regex:    FixedPhoneRegex,
			Priority: 940,
			Transformer: func(s string) string {
				return FixedPhoneDesensitize(s)
			},
		},
		{
			Name:        Email,
			Pattern:     EmailPattern,
			Regex:       EmailRegex,
			Priority:    930,
			FastFilters: []string{"@"},
			Transformer: func(s string) string {
				return EmailDesensitize(s)
			},
		},
		{
			Name:     Address,
			Pattern:  AddressPattern,
			Regex:    AddressRegex,
			Priority: 920,
			Transformer: func(s string) string {
				return AddressDesensitize(s)
			},
		},
		{
			Name:     PostalCode,
			Pattern:  PostalCodePattern,
			Regex:    PostalCodeRegex,
			Priority: 910,
			Transformer: func(s string) string {
				return PostalCodeDesensitize(s)
			},
		},
		{
			Name:     BankCard,
			Pattern:  BankCardPattern,
			Regex:    BankCardRegex,
			Priority: 975, // 提高优先级，高于社会保障号(970)
			Transformer: func(s string) string {
				return BankCardDesensitize(s)
			},
		},
		{
			Name:     CreditCard,
			Pattern:  CreditCardPattern,
			Regex:    CreditCardRegex,
			Priority: 965, // 高优先级，仅次于中国银行卡(975)
			Transformer: func(s string) string {
				return CreditCardDesensitize(s)
			},
		},
		{
			Name:        IPv4,
			Pattern:     IPv4Pattern,
			Regex:       IPv4Regex,
			Priority:    880,
			FastFilters: []string{"."},
			Transformer: func(s string) string {
				return IPv4Desensitize(s)
			},
		},
		{
			Name:        IPv6,
			Pattern:     IPv6Pattern,
			Regex:       IPv6Regex,
			Priority:    870,
			FastFilters: []string{":"},
			Transformer: func(s string) string {
				return IPv6Desensitize(s)
			},
		},
		{
			Name:        MAC,
			Pattern:     MACPattern,
			Regex:       MACRegex,
			Priority:    860,
			FastFilters: []string{":", "-"},
			Transformer: func(s string) string {
				return MACDesensitize(s)
			},
		},
		{
			Name:     IMEI,
			Pattern:  IMEIPattern,
			Regex:    IMEIRegex,
			Priority: 200, // 大幅降低优先级，避免干扰银行卡
			Transformer: func(s string) string {
				return IMEIDesensitize(s)
			},
		},
		{
			Name:     LicensePlate,
			Pattern:  LicensePlatePattern,
			Regex:    LicensePlateRegex,
			Priority: 840,
			Transformer: func(s string) string {
				return LicensePlateDesensitize(s)
			},
		},
		{
			Name:     VIN,
			Pattern:  VINPattern,
			Regex:    VINRegex,
			Priority: 100, // 大幅降低优先级，避免干扰银行卡
			Transformer: func(s string) string {
				return VINDesensitize(s)
			},
		},
		{
			Name:     APIKey,
			Pattern:  APIKeyPattern,
			Regex:    APIKeyRegex,
			Priority: 820,
			Transformer: func(s string) string {
				return APIKeyDesensitize(s)
			},
		},
		{
			Name:        JWT,
			Pattern:     JWTPattern,
			Regex:       JWTRegex,
			Priority:    810,
			FastFilters: []string{"eyJ"},
			Transformer: func(s string) string {
				return JWTDesensitize(s)
			},
		},
		{
			Name:     AccessToken,
			Pattern:  AccessTokenPattern,
			Regex:    AccessTokenRegex,
			Priority: 800,
			Transformer: func(s string) string {
				return AccessTokenDesensitize(s)
			},
		},
		{
			Name:     DeviceID,
			Pattern:  DeviceIDPattern,
			Regex:    DeviceIDRegex,
			Priority: 790,
			Transformer: func(s string) string {
				return DeviceIDDesensitize(s)
			},
		},
		{
			Name:     UUID,
			Pattern:  UUIDPattern,
			Regex:    UUIDRegex,
			Priority: 780,
			Transformer: func(s string) string {
				return UUIDDesensitize(s)
			},
		},
		{
			Name:     MD5,
			Pattern:  MD5Pattern,
			Regex:    MD5Regex,
			Priority: 770,
			Transformer: func(s string) string {
				return MD5Desensitize(s)
			},
		},
		{
			Name:     SHA1,
			Pattern:  SHA1Pattern,
			Regex:    SHA1Regex,
			Priority: 760,
			Transformer: func(s string) string {
				return SHA1Desensitize(s)
			},
		},
		{
			Name:     SHA256,
			Pattern:  SHA256Pattern,
			Regex:    SHA256Regex,
			Priority: 750,
			Transformer: func(s string) string {
				return SHA256Desensitize(s)
			},
		},
		{
			Name:     LatLng,
			Pattern:  LatLngPattern,
			Regex:    LatLngRegex,
			Priority: 740,
			Transformer: func(s string) string {
				return LatLngDesensitize(s)
			},
		},
		{
			Name:        URL,
			Pattern:     URLPattern,
			Regex:       URLRegex,
			Priority:    730,
			FastFilters: []string{"://"},
			Transformer: func(s string) string {
				return URLDesensitize(s)
			},
		},
		{
			Name:        Domain,
			Pattern:     DomainPattern,
			Regex:       DomainRegex,
			Priority:    720,
			FastFilters: []string{"."},
			Transformer: func(s string) string {
				return DomainDesensitize(s)
			},
		},
		{
			Name:     Password,
			Pattern:  PasswordPattern,
			Regex:    PasswordRegex,
			Priority: 710,
			Transformer: func(s string) string {
				return strings.Repeat("*", len(s))
			},
		},
		{
			Name:     Username,
			Pattern:  UsernamePattern,
			Regex:    UsernameRegex,
			Priority: 400, // 大幅降低优先级，避免干扰银行卡
			Transformer: func(s string) string {
				return UsernameDesensitize(s)
			},
		},
		{
			Name:     MedicalID,
			Pattern:  MedicalIDPattern,
			Regex:    MedicalIDRegex,
			Priority: 500, // 大幅降低优先级，避免干扰银行卡
			Transformer: func(s string) string {
				return MedicalIDDesensitize(s)
			},
		},
		{
			Name:     CompanyID,
			Pattern:  CompanyIDPattern,
			Regex:    CompanyIDRegex,
			Priority: 300, // 大幅降低优先级，避免干扰银行卡
			Transformer: func(s string) string {
				if len(s) > 2 {
					return strings.Repeat("*", len(s)-1) + s[len(s)-1:]
				}
				return strings.Repeat("*", len(s))
			},
		},
		{
			Name:     IBAN,
			Pattern:  IBANPattern,
			Regex:    IBANRegex,
			Priority: 670,
			Transformer: func(s string) string {
				if len(s) > 8 {
					return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
				}
				return strings.Repeat("*", len(s))
			},
		},
		{
			Name:     Swift,
			Pattern:  SwiftPattern,
			Regex:    SwiftRegex,
			Priority: 660,
			Transformer: func(s string) string {
				if len(s) > 4 {
					return s[:4] + strings.Repeat("*", len(s)-4)
				}
				return strings.Repeat("*", len(s))
			},
		},
		{
			Name:     GitRepo,
			Pattern:  GitRepoPattern,
			Regex:    GitRepoRegex,
			Priority: 650,
			Transformer: func(s string) string {
				// 保留协议和域名部分，隐藏仓库具体路径
				if idx := strings.Index(s, "://"); idx != -1 {
					protocol := s[:idx+3]
					rest := s[idx+3:]
					if domainEnd := strings.Index(rest, "/"); domainEnd != -1 {
						domain := rest[:domainEnd]
						return protocol + domain + "/****"
					}
				}
				return s
			},
		},
	}
	// 计算复杂度
	for _, m := range matchers {
		m.Complexity = calculateComplexity(m.Pattern)
	}

	// 根据复杂度和优先级排序
	sort.Slice(matchers, func(i, j int) bool {
		if matchers[i].Complexity == matchers[j].Complexity {
			return matchers[i].Priority > matchers[j].Priority
		}
		return matchers[i].Complexity > matchers[j].Complexity
	})

	// 添加到匹配器列表
	for _, m := range matchers {
		if err := s.AddMatcher(m); err != nil {
			return fmt.Errorf("failed to add matcher %s: %v", m.Name, err)
		}
	}

	return nil
}

// isRuleName 检查是否为规则名称
func isRuleName(text string) bool {
	ruleNames := []string{
		// 个人信息
		"chinese_name",
		"id_card",
		"passport",
		"drivers_license",
		"nickname",
		"biography",
		"signature",
		"social_security",

		// 联系方式
		"mobile_phone",
		"landline",
		"email",
		"address",

		// 账户信息
		"bank_card",
		"credit_card",
		"username",
		"password",

		// 设备信息
		"ipv4",
		"ipv6",
		"mac",
		"device_id",
		"imei",

		// 证件信息
		"medical_id",
		"company_id",
		"postal_code",

		// 车辆信息
		"plate",
		"vin",

		// 安全凭证
		"jwt",
		"access_token",
		"refresh_token",
		"private_key",
		"public_key",
		"certificate",

		// 内容相关
		"comment",
		"coordinate",

		// 通用处理
		"url",
		"first_mask",
		"null",
		"empty",
	}

	for _, name := range ruleNames {
		if text == name {
			return true
		}
	}
	return false
}

// getTypesVersion 获取类型版本号（用于缓存失效）
func (s *RegexSearcher) getTypesVersion() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// ReplaceAllTypes 一次性处理所有类型的敏感信息（重新设计的算法）
func (s *RegexSearcher) ReplaceAllTypes(text string) string {
	if text == "" {
		return text
	}

	// 跳过规则名称
	if isRuleName(text) {
		return text
	}

	// 跳过已脱敏的内容
	if strings.Contains(text, "****") {
		return text
	}

	s.mu.RLock()
	matchers := s.matchers
	s.mu.RUnlock()

	// 第一步：收集所有匹配
	type Match struct {
		Start    int
		End      int
		Content  string
		Matcher  *Matcher
		Priority int
	}

	var allMatches []Match

	// 收集所有类型的匹配
	for _, matcher := range matchers {
		// 快速预筛选
		if !s.quickFilter(text, matcher) {
			continue
		}

		matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			content := text[match[0]:match[1]]

			// 验证匹配内容
			if matcher.Validator != nil && !matcher.Validator(content) {
				continue
			}

			allMatches = append(allMatches, Match{
				Start:    match[0],
				End:      match[1],
				Content:  content,
				Matcher:  matcher,
				Priority: matcher.Priority,
			})
		}
	}

	if len(allMatches) == 0 {
		return text
	}

	// 第二步：按位置排序，解决重叠冲突
	// 先按开始位置排序，再按优先级排序
	sort.Slice(allMatches, func(i, j int) bool {
		if allMatches[i].Start == allMatches[j].Start {
			// 开始位置相同，优先级高的在前
			return allMatches[i].Priority > allMatches[j].Priority
		}
		return allMatches[i].Start < allMatches[j].Start
	})

	// 第三步：解决重叠冲突，保留优先级最高的匹配
	var finalMatches []Match
	for _, match := range allMatches {
		// 检查是否与已选择的匹配重叠
		overlap := false
		for _, final := range finalMatches {
			if match.Start < final.End && match.End > final.Start {
				// 有重叠，跳过当前匹配
				overlap = true
				break
			}
		}

		if !overlap {
			finalMatches = append(finalMatches, match)
		}
	}

	// 第四步：按位置倒序应用替换（避免位置偏移）
	sort.Slice(finalMatches, func(i, j int) bool {
		return finalMatches[i].Start > finalMatches[j].Start
	})

	result := text
	for _, match := range finalMatches {
		replacement := match.Matcher.Transformer(match.Content)
		result = result[:match.Start] + replacement + result[match.End:]
	}

	return result
}

// quickFilter 快速预筛选，避免不必要的正则表达式执行
func (s *RegexSearcher) quickFilter(text string, matcher *Matcher) bool {
	switch matcher.Name {
	case MobilePhone:
		// 手机号必须包含数字
		return strings.ContainsAny(text, "0123456789") && len(text) >= 11
	case Email:
		// 邮箱必须包含@和.
		return strings.Contains(text, "@") && strings.Contains(text, ".")
	case IPv4:
		// IPv4必须包含.
		return strings.Contains(text, ".")
	case ChineseName:
		// 中文姓名必须包含中文字符
		for _, r := range text {
			if r >= '\u4e00' && r <= '\u9fff' {
				// 检查是否在金融关键词附近，如果是则跳过姓名匹配
				if s.containsFinancialContext(text) {
					return false
				}
				return true
			}
		}
		return false
	case IDCard:
		// 身份证号必须包含数字且长度足够
		return strings.ContainsAny(text, "0123456789") && len(text) >= 15
	default:
		// 其他类型不做预筛选
		return true
	}
}

// containsFinancialContext 检查文本是否包含金融上下文关键词
func (s *RegexSearcher) containsFinancialContext(text string) bool {
	financialKeywords := []string{
		"银行", "银行卡", "信用卡", "借记卡", "储蓄卡", "金融卡",
		"工商银行", "建设银行", "农业银行", "中国银行", "交通银行",
		"招商银行", "浦发银行", "兴业银行", "民生银行", "光大银行",
		"中信银行", "华夏银行", "平安银行", "广发银行",
		"卡号", "账号", "账户", "余额", "转账", "支付", "收款",
	}

	for _, keyword := range financialKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// DetectAllTypes 一次性检测所有类型的敏感信息（优化版本）
func (s *RegexSearcher) DetectAllTypes(text string) map[string][]MatchResult {
	if text == "" {
		return nil
	}

	s.mu.RLock()
	matchers := make([]*Matcher, len(s.matchers))
	copy(matchers, s.matchers)
	s.mu.RUnlock()

	results := make(map[string][]MatchResult)

	// 并发检测所有类型
	for _, matcher := range matchers {
		var typeResults []MatchResult

		// 中文姓名特殊处理
		if matcher.Name == ChineseName {
			namePattern := regexp.MustCompile("[\u4e00-\u9fa5]{2,}")
			matches := namePattern.FindAllStringSubmatchIndex(text, -1)

			var filteredResults []MatchResult
			for _, match := range matches {
				content := text[match[0]:match[1]]
				chineseChars := []rune(content)

				if len(chineseChars) <= 3 {
					if matcher.Regex.MatchString(content) {
						filteredResults = append(filteredResults, MatchResult{
							Type:     matcher.Name,
							Content:  content,
							Position: [2]int{match[0], match[1]},
						})
					}
					continue
				}

				candidateNames := extractChineseNames(chineseChars)
				for _, name := range candidateNames {
					nameStr := string(name.chars)
					if matcher.Regex.MatchString(nameStr) {
						startPos := match[0] + name.startIndex*3
						endPos := startPos + len(name.chars)*3
						filteredResults = append(filteredResults, MatchResult{
							Type:     matcher.Name,
							Content:  nameStr,
							Position: [2]int{startPos, endPos},
						})
					}
				}
			}
			typeResults = filterOverlappingNames(filteredResults)
		} else {
			// 常规匹配
			matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)
			for _, match := range matches {
				content := text[match[0]:match[1]]

				if matcher.Validator != nil && !matcher.Validator(content) {
					continue
				}

				typeResults = append(typeResults, MatchResult{
					Type:     matcher.Name,
					Content:  content,
					Position: [2]int{match[0], match[1]},
				})
			}
		}

		if len(typeResults) > 0 {
			results[matcher.Name] = typeResults
		}
	}

	return results
}
