package rawpb

// maxVarintBytes is the maximum length of a valid protobuf varint for a 64-bit
// value. Anything longer is treated as a malformed message.
const maxVarintBytes = 10

type readerBody struct {
	body   []byte
	offset int
}

func newReaderBody(body []byte) *readerBody {
	return &readerBody{
		body: body,
	}
}

func (r *readerBody) varint() (uint64, error) {
	var ret uint64
	i := uint64(0)
	for r.next() {
		if i >= maxVarintBytes {
			return 0, ErrorInvalidMessage
		}
		ret += uint64(r.body[r.offset]&0x7f) << (7 * i)
		if r.body[r.offset]&0x80 == 0 { // last byte of varint
			r.offset++
			return ret, nil
		}
		r.offset++
		i++
	}
	return ret, ErrorTruncated
}

func (r *readerBody) next() bool {
	return r.offset < len(r.body)
}

func (r *readerBody) bytes(n uint64) ([]byte, error) {
	avail := uint64(len(r.body) - r.offset)
	if n > avail {
		return nil, ErrorTruncated
	}
	end := r.offset + int(n)
	v := r.body[r.offset:end]
	r.offset = end
	return v, nil
}

func (r *readerBody) lengthDelimited() ([]byte, error) {
	l, err := r.varint()
	if err != nil {
		return nil, err
	}
	return r.bytes(l)
}

func (r *readerBody) fixed64() (uint64, error) {
	p, err := r.bytes(8)
	if err != nil {
		return 0, err
	}
	u := uint64(p[0]) | (uint64(p[1]) << 8) | (uint64(p[2]) << 16) | (uint64(p[3]) << 24) |
		(uint64(p[4]) << 32) | (uint64(p[5]) << 40) | (uint64(p[6]) << 48) | (uint64(p[7]) << 56)
	return u, nil
}

func (r *readerBody) fixed32() (uint32, error) {
	p, err := r.bytes(4)
	if err != nil {
		return 0, err
	}
	u := uint32(p[0]) | (uint32(p[1]) << 8) | (uint32(p[2]) << 16) | (uint32(p[3]) << 24)
	return u, nil
}
