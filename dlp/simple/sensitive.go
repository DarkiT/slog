package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Sensitive 定义了用户信息结构体，并通过结构体标签指定了脱敏策略
type Sensitive struct {
	Name         string `dlp:"chinese_name" json:"name,omitempty"`
	IdCard       string `dlp:"id_card" json:"id_card,omitempty"`
	FixedPhone   string `dlp:"fixed_phone" json:"fixed_phone,omitempty"`
	MobilePhone  string `dlp:"mobile_phone" json:"mobile_phone,omitempty"`
	Address      string `dlp:"address" json:"address,omitempty"`
	Email        string `dlp:"email" json:"email,omitempty"`
	Password     string `dlp:"password" json:"password,omitempty"`
	CarLicense   string `dlp:"car_license" json:"car_license,omitempty"`
	BankCard     string `dlp:"bank_card" json:"bank_card,omitempty"`
	Ipv4         string `dlp:"ipv4" json:"ipv_4,omitempty"`
	Ipv6         string `dlp:"ipv6" json:"ipv_6,omitempty"`
	Base64       string `dlp:"base64" json:"base_64,omitempty"`
	URL          string `dlp:"url" json:"url,omitempty"`
	FirstMask    string `dlp:"first_mask" json:"first_mask,omitempty"`
	ClearToNull  string `dlp:"null" json:"null,omitempty"`
	ClearToEmpty string `dlp:"empty" json:"empty,omitempty"`
}

// ProcessSensitiveData 是一个泛型函数，处理任何结构体的脱敏
func ProcessSensitiveData[T any](v *T) {
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("dlp")

		if tag != "" {
			strategy := getStrategyFromTag(tag)
			if strategy != nil {
				field.SetString(strategy(field.String()))
			}
		}
	}
}

// 根据标签名称获取对应的脱敏策略
func getStrategyFromTag(tag string) func(data string) string {
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
		return Ipv4Desensitize
	case "url":
		return urlDesensitize
	case "first_mask":
		return FirstMaskDesensitize
	case "null":
		return ClearToNullDesensitize
	case "empty":
		return ClearToEmptyDesensitize
	case "base64":
		return Base64Desensitize // 确保 RSA 策略被正确处理
	default:
		return nil
	}
}

// ChineseNameDesensitize 对中文姓名进行脱敏处理，只保留第一个和最后一个字符，中间部分用 * 替换
func ChineseNameDesensitize(name string) string {
	if len(name) <= 1 {
		return name
	}
	// 计算字符串的 rune 长度
	length := utf8.RuneCountInString(name)
	if length <= 1 {
		return name
	}

	// 使用 utf8.DecodeRuneInString 获取每个字符
	firstChar, _ := utf8.DecodeRuneInString(name)
	lastChar, _ := utf8.DecodeLastRuneInString(name)

	return string(firstChar) + "*" + string(lastChar)
}

// IdCardDesensitize 对身份证号码进行脱敏处理，显示前5位和后5位，中间部分用 * 替换
func IdCardDesensitize(idCard string) string {
	if len(idCard) <= 10 {
		return idCard
	}

	prefix := idCard[:5]
	suffix := idCard[len(idCard)-5:]
	middle := strings.Repeat("*", len(idCard)-10)

	return prefix + middle + suffix
}

// FixedPhoneDesensitize 对固定电话进行脱敏处理，只保留前4位和后2位，中间部分用 * 替换
func FixedPhoneDesensitize(phone string) string {
	if len(phone) <= 6 {
		return phone
	}
	return phone[:4] + strings.Repeat("*", len(phone)-6) + phone[len(phone)-2:]
}

// MobilePhoneDesensitize 对手机号进行脱敏处理，只保留前三位和后四位，中间部分用 * 替换
func MobilePhoneDesensitize(phone string) string {
	if len(phone) <= 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}

// AddressDesensitize 对地址进行脱敏处理，只显示前面部分，隐藏最后8个字符
func AddressDesensitize(address string) string {
	// 将地址转换为字符切片
	runes := []rune(address)
	length := len(runes)

	if length <= 8 {
		return strings.Repeat("*", length)
	}

	// 显示前面部分，隐藏最后8个字符
	return string(runes[:length-8]) + strings.Repeat("*", 8)
}

// EmailDesensitize 对电子邮件地址进行脱敏处理，保留首字母，其他部分用 * 替换
func EmailDesensitize(email string) string {
	parts := strings.Split(email, "@")
	if len(parts[0]) <= 1 {
		return email
	}
	return string(parts[0][0]) + strings.Repeat("*", len(parts[0])-1) + "@" + parts[1]
}

// PasswordDesensitize 对密码进行脱敏处理，将密码替换为同等长度的 *
func PasswordDesensitize(password string) string {
	return strings.Repeat("*", len(password))
}

// CarLicenseDesensitize 对车牌号进行脱敏处理，只保留前三位和最后一位，中间部分用 * 替换
func CarLicenseDesensitize(license string) string {
	// 处理车牌号的字符数
	if len(license) <= 3 {
		return license
	}
	// 将车牌号转换为字符切片
	runes := []rune(license)
	// 处理车牌号长度
	if len(runes) <= 4 {
		return string(runes)
	}
	// 处理车牌号的脱敏
	return string(runes[:3]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-1:])
}

// BankCardDesensitize 对银行卡号进行脱敏处理，只保留前4位和最后4位，中间部分用 * 替换
func BankCardDesensitize(card string) string {
	if len(card) <= 8 {
		return card
	}
	masked := card[:4] + " " + strings.Repeat("*", len(card)-8)
	for i := 4; i < len(card)-4; i += 4 {
		masked += " " + strings.Repeat("*", 4)
	}
	return masked + " " + card[len(card)-4:]
}

