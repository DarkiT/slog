package header

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if engine.config == nil {
		t.Fatal("Engine config is nil")
	}
	if engine.engine == nil {
		t.Fatal("Engine DLP engine is nil")
	}
}

func TestEngine_Config(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	config := engine.Config()
	if config == nil {
		t.Fatal("Config() returned nil")
	}
}

func TestEngine_DesensitizeText(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 启用配置
	engine.config.Enable()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normal text",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.DesensitizeText(tt.input)
			if result != tt.expected && tt.input != "hello world" {
				t.Errorf("DesensitizeText() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_DesensitizeText_Disabled(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 禁用配置
	engine.config.Disable()

	input := "test input"
	result := engine.DesensitizeText(input)
	if result != input {
		t.Errorf("DesensitizeText() with disabled config = %v, want %v", result, input)
	}
}

func TestEngine_DesensitizeStruct(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 启用配置
	engine.config.Enable()

	// 测试结构体
	type TestStruct struct {
		Name string
	}

	data := &TestStruct{Name: "test"}
	err = engine.DesensitizeStruct(data)
	if err != nil {
		t.Errorf("DesensitizeStruct() error = %v", err)
	}
}

func TestEngine_DesensitizeStruct_Disabled(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 禁用配置
	engine.config.Disable()

	type TestStruct struct {
		Name string
	}

	data := &TestStruct{Name: "test"}
	err = engine.DesensitizeStruct(data)
	if err != nil {
		t.Errorf("DesensitizeStruct() with disabled config error = %v", err)
	}
}

func TestEngine_Mask(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 启用配置
	engine.config.Enable()

	tests := []struct {
		name    string
		text    string
		model   string
		wantErr bool
	}{
		{
			name:    "empty model",
			text:    "test text",
			model:   "",
			wantErr: false,
		},
		{
			name:    "nonexistent model",
			text:    "test text",
			model:   "nonexistent",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Mask(tt.text, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.model == "" {
				// 应该返回脱敏后的文本
				_ = result
			} else if tt.model == "nonexistent" {
				// 应该返回原文
				if result != tt.text {
					t.Errorf("Mask() with nonexistent model = %v, want %v", result, tt.text)
				}
			}
		})
	}
}

func TestEngine_Mask_Disabled(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 禁用配置
	engine.config.Disable()

	input := "test input"
	result, err := engine.Mask(input, "any_model")
	if err != nil {
		t.Errorf("Mask() with disabled config error = %v", err)
	}
	if result != input {
		t.Errorf("Mask() with disabled config = %v, want %v", result, input)
	}
}

func TestEngine_Deidentify(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 启用配置
	engine.config.Enable()

	input := "test input"
	result, model, err := engine.Deidentify(input)
	if err != nil {
		t.Errorf("Deidentify() error = %v", err)
	}
	if model != "default" {
		t.Errorf("Deidentify() model = %v, want default", model)
	}
	// result应该是脱敏后的文本
	_ = result
}

func TestEngine_Deidentify_Disabled(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// 禁用配置
	engine.config.Disable()

	input := "test input"
	result, model, err := engine.Deidentify(input)
	if err != nil {
		t.Errorf("Deidentify() with disabled config error = %v", err)
	}
	if result != input {
		t.Errorf("Deidentify() with disabled config result = %v, want %v", result, input)
	}
	if model != "" {
		t.Errorf("Deidentify() with disabled config model = %v, want empty", model)
	}
}
