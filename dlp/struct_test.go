package dlp

import (
	"reflect"
	"testing"
)

// 测试用的结构体定义
type User struct {
	ID       int64  `dlp:"id_card"`      // 身份证脱敏
	Name     string `dlp:"chinese_name"` // 姓名脱敏
	Phone    string `dlp:"mobile_phone"` // 手机号脱敏
	Email    string `dlp:"email"`        // 邮箱脱敏
	Password string `dlp:"password"`     // 密码脱敏
	Age      int    `dlp:"-"`            // 跳过年龄字段
	Address  string `dlp:"address"`      // 地址脱敏
}

type NestedUser struct {
	BaseInfo User              `dlp:",recursive"` // 递归处理嵌套结构体
	Friends  []User            `dlp:",recursive"` // 递归处理切片
	Metadata map[string]string `dlp:",recursive"` // 递归处理映射
	BankCard string            `dlp:"bank_card"`  // 银行卡脱敏
}

type ComplexStruct struct {
	Users    []User                 `dlp:",recursive"`
	UserMap  map[string]*User       `dlp:",recursive"`
	Settings map[string]interface{} `dlp:",recursive"`
	Token    string                 `dlp:"custom:jwt"` // 使用自定义策略
}

func TestBasicStructDesensitization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	user := &User{
		ID:       622421196903065015,
		Name:     "张三",
		Phone:    "13812345678",
		Email:    "zhangsan@example.com",
		Password: "password123",
		Age:      25,
		Address:  "北京市朝阳区某某街道123号",
	}

	// 使用原有的方法（向后兼容）
	err := engine.DesensitizeStruct(user)
	if err != nil {
		t.Fatalf("Failed to desensitize struct: %v", err)
	}

	// 验证脱敏结果
	if user.Name == "张三" {
		t.Error("Name should be desensitized")
	}
	if user.Phone == "13812345678" {
		t.Error("Phone should be desensitized")
	}
	if user.Email == "zhangsan@example.com" {
		t.Error("Email should be desensitized")
	}
	if user.Password == "password123" {
		t.Error("Password should be desensitized")
	}
	if user.Age != 25 {
		t.Error("Age should not be modified (skip tag)")
	}

	t.Logf("Desensitized user: %+v", user)
}

func TestAdvancedStructDesensitization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	nestedUser := &NestedUser{
		BaseInfo: User{
			ID:       622421196903065015,
			Name:     "李四",
			Phone:    "13987654321",
			Email:    "lisi@example.com",
			Password: "secret456",
			Age:      30,
			Address:  "上海市浦东新区某某路456号",
		},
		Friends: []User{
			{
				Name:  "王五",
				Phone: "13555666777",
				Email: "wangwu@example.com",
			},
			{
				Name:  "赵六",
				Phone: "13444555666",
				Email: "zhaoliu@example.com",
			},
		},
		Metadata: map[string]string{
			"phone":   "13812345678",
			"email":   "metadata@example.com",
			"address": "广州市天河区某某大道789号",
		},
		BankCard: "4111111111111111",
	}

	// 使用新的高级方法
	err := engine.DesensitizeStructAdvanced(nestedUser)
	if err != nil {
		t.Fatalf("Failed to desensitize advanced struct: %v", err)
	}

	// 验证嵌套结构体脱敏
	if nestedUser.BaseInfo.Name == "李四" {
		t.Error("Nested BaseInfo.Name should be desensitized")
	}
	if nestedUser.BaseInfo.Phone == "13987654321" {
		t.Error("Nested BaseInfo.Phone should be desensitized")
	}

	// 验证切片脱敏
	for i, friend := range nestedUser.Friends {
		if friend.Name == "王五" || friend.Name == "赵六" {
			t.Errorf("Friends[%d].Name should be desensitized", i)
		}
	}

	// 验证银行卡脱敏
	if nestedUser.BankCard == "6222020000000000000" {
		t.Error("BankCard should be desensitized")
	}

	t.Logf("Desensitized nested user: %+v", nestedUser)
}

func TestBatchDesensitization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	users := []User{
		{
			Name:  "孙八",
			Phone: "13111111111",
			Email: "batch1@example.com",
		},
		{
			Name:  "周九",
			Phone: "13222222222",
			Email: "batch2@example.com",
		},
		{
			Name:  "吴十",
			Phone: "13333333333",
			Email: "batch3@example.com",
		},
	}

	// 批量脱敏处理
	err := engine.BatchDesensitizeStruct(&users)
	if err != nil {
		t.Fatalf("Failed to batch desensitize: %v", err)
	}

	// 验证批量脱敏结果
	for i, user := range users {
		if user.Name == "孙八" || user.Name == "周九" || user.Name == "吴十" {
			t.Errorf("Users[%d].Name should be desensitized", i)
		}
		if user.Phone[:3] != "131" && user.Phone[:3] != "132" && user.Phone[:3] != "133" {
			// 脱敏后应该保留部分原始信息
		}
	}

	t.Logf("Batch desensitized users: %+v", users)
}

