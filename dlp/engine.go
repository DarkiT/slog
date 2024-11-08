package dlp

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrInvalidMatcher = errors.New("invalid matcher configuration")
	ErrNotStruct      = errors.New("input must be a struct")
)

var textBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// DlpEngine 定义脱敏引擎结构体
type DlpEngine struct {
	config   *DlpConfig
	searcher *RegexSearcher
	mu       sync.RWMutex
	enabled  atomic.Bool
}

// NewDlpEngine 创建新的DLP引擎实例
func NewDlpEngine() *DlpEngine {
	engine := &DlpEngine{
		config:   GetConfig(),
		searcher: NewRegexSearcher(),
	}
	engine.enabled.Store(false)
	return engine
}

// Enable 启用DLP引擎
func (e *DlpEngine) Enable() {
	e.enabled.Store(true)
}

// Disable 禁用DLP引擎
func (e *DlpEngine) Disable() {
	e.enabled.Store(false)
}

// IsEnabled 检查DLP引擎是否启用
func (e *DlpEngine) IsEnabled() bool {
	return e.enabled.Load()
}

// DesensitizeText 对文本进行脱敏处理
func (e *DlpEngine) DesensitizeText(text string) string {
	if !e.IsEnabled() || text == "" {
		return text
	}

	builder := textBuilderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		textBuilderPool.Put(builder)
	}()

	result := text
	// 按优先级顺序处理不同类型的敏感信息
	sensitiveTypes := e.searcher.GetAllSupportedTypes()
	for _, typeName := range sensitiveTypes {
		result = e.searcher.ReplaceParallel(result, typeName)
		builder.WriteString(result)
		result = builder.String()
		builder.Reset()
	}

	return result
}

// DesensitizeSpecificType 对指定类型的敏感信息进行脱敏
func (e *DlpEngine) DesensitizeSpecificType(text string, sensitiveType string) string {
	if !e.IsEnabled() || text == "" {
		return text
	}
	return e.searcher.ReplaceParallel(text, sensitiveType)
}

// DesensitizeStruct 对结构体进行脱敏处理
func (e *DlpEngine) DesensitizeStruct(data interface{}) error {
	if !e.IsEnabled() {
		return nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		tag := typ.Field(i).Tag.Get("dlp")
		if tag == "" {
			continue
		}

		if field.Kind() == reflect.String {
			desensitized := e.DesensitizeSpecificType(field.String(), tag)
			field.SetString(desensitized)
		}
	}

	return nil
}

// DetectSensitiveInfo 检测文本中的所有敏感信息
func (e *DlpEngine) DetectSensitiveInfo(text string) map[string][]MatchResult {
	if !e.IsEnabled() || text == "" {
		return nil
	}

	results := make(map[string][]MatchResult)
	sensitiveTypes := e.searcher.GetAllSupportedTypes()

	for _, typeName := range sensitiveTypes {
		matches := e.searcher.SearchSensitiveByType(text, typeName)
		if len(matches) > 0 {
			results[typeName] = matches
		}
	}

	return results
}

// RegisterCustomMatcher 注册自定义匹配器
func (e *DlpEngine) RegisterCustomMatcher(matcher *Matcher) error {
	if matcher.Pattern == "" || matcher.Name == "" {
		return ErrInvalidMatcher
	}

	regex, err := regexp.Compile(matcher.Pattern)
	if err != nil {
		return err
	}

	matcher.Regex = regex
	e.searcher.AddMatcher(matcher)
	return nil
}

// GetSupportedTypes 获取所有支持的敏感信息类型
func (e *DlpEngine) GetSupportedTypes() []string {
	return e.searcher.GetAllSupportedTypes()
}
