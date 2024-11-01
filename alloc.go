package rawpb

import (
	"slices"
)

type Allocator interface {
	Alloc(n int) []byte
}

type HeapAllocator struct {
}

func (a *HeapAllocator) Alloc(n int) []byte {
	return make([]byte, n)
}

type LinearAllocator struct {
	offset int
	buf    []byte
}

func NewLinearAllocator() *LinearAllocator {
	return &LinearAllocator{buf: make([]byte, 0)}
}

func (a *LinearAllocator) Alloc(n int) []byte {
	if a.offset+n <= len(a.buf) {
		p := a.buf[a.offset : a.offset+n]
		a.offset += n
		return p
	}

	a.buf = slices.Grow(a.buf, n-len(a.buf)+a.offset)
	a.buf = a.buf[:cap(a.buf)]
	p := a.buf[a.offset : a.offset+n]
	a.offset += n
	return p
}

func (a *LinearAllocator) Reset() {
	a.offset = 0
}

func (a *LinearAllocator) Grow(size int) {
	a.buf = slices.Grow(a.buf, size)
	a.buf = a.buf[:cap(a.buf)]
}
