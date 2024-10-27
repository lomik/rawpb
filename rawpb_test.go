package rawpb

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pluto-metrics/rawpb/test"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	var msg test.Main

	simple(t, 1, Int32, 42, &msg.SimpleInt32, &msg)
	simple(t, 1, Int32, -42, &msg.SimpleInt32, &msg)

	simple(t, 2, Int64, 42, &msg.SimpleInt64, &msg)
	simple(t, 2, Int64, -42, &msg.SimpleInt64, &msg)

	simple(t, 3, Uint32, 42, &msg.SimpleUint32, &msg)

	simple(t, 4, Uint64, 123442, &msg.SimpleUint64, &msg)

	simple(t, 5, Sint32, 123442, &msg.SimpleSint32, &msg)
	simple(t, 5, Sint32, -123442, &msg.SimpleSint32, &msg)

	simple(t, 6, Sint64, 123442, &msg.SimpleSint64, &msg)
	simple(t, 6, Sint64, -123442, &msg.SimpleSint64, &msg)

	simple(t, 7, Bool, true, &msg.SimpleBool, &msg)
	// simple(t, 7, Bool, false, &msg.SimpleBool, &msg) // zero value not encoded

	func() {
		var v int32
		var cnt int
		assert.NoError(withMain(
			func(msg *test.Main) {
				msg.SimpleEnum = test.EnumType_ENUM_TYPE_VALUE2
			},
			Enum(8, func(i int32) error {
				cnt++
				v = i
				return nil
			})),
		)

		assert.Equal(int32(test.EnumType_ENUM_TYPE_VALUE2), v)
		assert.Equal(1, cnt)
	}()

	simple(t, 9, Fixed64, 123442, &msg.SimpleFixed64, &msg)

	simple(t, 10, Sfixed64, 123442, &msg.SimpleSfixed64, &msg)
	simple(t, 10, Sfixed64, -123442, &msg.SimpleSfixed64, &msg)

	simple(t, 11, Double, 123442, &msg.SimpleDouble, &msg)
	simple(t, 11, Double, -123442, &msg.SimpleDouble, &msg)
	// @TODO check nan
	// simple(t, 11, Double, math.NaN(), &msg.SimpleDouble, &msg)

	simple(t, 12, String, "Hello world", &msg.SimpleString, &msg)

	simple(t, 13, Bytes, []byte("Hello world"), &msg.SimpleBytes, &msg)

	simple(t, 14, Fixed32, 123442, &msg.SimpleFixed32, &msg)

	simple(t, 15, Sfixed32, 123442, &msg.SimpleSfixed32, &msg)
	simple(t, 15, Sfixed32, -123442, &msg.SimpleSfixed32, &msg)

	simple(t, 16, Float, 123442, &msg.SimpleFloat, &msg)
	simple(t, 16, Float, -123442, &msg.SimpleFloat, &msg)

	repeated(t, 18, Uint32, []uint32{1, 43, 2, 123123}, &msg.RepeatedUint32, &msg)

	repeated(t, 19, String, []string{"asde", "", "hello world"}, &msg.RepeatedString, &msg)

	repeated(t, 20, Uint32, []uint32{1, 43, 2, 123123}, &msg.RepeatedPackedUint32, &msg)

	repeated(t, 21, Float, []float32{1, 43, 2.42, 123123}, &msg.RepeatedPackedFloat, &msg)

	repeated(t, 22, Double, []float64{1, 43, 2.42, 123123}, &msg.RepeatedPackedDouble, &msg)

	// big field numbers
	simple(t, 12313, Uint64, 123123, &msg.BigNumberVarint, &msg)
	simple(t, 12314, Fixed32, 123123, &msg.BigNumberFixed32, &msg)
	simple(t, 12315, Fixed64, 123123, &msg.BigNumberFixed64, &msg)
	simple(t, 12316, String, "hello world", &msg.BigNumberString, &msg)

	// unknown fields
	assert.NoError(withMain(
		func(msg *test.Main) {
			msg.SimpleUint64 = 1234
			msg.SimpleFixed32 = 97663
			msg.SimpleFixed64 = 12311
			msg.SimpleString = "hello world"
		},
		UnknownVarint(func(num int, v uint64) error {
			assert.Equal(4, num)
			assert.Equal(uint64(1234), v)
			return nil
		}),
		UnknownFixed32(func(num int, v uint32) error {
			assert.Equal(14, num)
			assert.Equal(uint32(97663), v)
			return nil
		}),
		UnknownFixed64(func(num int, v uint64) error {
			assert.Equal(9, num)
			assert.Equal(uint64(12311), v)
			return nil
		}),
		UnknownBytes(func(num int, v []byte) error {
			assert.Equal(12, num)
			assert.Equal([]byte("hello world"), v)
			return nil
		}),
	))

	// big unknown fields
	assert.NoError(withMain(
		func(msg *test.Main) {
			msg.BigNumberVarint = 1234
			msg.BigNumberFixed32 = 97663
			msg.BigNumberFixed64 = 12311
			msg.BigNumberString = "hello world"
		},
		UnknownVarint(func(num int, v uint64) error {
			assert.Equal(12313, num)
			assert.Equal(uint64(1234), v)
			return nil
		}),
		UnknownFixed32(func(num int, v uint32) error {
			assert.Equal(12314, num)
			assert.Equal(uint32(97663), v)
			return nil
		}),
		UnknownFixed64(func(num int, v uint64) error {
			assert.Equal(12315, num)
			assert.Equal(uint64(12311), v)
			return nil
		}),
		UnknownBytes(func(num int, v []byte) error {
			assert.Equal(12316, num)
			assert.Equal([]byte("hello world"), v)
			return nil
		}),
	))

	// without callback
	assert.NoError(withMain(
		func(msg *test.Main) {
			msg.SimpleUint64 = 1234
			msg.SimpleFixed32 = 97663
			msg.SimpleFixed64 = 12311
			msg.SimpleString = "hello world"
		},
	))

	// empty field in array
	assert.NoError(withMain(
		func(msg *test.Main) {
			msg.SimpleInt32 = 1
			msg.SimpleString = "hello world"
		},
		String(12, func(s string) error { return nil }),
	))

	// nil callback
	assert.NoError(withMain(
		func(msg *test.Main) {
			msg.SimpleInt32 = 1
			msg.SimpleString = "hello world"
		},
		String(12, nil),
	))

}

