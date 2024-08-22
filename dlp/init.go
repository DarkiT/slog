package dlp

import (
	"net"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/darkit/slog/dlp/dlpheader"
)

type dlp struct {
	dlpheader.EngineAPI
	UrlQueryArgs []string
}

func DlpInit(name ...string) (*dlp, error) {
	callID := strings.Join(name, ".")
	if callID == "" {
		callID = "slog.caller"
	}
	eng, err := NewEngine(callID)
	if err == nil {
		_ = eng.ApplyConfigDefault()
	}
	dlp := new(dlp)
	dlp.EngineAPI = eng
	_ = dlp.RegisterIP()
	_ = dlp.RegisterUrl()

	return dlp, nil
}

// RegisterUrl 对URL进行脱敏处理，隐藏账号、密码、IP等敏感信息
func (d *dlp) SetUrlQueryArgs(args ...string) {
	d.UrlQueryArgs = args
}

func (d *dlp) RegisterUrl() error {
	return d.RegisterMasker("URL", func(in string) (string, error) {
		parsedUrl, err := url.Parse(in)
		if err != nil {
			return "", err
		}

		// 脱敏用户名和密码
		if parsedUrl.User != nil {
			parsedUrl.User = url.UserPassword("username", "password")
		}

		// 脱敏IP地址
		parsedUrl.Host = desensitizeIP(parsedUrl.Host)

		// 脱敏查询参数中的敏感信息（自定义规则）
		if len(d.UrlQueryArgs) > 0 {
			parsedUrl.RawQuery = desensitizeQuery(parsedUrl.RawQuery, d.UrlQueryArgs...)
		}

		return url.QueryUnescape(parsedUrl.String())
	})
}

func (d *dlp) RegisterIP() error {
	return d.RegisterMasker("IP", func(in string) (string, error) {
		return desensitizeIP(in), nil
	})
}

// desensitizeIP 对 IP 地址进行脱敏处理
func desensitizeIP(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	} else if ip.To4() != nil {
		return ipv4Desensitize(host)
	} else if ip.To16() != nil {
		return ipv6Desensitize(host)
	}
	return host
}

// ipv4Desensitize 对 IPv4 地址进行脱敏处理，只保留首段和末段，中间部分用 * 替换
func ipv4Desensitize(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + ".*.*." + parts[3]
}

// ipv6Desensitize 对 IPv6 地址进行脱敏处理，隐藏中间部分，只显示前后两段
func ipv6Desensitize(ip string) string {
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
				values.Set(key, firstMaskDesensitize(values.Get(key)))
				break // 找到匹配的关键词后停止检查
			}
		}
	}
	return values.Encode()
}

// firstMaskDesensitize 对字符串进行脱敏处理，只保留第一个字符，其他部分用 * 替换
func firstMaskDesensitize(data string) string {
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