// Ipv4Desensitize 对 IPv4 地址进行脱敏处理，只保留首段和末段，中间部分用 * 替换
func Ipv4Desensitize(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + ".*.*." + parts[3]
}

// Ipv6Desensitize 对 IPv6 地址进行脱敏处理，隐藏中间部分，只显示前后两段
func Ipv6Desensitize(ip string) string {
	parts := strings.Split(ip, ":")
	// IPv6 地址通常由 8 段组成，若不是8段，返回原始字符串
	if len(parts) != 8 {
		return ip
	}
	// 保留第一段和最后一段，其他部分替换为 "*"
	for i := 2; i < len(parts)-2; i++ {
		parts[i] = "*"
	}
	// 将处理后的地址重新组合
	return strings.Join(parts, ":")
}

// Base64Desensitize 对数据进行 Base64 编码
func Base64Desensitize(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// AesDesensitize 对数据进行 AES 加密脱敏
func AesDesensitize(data, aesKey []byte) string {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return string(data) // Handle error appropriately
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return string(data) // Handle error appropriately
	}

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// DesDesensitize 对数据进行 DES 加密脱敏
func DesDesensitize(data string, desKey []byte) string {
	block, err := des.NewCipher(desKey)
	if err != nil {
		return data
	}

	iv := make([]byte, des.BlockSize)
	ciphertext := make([]byte, len(data))
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext, []byte(data))
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// RsaDesensitize 对数据进行 RSA 加密脱敏
func RsaDesensitize(data []byte, publicKey *rsa.PublicKey) (string, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// FirstMaskDesensitize 对字符串进行脱敏处理，只保留第一个字符，其他部分用 * 替换
func FirstMaskDesensitize(data string) string {
	if len(data) == 0 {
		return data
	}

	// 获取第一个字符的长度
	_, size := utf8.DecodeRuneInString(data)

	// 如果字符串长度小于或等于一个字符的长度
	if len(data) <= size {
		return data
	}

	// 返回第一个字符和其余部分用 * 替换
	return data[:size] + strings.Repeat("*", len(data)-size)
}

// CustomizeKeepLengthDesensitize 对字符串进行自定义长度的脱敏处理，保留前后指定长度，中间部分用 * 替换
func CustomizeKeepLengthDesensitize(data string, preKeep, postKeep int) string {
	runes := []rune(data) // 将字符串转换为 []rune 以支持中文字符
	length := len(runes)

	if length <= preKeep+postKeep {
		return data
	}

	return string(runes[:preKeep]) + strings.Repeat("*", length-preKeep-postKeep) + string(runes[length-postKeep:])
}

// StringDesensitize 对字符串进行自定义词汇的脱敏处理，将指定的词汇替换为 *
func StringDesensitize(data string, filterWords ...string) string {
	for _, word := range filterWords {
		replacement := strings.Repeat("*", utf8.RuneCountInString(word))
		data = replaceWord(data, word, replacement)
	}
	return data
}

// UrlDesensitize 对URL进行脱敏处理，隐藏账号、密码、IP等敏感信息
func UrlDesensitize(rawUrl string, args ...string) string {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return rawUrl
	}

	// 脱敏用户名和密码
	if parsedUrl.User != nil {
		parsedUrl.User = url.UserPassword("username", "password")
	}

	// 脱敏IP地址
	parsedUrl.Host = desensitizeIP(parsedUrl.Host)

	// 脱敏查询参数中的敏感信息（自定义规则）
	if len(args) > 0 {
		parsedUrl.RawQuery = desensitizeQuery(parsedUrl.RawQuery, args...)
	}

	decodedURL, _ := url.QueryUnescape(parsedUrl.String())
	return decodedURL
}

// UrlDesensitize 对URL进行脱敏处理，隐藏账号、密码、IP等敏感信息(不包含Query参数)
func urlDesensitize(rawUrl string) string {
	return UrlDesensitize(rawUrl)
}

// ClearToNullDesensitize 实现逻辑：返回空字符串
func ClearToNullDesensitize(_ string) string {
	return ""
}

// ClearToEmptyDesensitize 实现逻辑：返回空字符串
func ClearToEmptyDesensitize(_ string) string {
	return ""
}

// desensitizeIP 对 IP 地址进行脱敏处理
func desensitizeIP(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	} else if ip.To4() != nil {
		return Ipv4Desensitize(host)
	} else if ip.To16() != nil {
		return Ipv6Desensitize(host)
	}
	return host
}

// desensitizeQuery 对查询参数中的敏感信息进行脱敏处理（可根据需求自定义规则）
func desensitizeQuery(query string, args ...string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return query // 如果解析查询参数失败，返回原始查询参数
	}

	for key := range values {
		// 遍历关键词列表
		for _, keyword := range args {
			if strings.Contains(key, keyword) {
				values.Set(key, FirstMaskDesensitize(values.Get(key)))
				break // 找到匹配的关键词后停止检查
			}
		}
	}
	return values.Encode()
}

// replaceWord 替换字符串中的所有出现的词汇
func replaceWord(data, oldWord, newWord string) string {
	var result strings.Builder
	for len(data) > 0 {
		index := strings.Index(data, oldWord)
		if index == -1 {
			result.WriteString(data)
			break
		}
		result.WriteString(data[:index])
		result.WriteString(newWord)
		data = data[index+len(oldWord):]
	}
	return result.String()
}