func TestErrors(t *testing.T) {
	assert := assert.New(t)

	// natural number field
	assert.Panics(
		func() {
			withMain(func(msg *test.Main) {}, Uint64(-1, func(u uint64) error { return nil }))
		},
	)

	assert.Panics(
		func() {
			withMain(func(msg *test.Main) {}, Uint64(0, func(u uint64) error { return nil }))
		},
	)

	// check type mismatch
	assert.ErrorContains(withMain(
		func(msg *test.Main) {
			msg.SimpleInt32 = 42
		},
		Fixed64(1, func(u uint64) error { return nil }),
	), "field 1: varint received, but fixed64 expected")

}

func simple[T any](t *testing.T, num int, opt func(num int, f func(T) error) Option, value T, field *T, msg *test.Main) {
	assert := assert.New(t)
	msg.Reset()

	*field = value

	body, err := proto.Marshal(msg)
	if !assert.NoError(err) {
		return
	}

	cnt := 0
	var pv T

	p := New(opt(num, func(v T) error {
		pv = v
		cnt++
		return nil
	}))

	assert.NoError(p.Parse(body))
	assert.Equal(1, cnt)
	assert.Equal(value, pv)

	// and nil
	p2 := New(opt(num, nil))
	assert.NoError(p2.Parse(body))
}

func repeated[T any](t *testing.T, num int, opt func(num int, f func(T) error) Option, value []T, field *[]T, msg *test.Main) {
	assert := assert.New(t)
	msg.Reset()

	*field = value

	body, err := proto.Marshal(msg)
	if !assert.NoError(err) {
		return
	}

	var pv []T

	p := New(opt(num, func(v T) error {
		pv = append(pv, v)
		return nil
	}))

	assert.NoError(p.Parse(body))
	assert.Equal(value, pv)

	// and nil
	p2 := New(opt(num, nil))
	assert.NoError(p2.Parse(body))

}

func withMain(setter func(msg *test.Main), opts ...Option) error {
	msg := new(test.Main)
	setter(msg)

	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	p := New(opts...)

	return p.Parse(body)
}

func withBody(setter func(msg *test.Main), parser func(p []byte) error) error {
	msg := new(test.Main)
	setter(msg)

	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return parser(body)
}
