package rawpb

import (
	"math"
	"unsafe"
)

/*
@todo
1	I64	sfixed64,
2	LEN	packed repeated fields
5	I32	sfixed32,
*/

func Bytes(num int, f func([]byte) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			bytes: f,
		})
	}
}

func Varint(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			varint: func(v uint64) error {
				return f(v)
			},
		})
	}
}

func Fixed64(num int, f func(uint64) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			fixed64: func(v uint64) error {
				return f(v)
			},
		})
	}
}

func Fixed32(num int, f func(uint32) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			fixed32: func(v uint32) error {
				return f(v)
			},
		})
	}
}

func Uint64(num int, f func(uint64) error) Option {
	return Varint(num, f)
}

func Message(num int, n *RawPB) Option {
	return Bytes(num, n.Parse)
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func String(num int, f func(string) error) Option {
	return Bytes(num, func(b []byte) error {
		return f(unsafeString(b))
	})
}

func Int64(num int, f func(int64) error) Option {
	return Varint(num, func(u uint64) error {
		return f(int64(u))
	})
}

func Double(num int, f func(float64) error) Option {
	return Fixed64(num, func(u uint64) error {
		return f(math.Float64frombits(u))
	})
}

func Float(num int, f func(float32) error) Option {
	return Fixed32(num, func(u uint32) error {
		return f(math.Float32frombits(u))
	})
}

func Bool(num int, f func(bool) error) Option {
	return Int32(num, func(u int32) error {
		return f(u != 0)
	})
}

func Enum(num int, f func(int32) error) Option {
	return Int32(num, f)
}

func Uint32(num int, f func(uint32) error) Option {
	return Varint(num, func(u uint64) error {
		return f(uint32(u))
	})
}

func Int32(num int, f func(int32) error) Option {
	return Uint32(num, func(u uint32) error {
		return f(int32(u))
	})
}

func Sint32(num int, f func(int32) error) Option {
	return Uint32(num, func(u uint32) error {
		if u%2 == 0 {
			return f(int32(u / 2))
		}
		return f(-int32(u/2) - 1)
	})
}

func Sint64(num int, f func(int64) error) Option {
	return Uint64(num, func(u uint64) error {
		if u%2 == 0 {
			return f(int64(u / 2))
		}
		return f(-int64(u/2) - 1)
	})
}

func Sfixed64(num int, f func(int64) error) Option {
	return Fixed64(num, func(u uint64) error {
		return f(int64(u))
	})
}

func Sfixed32(num int, f func(int32) error) Option {
	return Fixed32(num, func(u uint32) error {
		return f(int32(u))
	})
}
