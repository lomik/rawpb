package rawpb

import "math"

// Wire type constants as defined by the protobuf wire format.
const (
	WireVarint  = 0
	WireFixed64 = 1
	WireLen     = 2
	WireFixed32 = 5
)

// Decoder is a pull-based, zero-allocation reader over a protobuf-encoded
// byte slice. It is an alternative to the callback-based RawPB API: instead
// of registering per-field callbacks up front, the caller drives a Next
// loop and picks off values with typed accessors.
//
// Typical usage:
//
//	var d rawpb.Decoder
//	d.Reset(input)
//	for d.Next() {
//	    switch d.Num() {
//	    case 1:
//	        name = d.UnsafeString()
//	    case 2:
//	        ts = d.Int64()
//	    case 3:
//	        xs = append(xs, d.Int32())   // works for packed and unpacked
//	    case 4:
//	        sub := d.Submessage()
//	        for sub.Next() { ... }
//	        if err := sub.Err(); err != nil { return err }
//	    }
//	}
//	if err := d.Err(); err != nil { return err }
//
// Repeated scalar fields are transparent: calling a scalar accessor
// (Int32, Uint64, Double, Fixed32, ...) on a length-delimited field
// interprets the payload as a packed sequence of that type and returns the
// first value. Subsequent Next calls yield the remaining packed values one
// at a time; when the payload is exhausted Next resumes reading the outer
// stream. If the caller does not touch the length-delimited field with a
// scalar accessor, Next moves straight to the next tag on its return —
// packing is never "expanded" without an explicit ask.
//
// Edge case: an empty packed field (LEN with zero payload) still causes
// one Next iteration; a scalar accessor on it returns the type's zero
// value and does not enter packed continuation. In practice encoders omit
// empty packed fields entirely, so this rarely surfaces.
//
// The zero value of Decoder is a valid empty decoder; use Reset to bind
// input. The type may be freely embedded in structs or allocated on the
// stack — the whole hot path is allocation-free.
//
// Errors are sticky: once an accessor or Next encounters a problem, Err
// returns it and Next returns false thereafter. Wire-type mismatches
// (e.g. Uint64 on a length-delimited value once packed unpacking failed
// to apply) count as errors — there are no unchecked accessors.
//
// Concurrent use from multiple goroutines is not supported.
type Decoder struct {
	body   []byte
	offset int    // where Next() reads the next tag from
	scalar uint64 // last-decoded value for varint / fixed32 / fixed64
	slice  []byte // last-decoded payload for length-delimited
	num    int
	wt     int
	err    error

	// Packed continuation state. When packedRem holds bytes, Next yields
	// the next packed value from it before advancing the outer stream.
	// Set by scalar accessors when called on a LEN field with a non-empty
	// payload; cleared once the payload is exhausted or Reset is called.
	packedRem  []byte
	packedNum  int
	packedWire int
}

// NewDecoder returns a Decoder over body. Equivalent to (&Decoder{}).Reset(body).
func NewDecoder(body []byte) *Decoder {
	return &Decoder{body: body}
}

// Reset rebinds the decoder to a new input and clears prior state.
func (d *Decoder) Reset(body []byte) {
	d.body = body
	d.offset = 0
	d.scalar = 0
	d.slice = nil
	d.num = 0
	d.wt = 0
	d.err = nil
	d.packedRem = nil
	d.packedNum = 0
	d.packedWire = 0
}

// Err returns the first error encountered by Next or by an accessor, or nil
// if none. Safe to call at any time.
func (d *Decoder) Err() error {
	return d.err
}

// Num returns the field number of the current field. Valid only immediately
// after Next returns true.
func (d *Decoder) Num() int {
	return d.num
}

// WireType returns the wire type of the current field (one of WireVarint,
// WireFixed64, WireLen, WireFixed32). Valid only immediately after Next
// returns true. For a packed continuation, WireType surfaces as the
// value-level wire type (WireVarint / WireFixed32 / WireFixed64), not
// WireLen.
func (d *Decoder) WireType() int {
	return d.wt
}

