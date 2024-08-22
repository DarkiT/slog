### 脱敏库使用说明

使用该脱敏库可以对用户信息进行脱敏处理。通过定义数据结构并使用结构体标签指定脱敏策略，您可以轻松处理敏感信息。

#### 主要功能
- 支持多种类型的数据脱敏，如姓名、身份证号、手机号、邮箱、地址等。
- 通过结构体标签自定义脱敏策略。
- 提供自定义脱敏规则，支持中文字符处理。

#### 结构体标签说明
每个字段都可以使用 `dlp` 标签指定脱敏策略。可用的策略包括：

| 标签名称       | 说明                                |
| ------------- | ----------------------------------- |
| chinese_name  | 脱敏中文姓名                        |
| id_card       | 脱敏身份证号                        |
| fixed_phone   | 脱敏固定电话                        |
| mobile_phone  | 脱敏手机号                          |
| address       | 脱敏地址                            |
| email         | 脱敏邮箱                            |
| password      | 脱敏密码                            |
| car_license   | 脱敏车牌号                          |
| bank_card     | 脱敏银行卡号                        |
| ipv4          | 脱敏IPv4地址                        |
| ipv6          | 脱敏IPv6地址                        |
| url           | 脱敏URL                             |
| base64        | Base64编码数据                      |
| first_mask    | 仅保留第一个字符，其他字符替换为 *   |
| null          | 清空字段                            |
| empty         | 将字段设置为空字符串                |

#### 使用示例

以下是一个简单的示例，展示了如何使用此库对用户数据进行脱敏处理。

```go
package main

import (
	"fmt"
)

func main() {
	user := Sensitive{
		Name:         "张三丰",
		IdCard:       "110101199003079876",
		FixedPhone:   "010-12345678",
		MobilePhone:  "13812345678",
		Address:      "北京市朝阳区幸福大街123号",
		Email:        "zhangsan@example.com",
		Password:     "123456",
		CarLicense:   "粤A66666",
		BankCard:     "9988002866797031",
		Ipv4:         "192.168.0.1",
		Ipv6:         "2409:8a55:488b:221:ba10::8",
		Base64:       "Hello, World!",
		URL:          "http://user:password@www.example.com/path?query=123",
		FirstMask:    "SensitiveData",
		ClearToNull:  "ShouldBeNull",
		ClearToEmpty: "ShouldBeEmpty",
	}

	ProcessSensitiveData(&user)

	fmt.Printf("%+v\n", user)
}
```

#### 运行结果
```json
{
  "name": "张*丰",
  "id_card": "11010******9876",
  "fixed_phone": "0101****78",
  "mobile_phone": "138****5678",
  "address": "北京市朝阳区幸福大****",
  "email": "z*********@example.com",
  "password": "******",
  "car_license": "粤A6***6",
  "bank_card": "9988 **** **** 7031",
  "ipv_4": "192.*.*.1",
  "ipv_6": "2409:8a55:*:*:ba10::8",
  "base_64": "SGVsbG8sIFdvcmxkIQ==",
  "url": "http://username:password@www.example.com/path?query=123",
  "first_mask": "S***********",
  "null": "",
  "empty": ""
}
```

### 导出函数说明

该脱敏库提供了一些导出函数用于处理各种类型的数据脱敏操作。以下是各个导出函数的独立用法说明。

#### 1. `ProcessSensitiveData`

**功能：**  
对传入的结构体进行脱敏处理，根据 `dlp` 标签指定的策略处理每个字段。

**用法：**
```go
user := Sensitive{
    Name: "张三丰",
    // 初始化其他字段
}
ProcessSensitiveData(&user)
```

**说明：**  
该函数会遍历传入结构体的所有字段，并根据 `dlp` 标签指定的策略进行脱敏。

#### 2. `ChineseNameDesensitize`

**功能：**  
对中文姓名进行脱敏处理，只保留第一个和最后一个字符，中间部分用 `*` 替换。

