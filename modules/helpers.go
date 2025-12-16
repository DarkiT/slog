package modules

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// WriteSyncer 兼容 io.Writer，用于输出 handler。
type WriteSyncer interface {
	io.Writer
}

// NewStdWriter 返回标准输出 writer。
func NewStdWriter() WriteSyncer {
	return os.Stdout
}

// Frame 获取调用帧。
func Frame(pc uintptr) runtime.Frame {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return f
}

// SourceLabel 将 frame 转为短路径标签。
func SourceLabel(f runtime.Frame) string {
	return filepath.Base(f.File) + ":" + itoa(int(f.Line))
}

// itoa 简单整数转字符串，避免 fmt 依赖。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	n := i
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
