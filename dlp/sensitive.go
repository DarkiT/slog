package dlp

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

type Sensitive struct {
	Name           string `dlp:"chinese_name" json:"name,omitempty"`               // 姓名
	IdCard         string `dlp:"id_card" json:"id_card,omitempty"`                 // 身份证
	FixedPhone     string `dlp:"fixed_phone" json:"fixed_phone,omitempty"`         // 固定电话
	MobilePhone    string `dlp:"mobile_phone" json:"mobile_phone,omitempty"`       // 手机号
	Address        string `dlp:"address" json:"address,omitempty"`                 // 地址
	Email          string `dlp:"email" json:"email,omitempty"`                     // 邮箱
	Password       string `dlp:"password" json:"password,omitempty"`               // 密码
	CarLicense     string `dlp:"car_license" json:"car_license,omitempty"`         // 车牌
	BankCard       string `dlp:"bank_card" json:"bank_card,omitempty"`             // 银行卡
	Ipv4           string `dlp:"ipv4" json:"ipv_4,omitempty"`                      // IPv4
	Ipv6           string `dlp:"ipv6" json:"ipv_6,omitempty"`                      // IPv6
	Base64         string `dlp:"base64" json:"base_64,omitempty"`                  // Base64编码
	URL            string `dlp:"url" json:"url,omitempty"`                         // URL
	FirstMask      string `dlp:"first_mask" json:"first_mask,omitempty"`           // 仅保留首字符
	ClearToNull    string `dlp:"null" json:"null,omitempty"`                       // 清空为null
	ClearToEmpty   string `dlp:"empty" json:"empty,omitempty"`                     // 清空为空字符串
	JWT            string `dlp:"jwt" json:"jwt,omitempty"`                         // JWT令牌
	SocialSecurity string `dlp:"social_security" json:"social_security,omitempty"` // 社会保障号
	Passport       string `dlp:"passport" json:"passport,omitempty"`               // 护照号
	DriversLicense string `dlp:"drivers_license" json:"drivers_license,omitempty"` // 驾驶证号
	MedicalID      string `dlp:"medical_id" json:"medical_id,omitempty"`           // 医保卡号
	CompanyID      string `dlp:"company_id" json:"company_id,omitempty"`           // 公司编号
	DeviceID       string `dlp:"device_id" json:"device_id,omitempty"`             // 设备ID
	MAC            string `dlp:"mac" json:"mac,omitempty"`                         // MAC地址
	VIN            string `dlp:"vin" json:"vin,omitempty"`                         // 车架号
	IMEI           string `dlp:"imei" json:"imei,omitempty"`                       // IMEI号
	Coordinate     string `dlp:"coordinate" json:"coordinate,omitempty"`           // 地理坐标
	AccessToken    string `dlp:"access_token" json:"access_token,omitempty"`       // 访问令牌
	RefreshToken   string `dlp:"refresh_token" json:"refresh_token,omitempty"`     // 刷新令牌
	PrivateKey     string `dlp:"private_key" json:"private_key,omitempty"`         // 私钥
	PublicKey      string `dlp:"public_key" json:"public_key,omitempty"`           // 公钥
	Certificate    string `dlp:"certificate" json:"certificate,omitempty"`         // 证书
	Username       string `dlp:"username" json:"username,omitempty"`               // 用户名
	Nickname       string `dlp:"nickname" json:"nickname,omitempty"`               // 昵称
	Biography      string `dlp:"biography" json:"biography,omitempty"`             // 个人简介
	Signature      string `dlp:"signature" json:"signature,omitempty"`             // 个性签名
	Comment        string `dlp:"comment" json:"comment,omitempty"`                 // 评论内容
}

// 错误定义
var (
	ErrNotStruct   = errors.New("input must be a struct")
	ErrInvalidKey  = errors.New("invalid key size")
	ErrInvalidData = errors.New("invalid data")

	// 正则表达式缓存
	regexpCache sync.Map
	// 脱敏处理缓存池
	maskPool = sync.Pool{
		New: func() interface{} {
			return new(strings.Builder)
		},
	}
	// 全局变量，存储需要脱敏的URL参数名
	sensitiveURLParams = []string{
		"token",
		"key",
		"secret",
		"password",
		"auth",
		"access_token",
		"refresh_token",
		"api_key",
	}
)

