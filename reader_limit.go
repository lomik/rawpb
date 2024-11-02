package rawpb

import (
	"io"
)

type Reader interface {
	io.Reader
	io.ByteScanner
}

type readerLimit struct {
	w     Reader
	mem   Allocator
	buf   [10]byte
	limit uint64
}

func newReaderLimit(w Reader, mem Allocator, limit uint64) *readerLimit {
	return &readerLimit{
		w:     w,
		mem:   mem,
		limit: limit,
	}
}

func (r *readerLimit) varintOrBreak() (uint64, bool, error) {
	var ret uint64
	var b byte
	var err error
	i := uint64(0)
	for {
		if r.limit == 0 {
			if i == 0 {
				// can't read first byte. stream ended
				return 0, true, nil
			}
			return ret, true, ErrorTruncated
		}
		b, err = r.w.ReadByte()
		if err != nil {
			if i == 0 && err == io.EOF {
				// can't read first byte. stream ended
				return 0, true, nil
			}
			return ret, true, ErrorTruncated
		}
		r.limit--
		ret += uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 { // last byte of varint
			return ret, false, nil
		}
		i++
	}
}

func (r *readerLimit) varint() (uint64, error) {
	var ret uint64
	var b byte
	var err error
	i := uint64(0)
	for {
		if r.limit == 0 {
			return ret, ErrorTruncated
		}
		b, err = r.w.ReadByte()
		if err != nil {
			return ret, ErrorTruncated
		}
		r.limit--
		ret += uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 { // last byte of varint
			return ret, nil
		}
		i++
	}
}

func (r *readerLimit) next() bool {
	if r.limit == 0 {
		return false
	}
	_, err := r.w.ReadByte()
	if err != nil {
		return false
	}
	if r.w.UnreadByte() != nil {
		return false
	}
	return true
}

func (r *readerLimit) skip(n uint64) error {
	_, err := r.bytes(n)
	return err
}

func (r *readerLimit) bytes(n uint64) ([]byte, error) {
	if n > r.limit {
		return nil, ErrorTruncated
	}
	p := r.mem.Alloc(int(n))

	_, err := io.ReadAtLeast(r.w, p, int(n))
	if err != nil {
		return p, ErrorTruncated
	}
	r.limit -= n
	return p, nil
}

func (r *readerLimit) fixed64() (uint64, error) {
	if r.limit < 8 {
		return 0, ErrorTruncated
	}
	_, err := io.ReadAtLeast(r.w, r.buf[:8], 8)
	if err != nil {
		return 0, ErrorTruncated
	}

	u := uint64(r.buf[0]) | (uint64(r.buf[1]) << 8) | (uint64(r.buf[2]) << 16) | (uint64(r.buf[3]) << 24) |
		(uint64(r.buf[4]) << 32) | (uint64(r.buf[5]) << 40) | (uint64(r.buf[6]) << 48) | (uint64(r.buf[7]) << 56)
	r.limit -= 8
	return u, nil
}

func (r *readerLimit) fixed32() (uint32, error) {
	if r.limit < 4 {
		return 0, ErrorTruncated
	}
	_, err := io.ReadAtLeast(r.w, r.buf[:4], 4)
	if err != nil {
		return 0, ErrorTruncated
	}
	u := uint32(r.buf[0]) | (uint32(r.buf[1]) << 8) | (uint32(r.buf[2]) << 16) | (uint32(r.buf[3]) << 24)
	r.limit -= 4
	return u, nil
}
