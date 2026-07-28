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

// LinearAllocator is an arena allocator: every call to Alloc bumps an
// internal offset into a single growable buffer, making per-field allocation
// essentially free at the cost of an all-or-nothing lifetime — the whole
// arena is reclaimed with a single Reset.
//
// Lifetime contract for slices returned by Alloc:
//
//   - Between two Reset calls, all returned slices remain valid and
//     non-overlapping.
//
//   - Reset does NOT zero the buffer and does NOT invalidate returned slices
//     at the Go type-system level. It only rewinds the offset. Subsequent
//     Alloc calls will start handing out memory from the beginning of the
//     buffer again, silently overwriting the bytes referenced by any slice
//     you retained from before the Reset. Treat every previously returned
//     slice as logically invalid the moment Reset is called; copy anything
//     you need to keep (bytes.Clone, strings.Clone, CopyString) BEFORE
//     calling Reset.
//
//   - Growth via slices.Grow may reallocate the underlying array. When that
//     happens, slices handed out before the growth stay alive via their own
//     reference to the old array — they don't observe writes to the new one.
//     Do not rely on this side effect for safety; the contract above is what
//     you can depend on.
//
//   - LinearAllocator is not safe for concurrent use. Serialize Alloc,
//     Reset, and Grow calls externally if the allocator is shared across
//     goroutines.
//
// Typical usage: create one LinearAllocator per parsing goroutine, pass it
// to RawPB.Read, and call Reset between messages when parsing a stream.
type LinearAllocator struct {
	offset int
	buf    []byte
}

// NewLinearAllocator creates a linear allocator with initial capacity
func NewLinearAllocator() *LinearAllocator {
	return &LinearAllocator{buf: make([]byte, 0)}
}

// Alloc returns a fresh slice of exactly n bytes carved from the arena,
// growing the backing buffer if needed. The returned slice is valid until
// the next Reset; see LinearAllocator for the full lifetime contract.
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

// Reset rewinds the arena so the next Alloc reuses memory from the beginning
// of the buffer.
//
// Reset does NOT zero the buffer. Slices previously returned by Alloc are
// logically invalidated: subsequent Alloc calls will overwrite their
// contents. Copy anything you need to retain across Reset before calling
// this method. See LinearAllocator for the full lifetime contract.
func (a *LinearAllocator) Reset() {
	a.offset = 0
}

// Grow pre-allocates at least size bytes of arena capacity so that Alloc
// calls up to that total avoid mid-parse reallocations. Useful when the
// maximum message size is known in advance.
func (a *LinearAllocator) Grow(size int) {
	a.buf = slices.Grow(a.buf, size)
	a.buf = a.buf[:cap(a.buf)]
}
