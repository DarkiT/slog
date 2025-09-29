package dlp

import (
	"testing"
)

func TestFinalStructDesensitization(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 使用真正的中文姓名进行测试
	t.Run("Real Chinese names", func(t *testing.T) {
		type User struct {
			Name string `dlp:"chinese_name"`
		}

		user := &User{Name: "张三"}
		t.Logf("Before: %+v", user)

		err := engine.DesensitizeStructAdvanced(user)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", user)
		if user.Name == "张三" {
			t.Error("Name should be desensitized")
		}
	})

	// 测试嵌套结构体
	t.Run("Nested structs with real names", func(t *testing.T) {
		type Profile struct {
			Name string `dlp:"chinese_name"`
		}

		type User struct {
			Profile Profile `dlp:",recursive"`
		}

		user := &User{
			Profile: Profile{Name: "李四"},
		}
		t.Logf("Before: %+v", user)

		err := engine.DesensitizeStructAdvanced(user)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", user)
		if user.Profile.Name == "李四" {
			t.Error("Nested name should be desensitized")
		}
	})

	// 测试切片
	t.Run("Slice with real names", func(t *testing.T) {
		type User struct {
			Name string `dlp:"chinese_name"`
		}

		type Group struct {
			Users []User `dlp:",recursive"`
		}

		group := &Group{
			Users: []User{
				{Name: "王五"},
				{Name: "赵六"},
			},
		}
		t.Logf("Before: %+v", group)

		err := engine.DesensitizeStructAdvanced(group)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", group)
		for i, user := range group.Users {
			if user.Name == "王五" || user.Name == "赵六" {
				t.Errorf("Users[%d].Name should be desensitized", i)
			}
		}
	})

	// 测试多种字段类型
	t.Run("Multiple field types", func(t *testing.T) {
		type User struct {
			Name  string `dlp:"chinese_name"`
			Phone string `dlp:"mobile_phone"`
			Email string `dlp:"email"`
		}

		user := &User{
			Name:  "陈七",
			Phone: "13812345678",
			Email: "chenqi@example.com",
		}
		t.Logf("Before: %+v", user)

		err := engine.DesensitizeStructAdvanced(user)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", user)
		if user.Name == "陈七" {
			t.Error("Name should be desensitized")
		}
		if user.Phone == "13812345678" {
			t.Error("Phone should be desensitized")
		}
		if user.Email == "chenqi@example.com" {
			t.Error("Email should be desensitized")
		}
	})
}
