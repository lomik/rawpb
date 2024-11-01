package rawpb

import (
	"math"
	"unsafe"
)

func Bytes(num int, f func([]byte) error) Option {
	return func(p *RawPB) {
		p.schema.setBytes(num, f)
	}
}

func Varint(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.schema.setVarint(num, f)
	}
}

func Fixed64(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.schema.setFixed64(num, f)
	}
}

func Fixed32(num int, f func(uint32) error) Option {
	return func(p *RawPB) {
		p.schema.setFixed32(num, f)
	}
}

func Message(num int, n *RawPB) Option {
	return func(p *RawPB) {
		p.schema.setMessage(num, n)
	}
}

func Uint64(num int, f func(uint64) error) Option {
	return Varint(num, f)
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func String(num int, f func(string) error) Option {
	return Bytes(num, func(b []byte) error {
		if f == nil {
			return nil
		}
		return f(unsafeString(b))
	})
}

func Int64(num int, f func(int64) error) Option {
	return Varint(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(int64(u))
	})
}

func Double(num int, f func(float64) error) Option {
	return Fixed64(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(math.Float64frombits(u))
	})
}

func Float(num int, f func(float32) error) Option {
	return Fixed32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(math.Float32frombits(u))
	})
}

func Bool(num int, f func(bool) error) Option {
	return Int32(num, func(u int32) error {
		if f == nil {
			return nil
		}
		return f(u != 0)
	})
}

func Enum(num int, f func(int32) error) Option {
	return Int32(num, f)
}

func Uint32(num int, f func(uint32) error) Option {
	return Varint(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(uint32(u))
	})
}

func Int32(num int, f func(int32) error) Option {
	return Uint32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(int32(u))
	})
}

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

func Sfixed64(num int, f func(int64) error) Option {
	return Fixed64(num, func(u uint64) error {
		if f == nil {
			return nil
		}
		return f(int64(u))
	})
}

func Sfixed32(num int, f func(int32) error) Option {
	return Fixed32(num, func(u uint32) error {
		if f == nil {
			return nil
		}
		return f(int32(u))
	})
}
