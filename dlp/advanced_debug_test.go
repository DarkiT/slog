package dlp

import (
	"testing"
)

func TestAdvancedDesensitizationDebugging(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()
	engine.EnablePluginArchitecture()

	// 测试最简单的高级脱敏
	t.Run("Simple advanced test", func(t *testing.T) {
		type SimpleStruct struct {
			Phone string `dlp:"phone"`
			Email string `dlp:"email"`
		}

		obj := &SimpleStruct{
			Phone: "13812345678",
			Email: "test@example.com",
		}
		t.Logf("Before: %+v", obj)

		err := engine.DesensitizeStructAdvanced(obj)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", obj)
		if obj.Phone == "13812345678" {
			t.Error("Phone should be desensitized")
		}
		if obj.Email == "test@example.com" {
			t.Error("Email should be desensitized")
		}
	})

	// 测试递归脱敏
	t.Run("Recursive test", func(t *testing.T) {
		type NestedStruct struct {
			Phone string `dlp:"phone"`
		}

		type ParentStruct struct {
			Child NestedStruct `dlp:",recursive"`
		}

		obj := &ParentStruct{
			Child: NestedStruct{Phone: "13812345678"},
		}
		t.Logf("Before: %+v", obj)

		err := engine.DesensitizeStructAdvanced(obj)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", obj)
		if obj.Child.Phone == "13812345678" {
			t.Error("Child phone should be desensitized")
		}
	})

	// 测试切片脱敏
	t.Run("Slice test", func(t *testing.T) {
		type User struct {
			Phone string `dlp:"phone"`
			Email string `dlp:"email"`
		}

		type Container struct {
			Users []User `dlp:",recursive"`
		}

		obj := &Container{
			Users: []User{
				{Phone: "13812345678", Email: "user1@example.com"},
				{Phone: "13987654321", Email: "user2@example.com"},
			},
		}
		t.Logf("Before: %+v", obj)

		err := engine.DesensitizeStructAdvanced(obj)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		t.Logf("After: %+v", obj)
		for i, user := range obj.Users {
			if user.Phone == "13812345678" || user.Phone == "13987654321" {
				t.Errorf("Users[%d].Phone should be desensitized", i)
			}
			if user.Email == "user1@example.com" || user.Email == "user2@example.com" {
				t.Errorf("Users[%d].Email should be desensitized", i)
			}
		}
	})
}
