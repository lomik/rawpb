package rawpb

import (
	"fmt"
	"slices"
)

const maxFieldListItems = 128

type field struct {
	varint  func(v uint64) error
	fixed64 func(v uint64) error
	bytes   func(v []byte) error
	fixed32 func(v uint32) error
}

type callbacks struct {
	lst []field       // fields 1-128
	mp  map[int]field // fields 129+

	def struct { // defaults
		varint  func(num int, v uint64) error
		fixed64 func(num int, v uint64) error
		bytes   func(num int, v []byte) error
		fixed32 func(num int, v uint32) error
	}
}

func (cb *callbacks) set(num int, f field) {
	if num < 1 {
		panic(fmt.Sprintf("field number should be natural number, invalid value: %d", num))
	}
	if num > maxFieldListItems {
		if cb.mp == nil {
			cb.mp = make(map[int]field)
		}
		cb.mp[num] = f
		return
	}

	if cb.lst == nil {
		cb.lst = make([]field, num)
	}

	if len(cb.lst) < num {
		cb.lst = slices.Grow(cb.lst, num-len(cb.lst))
		cb.lst = cb.lst[:cap(cb.lst)]
	}

	cb.lst[num-1] = f
}

func fieldExpected(f field) string {
	if f.varint != nil {
		return "varint"
	}
	if f.bytes != nil {
		return "length-delimited"
	}
	if f.fixed32 != nil {
		return "fixed32"
	}
	if f.fixed64 != nil {
		return "fixed64"
	}
	return "none"
}

func (cb *callbacks) varintDefault(num int, v uint64) error {
	if cb.def.varint != nil {
		return cb.def.varint(num, v)
	}
	return nil
}
func (cb *callbacks) fixed64Default(num int, v uint64) error {
	if cb.def.fixed64 != nil {
		return cb.def.fixed64(num, v)
	}
	return nil
}
func (cb *callbacks) fixed32Default(num int, v uint32) error {
	if cb.def.fixed32 != nil {
		return cb.def.fixed32(num, v)
	}
	return nil
}
func (cb *callbacks) bytesDefault(num int, v []byte) error {
	if cb.def.bytes != nil {
		return cb.def.bytes(num, v)
	}
	return nil
}

func (cb *callbacks) varint(num int, v uint64) error {
	if num > maxFieldListItems {
		if cb.mp == nil {
			return cb.varintDefault(num, v)
		}
		if c, ok := cb.mp[num]; ok {
			if c.varint == nil {
				return fmt.Errorf("field %d: varint received, but %s expected", num, fieldExpected(c))
			}
			return c.varint(v)
		}
		return cb.varintDefault(num, v)
	}

	if num > len(cb.lst) {
		return cb.varintDefault(num, v)
	}

	c := cb.lst[num-1]
	if c.varint == nil {
		return fmt.Errorf("field %d: varint received, but %s expected", num, fieldExpected(c))
	}
	return c.varint(v)
}

func (cb *callbacks) fixed64(num int, v uint64) error {
	if num > maxFieldListItems {
		if cb.mp == nil {
			return cb.fixed64Default(num, v)
		}
		if c, ok := cb.mp[num]; ok {
			if c.fixed64 == nil {
				return fmt.Errorf("field %d: fixed64 received, but %s expected", num, fieldExpected(c))
			}
			return c.fixed64(v)
		}
		return cb.fixed64Default(num, v)
	}

	if num > len(cb.lst) {
		return cb.fixed64Default(num, v)
	}

	c := cb.lst[num-1]
	if c.fixed64 == nil {
		return fmt.Errorf("field %d: fixed64 received, but %s expected", num, fieldExpected(c))
	}
	return c.fixed64(v)
}

func (cb *callbacks) fixed32(num int, v uint32) error {
	if num > maxFieldListItems {
		if cb.mp == nil {
			return cb.fixed32Default(num, v)
		}
		if c, ok := cb.mp[num]; ok {
			if c.fixed32 == nil {
				return fmt.Errorf("field %d: fixed32 received, but %s expected", num, fieldExpected(c))
			}
			return c.fixed32(v)
		}
		return cb.fixed32Default(num, v)
	}

	if num > len(cb.lst) {
		return cb.fixed32Default(num, v)
	}

	c := cb.lst[num-1]
	if c.fixed32 == nil {
		return fmt.Errorf("field %d: fixed32 received, but %s expected", num, fieldExpected(c))
	}
	return c.fixed32(v)
}

func (cb *callbacks) packedVarint(f func(v uint64) error, p []byte) error {
	r := newReader(p)
	for r.next() {
		v, err := r.varint()
		if err != nil {
			return err
		}
		if err = f(v); err != nil {
			return err
		}
	}
	return nil
}

func (cb *callbacks) packedFixed32(f func(v uint32) error, p []byte) error {
	r := newReader(p)
	for r.next() {
		v, err := r.fixed32()
		if err != nil {
			return err
		}
		if err = f(v); err != nil {
			return err
		}
	}
	return nil
}

func (cb *callbacks) packedFixed64(f func(v uint64) error, p []byte) error {
	r := newReader(p)
	for r.next() {
		v, err := r.fixed64()
		if err != nil {
			return err
		}
		if err = f(v); err != nil {
			return err
		}
	}
	return nil
}

func (cb *callbacks) packed(f field, v []byte) error {
	if f.fixed32 != nil {
		return cb.packedFixed32(f.fixed32, v)
	}
	if f.fixed64 != nil {
		return cb.packedFixed64(f.fixed64, v)
	}
	if f.varint != nil {
		return cb.packedVarint(f.varint, v)
	}
	return nil
}

func (cb *callbacks) bytes(num int, v []byte) error {
	if num > maxFieldListItems {
		if cb.mp == nil {
			return cb.bytesDefault(num, v)
		}
		if c, ok := cb.mp[num]; ok {
			if c.bytes == nil {
				return cb.packed(c, v)
			}
			return c.bytes(v)
		}
		return cb.bytesDefault(num, v)
	}

	if num > len(cb.lst) {
		return cb.bytesDefault(num, v)
	}

	c := cb.lst[num-1]
	if c.bytes == nil {
		return cb.packed(c, v)
	}
	return c.bytes(v)
}
