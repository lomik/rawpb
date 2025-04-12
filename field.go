package rawpb

import (
	"math"
	"unsafe"
)

// Bytes registers a callback for length-delimited (bytes/string) fields
func Bytes(num int, f func([]byte) error) Option {
	return func(p *RawPB) {
		p.schema.setBytes(num, f)
	}
}

// Varint registers a callback for varint-encoded fields
func Varint(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.schema.setVarint(num, f)
	}
}

// Fixed64 registers a callback for 64-bit fixed-size fields
func Fixed64(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.schema.setFixed64(num, f)
	}
}

// Fixed32 registers a callback for 32-bit fixed-size fields
func Fixed32(num int, f func(uint32) error) Option {
	return func(p *RawPB) {
		p.schema.setFixed32(num, f)
	}
}

// Message registers a nested message parser for length-delimited fields
func Message(num int, n *RawPB) Option {
	return func(p *RawPB) {
		p.schema.setMessage(num, n)
	}
}

// Uint64 registers a callback for unsigned 64-bit integers using varint encoding
func Uint64(num int, f func(uint64) error) Option {
	return Varint(num, f)
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// String registers a callback for string values from length-delimited fields.
// Uses unsafe conversion from byte slice to avoid allocation.
func String(num int, f func(string) error) Option {
	return Bytes(num, func(b []byte) error {
		if f == nil {
			return nil
		}
		return f(unsafeString(b))
	})
}

// Int64 registers a callback for signed 64-bit integers using varint encoding
func Int64(num int, f func(int64) error) Option {
	return Varint(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(int64(u))
	})
}

// Double registers a callback for double-precision floating point numbers using fixed64 encoding
func Double(num int, f func(float64) error) Option {
	return Fixed64(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(math.Float64frombits(u))
	})
}

// Float registers a callback for single-precision floating point numbers using fixed32 encoding
func Float(num int, f func(float32) error) Option {
	return Fixed32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(math.Float32frombits(u))
	})
}

// Bool registers a callback for boolean values decoded from varint-encoded integers
func Bool(num int, f func(bool) error) Option {
	return Int32(num, func(u int32) error {
		if f == nil {
			return nil
		}
		return f(u != 0)
	})
}

// Enum registers a callback for protocol buffer enum values encoded as varints
func Enum(num int, f func(int32) error) Option {
	return Int32(num, f)
}

// Uint32 registers a callback for unsigned 32-bit integers using varint encoding
func Uint32(num int, f func(uint32) error) Option {
	return Varint(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(uint32(u))
	})
}

// Int32 registers a callback for signed 32-bit integers using varint encoding
func Int32(num int, f func(int32) error) Option {
	return Uint32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(int32(u))
	})
}

// Sint32 registers a callback for zigzag-encoded signed 32-bit integers
func Sint32(num int, f func(int32) error) Option {
	return Uint32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		if u%2 == 0 {
			return f(int32(u / 2))
		}
		return f(-int32(u/2) - 1)
	})
}

// Sint64 registers a callback for zigzag-encoded signed 64-bit integers
func Sint64(num int, f func(int64) error) Option {
	return Uint64(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		if u%2 == 0 {
			return f(int64(u / 2))
		}
		return f(-int64(u/2) - 1)
	})
}

// Sfixed64 registers a callback for signed 64-bit integers using fixed64 encoding
func Sfixed64(num int, f func(int64) error) Option {
	return Fixed64(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(int64(u))
	})
}

// Sfixed32 registers a callback for signed 32-bit integers using fixed32 encoding
func Sfixed32(num int, f func(int32) error) Option {
	return Fixed32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(int32(u))
	})
}
