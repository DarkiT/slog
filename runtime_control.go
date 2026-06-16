package slog

import (
	"errors"
	"strings"
)

// RuntimeSnapshot 描述当前运行时开关状态，便于面板/CLI 展示。
type RuntimeSnapshot struct {
	Level       Level  `json:"level"`
	TextEnabled bool   `json:"text_enabled"`
	JSONEnabled bool   `json:"json_enabled"`
	DLPEnabled  bool   `json:"dlp_enabled"`
	DLPVersion  int64  `json:"dlp_version"`
	Message     string `json:"message,omitempty"`
}

// GetRuntimeSnapshot 返回当前运行时状态快照。
func GetRuntimeSnapshot() RuntimeSnapshot {
	var dlpVersion int64
	if ext != nil && ext.dlpEngine != nil {
		dlpVersion = ext.dlpEngine.Version()
	}
	return RuntimeSnapshot{
		Level:       levelVar.Level(),
		TextEnabled: isGlobalTextEnabled(),
		JSONEnabled: isGlobalJSONEnabled(),
		DLPEnabled:  ext != nil && ext.dlpEnabled.Load(),
		DLPVersion:  dlpVersion,
	}
}

// ApplyRuntimeOption 通过字符串选项调整全局开关，返回更新后的状态。
func ApplyRuntimeOption(option, value string) (RuntimeSnapshot, error) {
	switch strings.ToLower(option) {
	case "level":
		if err := SetLevel(value); err != nil {
			return GetRuntimeSnapshot(), err
		}
	case "text":
		if strings.ToLower(value) == "on" || strings.ToLower(value) == "true" {
			EnableTextLogger()
		} else {
			DisableTextLogger()
		}
	case "json":
		if strings.ToLower(value) == "on" || strings.ToLower(value) == "true" {
			EnableJSONLogger()
		} else {
			DisableJSONLogger()
		}
	case "dlp":
		if strings.ToLower(value) == "on" || strings.ToLower(value) == "true" {
			EnableDLPLogger()
		} else {
			DisableDLPLogger()
		}
	default:
		return GetRuntimeSnapshot(), errors.New("unknown runtime option")
	}
	return GetRuntimeSnapshot(), nil
}