// DesensitizeFunc 定义脱敏函数类型
type DesensitizeFunc func(string) string

// ProcessSensitiveData 处理结构体的脱敏
func ProcessSensitiveData(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ErrNotStruct
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

		if strategy := getDesensitizeFunc(tag); strategy != nil {
			if field.Kind() == reflect.String {
				field.SetString(strategy(field.String()))
			}
		}
	}

	return nil
}

// 获取脱敏策略函数
func getDesensitizeFunc(tag string) DesensitizeFunc {
	switch tag {
	case "chinese_name":
		return ChineseNameDesensitize
	case "id_card":
		return IdCardDesensitize
	case "fixed_phone":
		return FixedPhoneDesensitize
	case "mobile_phone":
		return MobilePhoneDesensitize
	case "address":
		return AddressDesensitize
	case "email":
		return EmailDesensitize
	case "password":
		return PasswordDesensitize
	case "car_license":
		return CarLicenseDesensitize
	case "bank_card":
		return BankCardDesensitize
	case "ipv4":
		return Ipv4Desensitize
	case "ipv6":
		return Ipv6Desensitize
	case "url":
		return urlDesensitize
	case "first_mask":
		return FirstMaskDesensitize
	case "null":
		return ClearToNullDesensitize
	case "empty":
		return ClearToEmptyDesensitize
	case "jwt":
		return JWTDesensitize
	case "social_security":
		return SocialSecurityDesensitize
	case "passport":
		return PassportDesensitize
	case "drivers_license":
		return DriversLicenseDesensitize
	case "medical_id":
		return MedicalIDDesensitize
	case "company_id":
		return CompanyIDDesensitize
	case "device_id":
		return DeviceIDDesensitize
	case "mac":
		return MACDesensitize
	case "vin":
		return VINDesensitize
	case "imei":
		return IMEIDesensitize
	case "coordinate":
		return CoordinateDesensitize
	case "access_token":
		return AccessTokenDesensitize
	case "refresh_token":
		return RefreshTokenDesensitize
	case "private_key":
		return PrivateKeyDesensitize
	case "public_key":
		return PublicKeyDesensitize
	case "certificate":
		return CertificateDesensitize
	case "username":
		return UsernameDesensitize
	case "nickname":
		return NicknameDesensitize
	case "biography":
		return BiographyDesensitize
	case "signature":
		return SignatureDesensitize
	case "comment":
		return CommentDesensitize
	default:
		return nil
	}
}

// 以下是所有脱敏策略的具体实现

// urlDesensitize URL脱敏实现
func urlDesensitize(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// 脱敏用户名和密码
	if parsedURL.User != nil {
		parsedURL.User = url.UserPassword("****", "****")
	}

	// 脱敏主机名中的IP地址
	host := parsedURL.Hostname()
	port := parsedURL.Port()

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			host = Ipv4Desensitize(host)
		} else {
			host = Ipv6Desensitize(host)
		}

		if port != "" {
			parsedURL.Host = net.JoinHostPort(host, port)
		} else {
			parsedURL.Host = host
		}
	}

	// 脱敏查询参数中的敏感信息
	values := parsedURL.Query()
	sensitiveParams := []string{"key", "token", "secret", "password", "auth"}

	for key := range values {
		for _, param := range sensitiveParams {
			if strings.Contains(strings.ToLower(key), param) {
				values.Set(key, "****")
			}
		}
	}
	parsedURL.RawQuery = values.Encode()

	return parsedURL.String()
}

// ClearToNullDesensitize 清空为null的脱敏实现
func ClearToNullDesensitize(data string) string {
	return ""
}

// ClearToEmptyDesensitize 清空为空字符串的脱敏实现
func ClearToEmptyDesensitize(data string) string {
	return ""
}

// RegisterURLSensitiveParams 添加一个便捷方法用于注册自定义的URL参数脱敏规则
func RegisterURLSensitiveParams(params ...string) {
	sensitiveURLParams = append(sensitiveURLParams, params...)
}