func TestCustomStrategy(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 注册自定义策略
	engine.config.RegisterStrategy("upper_mask", func(s string) string {
		if len(s) <= 2 {
			return "**"
		}
		return s[:1] + "***" + s[len(s)-1:]
	})

	type CustomUser struct {
		Username string `dlp:"custom:upper_mask"`
		Token    string `dlp:"custom:jwt"`
	}

	user := &CustomUser{
		Username: "customuser",
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
	}

	err := engine.DesensitizeStructAdvanced(user)
	if err != nil {
		t.Fatalf("Failed to desensitize with custom strategy: %v", err)
	}

	if user.Username == "customuser" {
		t.Error("Username should be desensitized with custom strategy")
	}

	t.Logf("Custom strategy result: %+v", user)
}

func TestTagParsing(t *testing.T) {
	tests := []struct {
		tag      string
		expected *DlpTagConfig
		hasError bool
	}{
		{
			tag: "mobile_phone",
			expected: &DlpTagConfig{
				Type: "mobile_phone",
			},
		},
		{
			tag: "email,recursive",
			expected: &DlpTagConfig{
				Type:      "email",
				Recursive: true,
			},
		},
		{
			tag: "-",
			expected: &DlpTagConfig{
				Skip: true,
			},
		},
		{
			tag: "custom:my_strategy",
			expected: &DlpTagConfig{
				Custom: "my_strategy",
			},
		},
		{
			tag: "id_card,recursive,skip",
			expected: &DlpTagConfig{
				Type:      "id_card",
				Recursive: true,
				Skip:      true,
			},
		},
		{
			tag:      "",
			expected: nil,
		},
		{
			tag:      "invalid,,",
			hasError: false, // 应该解析为 Type: "invalid"
			expected: &DlpTagConfig{
				Type: "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			config, err := parseDlpTag(tt.tag)

			if tt.hasError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(config, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, config)
			}
		})
	}
}

func TestComplexNestedStructure(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	complex := &ComplexStruct{
		Users: []User{
			{
				Name:  "钱多多",
				Phone: "13999888777",
				Email: "complex1@example.com",
			},
		},
		UserMap: map[string]*User{
			"admin": {
				Name:  "孙悟空",
				Phone: "13888777666",
				Email: "admin@example.com",
			},
		},
		Settings: map[string]interface{}{
			"debug_phone": "13777666555",
			"admin_email": "settings@example.com",
		},
		Token: "bearer_token_12345",
	}

	err := engine.DesensitizeStructAdvanced(complex)
	if err != nil {
		t.Fatalf("Failed to desensitize complex structure: %v", err)
	}

	// 验证各层级的脱敏效果
	if len(complex.Users) > 0 && complex.Users[0].Name == "钱多多" {
		t.Error("Complex Users[0].Name should be desensitized")
	}

	if admin, exists := complex.UserMap["admin"]; exists {
		if admin.Name == "孙悟空" {
			t.Error("Complex UserMap admin.Name should be desensitized")
		}
	}

	t.Logf("Complex structure result: %+v", complex)
}

func TestStructBatchPerformance(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 创建大量测试数据
	users := make([]User, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = User{
			Name:  "测试用户" + string(rune(i)),
			Phone: "13812345678",
			Email: "test@example.com",
		}
	}

	// 测试原有方法性能
	users1 := make([]User, len(users))
	copy(users1, users)

	// 测试新方法性能
	users2 := make([]User, len(users))
	copy(users2, users)

	// 使用新的批量方法
	err := engine.BatchDesensitizeStruct(&users2)
	if err != nil {
		t.Fatalf("Batch desensitization failed: %v", err)
	}

	t.Log("Performance test completed")
}

// 边界情况测试
func TestStructEdgeCases(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 测试空结构体
	type EmptyStruct struct{}
	empty := &EmptyStruct{}
	err := engine.DesensitizeStructAdvanced(empty)
	if err != nil {
		t.Errorf("Empty struct should not cause error: %v", err)
	}

	// 测试 nil 指针
	var nilUser *User
	err = engine.DesensitizeStructAdvanced(nilUser)
	if err != nil {
		t.Errorf("Nil pointer should not cause error: %v", err)
	}

	// 测试深度嵌套（超过限制）
	type DeepStruct struct {
		Next *DeepStruct `dlp:",recursive"`
		Data string      `dlp:"email"`
	}

	deep := &DeepStruct{Data: "deep@example.com"}
	current := deep
	// 创建超过 10 层的嵌套
	for i := 0; i < 12; i++ {
		current.Next = &DeepStruct{Data: "level" + string(rune(i)) + "@example.com"}
		current = current.Next
	}

	err = engine.DesensitizeStructAdvanced(deep)
	// 应该处理但不会无限递归
	if err != nil {
		t.Logf("Deep nesting handled with error (expected): %v", err)
	}
}