**用法：**
```go
desensitizedName := ChineseNameDesensitize("张三丰")
fmt.Println(desensitizedName) // 输出: 张*丰
```

**说明：**  
适用于中文姓名的脱敏处理。

#### 3. `IdCardDesensitize`

**功能：**  
对身份证号码进行脱敏处理，显示前5位和后5位，中间部分用 `*` 替换。

**用法：**
```go
desensitizedIdCard := IdCardDesensitize("110101199003079876")
fmt.Println(desensitizedIdCard) // 输出: 11010******9876
```

**说明：**  
适用于身份证号码的脱敏处理。

#### 4. `FixedPhoneDesensitize`

**功能：**  
对固定电话进行脱敏处理，只保留前4位和后2位，中间部分用 `*` 替换。

**用法：**
```go
desensitizedPhone := FixedPhoneDesensitize("010-12345678")
fmt.Println(desensitizedPhone) // 输出: 0101****78
```

**说明：**  
适用于固定电话的脱敏处理。

#### 5. `MobilePhoneDesensitize`

**功能：**  
对手机号进行脱敏处理，只保留前三位和后四位，中间部分用 `*` 替换。

**用法：**
```go
desensitizedMobile := MobilePhoneDesensitize("13812345678")
fmt.Println(desensitizedMobile) // 输出: 138****5678
```

**说明：**  
适用于手机号的脱敏处理。

#### 6. `AddressDesensitize`

**功能：**  
对地址进行脱敏处理，只显示前面部分，隐藏最后8个字符。

**用法：**
```go
desensitizedAddress := AddressDesensitize("北京市朝阳区幸福大街123号")
fmt.Println(desensitizedAddress) // 输出: 北京市朝阳区幸福大****
```

**说明：**  
适用于地址的脱敏处理。

#### 7. `EmailDesensitize`

**功能：**  
对电子邮件地址进行脱敏处理，保留首字母，其他部分用 `*` 替换。

**用法：**
```go
desensitizedEmail := EmailDesensitize("zhangsan@example.com")
fmt.Println(desensitizedEmail) // 输出: z*********@example.com
```

**说明：**  
适用于邮箱地址的脱敏处理。

#### 8. `PasswordDesensitize`

**功能：**  
对密码进行脱敏处理，将密码替换为同等长度的 `*`。

**用法：**
```go
desensitizedPassword := PasswordDesensitize("123456")
fmt.Println(desensitizedPassword) // 输出: ******
```

**说明：**  
适用于密码的脱敏处理。

#### 9. `CarLicenseDesensitize`

**功能：**  
对车牌号进行脱敏处理，只保留前三位和最后一位，中间部分用 `*` 替换。

**用法：**
```go
desensitizedLicense := CarLicenseDesensitize("粤A66666")
fmt.Println(desensitizedLicense) // 输出: 粤A6***6
```

**说明：**  
适用于车牌号的脱敏处理。

#### 10. `BankCardDesensitize`

**功能：**  
对银行卡号进行脱敏处理，只保留前4位和最后4位，中间部分用 `*` 替换。

**用法：**
```go
desensitizedBankCard := BankCardDesensitize("9988002866797031")
fmt.Println(desensitizedBankCard) // 输出: 9988 **** **** 7031
```

**说明：**  
适用于银行卡号的脱敏处理。

#### 11. `Ipv4Desensitize`

**功能：**  
对 IPv4 地址进行脱敏处理，只保留首段和末段，中间部分用 `*` 替换。

**用法：**
```go
desensitizedIPv4 := Ipv4Desensitize("192.168.0.1")
fmt.Println(desensitizedIPv4) // 输出: 192.*.*.1
```

**说明：**  
适用于 IPv4 地址的脱敏处理。

#### 12. `Ipv6Desensitize`

**功能：**  
对 IPv6 地址进行脱敏处理，隐藏中间部分，只显示前后两段。

