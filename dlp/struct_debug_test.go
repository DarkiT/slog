package dlp

import (
	"reflect"
	"testing"
)

func TestStructDebugDetailed(t *testing.T) {
	engine := NewDlpEngine()
	engine.Enable()

	// 手动测试标签解析
	t.Run("Tag parsing", func(t *testing.T) {
		config, err := parseDlpTag("chinese_name")
		if err != nil {
			t.Fatalf("Tag parsing error: %v", err)
		}
		t.Logf("Parsed config: %+v", config)

		if config == nil {
			t.Error("Config should not be nil")
		}
		if config != nil && config.Type != "chinese_name" {
			t.Errorf("Expected type 'chinese_name', got '%s'", config.Type)
		}
	})

	// 手动测试字段处理
	t.Run("Manual field processing", func(t *testing.T) {
		type TestStruct struct {
			Name string `dlp:"chinese_name"`
		}

		obj := &TestStruct{Name: "手动测试"}
		val := reflect.ValueOf(obj).Elem()
		field := val.Field(0)
		fieldType := val.Type().Field(0)

		t.Logf("Field name: %s", fieldType.Name)
		t.Logf("Field tag: %s", fieldType.Tag.Get("dlp"))
		t.Logf("Field can set: %v", field.CanSet())
		t.Logf("Field kind: %v", field.Kind())
		t.Logf("Field value: %s", field.String())

		// 手动解析标签
		tag := fieldType.Tag.Get("dlp")
		config, err := parseDlpTag(tag)
		if err != nil {
			t.Fatalf("Tag parsing failed: %v", err)
		}
		t.Logf("Parsed config: %+v", config)

		// 手动调用脱敏
		if config != nil && config.Type != "" {
			original := field.String()
			desensitized := engine.DesensitizeSpecificType(original, config.Type)
			t.Logf("Original: %s, Desensitized: %s", original, desensitized)
			field.SetString(desensitized)
			t.Logf("Field after setting: %s", field.String())
		}

		t.Logf("Final object: %+v", obj)
	})

	// 测试高级方法的各个步骤
	t.Run("Advanced method steps", func(t *testing.T) {
		type TestStruct struct {
			Name string `dlp:"chinese_name"`
		}

		obj := &TestStruct{Name: "步骤测试"}
		t.Logf("Original: %+v", obj)

		structProcessor := NewStructDesensitizer(engine)

		// 手动调用方法链
		val := reflect.ValueOf(obj)
		t.Logf("Initial val kind: %v", val.Kind())

		err := structProcessor.desensitizeValue(val, 0)
		if err != nil {
			t.Fatalf("DesensitizeValue error: %v", err)
		}

		t.Logf("After desensitizeValue: %+v", obj)
	})
}
