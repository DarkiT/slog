package main

import "regexp"

// 正则表达式模式
const (
	DatePattern           = `(?i)(?:[0-3]?\d(?:st|nd|rd|th)?\s+(?:of\s+)?(?:jan\.?|january|feb\.?|february|mar\.?|march|apr\.?|april|may|jun\.?|june|jul\.?|july|aug\.?|august|sep\.?|september|oct\.?|october|nov\.?|november|dec\.?|december)|(?:jan\.?|january|feb\.?|february|mar\.?|march|apr\.?|april|may|jun\.?|june|jul\.?|july|aug\.?|august|sep\.?|september|oct\.?|october|nov\.?|november|dec\.?|december)\s+[0-3]?\d(?:st|nd|rd|th)?)(?:\,)?\s*(?:\d{4})?|[0-3]?\d[-\./][0-3]?\d[-\./]\d{2,4}`
	TimePattern           = `(?i)\d{1,2}:\d{2} ?(?:[ap]\.?m\.?)?|\d[ap]\.?m\.?`
	PhonePattern          = `(?:(?:\+?\d{1,3}[-.\s*]?)?(?:\(?\d{3}\)?[-.\s*]?)?\d{3}[-.\s*]?\d{4,6})|(?:(?:(?:\(\+?\d{2}\))|(?:\+?\d{2}))\s*\d{2}\s*\d{3}\s*\d{4})`
	PhonesWithExtsPattern = `(?i)(?:(?:\+?1\s*(?:[.-]\s*)?)?(?:\(\s*(?:[2-9]1[02-9]|[2-9][02-8]1|[2-9][02-8][02-9])\s*\)|(?:[2-9]1[02-9]|[2-9][02-8]1|[2-9][02-8][02-9]))\s*(?:[.-]\s*)?)?(?:[2-9]1[02-9]|[2-9][02-9]1|[2-9][02-9]{2})\s*(?:[.-]\s*)?(?:[0-9]{4})(?:\s*(?:#|x\.?|ext\.?|extension)\s*(?:\d+)?)`
	LinkPattern           = `(?:(?:https?:\/\/)?(?:[a-z0-9.\-]+|www|[a-z0-9.\-])[.](?:[^\s()<>]+|\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\))+(?:\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))`
	EmailPattern          = `(?i)([A-Za-z0-9!#$%&'*+\/=?^_{|.}~-]+@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)`
	IPv4Pattern           = `(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`
	IPv6Pattern           = `(?:(?:(?:[0-9A-Fa-f]{1,4}:){7}(?:[0-9A-Fa-f]{1,4}|:))|(?:(?:[0-9A-Fa-f]{1,4}:){6}(?::[0-9A-Fa-f]{1,4}|(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(?:(?:[0-9A-Fa-f]{1,4}:){5}(?:(?:(?::[0-9A-Fa-f]{1,4}){1,2})|:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(?:(?:[0-9A-Fa-f]{1,4}:){4}(?:(?:(?::[0-9A-Fa-f]{1,4}){1,3})|(?:(?::[0-9A-Fa-f]{1,4})?:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(?:(?:[0-9A-Fa-f]{1,4}:){3}(?:(?:(?::[0-9A-Fa-f]{1,4}){1,4})|(?:(?::[0-9A-Fa-f]{1,4}){0,2}:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(?:(?:[0-9A-Fa-f]{1,4}:){2}(?:(?:(?::[0-9A-Fa-f]{1,4}){1,5})|(?:(?::[0-9A-Fa-f]{1,4}){0,3}:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(?:(?:[0-9A-Fa-f]{1,4}:){1}(?:(?:(?::[0-9A-Fa-f]{1,4}){1,6})|(?:(?::[0-9A-Fa-f]{1,4}){0,4}:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(?::(?:(?:(?::[0-9A-Fa-f]{1,4}){1,7})|(?:(?::[0-9A-Fa-f]{1,4}){0,5}:(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(?:%.+)?\s*`
	IPPattern             = IPv4Pattern + `|` + IPv6Pattern
	NotKnownPortPattern   = `6[0-5]{2}[0-3][0-5]|[1-5][\d]{4}|[2-9][\d]{3}|1[1-9][\d]{2}|10[3-9][\d]|102[4-9]`
	PricePattern          = `[$]\s?[+-]?[0-9]{1,3}(?:(?:,?[0-9]{3}))*(?:\.[0-9]{1,2})?`
	HexColorPattern       = `(?:#?([0-9a-fA-F]{6}|[0-9a-fA-F]{3}))`
	CreditCardPattern     = `(?:(?:(?:\d{4}[- ]?){3}\d{4}|\d{15,16}))`
	VISACreditCardPattern = `4\d{3}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`
	MCCreditCardPattern   = `5[1-5]\d{2}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`
	BtcAddressPattern     = `[13][a-km-zA-HJ-NP-Z1-9]{25,34}`
	StreetAddressPattern  = `\d{1,4} [\w\s]{1,20}(?:street|st|avenue|ave|road|rd|highway|hwy|square|sq|trail|trl|drive|dr|court|ct|park|parkway|pkwy|circle|cir|boulevard|blvd)\W?`
	ZipCodePattern        = `\b\d{5}(?:[-\s]\d{4})?\b`
	PoBoxPattern          = `(?i)P\.? ?O\.? Box \d+`
	SSNPattern            = `(?:\d{3}-\d{2}-\d{4})`
	MD5HexPattern         = `[0-9a-fA-F]{32}`
	SHA1HexPattern        = `[0-9a-fA-F]{40}`
	SHA256HexPattern      = `[0-9a-fA-F]{64}`
	GUIDPattern           = `[0-9a-fA-F]{8}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{12}`
	ISBN13Pattern         = `(?:[\d]-?){12}[\dxX]`
	ISBN10Pattern         = `(?:[\d]-?){9}[\dxX]`
	MACAddressPattern     = `(([a-fA-F0-9]{2}[:-]){5}([a-fA-F0-9]{2}))`
	IBANPattern           = `[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z\d]?){0,16}`
	GitRepoPattern        = `((git|ssh|http(s)?)|(git@[\w\.]+))(:(\/\/)?)([\w\.@\:/\-~]+)(\.git)(\/)?`
	URLPattern            = "([ a-zA-Z ]{ 1,6 }):\\/\\/([^:@]*:[^:@]+@)?(\\d{1,3})\\.(\\d{1,3})\\.(\\d{1,3})\\.(\\d{1,3}) | ([a-zA-Z]{1,6}):\\/\\/([^:@]*:[^:@]+@)?(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]"
)

