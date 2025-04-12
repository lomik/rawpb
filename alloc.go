package rawpb

import (
	"slices"
)

// Allocator manages byte buffer allocations for protocol buffer parsing
type Allocator interface {
	// Alloc returns a byte slice of at least n bytes
	Alloc(n int) []byte
}

// HeapAllocator uses Go's built-in memory allocation
type HeapAllocator struct {
}

func (a *HeapAllocator) Alloc(n int) []byte {
	return make([]byte, n)
}

// LinearAllocator uses a single growing buffer for allocations
type LinearAllocator struct {
	offset int
	buf    []byte
}

// NewLinearAllocator creates a linear allocator with initial capacity
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

// Reset recycles the allocation buffer for reuse
func (a *LinearAllocator) Reset() {
	a.offset = 0
}

// Grow pre-allocates buffer space for expected allocations
func (a *LinearAllocator) Grow(size int) {
	a.buf = slices.Grow(a.buf, size)
	a.buf = a.buf[:cap(a.buf)]
}
