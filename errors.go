package slog

import (
	"fmt"
)

// ErrorType 错误类型枚举
type ErrorType int

const (
	ErrorTypeInvalidInput ErrorType = iota
	ErrorTypeProcessing
	ErrorTypeConfiguration
	ErrorTypeInitialization
	ErrorTypeInternal
)

// String 返回错误类型的字符串表示
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeInvalidInput:
		return "InvalidInput"
	case ErrorTypeProcessing:
		return "Processing"
	case ErrorTypeConfiguration:
		return "Configuration"
	case ErrorTypeInitialization:
		return "Initialization"
	case ErrorTypeInternal:
		return "Internal"
	default:
		return "Unknown"
	}
}

// SlogError 结构化错误类型
// 提供更丰富的错误上下文信息，便于调试和错误处理
type SlogError struct {
	Type      ErrorType
	Component string
	Operation string
	Field     string
	Expected  string
	Actual    string
	Details   map[string]interface{}
	Cause     error
}

// Error 实现error接口
func (e *SlogError) Error() string {
	var msg string

	switch e.Type {
	case ErrorTypeInvalidInput:
		if e.Field != "" {
			msg = fmt.Sprintf("slog/%s: invalid input for field '%s'",
				e.Component, e.Field)
			if e.Expected != "" && e.Actual != "" {
				msg += fmt.Sprintf(" - expected %s, got %s", e.Expected, e.Actual)
			}
		} else {
			msg = fmt.Sprintf("slog/%s: invalid input in operation '%s'",
				e.Component, e.Operation)
		}
	case ErrorTypeProcessing:
		msg = fmt.Sprintf("slog/%s: processing failed in operation '%s'",
			e.Component, e.Operation)
		if e.Field != "" {
			msg += fmt.Sprintf(" for field '%s'", e.Field)
		}
	case ErrorTypeConfiguration:
		msg = fmt.Sprintf("slog/%s: configuration error", e.Component)
		if e.Field != "" {
			msg += fmt.Sprintf(" in field '%s'", e.Field)
		}
	case ErrorTypeInitialization:
		msg = fmt.Sprintf("slog/%s: initialization failed", e.Component)
		if e.Operation != "" {
			msg += fmt.Sprintf(" during '%s'", e.Operation)
		}
	case ErrorTypeInternal:
		msg = fmt.Sprintf("slog/%s: internal error", e.Component)
		if e.Operation != "" {
			msg += fmt.Sprintf(" in operation '%s'", e.Operation)
		}
	default:
		msg = fmt.Sprintf("slog/%s: unknown error", e.Component)
	}

	if e.Cause != nil {
		msg += fmt.Sprintf(" - %v", e.Cause)
	}

	return msg
}

// Unwrap 实现errors.Unwrap接口，支持错误链
func (e *SlogError) Unwrap() error {
	return e.Cause
}

// WithDetails 添加详细信息
func (e *SlogError) WithDetails(key string, value interface{}) *SlogError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// GetDetails 获取详细信息
func (e *SlogError) GetDetails() map[string]interface{} {
	if e.Details == nil {
		return make(map[string]interface{})
	}
	return e.Details
}

// 错误构造函数

// NewInvalidInputError 创建输入无效错误
func NewInvalidInputError(field, expected, actual string) *SlogError {
	return &SlogError{
		Type:      ErrorTypeInvalidInput,
		Component: "core",
		Field:     field,
		Expected:  expected,
		Actual:    actual,
	}
}

// NewProcessingError 创建处理错误
func NewProcessingError(component, operation string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeProcessing,
		Component: component,
		Operation: operation,
		Cause:     cause,
	}
}

// NewConfigurationError 创建配置错误
func NewConfigurationError(component, field string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeConfiguration,
		Component: component,
		Field:     field,
		Cause:     cause,
	}
}

// NewInitializationError 创建初始化错误
func NewInitializationError(component, operation string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeInitialization,
		Component: component,
		Operation: operation,
		Cause:     cause,
	}
}

// NewInternalError 创建内部错误
func NewInternalError(component, operation string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeInternal,
		Component: component,
		Operation: operation,
		Cause:     cause,
	}
}

// 特定领域的错误构造函数

// NewDLPError 创建DLP相关错误
func NewDLPError(operation, field string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeProcessing,
		Component: "dlp",
		Operation: operation,
		Field:     field,
		Cause:     cause,
	}
}

// NewModuleError 创建模块相关错误
func NewModuleError(moduleName, operation string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeProcessing,
		Component: fmt.Sprintf("module/%s", moduleName),
		Operation: operation,
		Cause:     cause,
	}
}

// NewFormatterError 创建格式化器相关错误
func NewFormatterError(operation string, cause error) *SlogError {
	return &SlogError{
		Type:      ErrorTypeProcessing,
		Component: "formatter",
		Operation: operation,
		Cause:     cause,
	}
}

// IsErrorType 检查错误是否为指定类型
func IsErrorType(err error, errorType ErrorType) bool {
	if slogErr, ok := err.(*SlogError); ok {
		return slogErr.Type == errorType
	}
	return false
}

// GetErrorComponent 获取错误组件名称
func GetErrorComponent(err error) string {
	if slogErr, ok := err.(*SlogError); ok {
		return slogErr.Component
	}
	return ""
}

// GetErrorOperation 获取错误操作名称
func GetErrorOperation(err error) string {
	if slogErr, ok := err.(*SlogError); ok {
		return slogErr.Operation
	}
	return ""
}