// 编译的正则表达式
var (
	DateRegex           = regexp.MustCompile(DatePattern)
	TimeRegex           = regexp.MustCompile(TimePattern)
	PhoneRegex          = regexp.MustCompile(PhonePattern)
	PhonesWithExtsRegex = regexp.MustCompile(PhonesWithExtsPattern)
	LinkRegex           = regexp.MustCompile(LinkPattern)
	EmailRegex          = regexp.MustCompile(EmailPattern)
	IPv4Regex           = regexp.MustCompile(IPv4Pattern)
	IPv6Regex           = regexp.MustCompile(IPv6Pattern)
	IPRegex             = regexp.MustCompile(IPPattern)
	NotKnownPortRegex   = regexp.MustCompile(NotKnownPortPattern)
	PriceRegex          = regexp.MustCompile(PricePattern)
	HexColorRegex       = regexp.MustCompile(HexColorPattern)
	CreditCardRegex     = regexp.MustCompile(CreditCardPattern)
	BtcAddressRegex     = regexp.MustCompile(BtcAddressPattern)
	StreetAddressRegex  = regexp.MustCompile(StreetAddressPattern)
	ZipCodeRegex        = regexp.MustCompile(ZipCodePattern)
	PoBoxRegex          = regexp.MustCompile(PoBoxPattern)
	SSNRegex            = regexp.MustCompile(SSNPattern)
	MD5HexRegex         = regexp.MustCompile(MD5HexPattern)
	SHA1HexRegex        = regexp.MustCompile(SHA1HexPattern)
	SHA256HexRegex      = regexp.MustCompile(SHA256HexPattern)
	GUIDRegex           = regexp.MustCompile(GUIDPattern)
	ISBN13Regex         = regexp.MustCompile(ISBN13Pattern)
	ISBN10Regex         = regexp.MustCompile(ISBN10Pattern)
	VISACreditCardRegex = regexp.MustCompile(VISACreditCardPattern)
	MCCreditCardRegex   = regexp.MustCompile(MCCreditCardPattern)
	MACAddressRegex     = regexp.MustCompile(MACAddressPattern)
	IBANRegex           = regexp.MustCompile(IBANPattern)
	GitRepoRegex        = regexp.MustCompile(GitRepoPattern)
	URLRegex            = regexp.MustCompile(URLPattern)
)

