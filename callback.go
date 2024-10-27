package rawpb

import (
	"fmt"
	"slices"
)

const maxFieldListItems = 128

type callbackFunc interface {
	func(v uint64) error | func(v uint32) error | func(v []byte) error
}

type callbackType int

const (
	callbackTypeNone    callbackType = 0
	callbackTypeVarint  callbackType = 1
	callbackTypeFixed64 callbackType = 2
	callbackTypeFixed32 callbackType = 3
	callbackTypeBytes   callbackType = 4
)

var callbackTypeString = map[callbackType]string{
	callbackTypeNone:    "none",
	callbackTypeVarint:  "varint",
	callbackTypeFixed64: "fixed64",
	callbackTypeFixed32: "fixed32",
	callbackTypeBytes:   "length-delimited",
}

type callback struct {
	tp         callbackType
	funcUint64 func(v uint64) error
	funcUint32 func(v uint32) error
	funcBytes  func(v []byte) error
}

type callbacks struct {
	lst []callback       // fields 1-128
	mp  map[int]callback // fields 129+

	unknown struct { // defaults
		varint  func(num int, v uint64) error
		fixed64 func(num int, v uint64) error
		bytes   func(num int, v []byte) error
		fixed32 func(num int, v uint32) error
	}
}

var emptyCallback = callback{}

func (c callback) isEmpty() bool {
	return c.tp == callbackTypeNone
}

func (cb *callbacks) set(num int, c callback) {
	if num < 1 {
		panic(fmt.Sprintf("field number should be natural number, invalid value: %d", num))
	}
	if num > maxFieldListItems {
		if cb.mp == nil {
			cb.mp = make(map[int]callback)
		}
		cb.mp[num] = c
		return
	}

	if cb.lst == nil {
		cb.lst = make([]callback, num)
	}

	if len(cb.lst) < num {
		cb.lst = slices.Grow(cb.lst, num-len(cb.lst))
		cb.lst = cb.lst[:cap(cb.lst)]
	}

	cb.lst[num-1] = c
}

func (cb *callbacks) setVarint(num int, f func(v uint64) error) {
	cb.set(num, callback{
		tp:         callbackTypeVarint,
		funcUint64: f,
	})
}

func (cb *callbacks) setFixed64(num int, f func(v uint64) error) {
	cb.set(num, callback{
		tp:         callbackTypeFixed64,
		funcUint64: f,
	})
}

func (cb *callbacks) setFixed32(num int, f func(v uint32) error) {
	cb.set(num, callback{
		tp:         callbackTypeFixed32,
		funcUint32: f,
	})
}

func (cb *callbacks) setBytes(num int, f func(v []byte) error) {
	cb.set(num, callback{
		tp:        callbackTypeBytes,
		funcBytes: f,
	})
}

func (c *callback) wireType() string {
	return callbackTypeString[c.tp]
}

func (cb *callbacks) get(num int) callback {
	if num > maxFieldListItems {
		return cb.mp[num]
	}

	if num <= len(cb.lst) {
		return cb.lst[num-1]
	}

	return emptyCallback
}

func call[T any](f func(v T) error, v T) error {
	if f == nil {
		return nil
	}
	return f(v)
}

func callUnknown[T any](f func(num int, v T) error, num int, v T) error {
	if f == nil {
		return nil
	}
	return f(num, v)
}

func (cb *callbacks) varint(num int, v uint64) error {
	c := cb.get(num)
	if c.isEmpty() {
		return callUnknown(cb.unknown.varint, num, v)
	}
	if c.tp == callbackTypeVarint {
		return call(c.funcUint64, v)
	}
	return fmt.Errorf("field %d: varint received, but %s expected", num, c.wireType())
}

func (cb *callbacks) fixed64(num int, v uint64) error {
	c := cb.get(num)
	if c.isEmpty() {
		return callUnknown(cb.unknown.fixed64, num, v)
	}
	if c.tp == callbackTypeFixed64 {
		return call(c.funcUint64, v)
	}
	return fmt.Errorf("field %d: fixed64 received, but %s expected", num, c.wireType())
}

func (cb *callbacks) fixed32(num int, v uint32) error {
	c := cb.get(num)
	if c.isEmpty() {
		return callUnknown(cb.unknown.fixed32, num, v)
	}
	if c.tp == callbackTypeFixed32 {
		return call(c.funcUint32, v)
	}
	return fmt.Errorf("field %d: fixed32 received, but %s expected", num, c.wireType())
}

func (cb *callbacks) bytes(num int, v []byte) error {
	c := cb.get(num)
	if c.isEmpty() {
		return callUnknown(cb.unknown.bytes, num, v)
	}

	if c.tp == callbackTypeBytes {
		return call(c.funcBytes, v)
	}

	// packed
	switch c.tp {
	case callbackTypeVarint:
		r := newReader(v)
		for r.next() {
			vv, err := r.varint()
			if err != nil {
				return fmt.Errorf("field %d: parse packed failed: %s", num, err.Error())
			}
			if err = call(c.funcUint64, vv); err != nil {
				return err
			}
		}
	case callbackTypeFixed64:
		r := newReader(v)
		for r.next() {
			vv, err := r.fixed64()
			if err != nil {
				return fmt.Errorf("field %d: parse packed failed: %s", num, err.Error())
			}
			if err = call(c.funcUint64, vv); err != nil {
				return err
			}
		}
	case callbackTypeFixed32:
		r := newReader(v)
		for r.next() {
			vv, err := r.fixed32()
			if err != nil {
				return fmt.Errorf("field %d: parse packed failed: %s", num, err.Error())
			}
			if err = call(c.funcUint32, vv); err != nil {
				return err
			}
		}
	default:
		panic("unknown callback type")
	}

	return nil
}