func ChineseNameDesensitize(name string) string {
	if len(name) <= 1 {
		return name
	}
	runes := []rune(name)
	if len(runes) <= 1 {
		return name
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

func IdCardDesensitize(idCard string) string {
	if len(idCard) <= 10 {
		return idCard
	}
	return idCard[:6] + strings.Repeat("*", len(idCard)-10) + idCard[len(idCard)-4:]
}

func FixedPhoneDesensitize(phone string) string {
	if len(phone) <= 6 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-5) + phone[len(phone)-2:]
}

func MobilePhoneDesensitize(phone string) string {
	if len(phone) <= 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}

func AddressDesensitize(address string) string {
	runes := []rune(address)
	length := len(runes)
	if length <= 8 {
		return strings.Repeat("*", length)
	}
	return string(runes[:length-8]) + strings.Repeat("*", 8)
}

func EmailDesensitize(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	username := parts[0]
	if len(username) <= 1 {
		return email
	}
	return username[:1] + strings.Repeat("*", len(username)-1) + "@" + parts[1]
}

func PasswordDesensitize(password string) string {
	if len(password) == 0 {
		return password
	}
	return strings.Repeat("*", len(password))
}

func CarLicenseDesensitize(license string) string {
	runes := []rune(license)
	if len(runes) <= 4 {
		return license
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-3) + string(runes[len(runes)-1:])
}

func BankCardDesensitize(card string) string {
	if len(card) <= 8 {
		return card
	}
	return card[:6] + strings.Repeat("*", len(card)-10) + card[len(card)-4:]
}

func Ipv4Desensitize(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + ".*.*." + parts[3]
}

func Ipv6Desensitize(ip string) string {
	parts := strings.Split(ip, ":")
	if len(parts) < 4 {
		return ip
	}
	return parts[0] + ":" + parts[1] + ":****:" + parts[len(parts)-1]
}

func JWTDesensitize(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}
	return parts[0] + ".****." + parts[2]
}

func SocialSecurityDesensitize(ssn string) string {
	if len(ssn) != 11 {
		return ssn
	}
	return ssn[:3] + "-**-" + ssn[7:]
}

func PassportDesensitize(passport string) string {
	if len(passport) < 6 {
		return passport
	}
	return passport[:2] + strings.Repeat("*", len(passport)-4) + passport[len(passport)-2:]
}

func DriversLicenseDesensitize(license string) string {
	if len(license) < 8 {
		return license
	}
	return license[:4] + strings.Repeat("*", len(license)-6) + license[len(license)-2:]
}

func MedicalIDDesensitize(id string) string {
	if len(id) < 8 {
		return id
	}
	return id[:3] + strings.Repeat("*", len(id)-6) + id[len(id)-3:]
}

func CompanyIDDesensitize(id string) string {
	if len(id) < 6 {
		return id
	}
	return id[:2] + strings.Repeat("*", len(id)-4) + id[len(id)-2:]
}

func DeviceIDDesensitize(id string) string {
	if len(id) < 8 {
		return id
	}
	return id[:4] + strings.Repeat("*", len(id)-8) + id[len(id)-4:]
}

func MACDesensitize(mac string) string {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return mac
	}
	return parts[0] + ":**:**:**:**:" + parts[5]
}

func VINDesensitize(vin string) string {
	if len(vin) != 17 {
		return vin
	}
	return vin[:3] + strings.Repeat("*", 11) + vin[14:]
}

func IMEIDesensitize(imei string) string {
	if len(imei) != 15 {
		return imei
	}
	return imei[:4] + strings.Repeat("*", 7) + imei[11:]
}

func CoordinateDesensitize(coord string) string {
	parts := strings.Split(coord, ",")
	if len(parts) != 2 {
		return coord
	}
	return "**.****,**.****"
}