func match(text string, regex *regexp.Regexp) []string {
	parsed := regex.FindAllString(text, -1)
	return parsed
}

// Date 查找所有日期字符串
func Date(text string) []string {
	return match(text, DateRegex)
}

// Time 查找所有时间字符串
func Time(text string) []string {
	return match(text, TimeRegex)
}

// Phones 查找所有电话号码
func Phones(text string) []string {
	return match(text, PhoneRegex)
}

// PhonesWithExts 查找所有带有扩展号的电话号码
func PhonesWithExts(text string) []string {
	return match(text, PhonesWithExtsRegex)
}

// Links 查找所有链接字符串
func Links(text string) []string {
	return match(text, LinkRegex)
}

// Emails 查找所有电子邮件字符串
func Emails(text string) []string {
	return match(text, EmailRegex)
}

// IPv4s 查找所有 IPv4 地址
func IPv4s(text string) []string {
	return match(text, IPv4Regex)
}

// IPv6s 查找所有 IPv6 地址
func IPv6s(text string) []string {
	return match(text, IPv6Regex)
}

// IPs 查找所有 IP 地址（包括 IPv4 和 IPv6）
func IPs(text string) []string {
	return match(text, IPRegex)
}

// NotKnownPorts 查找所有未知端口号
func NotKnownPorts(text string) []string {
	return match(text, NotKnownPortRegex)
}

// Prices 查找所有价格字符串
func Prices(text string) []string {
	return match(text, PriceRegex)
}

// HexColors 查找所有十六进制颜色值
func HexColors(text string) []string {
	return match(text, HexColorRegex)
}

// CreditCards 查找所有信用卡号码
func CreditCards(text string) []string {
	return match(text, CreditCardRegex)
}

// BtcAddresses 查找所有比特币地址
func BtcAddresses(text string) []string {
	return match(text, BtcAddressRegex)
}

// StreetAddresses 查找所有街道地址
func StreetAddresses(text string) []string {
	return match(text, StreetAddressRegex)
}

// ZipCodes 查找所有邮政编码
func ZipCodes(text string) []string {
	return match(text, ZipCodeRegex)
}

// PoBoxes 查找所有邮政信箱字符串
func PoBoxes(text string) []string {
	return match(text, PoBoxRegex)
}

// SSNs 查找所有社会保障号码字符串
func SSNs(text string) []string {
	return match(text, SSNRegex)
}

// MD5Hexes 查找所有 MD5 十六进制字符串
func MD5Hexes(text string) []string {
	return match(text, MD5HexRegex)
}

// SHA1Hexes 查找所有 SHA1 十六进制字符串
func SHA1Hexes(text string) []string {
	return match(text, SHA1HexRegex)
}

// SHA256Hexes 查找所有 SHA256 十六进制字符串
func SHA256Hexes(text string) []string {
	return match(text, SHA256HexRegex)
}

// GUIDs 查找所有 GUID 字符串
func GUIDs(text string) []string {
	return match(text, GUIDRegex)
}

// ISBN13s 查找所有 ISBN13 字符串
func ISBN13s(text string) []string {
	return match(text, ISBN13Regex)
}

// ISBN10s 查找所有 ISBN10 字符串
func ISBN10s(text string) []string {
	return match(text, ISBN10Regex)
}

// VISACreditCards 查找所有 VISA 信用卡号码
func VISACreditCards(text string) []string {
	return match(text, VISACreditCardRegex)
}

// MCCreditCards 查找所有 MasterCard 信用卡号码
func MCCreditCards(text string) []string {
	return match(text, MCCreditCardRegex)
}

// MACAddresses 查找所有 MAC 地址
func MACAddresses(text string) []string {
	return match(text, MACAddressRegex)
}

// IBANs 查找所有 IBAN 字符串
func IBANs(text string) []string {
	return match(text, IBANRegex)
}

// GitRepos 查找所有带协议前缀的 git 仓库地址
func GitRepos(text string) []string {
	return match(text, GitRepoRegex)
}

// Url 查找所有带协议前缀的 URL 地址
func Url(text string) []string {
	return match(text, URLRegex)
}
