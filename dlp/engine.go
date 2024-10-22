package dlp

import (
	"reflect"
	"regexp"
	"sync"
	"sync/atomic"
)

// DlpEngine 定义脱敏引擎的完整功能
type DlpEngine struct {
	config   *DlpConfig
	searcher *RegexSearcher
	mu       sync.RWMutex
	enabled  atomic.Bool
	patterns map[string]*regexp.Regexp
}

func NewDlpEngine() *DlpEngine {
	engine := &DlpEngine{
		config: GetConfig(),
	}
	engine.initPatterns()
	return engine
}

// initPatterns 初始化正则表达式模式
func (e *DlpEngine) initPatterns() {
	e.patterns = make(map[string]*regexp.Regexp)

	// 加载所有预定义的正则表达式
	patterns := map[string]string{
		"chinese_name":    ChineseNamePattern,    // 中文姓名
		"id_card":         ChineseIDCardPattern,  // 身份证
		"passport":        PassportPattern,       // 护照号
		"social_security": SocialSecurityPattern, // 社会保障号
		"drivers_license": DriversLicensePattern, // 驾驶证号
		"mobile_phone":    MobilePhonePattern,    // 手机号
		"fixed_phone":     FixedPhonePattern,     // 固定电话
		"email":           EmailPattern,          // 电子邮箱
		"address":         AddressPattern,        // 详细地址
		"postal_code":     PostalCodePattern,     // 邮政编码
		"bank_card":       BankCardPattern,       // 银行卡
		"credit_card":     CreditCardPattern,     // 信用卡
		"ipv4":            IPv4Pattern,           // IPv4
		"ipv6":            IPv6Pattern,           // IPv6
		"mac":             MACPattern,            // MAC地址
		"imei":            IMEIPattern,           // IMEI号
		"car_license":     CarLicensePattern,     // 车牌号
		"vin":             VINPattern,            // 车架号
		"api_key":         APIKeyPattern,         // API密钥
		"jwt":             JWTPattern,            // JWT令牌
		"access_token":    AccessTokenPattern,    // 访问令牌
		"device_id":       DeviceIDPattern,       // 设备ID
		"uuid":            UUIDPattern,           // UUID
		"md5":             MD5Pattern,            // MD5哈希
		"sha1":            SHA1Pattern,           // SHA1哈希
		"sha256":          SHA256Pattern,         // SHA256哈希
		"base64":          Base64Pattern,         // Base64编码
		"lat_lng":         LatLngPattern,         // 经纬度
		"url":             URLPattern,            // URL
		"domain":          DomainPattern,         // 域名
		"password":        PasswordPattern,       // 密码
		"username":        UsernamePattern,       // 用户名
		"medical_id":      MedicalIDPattern,      // 医保卡号
		"company_id":      CompanyIDPattern,      // 统一社会信用代码
		"iban":            IBANPattern,           // IBAN号码
		"swift":           SwiftPattern,          // SWIFT代码
		"git_repo":        GitRepoPattern,        // Git仓库
		"commit_hash":     CommitHashPattern,     // Git提交哈希
		"date":            DatePattern,           // 日期
		"time":            TimePattern,           // 时间
		"ip_port":         IPPortPattern,         // 端口号
	}

	for name, pattern := range patterns {
		if reg, err := regexp.Compile(pattern); err == nil {
			e.patterns[name] = reg
		}
	}
}

// getStrategyForPattern 根据正则表达式获取对应的脱敏策略
func (e *DlpEngine) getStrategyForPattern(pattern *regexp.Regexp) (DesensitizeFunc, bool) {
	patternStr := pattern.String()
	for name, p := range e.patterns {
		if p.String() == patternStr {
			return e.config.GetStrategy(name)
		}
	}
	return nil, false
}

// DesensitizeStruct 对结构体进行脱敏
func (e *DlpEngine) DesensitizeStruct(data interface{}) error {
	if !e.config.IsEnabled() {
		return nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("dlp")
		if tag == "" {
			continue
		}

		if strategy, ok := e.config.GetStrategy(tag); ok {
			if field.Kind() == reflect.String {
				field.SetString(strategy(field.String()))
			}
		}
	}

	return nil
}

// DesensitizeText 对文本内容进行脱敏
func (e *DlpEngine) DesensitizeText(text string) string {
	if !e.config.IsEnabled() {
		return text
	}

	// 使用正则表达式进行内容匹配和脱敏
	for _, pattern := range e.patterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			if strategy, ok := e.getStrategyForPattern(pattern); ok {
				return strategy(match)
			}
			return match
		})
	}

	return text
}
