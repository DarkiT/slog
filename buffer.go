package slog

import (
	"io"
	"sync"
	"sync/atomic"
)

const (
	defaultBufferSize  = 4096     // 默认buffer大小(1KB)
	maxBufferSize      = 32 << 10 // 最大buffer大小(16KB)
	initialBufferSlice = 128      // 初始切片大小,用于小对象优化
)

// buffer 定义了一个高效的字节缓冲区
// 包含主缓冲区、临时缓冲区(用于小对象优化)和操作计数器
type buffer struct {
	buf []byte                   // 主缓冲区,存储实际数据
	tmp [initialBufferSlice]byte // 临时缓冲区,用于优化小对象写入
	n   int32                    // 原子操作计数器,记录写入次数
}

// bufferPool 全局buffer池实例
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &buffer{
			buf: make([]byte, 0, defaultBufferSize),
		}
	},
}

// newBuffer 从全局池中获取一个buffer实例
// 如果池中没有可用的buffer,会创建一个新的
func newBuffer() *buffer {
	return bufferPool.Get().(*buffer)
}

// Free 释放buffer资源
// 如果buffer大小合适且使用次数未超过阈值,则放回池中复用
// 否则直接丢弃以防止内存泄漏
func (b *buffer) Free() {
	// 避免过大的buffer进入池
	if cap(b.buf) <= maxBufferSize && atomic.LoadInt32(&b.n) < 1000 {
		b.Reset()
		bufferPool.Put(b)
		return
	}
	// 过大或使用次数过多的buffer直接丢弃
	b.buf = nil
}

// Reset 重置buffer状态
// 清空内容并重置计数器,但保留已分配的容量
func (b *buffer) Reset() {
	b.buf = b.buf[:0]
	atomic.StoreInt32(&b.n, 0)
}

// Write 实现了io.Writer接口的写入方法
// 包含小对象优化和自动扩容机制
func (b *buffer) Write(p []byte) (n int, err error) {
	atomic.AddInt32(&b.n, 1)
	// 小对象优化:对于小于临时缓冲区的数据,先复制到临时区
	if len(p) <= len(b.tmp) {
		copy(b.tmp[:], p)
		b.buf = append(b.buf, b.tmp[:len(p)]...)
		return len(p), nil
	}

	// 容量不足时进行扩容
	if cap(b.buf)-len(b.buf) < len(p) {
		newCap := cap(b.buf) * 2
		if newCap < cap(b.buf)+len(p) {
			newCap = cap(b.buf) + len(p)
		}
		if newCap > maxBufferSize {
			newCap = maxBufferSize
		}
		newBuf := make([]byte, len(b.buf), newCap)
		copy(newBuf, b.buf)
		b.buf = newBuf
	}

	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteStringIf 条件写入字符串
// 只有在条件为true时才执行写入操作,可以避免不必要的内存分配
// 参数:
//   - ok: 写入条件
//   - str: 要写入的字符串
func (b *buffer) WriteStringIf(ok bool, str string) {
	if !ok {
		return
	}
	b.buf = append(b.buf, str...)
}

// WriteString 写入字符串到buffer
// 实现了io.StringWriter接口
func (b *buffer) WriteString(s string) (n int, err error) {
	atomic.AddInt32(&b.n, 1)
	b.buf = append(b.buf, s...)
	return len(s), nil
}

// WriteByte 写入单个字节到buffer
// 实现了io.ByteWriter接口
func (b *buffer) WriteByte(c byte) error {
	atomic.AddInt32(&b.n, 1)
	b.buf = append(b.buf, c)
	return nil
}

// WriteTo 将buffer中的数据写入到io.Writer
// 实现了io.WriterTo接口
func (b *buffer) WriteTo(w io.Writer) (n int64, err error) {
	if len(b.buf) == 0 {
		return 0, nil
	}
	nBytes, err := w.Write(b.buf)
	return int64(nBytes), err
}

// String 返回buffer中的内容作为字符串
// 实现了fmt.Stringer接口
func (b *buffer) String() string {
	return string(b.buf)
}
