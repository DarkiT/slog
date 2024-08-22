package slog

import (
	"fmt"

	"github.com/darkit/slog/dlp"
	"github.com/darkit/slog/dlp/dlpheader"
)

var dlpEngine dlpheader.EngineAPI

func (h *addons) dlpInit() {
	eng, err := dlp.NewEngine("slog.caller")
	if err == nil {
		_ = eng.ApplyConfigDefault()
	}
	dlpEngine = eng
}

// Mask 脱敏传入的字符串并且返回脱敏后的结果,这里用godlp实现，所有的识别及脱敏算法全都用godlp的开源内容，当然也可以自己写或者扩展
func (h *addons) Mask(inStr string) string {
	if dlpEngine == nil {
		fmt.Println("dlp engine is nil")
		return inStr
	}
	if outStr, _, err := dlpEngine.Deidentify(inStr); err == nil {
		return customSanitize(outStr)
	}
	return inStr
}

// Close 关闭脱敏处理器
func (h *addons) Close() {
	dlpEngine.Close()
}
