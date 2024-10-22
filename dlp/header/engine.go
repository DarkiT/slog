package header

import "github.com/darkit/slog/dlp"

// Config 配置接口定义
type Config interface {
	// Enable 启用脱敏功能
	Enable()

	// Disable 禁用脱敏功能
	Disable()

	// IsEnabled 检查脱敏功能是否已启用
	IsEnabled() bool

	// RegisterStrategy 注册新的脱敏策略
	RegisterStrategy(name string, strategy dlp.DesensitizeFunc)
}

// EngineAPI 定义了DLP引擎必须实现的接口
type EngineAPI interface {
	// Config 获取配置接口
	Config() Config

	// DesensitizeText 对文本内容进行脱敏处理
	DesensitizeText(text string) string

	// DesensitizeStruct 对结构体进行脱敏处理
	DesensitizeStruct(data interface{}) error

	// Mask 使用指定模型对文本进行脱敏
	Mask(text string, model string) (string, error)

	// Deidentify 对文本进行默认脱敏处理
	Deidentify(text string) (string, string, error)
}

// Engine 实现了 EngineAPI 接口
type Engine struct {
	config *dlp.DlpConfig
	engine *dlp.DlpEngine
}

// NewEngine 创建新的DLP引擎实例
func NewEngine() (*Engine, error) {
	engine := &Engine{
		config: dlp.GetConfig(),
		engine: dlp.NewDlpEngine(),
	}
	return engine, nil
}

// Config 获取配置接口
func (e *Engine) Config() Config {
	return e.config
}

// DesensitizeText 对文本进行脱敏处理
func (e *Engine) DesensitizeText(text string) string {
	if !e.config.IsEnabled() {
		return text
	}
	return e.engine.DesensitizeText(text)
}

// DesensitizeStruct 对结构体进行脱敏处理
func (e *Engine) DesensitizeStruct(data interface{}) error {
	if !e.config.IsEnabled() {
		return nil
	}
	return e.engine.DesensitizeStruct(data)
}

// Mask 使用指定模型对文本进行脱敏
func (e *Engine) Mask(text string, model string) (string, error) {
	if !e.config.IsEnabled() {
		return text, nil
	}

	// 如果没有指定模型,使用默认脱敏
	if model == "" {
		return e.DesensitizeText(text), nil
	}

	// 获取指定模型的脱敏策略
	if strategy, ok := e.config.GetStrategy(model); ok {
		return strategy(text), nil
	}

	// 找不到指定模型,返回原文
	return text, nil
}

// Deidentify 对文本进行默认脱敏处理
func (e *Engine) Deidentify(text string) (string, string, error) {
	if !e.config.IsEnabled() {
		return text, "", nil
	}

	// 执行默认脱敏
	result := e.DesensitizeText(text)

	// 记录使用的模型,这里返回默认模型标识
	return result, "default", nil
}
