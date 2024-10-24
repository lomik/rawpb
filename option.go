package rawpb

type Option func(*RawPB)

func Begin(f func() error) Option {
	return func(p *RawPB) {
		p.beginFunc = f
	}
}

func End(f func() error) Option {
	return func(p *RawPB) {
		p.endFunc = f
	}
}

func UnknownVarint(f func(num int, v uint64) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.varint = f
	}
}

func UnknownFixed32(f func(num int, v uint32) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.fixed32 = f
	}
}

func UnknownFixed64(f func(num int, v uint64) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.fixed64 = f
	}
}

func UnknownBytes(f func(num int, v []byte) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.bytes = f
	}
}
