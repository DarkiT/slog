package slog

import (
	"encoding/json"
	"errors"
	"net/http"
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
		TextEnabled: textEnabled,
		JSONEnabled: jsonEnabled,
		DLPEnabled:  ext != nil && ext.dlpEnabled,
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

// StartRuntimePanel 启动一个简易 HTTP 面板，暴露运行时开关。
// GET  /slog/runtime   -> 返回状态
// POST /slog/runtime   -> 通过 query/form 字段 level/text/json/dlp 调整开关
func StartRuntimePanel(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slog/runtime", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(GetRuntimeSnapshot())
			return
		}
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(RuntimeSnapshot{Message: err.Error()})
				return
			}
			var lastErr error
			for key, vals := range r.Form {
				if len(vals) == 0 {
					continue
				}
				if _, err := ApplyRuntimeOption(key, vals[0]); err != nil {
					lastErr = err
				}
			}
			snap := GetRuntimeSnapshot()
			if lastErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				snap.Message = lastErr.Error()
			}
			_ = json.NewEncoder(w).Encode(snap)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	return srv, nil
}
