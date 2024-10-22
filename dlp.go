package slog

import (
	"sync"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/dlp/header"
)

var (
	dlpOnce   sync.Once
	dlpEngine header.EngineAPI
)

// EnableDLP 启用日志脱敏功能
func EnableDLP() {
	dlpOnce.Do(func() {
		if engine, err := header.NewEngine(); err == nil {
			dlpEngine = engine
		}
	})
}

// DisableDLP 禁用日志脱敏功能
func DisableDLP() {
	if dlpEngine != nil {
		dlpEngine.Config().Disable()
	}
}

// RegisterDLPStrategy 注册自定义脱敏策略
func RegisterDLPStrategy(name string, strategy dlp.DesensitizeFunc) {
	if dlpEngine != nil {
		dlpEngine.Config().RegisterStrategy(name, strategy)
	}
}

// IsDLPEnabled 检查脱敏功能是否已启用
func IsDLPEnabled() bool {
	if dlpEngine != nil {
		return dlpEngine.Config().IsEnabled()
	}
	return false
}

// DlpMask 执行文本脱敏处理
func DlpMask(text string) string {
	if dlpEngine != nil && dlpEngine.Config().IsEnabled() {
		return dlpEngine.DesensitizeText(text)
	}
	return text
}