// Next advances to the next field. Returns false at end of input or after
// an error (see Err). Fields whose value the caller did not consume are
// implicitly skipped.
//
// If a scalar accessor previously entered packed continuation, Next yields
// the next packed value (same field number, value-level wire type). Once
// the packed payload is exhausted, Next resumes reading the outer stream.
func (d *Decoder) Next() bool {
	if d.err != nil {
		return false
	}
	if len(d.packedRem) > 0 {
		return d.nextPackedValue()
	}
	// Not in packed mode (or just consumed the last packed value).
	d.packedRem = nil

	if d.offset >= len(d.body) {
		return false
	}

	tag, n, err := decodeVarint(d.body[d.offset:])
	if err != nil {
		d.err = err
		return false
	}
	d.offset += n

	wt := int(tag & 7)
	num := int(tag >> 3)
	if num < 1 || num > maxFieldNumber {
		d.err = ErrorInvalidMessage
		return false
	}
	d.num = num
	d.wt = wt
	d.slice = nil

	switch wt {
	case WireVarint:
		v, vn, err := decodeVarint(d.body[d.offset:])
		if err != nil {
			d.err = err
			return false
		}
		d.scalar = v
		d.offset += vn
	case WireFixed64:
		if d.offset+8 > len(d.body) {
			d.err = ErrorTruncated
			return false
		}
		b := d.body[d.offset:]
		d.scalar = uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
		d.offset += 8
	case WireFixed32:
		if d.offset+4 > len(d.body) {
			d.err = ErrorTruncated
			return false
		}
		b := d.body[d.offset:]
		d.scalar = uint64(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
		d.offset += 4
	case WireLen:
		l, ln, err := decodeVarint(d.body[d.offset:])
		if err != nil {
			d.err = err
			return false
		}
		d.offset += ln
		if l > uint64(math.MaxInt) {
			d.err = ErrorInvalidMessage
			return false
		}
		if uint64(len(d.body)-d.offset) < l {
			d.err = ErrorTruncated
			return false
		}
		end := d.offset + int(l)
		d.slice = d.body[d.offset:end]
		d.offset = end
	default:
		d.err = ErrorWrongWireType
		return false
	}
	return true
}

// nextPackedValue pulls the next scalar out of the packed payload set up
// by a previous scalar-accessor call on a LEN field. Populates num / wt /
// scalar exactly as if a matching primitive field had just been read.
func (d *Decoder) nextPackedValue() bool {
	d.num = d.packedNum
	d.wt = d.packedWire
	d.slice = nil
	switch d.packedWire {
	case WireVarint:
		v, n, err := decodeVarint(d.packedRem)
		if err != nil {
			d.err = err
			return false
		}
		d.scalar = v
		d.packedRem = d.packedRem[n:]
	case WireFixed64:
		if len(d.packedRem) < 8 {
			d.err = ErrorTruncated
			return false
		}
		b := d.packedRem
		d.scalar = uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
		d.packedRem = d.packedRem[8:]
	case WireFixed32:
		if len(d.packedRem) < 4 {
			d.err = ErrorTruncated
			return false
		}
		b := d.packedRem
		d.scalar = uint64(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
		d.packedRem = d.packedRem[4:]
	}
	return true
}

// decodeVarint reads a varint from the start of b, returning the value and
// the number of bytes consumed. Rejects varints longer than 10 bytes.
func decodeVarint(b []byte) (uint64, int, error) {
	var v uint64
	limit := len(b)
	if limit > maxVarintBytes {
		limit = maxVarintBytes
	}
	for i := 0; i < limit; i++ {
		v |= uint64(b[i]&0x7f) << (7 * uint(i))
		if b[i]&0x80 == 0 {
			return v, i + 1, nil
		}
	}
	if len(b) < maxVarintBytes {
		return 0, 0, ErrorTruncated
	}
	return 0, 0, ErrorInvalidMessage
}

// --- Varint accessors ---

// Uint64 returns the current varint value.
//
// On a LEN field with a non-empty payload, Uint64 treats the payload as a
// packed sequence of varints: it decodes and returns the first value,
// stashing the remainder for subsequent Next calls to yield. On an empty
// LEN payload it returns 0 without entering packed continuation. On any
// other wire type it sets ErrorWrongWireType.
func (d *Decoder) Uint64() uint64 {
	if d.err != nil {
		return 0
	}
	switch d.wt {
	case WireVarint:
		return d.scalar
	case WireLen:
		if len(d.slice) == 0 {
			return 0
		}
		v, n, err := decodeVarint(d.slice)
		if err != nil {
			d.err = err
			return 0
		}
		d.packedRem = d.slice[n:]
		d.packedNum = d.num
		d.packedWire = WireVarint
		d.slice = nil
		d.scalar = v
		d.wt = WireVarint
		return v
	}
	d.err = ErrorWrongWireType
	return 0
}

// Int64 returns the current varint value reinterpreted as int64.
func (d *Decoder) Int64() int64 { return int64(d.Uint64()) }

// Uint32 returns the current varint value truncated to uint32.
func (d *Decoder) Uint32() uint32 { return uint32(d.Uint64()) }

// Int32 returns the current varint value reinterpreted as int32.
func (d *Decoder) Int32() int32 { return int32(d.Uint64()) }

// Bool returns the current varint value != 0.
func (d *Decoder) Bool() bool { return d.Uint64() != 0 }

// Sint32 decodes the current varint using zigzag encoding.
func (d *Decoder) Sint32() int32 {
	u := d.Uint32()
	return int32(u>>1) ^ -int32(u&1)
}

// Sint64 decodes the current varint using zigzag encoding.
func (d *Decoder) Sint64() int64 {
	u := d.Uint64()
	return int64(u>>1) ^ -int64(u&1)
}

// --- Fixed64 accessors ---

// Fixed64 returns the current 64-bit fixed value.
//
// On a LEN field, Fixed64 treats the payload as a packed sequence of
// fixed64 values (payload length must be a multiple of 8): it returns the
// first value and stashes the rest for subsequent Next calls. On an empty
// LEN payload it returns 0. On any other wire type it sets
// ErrorWrongWireType.
func (d *Decoder) Fixed64() uint64 {
	if d.err != nil {
		return 0
	}
	switch d.wt {
	case WireFixed64:
		return d.scalar
	case WireLen:
		if len(d.slice) == 0 {
			return 0
		}
		if len(d.slice)%8 != 0 {
			d.err = ErrorInvalidMessage
			return 0
		}
		b := d.slice
		v := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
		d.packedRem = d.slice[8:]
		d.packedNum = d.num
		d.packedWire = WireFixed64
		d.slice = nil
		d.scalar = v
		d.wt = WireFixed64
		return v
	}
	d.err = ErrorWrongWireType
	return 0
}

// Sfixed64 returns the current 64-bit fixed value as int64.
func (d *Decoder) Sfixed64() int64 { return int64(d.Fixed64()) }

// Double returns the current 64-bit fixed value as float64.
func (d *Decoder) Double() float64 { return math.Float64frombits(d.Fixed64()) }

// --- Fixed32 accessors ---

// Fixed32 returns the current 32-bit fixed value.
//
// On a LEN field, Fixed32 treats the payload as a packed sequence of
// fixed32 values (payload length must be a multiple of 4): it returns the
// first value and stashes the rest for subsequent Next calls. On an empty
// LEN payload it returns 0. On any other wire type it sets
// ErrorWrongWireType.
func (d *Decoder) Fixed32() uint32 {
	if d.err != nil {
		return 0
	}
	switch d.wt {
	case WireFixed32:
		return uint32(d.scalar)
	case WireLen:
		if len(d.slice) == 0 {
			return 0
		}
		if len(d.slice)%4 != 0 {
			d.err = ErrorInvalidMessage
			return 0
		}
		b := d.slice
		v := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
		d.packedRem = d.slice[4:]
		d.packedNum = d.num
		d.packedWire = WireFixed32
		d.slice = nil
		d.scalar = uint64(v)
		d.wt = WireFixed32
		return v
	}
	d.err = ErrorWrongWireType
	return 0
}

// Sfixed32 returns the current 32-bit fixed value as int32.
func (d *Decoder) Sfixed32() int32 { return int32(d.Fixed32()) }

// Float returns the current 32-bit fixed value as float32.
func (d *Decoder) Float() float32 { return math.Float32frombits(d.Fixed32()) }

// --- Length-delimited accessors ---

// Bytes returns the raw payload of the current length-delimited field. The
// returned slice aliases the decoder's input buffer and is only safe to
// use while that buffer remains untouched. Copy it (bytes.Clone) if it
// must outlive the next mutation of the input.
func (d *Decoder) Bytes() []byte {
	if d.err != nil {
		return nil
	}
	if d.wt != WireLen {
		d.err = ErrorWrongWireType
		return nil
	}
	return d.slice
}

// BytesCopy returns a heap-allocated copy of the current length-delimited
// field. Safe to retain arbitrarily.
func (d *Decoder) BytesCopy() []byte {
	b := d.Bytes()
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// UnsafeString returns the current length-delimited payload as a string
// aliasing the decoder's input. Same lifetime constraints as Bytes.
func (d *Decoder) UnsafeString() string {
	b := d.Bytes()
	if b == nil {
		return ""
	}
	return unsafeString(b)
}

// CopyString returns the current length-delimited payload as a heap-copied
// string that safely outlives the decoder.
func (d *Decoder) CopyString() string {
	b := d.Bytes()
	if b == nil {
		return ""
	}
	return string(b)
}

// --- Submessage ---

// Submessage returns a Decoder scoped to the current length-delimited
// field's payload — the standard way to walk into a nested message. The
// returned Decoder is a value; store it in a local so its methods (which
// take *Decoder) can take its address without escaping to the heap.
//
//	sub := d.Submessage()
//	for sub.Next() {
//	    switch sub.Num() { ... }
//	}
//	if err := sub.Err(); err != nil { return err }
//
// If the current field is not length-delimited, ErrorWrongWireType is set
// on the parent decoder and an empty (immediately-terminating) sub-decoder
// is returned.
func (d *Decoder) Submessage() Decoder {
	b := d.Bytes()
	if b == nil {
		return Decoder{}
	}
	return Decoder{body: b}
}
