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
		"艾|安|敖|巴|白|班|包|暴|鲍|贝|贲|毕|边|卞|别|邴|伯|薄|卜|蔡|曹|岑|柴|昌|常|晁|车|陈|成|程|池|充|仇|储|楚|褚|淳|从|崔|戴|党|邓|狄|刁|丁|董|窦|杜|端|段|鄂|樊|范|方|房|费|丰|封|冯|凤|伏|扶|符|福|傅|甘|高|郜|戈|盖|葛|耿|龚|宫|勾|苟|辜|古|谷|顾|关|管|桂|郭|国|韩|杭|郝|何|和|贺|赫|衡|洪|侯|胡|扈|花|华|滑|怀|宦|黄|惠|霍|姬|嵇|吉|汲|籍|计|纪|季|贾|简|姜|江|蒋|焦|金|靳|荆|井|景|居|鞠|阚|康|柯|空|孔|寇|蒯|匡|邝|况|赖|蓝|郎|劳|雷|冷|黎|李|利|连|廉|练|梁|廖|林|蔺|凌|令|刘|柳|龙|隆|娄|卢|鲁|陆|路|逯|禄|吕|栾|罗|骆|麻|马|满|毛|茅|梅|蒙|孟|米|宓|闵|明|莫|牟|穆|倪|聂|年|宁|牛|钮|农|潘|庞|裴|彭|皮|平|蒲|濮|浦|戚|祁|齐|钱|强|乔|谯|秦|邱|裘|曲|屈|瞿|全|阙|冉|饶|任|荣|容|阮|芮|桑|沙|山|单|商|上|邵|佘|申|沈|盛|师|施|时|石|史|寿|殳|舒|束|双|水|司|松|宋|苏|宿|孙|索|邰|太|谈|谭|汤|唐|陶|滕|田|通|童|涂|屠|万|汪|王|危|韦|卫|魏|温|文|闻|翁|巫|邬|伍|武|务|西|席|夏|咸|向|项|萧|谢|辛|邢|幸|熊|徐|许|轩|宣|薛|荀|闫|严|言|阎|颜|晏|燕|杨|姚|叶|伊|易|殷|尹|应|庸|雍|尤|游|于|余|俞|虞|元|袁|岳|云|臧|曾|翟|詹|湛|张|章|赵|甄|郑|支|钟|仲|周|朱|诸|祝|庄|卓|子|宗|邹|祖|左" +
		")"
	ChineseNamePattern    = "(?:" + ChineseSurnames + ")[\u4e00-\u9fa5]{1,5}"                                     // 中文姓名：百家姓+名字
	ChineseIDCardPattern  = "[1-9]\\d{5}(?:18|19|20)\\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]" // 身份证
	PassportPattern       = "[a-zA-Z][0-9]{9}"                                                                    // 护照号
	SocialSecurityPattern = "[1-9]\\d{17}[\\dXx]"                                                                 // 社会保障号
	DriversLicensePattern = "[1-9]\\d{5}[a-zA-Z]\\d{6}"                                                           // 驾驶证号

	// 联系方式
	MobilePhonePattern = "(?:(?:\\+|00)86)?1(?:(?:3[\\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\\d])|(?:9[189]))\\d{8}" // 手机号
	FixedPhonePattern  = "(?:[\\d]{3,4}-)?[\\d]{7,8}(?:-[\\d]{1,4})?"                                                                   // 固定电话
	EmailPattern       = `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`                                                               // 电子邮箱

	// 地址信息
	AddressPattern    = "[\u4e00-\u9fa5]{2,}(?:省|自治区|市|特别行政区|自治州)?[\u4e00-\u9fa5]{2,}(?:市|区|县|镇|村|街道|路|号楼|栋|单元|室)" // 详细地址
	PostalCodePattern = `[1-9]\d{5}([^\d]|$)`                                                                      // 邮政编码

	// 金融信息
	BankCardPattern   = `(?:(?:4\d{12}(?:\d{3})?)|(?:5[1-5]\d{14})|(?:6(?:011|5\d{2})\d{12})|(?:3[47]\d{13})|(?:(?:30[0-5]|36\d|38\d)\d{11}))`                                // 银行卡
	CreditCardPattern = `(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\d{3})\d{11})` // 信用卡

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
	UsernamePattern = `[a-zA-Z0-9_-]{3,16}` // 用户名

	// 证件号码
	MedicalIDPattern = `[1-9]\d{7}`  // 医保卡号
	CompanyIDPattern = `[1-9]\d{14}` // 统一社会信用代码

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

	var results []MatchResult
	matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)
	if matches == nil {
		return nil
	}

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

	builder := textBuilderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		textBuilderPool.Put(builder)
	}()

	lastIndex := 0
	for _, matcher := range s.matchers {
		if matcher.Name != matchType {
			continue
		}

		matches := matcher.Regex.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			builder.WriteString(text[lastIndex:match[0]])
			content := text[match[0]:match[1]]

			if matcher.Validator != nil && !matcher.Validator(content) {
				builder.WriteString(content)
			} else {
				builder.WriteString(matcher.Transformer(content))
			}
			lastIndex = match[1]
		}
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
			Priority: 1000,
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
				return IdCardDesensitize(s)
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
			Name:     Email,
			Pattern:  EmailPattern,
			Regex:    EmailRegex,
			Priority: 930,
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
			Priority: 900,
			Transformer: func(s string) string {
				return BankCardDesensitize(s)
			},
		},
		{
			Name:     CreditCard,
			Pattern:  CreditCardPattern,
			Regex:    CreditCardRegex,
			Priority: 890,
			Validator: func(s string) bool {
				return validateCreditCard(s)
			},
			Transformer: func(s string) string {
				return CreditCardDesensitize(s)
			},
		},
		{
			Name:     IPv4,
			Pattern:  IPv4Pattern,
			Regex:    IPv4Regex,
			Priority: 880,
			Transformer: func(s string) string {
				return IPv4Desensitize(s)
			},
		},
		{
			Name:     IPv6,
			Pattern:  IPv6Pattern,
			Regex:    IPv6Regex,
			Priority: 870,
			Transformer: func(s string) string {
				return IPv6Desensitize(s)
			},
		},
		{
			Name:     MAC,
			Pattern:  MACPattern,
			Regex:    MACRegex,
			Priority: 860,
			Transformer: func(s string) string {
				return MACDesensitize(s)
			},
		},
		{
			Name:     IMEI,
			Pattern:  IMEIPattern,
			Regex:    IMEIRegex,
			Priority: 850,
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
			Priority: 830,
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
			Name:     JWT,
			Pattern:  JWTPattern,
			Regex:    JWTRegex,
			Priority: 810,
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
			Name:     URL,
			Pattern:  URLPattern,
			Regex:    URLRegex,
			Priority: 730,
			Transformer: func(s string) string {
				return URLDesensitize(s)
			},
		},
		{
			Name:     Domain,
			Pattern:  DomainPattern,
			Regex:    DomainRegex,
			Priority: 720,
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
			Priority: 700,
			Transformer: func(s string) string {
				return UsernameDesensitize(s)
			},
		},
		{
			Name:     MedicalID,
			Pattern:  MedicalIDPattern,
			Regex:    MedicalIDRegex,
			Priority: 690,
			Transformer: func(s string) string {
				return MedicalIDDesensitize(s)
			},
		},
		{
			Name:     CompanyID,
			Pattern:  CompanyIDPattern,
			Regex:    CompanyIDRegex,
			Priority: 680,
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
