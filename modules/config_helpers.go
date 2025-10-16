package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Bind 将通用 Config 映射到强类型结构体或其他合法的 JSON 目标对象。
//
// 它通过标准库的 JSON 编解码器完成类型转换，避免在业务代码中散布显式的类型断言。
// 使用示例：
//
//	var opts struct {
//	    Endpoint string        `json:"endpoint"`
//	    Timeout  time.Duration `json:"timeout"`
//	}
//	if err := cfg.Bind(&opts); err != nil { ... }
//
// 任何实现了 json.Unmarshaler / encoding.TextUnmarshaler 的类型，都会被透明支持。
func (c Config) Bind(target any) error {
	if target == nil {
		return fmt.Errorf("modules: bind target cannot be nil")
	}

	// 将 nil map 视为一个空对象，保持行为直观。
	if c == nil {
		return json.Unmarshal([]byte("{}"), target)
	}

	payload, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("modules: marshal config failed: %w", err)
	}

	if bytes.Equal(payload, []byte("null")) {
		payload = []byte("{}")
	}

	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("modules: unmarshal config failed: %w", err)
	}

	return nil
}
