package dlp

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Sensitive 定义敏感信息结构体
type Sensitive struct {
	Name           string `dlp:"chinese_name" json:"name,omitempty"`               // 姓名
	IdCard         string `dlp:"id_card" json:"id_card,omitempty"`                 // 身份证
	FixedPhone     string `dlp:"landline" json:"landline,omitempty"`               // 固定电话
	MobilePhone    string `dlp:"mobile" json:"mobile,omitempty"`                   // 手机号
	Address        string `dlp:"address" json:"address,omitempty"`                 // 地址
	Email          string `dlp:"email" json:"email,omitempty"`                     // 邮箱
	Password       string `dlp:"password" json:"password,omitempty"`               // 密码
	LicensePlate   string `dlp:"plate" json:"plate,omitempty"`                     // 车牌
	BankCard       string `dlp:"bank_card" json:"bank_card,omitempty"`             // 银行卡
	CreditCard     string `dlp:"credit_card" json:"credit_card,omitempty"`         // 信用卡
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
	DriversLicense string `dlp:"license_number" json:"license_number,omitempty"`   // 驾驶证号
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

var (
	// 脱敏处理缓存池
	maskPool = sync.Pool{
		New: func() interface{} {
			return new(strings.Builder)
		},
	}
	// 全局变量，存储需要脱敏的URL参数名
	sensitiveURLParams = []string{
		// 认证相关
		"token",
		"access_token",
		"refresh_token",
		"id_token",
		"bearer",
		"jwt",

		// 密钥相关
		"key",
		"api_key",
		"apikey",
		"secret",
		"secret_key",
		"private_key",
		"public_key",

		// 密码相关
		"password",
		"passwd",
		"pwd",
		"auth",
		"authentication",
		"credentials",
		"access_token",
		"refresh_token",
		"api_key",

		// 个人信息相关
		"ssn", // 社会安全号
		"sin", // 社会保险号
		"credit_card",
		"card_number",
		"account",
		"account_number",
		"phone",
		"mobile",
		"email",
		"address",
		"license_number",

		// Session相关
		"session",
		"session_id",
		"sessionid",
		"cookie",

		// 签名相关
		"sign",
		"signature",
		"hash",
		"digest",

		// OAuth相关
		"client_secret",
		"client_id",
		"code",
		"state",
		"nonce",

		// 其他敏感信息
		"certificate",
		"license",
		"passport",
		"device_id",
		"imei",
		"mac",
		"ip",
		"location",
		"coordinates",
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

// getDesensitizeFunc 获取脱敏策略函数
func getDesensitizeFunc(tag string) DesensitizeFunc {
	switch tag {
	case "chinese_name":
		return ChineseNameDesensitize
	case "id_card":
		return IdCardDesensitize
	case "landline":
		return FixedPhoneDesensitize
	case "mobile_phone":
		return MobilePhoneDesensitize
	case "address":
		return AddressDesensitize
	case "email":
		return EmailDesensitize
	case "password":
		return PasswordDesensitize
	case "plate":
		return LicensePlateDesensitize
	case "bank_card":
		return BankCardDesensitize
	case "ipv4":
		return IPv4Desensitize
	case "ipv6":
		return IPv6Desensitize
	case "url":
		return URLDesensitize
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
	case "license_number":
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

// URLDesensitize URL脱敏实现
func URLDesensitize(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// 脱敏用户名和密码
	var userInfo string
	if parsedURL.User != nil {
		userInfo = "****:****@"
	}

	// 脱敏主机名中的IP地址
	host := parsedURL.Hostname()
	port := parsedURL.Port()

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			host = IPv4Desensitize(host)
		} else {
			host = IPv6Desensitize(host)
		}
	}

	if port != "" {
		host = net.JoinHostPort(host, port)
	}

	// 脱敏查询参数中的敏感信息
	values := parsedURL.Query()
	for key := range values {
		for _, param := range sensitiveURLParams {
			if strings.Contains(strings.ToLower(key), param) {
				values.Set(key, "****")
			}
		}
	}

	// 手动构建URL字符串
	var buf strings.Builder
	if parsedURL.Scheme != "" {
		buf.WriteString(parsedURL.Scheme)
		buf.WriteString("://")
	}
	buf.WriteString(userInfo)
	buf.WriteString(host)
	if parsedURL.Path != "" {
		buf.WriteString(parsedURL.Path)
	}
	if len(values) > 0 {
		buf.WriteByte('?')
		buf.WriteString(values.Encode())
	}
	if parsedURL.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(parsedURL.Fragment)
	}

	return buf.String()
}

// ClearToNullDesensitize 清空为null的脱敏实现
func ClearToNullDesensitize(_ string) string {
	return ""
}

// ClearToEmptyDesensitize 清空为空字符串的脱敏实现
func ClearToEmptyDesensitize(_ string) string {
	return ""
}

// RegisterURLSensitiveParams 添加自定义的URL参数脱敏规则
func RegisterURLSensitiveParams(params ...string) {
	sensitiveURLParams = append(sensitiveURLParams, params...)
}

// ChineseNameDesensitize 中文姓名脱敏
func ChineseNameDesensitize(name string) string {
	runes := []rune(name)
	if len(runes) <= 1 {
		return name
	}
	return string(runes[0]) + strings.Repeat("*", utf8.RuneCountInString(name)-2) + string(runes[len(runes)-1])
}

// IdCardDesensitize 身份证脱敏
func IdCardDesensitize(idCard string) string {
	runes := []rune(idCard)
	if len(runes) <= 10 {
		return idCard
	}
	return string(runes[:6]) + strings.Repeat("*", len(runes)-10) + string(runes[len(runes)-4:])
}

// ChineseIDCardDesensitize 验证中国身份证号
func ChineseIDCardDesensitize(id string) bool {
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

// FixedPhoneDesensitize 固定电话脱敏
func FixedPhoneDesensitize(phone string) string {
	runes := []rune(phone)
	if len(runes) <= 6 {
		return phone
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-5) + string(runes[len(runes)-2:])
}

// MobilePhoneDesensitize 手机号脱敏
func MobilePhoneDesensitize(phone string) string {
	runes := []rune(phone)
	if len(runes) <= 7 {
		return phone
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-7) + string(runes[len(runes)-4:])
}

// AddressDesensitize 地址脱敏
func AddressDesensitize(address string) string {
	runes := []rune(address)
	length := len(runes)
	if length <= 8 {
		return strings.Repeat("*", length)
	}
	return string(runes[:length-8]) + strings.Repeat("*", 8)
}

// EmailDesensitize 邮箱脱敏，隐藏用户名中间3位，域名不打码
func EmailDesensitize(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	// 用户名处理
	runes := []rune(username)
	if len(runes) <= 5 { // 用户名太短则保持原样
		return email
	}

	// 保留前两位和后两位，中间用3个星号替换
	maskedUsername := string(runes[:2]) + "***" + string(runes[len(runes)-2:])

	return maskedUsername + "@" + domain
}

// PasswordDesensitize 密码脱敏
func PasswordDesensitize(password string) string {
	return strings.Repeat("*", utf8.RuneCountInString(password))
}

// LicensePlateDesensitize 车牌号脱敏
func LicensePlateDesensitize(license string) string {
	runes := []rune(license)
	if len(runes) <= 4 {
		return license
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-3) + string(runes[len(runes)-1:])
}

// BankCardDesensitize 银行卡脱敏，保留前6位和后4位，中间用*代替
func BankCardDesensitize(card string) string {
	runes := []rune(card)
	if len(runes) <= 10 { // 银行卡号太短
		return strings.Repeat("*", len(runes))
	}

	// 验证银行卡号有效性
	if !validateCreditCard(card) {
		return strings.Repeat("*", len(runes))
	}

	// 保留前6位和后4位，中间用*代替
	return string(runes[:6]) + strings.Repeat("*", len(runes)-10) + string(runes[len(runes)-4:])
}

// IPv4Desensitize IPv4地址脱敏
func IPv4Desensitize(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + ".*.*." + parts[3]
}

// IPv6Desensitize IPv6地址脱敏
func IPv6Desensitize(ip string) string {
	parts := strings.Split(ip, ":")
	if len(parts) < 4 {
		return ip
	}
	return parts[0] + ":" + parts[1] + ":****:" + parts[len(parts)-1]
}

// JWTDesensitize JWT令牌脱敏
func JWTDesensitize(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}
	return parts[0] + ".****." + parts[2]
}

// SocialSecurityDesensitize 社会保障号脱敏
func SocialSecurityDesensitize(ssn string) string {
	runes := []rune(ssn)
	if len(runes) != 11 {
		return ssn
	}
	return string(runes[:3]) + "-**-" + string(runes[7:])
}

// PassportDesensitize 护照号脱敏
func PassportDesensitize(passport string) string {
	runes := []rune(passport)
	if len(runes) < 6 {
		return passport
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

// DriversLicenseDesensitize 驾驶证号脱敏
func DriversLicenseDesensitize(license string) string {
	runes := []rune(license)
	if len(runes) < 8 {
		return license
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-6) + string(runes[len(runes)-2:])
}

// MedicalIDDesensitize 医保卡号脱敏
func MedicalIDDesensitize(id string) string {
	runes := []rune(id)
	if len(runes) < 8 {
		return id
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-6) + string(runes[len(runes)-3:])
}

// CompanyIDDesensitize 公司编号（统一信用代码）脱敏
func CompanyIDDesensitize(id string) string {
	runes := []rune(id)
	if len(runes) < 6 {
		return id
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

// DeviceIDDesensitize 设备ID脱敏
func DeviceIDDesensitize(id string) string {
	runes := []rune(id)
	if len(runes) < 8 {
		return id
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-8) + string(runes[len(runes)-4:])
}

// MACDesensitize MAC地址脱敏
func MACDesensitize(mac string) string {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return mac
	}
	return parts[0] + ":**:**:**:**:" + parts[5]
}

// VINDesensitize 车架号脱敏
func VINDesensitize(vin string) string {
	runes := []rune(vin)
	if len(runes) != 17 {
		return vin
	}
	return string(runes[:3]) + strings.Repeat("*", 11) + string(runes[14:])
}

// IMEIDesensitize IMEI号脱敏
func IMEIDesensitize(imei string) string {
	runes := []rune(imei)
	if len(runes) != 15 {
		return imei
	}
	return string(runes[:4]) + strings.Repeat("*", 7) + string(runes[11:])
}

// CoordinateDesensitize 地理坐标脱敏
func CoordinateDesensitize(coord string) string {
	parts := strings.Split(coord, ",")
	if len(parts) != 2 {
		return coord
	}
	return "**.****,**.****"
}

// AccessTokenDesensitize 访问令牌脱敏
func AccessTokenDesensitize(token string) string {
	runes := []rune(token)
	if len(runes) < 8 {
		return token
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-8) + string(runes[len(runes)-4:])
}

// RefreshTokenDesensitize 刷新令牌脱敏
func RefreshTokenDesensitize(token string) string {
	runes := []rune(token)
	if len(runes) < 8 {
		return token
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-8) + string(runes[len(runes)-4:])
}

// PrivateKeyDesensitize 私钥脱敏
func PrivateKeyDesensitize(_ string) string {
	return "[PRIVATE_KEY]"
}

// PublicKeyDesensitize 公钥脱敏
func PublicKeyDesensitize(key string) string {
	runes := []rune(key)
	if len(runes) < 20 {
		return key
	}
	return string(runes[:10]) + "..." + string(runes[len(runes)-10:])
}

// CertificateDesensitize 证书脱敏
func CertificateDesensitize(cert string) string {
	runes := []rune(cert)
	if len(runes) < 20 {
		return cert
	}
	return "-----BEGIN CERTIFICATE-----\n****\n-----END CERTIFICATE-----"
}

// UsernameDesensitize 用户名脱敏，显示前后两位
func UsernameDesensitize(username string) string {
	runes := []rune(username)
	if len(runes) <= 4 {
		return username // 如果用户名长度小于等于4，直接返回
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

// NicknameDesensitize 昵称脱敏
func NicknameDesensitize(nickname string) string {
	runes := []rune(nickname)
	if len(runes) <= 1 {
		return nickname
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

// BiographyDesensitize 个人简介脱敏
func BiographyDesensitize(bio string) string {
	runes := []rune(bio)
	if len(runes) <= 10 {
		return bio
	}
	return string(runes[:5]) + "..." + string(runes[len(runes)-5:])
}

// SignatureDesensitize 个性签名脱敏
func SignatureDesensitize(signature string) string {
	runes := []rune(signature)
	if len(runes) <= 4 {
		return signature
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

// CommentDesensitize 评论内容脱敏
func CommentDesensitize(comment string) string {
	runes := []rune(comment)
	if len(runes) <= 10 {
		return comment
	}
	return string(runes[:5]) + "..." + string(runes[len(runes)-5:])
}

// Base64Desensitize Base64编码脱敏方法
func Base64Desensitize(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// AesDesensitize AES加密脱敏方法
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

// DesDesensitize DES加密脱敏方法
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

// RsaDesensitize RSA加密脱敏方法
func RsaDesensitize(data []byte, publicKey *rsa.PublicKey) (string, error) {
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, data, nil)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// FirstMaskDesensitize 仅保留首字符脱敏
func FirstMaskDesensitize(data string) string {
	runes := []rune(data)
	if len(runes) <= 1 {
		return data
	}
	return string(runes[:1]) + strings.Repeat("*", len(runes)-1)
}

// CustomizeKeepLengthDesensitize 自定义保留长度的脱敏
func CustomizeKeepLengthDesensitize(data string, preKeep, postKeep int) string {
	runes := []rune(data)
	length := len(runes)

	if length <= preKeep+postKeep {
		return data
	}

	return string(runes[:preKeep]) + strings.Repeat("*", length-preKeep-postKeep) + string(runes[length-postKeep:])
}

// StringDesensitize 字符串脱敏
func StringDesensitize(data string, filterWords ...string) string {
	builder := maskPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		maskPool.Put(builder)
	}()

	for _, word := range filterWords {
		regex := regexp.MustCompile(regexp.QuoteMeta(word))
		data = regex.ReplaceAllStringFunc(data, func(match string) string {
			return strings.Repeat("*", utf8.RuneCountInString(match))
		})
	}

	return data
}

// PostalCodeDesensitize 处理邮政编码，隐藏中间三位
func PostalCodeDesensitize(code string) string {
	runes := []rune(code)
	if len(runes) == 6 {
		return string(runes[:3]) + "***" + string(runes[6:])
	}
	return code
}

// CreditCardDesensitize 处理信用卡号，只显示后四位
func CreditCardDesensitize(card string) string {
	runes := []rune(card)
	if len(runes) >= 4 {
		return strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-4:])
	}
	return card
}

// validateCreditCard 验证信用卡号，使用简单的Luhn算法
func validateCreditCard(card string) bool {
	sum := 0
	alternate := false
	for i := len(card) - 1; i >= 0; i-- {
		n := int(card[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}

// APIKeyDesensitize 处理API密钥，隐藏前面部分
func APIKeyDesensitize(key string) string {
	runes := []rune(key)
	if len(runes) >= 8 {
		return strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-4:])
	}
	return key
}

// UUIDDesensitize 处理UUID，隐藏中间部分
func UUIDDesensitize(uuid string) string {
	parts := strings.Split(uuid, "-")
	if len(parts) == 5 && len(parts[2]) >= 4 {
		return parts[0] + "-" + parts[1] + "-****-****-" + parts[4]
	}
	return uuid
}

// MD5Desensitize 计算输入的MD5哈希值并返回
func MD5Desensitize(input string) string {
	hash := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// SHA1Desensitize 计算输入的SHA1哈希值并返回
func SHA1Desensitize(input string) string {
	hash := sha1.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// SHA256Desensitize 处理输入，返回 SHA-256 哈希值
func SHA256Desensitize(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

// LatLngDesensitize 处理经纬度，隐藏具体数值
func LatLngDesensitize(latLng string) string {
	parts := strings.Split(latLng, ",")
	if len(parts) == 2 {
		return "**.****,**.****"
	}
	return latLng
}

// DomainDesensitize 处理域名，隐藏前面部分
func DomainDesensitize(domain string) string {
	if strings.Contains(domain, ".") {
		parts := strings.Split(domain, ".")
		if len(parts) > 1 {
			return "****." + parts[len(parts)-1]
		}
	}
	return domain
}

// MaskString 字符串遮罩处理
func MaskString(str string, start, end int, maskChar string) string {
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

// pkcs7Padding PKCS#7填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}
