package formatter

import (
	"log/slog"
	"time"

	"github.com/darkit/slog/modules"
)

// FormatterAdapter 格式化器模块适配器
type FormatterAdapter struct {
	*modules.BaseModule
	formatters []Formatter
}

// NewFormatterAdapter 创建格式化器适配器
func NewFormatterAdapter() *FormatterAdapter {
	return &FormatterAdapter{
		BaseModule: modules.NewBaseModule("formatter", modules.TypeFormatter, 10),
		formatters: make([]Formatter, 0),
	}
}

// Configure 配置格式化器模块
func (f *FormatterAdapter) Configure(config modules.Config) error {
	if err := f.BaseModule.Configure(config); err != nil {
		return err
	}

	var cfg struct {
		Type        string `json:"type"`
		Format      string `json:"format"`
		Replacement string `json:"replacement"`
	}

	if err := config.Bind(&cfg); err != nil {
		return err
	}

	// 根据配置创建格式化器
	switch cfg.Type {
	case "time":
		format := cfg.Format
		if format == "" {
			format = "2006-01-02 15:04:05"
		}
		f.formatters = append(f.formatters, TimeFormatter(format, time.Local))
	case "error":
		replacement := cfg.Replacement
		if replacement == "" {
			replacement = "error"
		}
		f.formatters = append(f.formatters, ErrorFormatter(replacement))
	case "pii":
		replacement := cfg.Replacement
		if replacement == "" {
			replacement = "*****"
		}
		f.formatters = append(f.formatters, PIIFormatter(replacement))
	}

	return nil
}

// GetFormatters 获取格式化器列表（返回兼容的函数类型）
func (f *FormatterAdapter) GetFormatters() interface{} {
	// 转换为兼容的函数类型
	result := make([]func([]string, slog.Attr) (slog.Value, bool), len(f.formatters))
	for i, formatter := range f.formatters {
		// 避免闭包问题，为每个迭代创建局部变量
		localFormatter := formatter
		result[i] = func(groups []string, attr slog.Attr) (slog.Value, bool) {
			return localFormatter(groups, attr)
		}
	}
	return result
}

// FormatterFunctions 实现 modules.FormatterProvider，避免反射与 interface{} 转换。
func (f *FormatterAdapter) FormatterFunctions() []func([]string, slog.Attr) (slog.Value, bool) {
	funcs := make([]func([]string, slog.Attr) (slog.Value, bool), 0, len(f.formatters))
	for _, formatter := range f.formatters {
		lf := formatter
		funcs = append(funcs, func(groups []string, attr slog.Attr) (slog.Value, bool) {
			return lf(groups, attr)
		})
	}
	return funcs
}

// init 注册formatter模块工厂
func init() {
	modules.RegisterFactory("formatter", func(config modules.Config) (modules.Module, error) {
		adapter := NewFormatterAdapter()
		return adapter, adapter.Configure(config)
	})
}
