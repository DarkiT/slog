package dlp

import (
	"sync"
	"sync/atomic"
)

type DlpConfig struct {
	enabled    atomic.Bool
	strategies sync.Map // 存储脱敏策略
}

var (
	globalConfig *DlpConfig
	configOnce   sync.Once
)

// GetConfig 获取全局配置实例
func GetConfig() *DlpConfig {
	configOnce.Do(func() {
		globalConfig = &DlpConfig{}
		globalConfig.init()
	})
	return globalConfig
}

func (c *DlpConfig) init() {
	// 初始化默认脱敏策略
	c.registerDefaultStrategies()
	c.enabled.Store(false)
}

// Enable 启用脱敏功能
func (c *DlpConfig) Enable() {
	c.enabled.Store(true)
}

// Disable 禁用脱敏功能
func (c *DlpConfig) Disable() {
	c.enabled.Store(false)
}

// IsEnabled 检查脱敏功能是否启用
func (c *DlpConfig) IsEnabled() bool {
	return c.enabled.Load()
}

// RegisterStrategy 注册自定义脱敏策略
func (c *DlpConfig) RegisterStrategy(name string, strategy DesensitizeFunc) {
	c.strategies.Store(name, strategy)
}

// GetStrategy 获取脱敏策略
func (c *DlpConfig) GetStrategy(name string) (DesensitizeFunc, bool) {
	if v, ok := c.strategies.Load(name); ok {
		return v.(DesensitizeFunc), true
	}
	return nil, false
}

// registerDefaultStrategies 注册默认的脱敏策略
func (c *DlpConfig) registerDefaultStrategies() {
	// 个人信息
	c.RegisterStrategy("chinese_name", ChineseNameDesensitize)
	c.RegisterStrategy("id_card", IdCardDesensitize)
	c.RegisterStrategy("passport", PassportDesensitize)
	c.RegisterStrategy("drivers_license", DriversLicenseDesensitize)

	// 联系方式
	c.RegisterStrategy("mobile_phone", MobilePhoneDesensitize)
	c.RegisterStrategy("fixed_phone", FixedPhoneDesensitize)
	c.RegisterStrategy("email", EmailDesensitize)
	c.RegisterStrategy("address", AddressDesensitize)

	// 账户信息
	c.RegisterStrategy("bank_card", BankCardDesensitize)
	c.RegisterStrategy("username", UsernameDesensitize)
	c.RegisterStrategy("password", PasswordDesensitize)

	// 设备信息
	c.RegisterStrategy("ipv4", IPv4Desensitize)
	c.RegisterStrategy("ipv6", IPv6Desensitize)
	c.RegisterStrategy("mac", MACDesensitize)
	c.RegisterStrategy("device_id", DeviceIDDesensitize)
	c.RegisterStrategy("imei", IMEIDesensitize)

	// 证件信息
	c.RegisterStrategy("social_security", SocialSecurityDesensitize)
	c.RegisterStrategy("medical_id", MedicalIDDesensitize)
	c.RegisterStrategy("company_id", CompanyIDDesensitize)

	// 车辆信息
	c.RegisterStrategy("car_license", CarLicenseDesensitize)
	c.RegisterStrategy("vin", VINDesensitize)

	// 安全凭证
	c.RegisterStrategy("jwt", JWTDesensitize)
	c.RegisterStrategy("access_token", AccessTokenDesensitize)
	c.RegisterStrategy("refresh_token", RefreshTokenDesensitize)
	c.RegisterStrategy("private_key", PrivateKeyDesensitize)
	c.RegisterStrategy("public_key", PublicKeyDesensitize)
	c.RegisterStrategy("certificate", CertificateDesensitize)

	// 其他
	c.RegisterStrategy("url", URLDesensitize)
	c.RegisterStrategy("first_mask", FirstMaskDesensitize)
	c.RegisterStrategy("null", ClearToNullDesensitize)
	c.RegisterStrategy("empty", ClearToEmptyDesensitize)
	c.RegisterStrategy("base64", Base64Desensitize)
}
