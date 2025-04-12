package rawpb

import (
	"errors"
	"fmt"
	"math"
)

var ErrorTruncated = errors.New("message truncated")
var ErrorInvalidMessage = errors.New("invalid message")
var ErrorWrongWireType = errors.New("wrong wire type")

// RawPB implements a low-level protocol buffer parser without code generation
type RawPB struct {
	beginFunc func() error
	endFunc   func() error
	schema    callbacks
	name      string
}

// New creates a new RawPB parser with optional configuration
func New(opts ...Option) *RawPB {
	r := &RawPB{}

	for _, o := range opts {
		o(r)
	}

	return r
}

// Read parses protocol buffer data from a stream with memory management via Allocator
func (pb *RawPB) Read(stream Reader, allocator Allocator) error {
	if allocator == nil {
		allocator = &HeapAllocator{}
	}

	return pb.doRead(newReaderLimit(stream, allocator, math.MaxUint64))
}

func (pb *RawPB) doRead(r *readerLimit) error {

	if pb.beginFunc != nil {
		if err := pb.beginFunc(); err != nil {
			return err
		}
	}

	for {
		// read wire type
		tag, abort, err := r.varintOrBreak()
		if err != nil {
			return err
		}
		if abort {
			break
		}
		wt := tag % 8
		num := int(tag >> 3)

		switch wt {
		case 0: // varint
			v, err := r.varint()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.varint(int(num), v); err != nil {
				return pb.wrapError(num, err)
			}
		case 1: // 64-bit
			v, err := r.fixed64()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.fixed64(int(num), v); err != nil {
				return pb.wrapError(num, err)
			}
		case 5: // 32-bit
			v, err := r.fixed32()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.fixed32(int(num), v); err != nil {
				return pb.wrapError(num, err)
			}
		case 2: // Length-delimited
			l, err := r.varint()
			if err != nil {
				return pb.wrapError(num, err)
			}

			c := pb.schema.get(num)

			// has callback
			// packed varint, fixed32, fixed64
			// bytes, string
			// submessage
			switch c.tp {
			case callbackTypeNone:
				if pb.schema.unknown.bytes != nil {
					v, err := r.bytes(l)
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = pb.schema.unknown.bytes(num, v); err != nil {
						return pb.wrapError(num, err)
					}
				} else {
					if err = r.skip(l); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeBytes:
				if c.funcBytes != nil {
					v, err := r.bytes(l)
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = c.funcBytes(v); err != nil {
						return pb.wrapError(num, err)
					}
				} else {
					if err = r.skip(l); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeMessage:
				if c.message != nil {
					currentLimit := r.limit
					if currentLimit < l {
						return pb.wrapError(num, ErrorTruncated)
					}
					r.limit = l
					if err = c.message.doRead(r); err != nil {
						return pb.wrapError(num, err)
					}
					// restore parent limit
					r.limit = currentLimit - l
				} else {
					if err = r.skip(l); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeVarint:
				currentLimit := r.limit
				if currentLimit < l {
					return pb.wrapError(num, ErrorTruncated)
				}
				r.limit = l
				for {
					vv, breakLoop, err := r.varintOrBreak()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if breakLoop {
						break
					}
					if err = call(c.funcUint64, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
				// restore parent limit
				r.limit = currentLimit - l
			case callbackTypeFixed64:
				currentLimit := r.limit
				if currentLimit < l {
					return pb.wrapError(num, ErrorTruncated)
				}
				r.limit = l
				for r.next() {
					vv, err := r.fixed64()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = call(c.funcUint64, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
				// restore parent limit
				r.limit = currentLimit - l
			case callbackTypeFixed32:
				currentLimit := r.limit
				if currentLimit < l {
					return pb.wrapError(num, ErrorTruncated)
				}
				r.limit = l
				for r.next() {
					vv, err := r.fixed32()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = call(c.funcUint32, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
				// restore parent limit
				r.limit = currentLimit - l
			default:
				panic("unknown callback type")
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

// Parse decodes protocol buffer data directly from a byte slice
func (pb *RawPB) Parse(body []byte) error {

	r := newReaderBody(body)

	if pb.beginFunc != nil {
		if err := pb.beginFunc(); err != nil {
			return err
		}
	}

	for r.next() {
		// read wire type
		tag, err := r.varint()
		if err != nil {
			return err
		}
		wt := tag % 8
		num := int(tag >> 3)

		switch wt {
		case 0: // varint
			v, err := r.varint()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.varint(num, v); err != nil {
				return pb.wrapError(num, err)
			}
		case 1: // 64-bit
			v, err := r.fixed64()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.fixed64(num, v); err != nil {
				return pb.wrapError(num, err)
			}
		case 5: // 32-bit
			v, err := r.fixed32()
			if err != nil {
				return pb.wrapError(num, err)
			}
			if err = pb.schema.fixed32(num, v); err != nil {
				return pb.wrapError(num, err)
			}
		case 2: // Length-delimited
			v, err := r.lengthDelimited()
			if err != nil {
				return pb.wrapError(num, err)
			}

			c := pb.schema.get(num)

			// has callback
			// packed varint, fixed32, fixed64
			// bytes, string
			// submessage
			switch c.tp {
			case callbackTypeNone:
				if err = callUnknown(pb.schema.unknown.bytes, num, v); err != nil {
					return pb.wrapError(num, err)
				}
			case callbackTypeBytes:
				if err = call(c.funcBytes, v); err != nil {
					return pb.wrapError(num, err)
				}
			case callbackTypeMessage:

				if c.message != nil {
					if err = c.message.Parse(v); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeVarint:
				sub := newReaderBody(v)
				for sub.next() {
					vv, err := sub.varint()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = call(c.funcUint64, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeFixed64:
				sub := newReaderBody(v)
				for sub.next() {
					vv, err := sub.fixed64()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = call(c.funcUint64, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
			case callbackTypeFixed32:
				sub := newReaderBody(v)
				for sub.next() {
					vv, err := sub.fixed32()
					if err != nil {
						return pb.wrapError(num, err)
					}
					if err = call(c.funcUint32, vv); err != nil {
						return pb.wrapError(num, err)
					}
				}
			default:
				panic("unknown callback type")
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

func (pb *RawPB) wrapError(num int, err error) error {
	if err == nil {
		return nil
	}
	name := pb.name
	if name == "" {
		name = "<unnamed>"
	}
	return fmt.Errorf("%s[%d]: %s", name, num, err.Error())
}