**用法：**
```go
desensitizedIPv6 := Ipv6Desensitize("2409:8a55:488b:221:ba10::8")
fmt.Println(desensitizedIPv6) // 输出: 2409:8a55:*:*:ba10::8
```

**说明：**  
适用于 IPv6 地址的脱敏处理。

#### 13. `Base64Desensitize`

**功能：**  
对数据进行 Base64 编码。

**用法：**
```go
base64Encoded := Base64Desensitize("Hello, World!")
fmt.Println(base64Encoded) // 输出: SGVsbG8sIFdvcmxkIQ==
```

**说明：**  
适用于任何字符串数据的 Base64 编码。

#### 14. `AesDesensitize`

**功能：**  
对数据进行 AES 加密脱敏。

**用法：**
```go
aesKey := []byte("examplekey123456")
aesEncrypted := AesDesensitize([]byte("SensitiveData"), aesKey)
fmt.Println(aesEncrypted) // 输出: AES 加密后的字符串
```

**说明：**  
适用于需要 AES 加密的脱敏处理。

#### 15. `DesDesensitize`

**功能：**  
对数据进行 DES 加密脱敏。

**用法：**
```go
desKey := []byte("examplek")
desEncrypted := DesDesensitize("SensitiveData", desKey)
fmt.Println(desEncrypted) // 输出: DES 加密后的字符串
```

**说明：**  
适用于需要 DES 加密的脱敏处理。

#### 16. `RsaDesensitize`

**功能：**  
对数据进行 RSA 加密脱敏。

**用法：**
```go
// publicKey 为已经初始化的 *rsa.PublicKey 对象
rsaEncrypted, err := RsaDesensitize([]byte("SensitiveData"), publicKey)
if err == nil {
    fmt.Println(rsaEncrypted) // 输出: RSA 加密后的字符串
}
```

**说明：**  
适用于需要 RSA 加密的脱敏处理。

#### 17. `FirstMaskDesensitize`

**功能：**  
对字符串进行脱敏处理，只保留第一个字符，其他部分用 `*` 替换。

**用法：**
```go
firstMasked := FirstMaskDesensitize("SensitiveData")
fmt.Println(firstMasked) // 输出: S***********
```

**说明：**  
适用于需要简单脱敏的字符串。

#### 18. `CustomizeKeepLengthDesensitize`

**功能：**  
自定义保留字符串的前后指定长度，中间部分用 `*` 替换。

**用法：**
```go
customMasked := CustomizeKeepLengthDesensitize("SensitiveData", 3, 4)
fmt.Println(customMasked) // 输出: Sen*****Data
```

**说明：**  
适用于需要灵活控制脱敏字符数的场景。

#### 19. `StringDesensitize`

**功能：**  
对字符串进行自定义词汇的脱敏处理，将指定的词汇替换为 `*`。

**用法：**
```go
filtered := StringDesensitize("Sensitive information", "Sensitive")
fmt.Println(filtered) // 输出: ********* information
```

**说明：**  
适用于基于关键词的脱敏处理。

#### 20. `UrlDesensitize`

**功能：**  
对 URL 进行脱敏处理，隐藏账号、密码、IP 等敏感信息。

**用法：**
```go
desensitizedURL := UrlDesensitize("http://admin:admin123@192.168.1.1/path?user=admin&token=123456","user","token")
fmt.Println(desensitizedURL) // 输出: http://username:password@192.*.*.1/path?query=a****&token=1*****
```

**说明：**  
适用于 URL 中的敏感信息脱敏处理。

#### 21. `ClearToNullDesensitize`

**功能：**  
将字符串字段设置为空。


## 依赖

该库使用了 Go 的反射功能和泛型，确保你的 Go 环境支持这些特性（Go 1.18 及以上版本）。

## 贡献

如果你有改进建议或发现问题，请提交 Issue 或 Pull Request。

## 联系方式

如有问题或建议，请联系：[i@shabi.in](mailto:i@shabi.in)
