package slog

import (
	"sync"
)

type buffer []byte

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*buffer)(&b)
	},
}

func newBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

func (b *buffer) Free() {
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

func (b *buffer) Reset() {
	*b = (*b)[:0]
}

func (b *buffer) Write(p []byte) {
	*b = append(*b, p...)
	return
}

func (b *buffer) WriteString(s string) {
	*b = append(*b, s...)
	return
}

func (b *buffer) WriteStringIf(ok bool, str string) {
	if !ok {
		return
	}
	b.WriteString(str)
}

func (b *buffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

func (b *buffer) WritePosInt(i int) {
	b.WritePosIntWidth(i, 0)
}

func (b *buffer) WritePosIntWidth(i, width int) {
	if i < 0 {
		panic("negative int")
	}

	var bb [20]byte
	bp := len(bb) - 1
	for i >= 10 || width > 1 {
		width--
		q := i / 10
		bb[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	bb[bp] = byte('0' + i)
	b.Write(bb[bp:])
}

func (b *buffer) String() string {
	return string(*b)
}
