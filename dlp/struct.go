package dlp

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedType = errors.New("unsupported field type for desensitization")
	ErrInvalidTag      = errors.New("invalid dlp tag configuration")
)

// StructDesensitizer 结构体脱敏器
type StructDesensitizer struct {
	engine *DlpEngine
}

// NewStructDesensitizer 创建结构体脱敏器
func NewStructDesensitizer(engine *DlpEngine) *StructDesensitizer {
	return &StructDesensitizer{
		engine: engine,
	}
}

// DlpTagConfig 脱敏标签配置
type DlpTagConfig struct {
	Type      string // 脱敏类型
	Recursive bool   // 是否递归处理嵌套结构体
	Skip      bool   // 是否跳过此字段
	Custom    string // 自定义脱敏策略名称
}

// parseDlpTag 解析 dlp 标签
// 支持格式：
// - "type"                    // 简单类型
// - "type,recursive"          // 类型+递归
// - "type,skip"              // 类型+跳过
// - "custom:strategy_name"    // 自定义策略
// - "-"                      // 跳过字段
func parseDlpTag(tag string) (*DlpTagConfig, error) {
	if tag == "" {
		return nil, nil
	}

	if tag == "-" {
		return &DlpTagConfig{Skip: true}, nil
	}

	config := &DlpTagConfig{}
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // 跳过空字符串
		}
		switch part {
		case "recursive":
			config.Recursive = true
		case "skip":
			config.Skip = true
		default:
			if strings.HasPrefix(part, "custom:") {
				config.Custom = strings.TrimPrefix(part, "custom:")
			} else if config.Type == "" && part != "" {
				config.Type = part
			}
		}
	}

	if config.Type == "" && config.Custom == "" && !config.Skip && !config.Recursive {
		return nil, ErrInvalidTag
	}

	return config, nil
}

// DesensitizeStructAdvanced 高级结构体脱敏处理
func (s *StructDesensitizer) DesensitizeStructAdvanced(data interface{}) error {
	if !s.engine.IsEnabled() {
		return nil
	}

	return s.desensitizeValue(reflect.ValueOf(data), 0)
}

// desensitizeValue 递归处理值
func (s *StructDesensitizer) desensitizeValue(val reflect.Value, depth int) error {
	// 防止无限递归
	if depth > 10 {
		return errors.New("maximum recursion depth exceeded")
	}

	// 处理指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return s.desensitizeStruct(val, depth)
	case reflect.Slice, reflect.Array:
		return s.desensitizeSlice(val, depth)
	case reflect.Map:
		return s.desensitizeMap(val, depth)
	default:
		// 基本类型不需要特殊处理
		return nil
	}
}

// desensitizeStruct 处理结构体
func (s *StructDesensitizer) desensitizeStruct(val reflect.Value, depth int) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可设置的字段
		if !field.CanSet() {
			continue
		}

		// 获取标签配置
		tag := fieldType.Tag.Get("dlp")
		config, err := parseDlpTag(tag)
		if err != nil {
			continue // 忽略无效标签
		}

		// 跳过标记为跳过的字段
		if config != nil && config.Skip {
			continue
		}

		// 处理字段
		if err := s.desensitizeField(field, config, depth); err != nil {
			// 记录错误但继续处理其他字段
			continue
		}
	}

	return nil
}

