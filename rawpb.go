package rawpb

import "errors"

var ErrorTruncated = errors.New("message truncated")
var ErrorInvalidMessage = errors.New("invalid message")
var ErrorWrongWireType = errors.New("wrong wire type")

type RawPB struct {
	beginFunc func() error
	endFunc   func() error
	schema    callbacks
}

func New(opts ...Option) *RawPB {
	r := &RawPB{}

	for _, o := range opts {
		o(r)
	}

	return r
}

func (pb *RawPB) Parse(body []byte) error {
	if pb.beginFunc != nil {
		if err := pb.beginFunc(); err != nil {
			return err
		}
	}

	r := newReader(body)

	for r.next() {
		// read wire type
		tag, err := r.varint()
		if err != nil {
			return err
		}
		wt := tag % 8
		num := tag >> 3
		switch wt {
		case 0: // varint
			v, err := r.varint()
			if err != nil {
				return err
			}
			if err = pb.schema.varint(int(num), v); err != nil {
				return err
			}
		case 1: // 64-bit
			v, err := r.fixed64()
			if err != nil {
				return err
			}
			if err = pb.schema.fixed64(int(num), v); err != nil {
				return err
			}
		case 2: // Length-delimited
			v, err := r.lengthDelimited()
			if err != nil {
				return err
			}
			if err = pb.schema.bytes(int(num), v); err != nil {
				return err
			}
		case 5: // 32-bit
			v, err := r.fixed32()
			if err != nil {
				return err
			}
			if err = pb.schema.fixed32(int(num), v); err != nil {
				return err
			}
		default:
			return ErrorWrongWireType
		}
	}

	if pb.endFunc != nil {
		if err := pb.endFunc(); err != nil {
			return err
		}
	}
	return nil
}
