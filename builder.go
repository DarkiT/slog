package slog

import (
	"io"
	"os"

	gelfmod "github.com/darkit/slog/modules/output/gelf"
)

// LoggerBuilder 通过链式方式快速构建 Logger，便于上层按需开启 Text/JSON/DLP、预置分组与字段。
type LoggerBuilder struct {
	cfg    *Config
	writer io.Writer
	module string
	groups []string
	attrs  []any
	mode   string // "", "logfmt", "gelf"
	gopts  *gelfmod.Options
}

// NewLoggerBuilder 创建一个新的构建器，默认输出到 stdout、启用文本日志。
func NewLoggerBuilder() *LoggerBuilder {
	cfg := DefaultConfig()
	return &LoggerBuilder{
		cfg:    cfg,
		writer: os.Stdout,
	}
}

// WithWriter 指定输出目标。
func (b *LoggerBuilder) WithWriter(w io.Writer) *LoggerBuilder {
	if w != nil {
		b.writer = w
	}
	return b
}

// WithConfig 使用自定义配置，内部会复制一份避免外部修改产生副作用。
func (b *LoggerBuilder) WithConfig(cfg *Config) *LoggerBuilder {
	if cfg == nil {
		return b
	}
	copyCfg := *cfg
	b.cfg = &copyCfg
	return b
}

// WithModule 为日志添加模块字段。
func (b *LoggerBuilder) WithModule(name string) *LoggerBuilder {
	b.module = name
	return b
}

// WithGroup 预置日志分组。
func (b *LoggerBuilder) WithGroup(name string) *LoggerBuilder {
	if name != "" {
		b.groups = append(b.groups, name)
	}
	return b
}

// WithAttrs 预置结构化字段。
func (b *LoggerBuilder) WithAttrs(attrs ...Attr) *LoggerBuilder {
	for _, a := range attrs {
		b.attrs = append(b.attrs, a)
	}
	return b
}

// EnableText 控制文本输出。
func (b *LoggerBuilder) EnableText(on bool) *LoggerBuilder {
	b.cfg.SetEnableText(on)
	return b
}

// EnableJSON 控制 JSON 输出。
func (b *LoggerBuilder) EnableJSON(on bool) *LoggerBuilder {
	b.cfg.SetEnableJSON(on)
	return b
}

// UseLogfmt 切换为 logfmt 输出。
func (b *LoggerBuilder) UseLogfmt() *LoggerBuilder {
	b.mode = "logfmt"
	return b
}

// UseGELF 切换为 GELF 输出，并可附带选项。
func (b *LoggerBuilder) UseGELF(opts *gelfmod.Options) *LoggerBuilder {
	b.mode = "gelf"
	b.gopts = opts
	return b
}

// EnableDLP 控制 DLP 脱敏能力。
func (b *LoggerBuilder) EnableDLP(on bool) *LoggerBuilder {
	if on {
		EnableDLPLogger()
	} else {
		DisableDLPLogger()
	}
	return b
}

// Build 构建 Logger 实例。
func (b *LoggerBuilder) Build() *Logger {
	var logger *Logger
	switch b.mode {
	case "logfmt":
		logger = NewLogfmtLogger(b.writer, nil)
	case "gelf":
		logger = NewGELFLogger(b.writer, nil, b.gopts)
	default:
		logger = NewLoggerWithConfig(b.writer, b.cfg)
	}
	if b.module != "" {
		logger = logger.With("module", b.module)
	}
	for _, g := range b.groups {
		logger = logger.WithGroup(g)
	}
	if len(b.attrs) > 0 {
		logger = logger.With(b.attrs...)
	}
	return logger
}
