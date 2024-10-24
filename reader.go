package rawpb

type reader struct {
	body   []byte
	offset int
}

func newReader(body []byte) *reader {
	return &reader{body: body}
}

func (r *reader) varint() (uint64, error) {
	var ret uint64
	i := 0
	for r.next() {
		ret += uint64(r.body[r.offset]&0x7f) << (7 * uint64(i))
		if r.body[r.offset]&0x80 == 0 { // last byte of varint
			r.offset++
			return ret, nil
		}
		r.offset++
		i++
	}
	return ret, ErrorTruncated
}

func (r *reader) next() bool {
	return r.offset < len(r.body)
}

func (r *reader) bytes(n int) ([]byte, error) {
	if r.offset+n > len(r.body) {
		return nil, ErrorTruncated
	}
	v := r.body[r.offset : r.offset+n]
	r.offset += n
	return v, nil
}

func (r *reader) lengthDelimited() ([]byte, error) {
	l, err := r.varint()
	if err != nil {
		return nil, err
	}
	return r.bytes(int(l))
}

func (r *reader) fixed64() (uint64, error) {
	p, err := r.bytes(8)
	if err != nil {
		return 0, err
	}
	u := uint64(p[0]) | (uint64(p[1]) << 8) | (uint64(p[2]) << 16) | (uint64(p[3]) << 24) |
		(uint64(p[4]) << 32) | (uint64(p[5]) << 40) | (uint64(p[6]) << 48) | (uint64(p[7]) << 56)
	return u, nil
}

func (r *reader) fixed32() (uint32, error) {
	p, err := r.bytes(4)
	if err != nil {
		return 0, err
	}
	u := uint32(p[0]) | (uint32(p[1]) << 8) | (uint32(p[2]) << 16) | (uint32(p[3]) << 24)
	return u, nil
}