func AccessTokenDesensitize(token string) string {
	if len(token) < 8 {
		return token
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

func RefreshTokenDesensitize(token string) string {
	if len(token) < 8 {
		return token
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

func PrivateKeyDesensitize(key string) string {
	return "[PRIVATE_KEY]"
}

func PublicKeyDesensitize(key string) string {
	if len(key) < 20 {
		return key
	}
	return key[:10] + "..." + key[len(key)-10:]
}

func CertificateDesensitize(cert string) string {
	if len(cert) < 20 {
		return cert
	}
	return "-----BEGIN CERTIFICATE-----\n****\n-----END CERTIFICATE-----"
}

func UsernameDesensitize(username string) string {
	if len(username) <= 2 {
		return username
	}
	return username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
}

func NicknameDesensitize(nickname string) string {
	runes := []rune(nickname)
	if len(runes) <= 1 {
		return nickname
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

func BiographyDesensitize(bio string) string {
	runes := []rune(bio)
	if len(runes) <= 10 {
		return bio
	}
	return string(runes[:5]) + "..." + string(runes[len(runes)-5:])
}

func SignatureDesensitize(signature string) string {
	runes := []rune(signature)
	if len(runes) <= 4 {
		return signature
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

func CommentDesensitize(comment string) string {
	runes := []rune(comment)
	if len(runes) <= 10 {
		return comment
	}
	return string(runes[:5]) + "..." + string(runes[len(runes)-5:])
}

// 高级加密脱敏方法
func Base64Desensitize(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func AesDesensitize(data, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return hex.EncodeToString(ciphertext), nil
}

func DesDesensitize(data string, key []byte) (string, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := make([]byte, des.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	padded := pkcs7Padding([]byte(data), des.BlockSize)
	ciphertext := make([]byte, len(padded))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	return hex.EncodeToString(append(iv, ciphertext...)), nil
}

func RsaDesensitize(data []byte, publicKey *rsa.PublicKey) (string, error) {
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, data, nil)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// 自定义脱敏方法
func FirstMaskDesensitize(data string) string {
	if len(data) <= 1 {
		return data
	}
	return data[:1] + strings.Repeat("*", len(data)-1)
}

func CustomizeKeepLengthDesensitize(data string, preKeep, postKeep int) string {
	runes := []rune(data)
	length := len(runes)

	if length <= preKeep+postKeep {
		return data
	}

	return string(runes[:preKeep]) + strings.Repeat("*", length-preKeep-postKeep) + string(runes[length-postKeep:])
}

func StringDesensitize(data string, filterWords ...string) string {
	builder := maskPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		maskPool.Put(builder)
	}()

	for _, word := range filterWords {
		replacement := strings.Repeat("*", utf8.RuneCountInString(word))
		data = strings.ReplaceAll(data, word, replacement)
	}

	return data
}

func UrlDesensitize(rawUrl string, args ...string) string {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return rawUrl
	}

	// 脱敏用户名和密码
	if parsedUrl.User != nil {
		parsedUrl.User = url.UserPassword("****", "****")
	}

	// 脱敏IP地址
	host, port, err := net.SplitHostPort(parsedUrl.Host)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if ip.To4() != nil {
				parsedUrl.Host = Ipv4Desensitize(host)
			} else {
				parsedUrl.Host = Ipv6Desensitize(host)
			}
			if port != "" {
				parsedUrl.Host += ":" + port
			}
		}
	}

	// 脱敏查询参数
	if len(args) > 0 {
		values := parsedUrl.Query()
		for key := range values {
			for _, sensitiveKey := range args {
				if strings.Contains(strings.ToLower(key), strings.ToLower(sensitiveKey)) {
					values.Set(key, "****")
				}
			}
		}
		parsedUrl.RawQuery = values.Encode()
	}

	return parsedUrl.String()
}

// 辅助方法
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// 正则表达式编译结果缓存
func getRegexp(pattern string) (*regexp.Regexp, error) {
	if v, ok := regexpCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexpCache.Store(pattern, reg)
	return reg, nil
}

// 性能优化：使用 strings.Builder 进行字符串拼接
func maskString(str string, start, end int, maskChar string) string {
	builder := maskPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		maskPool.Put(builder)
	}()

	builder.WriteString(str[:start])
	builder.WriteString(strings.Repeat(maskChar, len(str)-start-end))
	builder.WriteString(str[len(str)-end:])

	return builder.String()
}

// 初始化：预编译正则表达式
func init() {
	patterns := []string{
		`\d{17}[\dXx]`,                // 身份证
		`\d{11}`,                      // 手机号
		`[0-9a-zA-Z]{16,19}`,          // 银行卡
		`[\w\-.]+@[\w\-]+\.[a-zA-Z]+`, // 邮箱
		// ... 其他正则表达式
	}

	for _, pattern := range patterns {
		if reg, err := regexp.Compile(pattern); err == nil {
			regexpCache.Store(pattern, reg)
		}
	}
}
