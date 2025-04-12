package rawpb

// Option configures RawPB parser behavior
type Option func(*RawPB)

// Begin sets a callback function to execute before parsing starts
func Begin(f func() error) Option {
	return func(p *RawPB) {
		p.beginFunc = f
	}
}

// End sets a callback function to execute after parsing completes
func End(f func() error) Option {
	return func(p *RawPB) {
		p.endFunc = f
	}
}

// UnknownVarint handles unregistered varint fields
func UnknownVarint(f func(num int, v uint64) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.varint = f
	}
}

// UnknownFixed32 handles unregistered fixed32 fields
func UnknownFixed32(f func(num int, v uint32) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.fixed32 = f
	}
}

// UnknownFixed64 handles unregistered fixed64 fields
func UnknownFixed64(f func(num int, v uint64) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.fixed64 = f
	}
}

// UnknownBytes handles unregistered length-delimited fields
func UnknownBytes(f func(num int, v []byte) error) Option {
	return func(p *RawPB) {
		p.schema.unknown.bytes = f
	}
}

// Name sets the parser name for error reporting
func Name(name string) Option {
	return func(p *RawPB) {
		p.name = name
	}
}
