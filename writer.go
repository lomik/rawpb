package rawpb

import (
	"bytes"
	"io"
	"math"
	"unsafe"
)

const (
	wireVarint = 0
	wireI64    = 1
	wireLen    = 2
	wireI32    = 5
)

// Writer implements a low-level protocol buffer writer without code generation
type Writer struct {
	wrap      io.Writer
	buf       [10]byte
	subBuffer bytes.Buffer
	subWriter *Writer
	err       error
}

// Write proto message
func Write(out io.Writer, cb func(w *Writer) error) error {
	w := NewWriter(out)

	err := cb(w)

	if err != nil {
		return err
	}

	return w.Err()
}

// NewWriter creates a new Writer instance
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		wrap: w,
	}
}

// Message writes a protocol buffer submessage using a callback function
func (w *Writer) Message(num int, cb func(w *Writer) error) {
	if w.err != nil || cb == nil {
		return
	}

	if w.subWriter == nil {
		w.subWriter = NewWriter(&w.subBuffer)
	}

	w.subBuffer.Reset()

	err := cb(w.subWriter)
	if err != nil {
		w.err = err
		return
	}

	if w.subWriter.err != nil {
		w.err = w.subWriter.err
		return
	}

	w.Bytes(num, w.subBuffer.Bytes())
}

func (w *Writer) writeVarint(v uint64) error {
	n := 0
	for v >= 1<<7 {
		w.buf[n] = byte(v&0x7f | 0x80)
		v >>= 7
		n++
	}
	w.buf[n] = byte(v)

	if _, err := w.wrap.Write(w.buf[:n+1]); err != nil {
		w.err = err
		return err
	}

	return nil
}

func (w *Writer) writeTag(num int, wt int) error {
	if w.err != nil {
		return w.err
	}

	tag := (uint64(num) << 3) | uint64(wt)
	if err := w.writeVarint(tag); err != nil {
		w.err = err
		return err
	}

	return nil
}

// Bytes writes a length-delimited byte slice field
func (w *Writer) Bytes(num int, v []byte) {
	if err := w.writeTag(num, wireLen); err != nil {
		return
	}

	if err := w.writeVarint(uint64(len(v))); err != nil {
		w.err = err
		return
	}
	if len(v) > 0 {
		if _, err := w.wrap.Write(v); err != nil {
			w.err = err
		}
	}
}

// Fixed64 writes a 64-bit fixed-size field
func (w *Writer) Fixed64(num int, v uint64) {
	if err := w.writeTag(num, wireI64); err != nil {
		return
	}

	for i := 0; i < 8; i++ {
		w.buf[i] = byte(v >> (i * 8))
	}
	if _, err := w.wrap.Write(w.buf[:8]); err != nil {
		w.err = err
	}
}

// Fixed32 writes a 32-bit fixed-size field
func (w *Writer) Fixed32(num int, v uint32) {
	if err := w.writeTag(num, wireI32); err != nil {
		return
	}

	for i := 0; i < 4; i++ {
		w.buf[i] = byte(v >> (i * 8))
	}
	if _, err := w.wrap.Write(w.buf[:4]); err != nil {
		w.err = err
	}
}

// Uint64 writes an unsigned 64-bit integer field
func (w *Writer) Uint64(num int, v uint64) {
	if err := w.writeTag(num, wireVarint); err != nil {
		return
	}

	if err := w.writeVarint(v); err != nil {
		w.err = err
	}
}

// String writes a string field
func (w *Writer) String(num int, v string) {
	if err := w.writeTag(num, wireLen); err != nil {
		return
	}

	if err := w.writeVarint(uint64(len(v))); err != nil {
		w.err = err
		return
	}
	if len(v) > 0 {
		// unsafe string to bytes
		dataPtr := unsafe.StringData(v)
		// Create a byte slice from the pointer and string length
		byteSlice := unsafe.Slice(dataPtr, len(v))

		if _, err := w.wrap.Write(byteSlice); err != nil {
			w.err = err
		}
	}
}

// Int64 writes a signed 64-bit integer field
func (w *Writer) Int64(num int, v int64) {
	w.Uint64(num, uint64(v))
}

// Double writes a double-precision floating-point field
func (w *Writer) Double(num int, v float64) {
	w.Fixed64(num, math.Float64bits(v))
}

// Float writes a single-precision floating-point field
func (w *Writer) Float(num int, v float32) {
	w.Fixed32(num, math.Float32bits(v))
}

// Bool writes a boolean field
func (w *Writer) Bool(num int, v bool) {
	if v {
		w.Uint64(num, 1)
	} else {
		w.Uint64(num, 0)
	}
}

// Enum writes a protocol buffer enum field
func (w *Writer) Enum(num int, v int32) {
	w.Int32(num, v)
}

// Uint32 writes an unsigned 32-bit integer field
func (w *Writer) Uint32(num int, v uint32) {
	w.Uint64(num, uint64(v))
}

// Int32 writes a signed 32-bit integer field
func (w *Writer) Int32(num int, v int32) {
	w.Uint64(num, uint64(v))
}

// Sint32 writes a signed 32-bit integer field using zigzag encoding
func (w *Writer) Sint32(num int, v int32) {
	w.Uint32(num, uint32((v<<1)^(v>>31)))
}

// Sint64 writes a signed 64-bit integer field using zigzag encoding
func (w *Writer) Sint64(num int, v int64) {
	w.Uint64(num, uint64((v<<1)^(v>>63)))
}

// Sfixed64 writes a signed 64-bit fixed-size field
func (w *Writer) Sfixed64(num int, v int64) {
	w.Fixed64(num, uint64(v))
}

// Sfixed32 writes a signed 32-bit fixed-size field
func (w *Writer) Sfixed32(num int, v int32) {
	w.Fixed32(num, uint32(v))
}

// Err ...
func (w *Writer) Err() error {
	return w.err
}