// desensitizeField 处理单个字段
func (s *StructDesensitizer) desensitizeField(field reflect.Value, config *DlpTagConfig, depth int) error {
	if config == nil {
		// 没有标签配置，但可能需要递归处理
		return s.desensitizeValue(field, depth+1)
	}

	switch field.Kind() {
	case reflect.String:
		return s.desensitizeStringField(field, config)
	case reflect.Struct:
		if config.Recursive {
			return s.desensitizeValue(field, depth+1)
		}
	case reflect.Ptr:
		if !field.IsNil() && config.Recursive {
			return s.desensitizeValue(field, depth+1)
		}
	case reflect.Slice, reflect.Array:
		if config.Recursive {
			return s.desensitizeValue(field, depth+1)
		}
	case reflect.Map:
		if config.Recursive {
			return s.desensitizeValue(field, depth+1)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return s.desensitizeIntField(field, config)
	case reflect.Float32, reflect.Float64:
		return s.desensitizeFloatField(field, config)
	}

	return nil
}

// desensitizeStringField 处理字符串字段
func (s *StructDesensitizer) desensitizeStringField(field reflect.Value, config *DlpTagConfig) error {
	original := field.String()
	if original == "" {
		return nil
	}

	var desensitized string
	if config.Custom != "" {
		// 使用自定义策略
		if strategy, ok := s.engine.config.GetStrategy(config.Custom); ok {
			desensitized = strategy(original)
		} else {
			return fmt.Errorf("custom strategy '%s' not found", config.Custom)
		}
	} else if config.Type != "" {
		// 使用指定类型进行脱敏
		desensitized = s.engine.DesensitizeSpecificType(original, config.Type)
	} else {
		// 自动检测并脱敏
		desensitized = s.engine.DesensitizeText(original)
	}

	field.SetString(desensitized)
	return nil
}

// desensitizeIntField 处理整数字段
func (s *StructDesensitizer) desensitizeIntField(field reflect.Value, config *DlpTagConfig) error {
	if config.Type == "" {
		return nil
	}

	original := field.Int()
	originalStr := strconv.FormatInt(original, 10)

	var desensitized string
	if config.Custom != "" {
		if strategy, ok := s.engine.config.GetStrategy(config.Custom); ok {
			desensitized = strategy(originalStr)
		} else {
			return fmt.Errorf("custom strategy '%s' not found", config.Custom)
		}
	} else {
		desensitized = s.engine.DesensitizeSpecificType(originalStr, config.Type)
	}

	// 尝试转换回整数，如果失败则保留原值
	if newVal, err := strconv.ParseInt(desensitized, 10, 64); err == nil {
		field.SetInt(newVal)
	}

	return nil
}

// desensitizeFloatField 处理浮点数字段
func (s *StructDesensitizer) desensitizeFloatField(field reflect.Value, config *DlpTagConfig) error {
	if config.Type == "" {
		return nil
	}

	original := field.Float()
	originalStr := strconv.FormatFloat(original, 'f', -1, 64)

	var desensitized string
	if config.Custom != "" {
		if strategy, ok := s.engine.config.GetStrategy(config.Custom); ok {
			desensitized = strategy(originalStr)
		} else {
			return fmt.Errorf("custom strategy '%s' not found", config.Custom)
		}
	} else {
		desensitized = s.engine.DesensitizeSpecificType(originalStr, config.Type)
	}

	// 尝试转换回浮点数，如果失败则保留原值
	if newVal, err := strconv.ParseFloat(desensitized, 64); err == nil {
		field.SetFloat(newVal)
	}

	return nil
}

// desensitizeSlice 处理切片和数组
func (s *StructDesensitizer) desensitizeSlice(val reflect.Value, depth int) error {
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		if err := s.desensitizeValue(elem, depth+1); err != nil {
			continue // 继续处理其他元素
		}
	}
	return nil
}

// desensitizeMap 处理映射
func (s *StructDesensitizer) desensitizeMap(val reflect.Value, depth int) error {
	if !val.CanSet() {
		return nil
	}

	// 对于map，我们需要重新设置值，因为MapIndex返回的是不可设置的值
	for _, key := range val.MapKeys() {
		mapVal := val.MapIndex(key)

		// 创建一个新的可设置的值
		if mapVal.IsValid() {
			newVal := reflect.New(mapVal.Type()).Elem()
			newVal.Set(mapVal)

			if err := s.desensitizeValue(newVal, depth+1); err == nil {
				// 只有在脱敏成功时才更新map中的值
				val.SetMapIndex(key, newVal)
			}
		}
	}
	return nil
}

// BatchDesensitizeStruct 批量处理结构体切片
func (s *StructDesensitizer) BatchDesensitizeStruct(data interface{}) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Slice {
		return errors.New("input must be a slice")
	}

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		// 直接操作reflect.Value而不是Interface()，避免值拷贝问题
		if err := s.desensitizeValue(elem, 0); err != nil {
			// 记录错误但继续处理其他元素
			continue
		}
	}

	return nil
}
