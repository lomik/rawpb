package rawpb

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pluto-metrics/rawpb/test"
	"github.com/stretchr/testify/assert"
)

func readFixture(name string) []byte {
	gz, err := os.ReadFile("fixtures/" + name)
	if err != nil {
		panic(err)
	}
	gzReader, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		panic(err)
	}
	raw, err := io.ReadAll(gzReader)
	if err != nil {
		panic(err)
	}
	return raw
}
func TestRawPB(t *testing.T) {
	assert := assert.New(t)
	body := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")

	r := New(
		Begin(func() error { return nil }),
		End(func() error { return nil }),
		Message(1, New(
			Begin(func() error { return nil }),
			End(func() error { return nil }),
			Message(1, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				String(1, func(v string) error {
					return nil
				}),
				String(2, func(v string) error {
					return nil
				}),
			)),
			Message(2, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				Double(1, func(v float64) error {
					return nil
				}),
				Int64(2, func(v int64) error {
					return nil
				}),
			)),
		)),
	)

	err := r.Parse(body)
	assert.NoError(err)
}

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

	/*
			    uint64 big_number_varint = 12313;
		    uint64 big_number_fixed32 = 12314;
		    uint64 big_number_fixed64 = 12315;
		    uint64 big_number_string = 12315;
	*/

	simple(t, 12313, Uint64, 123123, &msg.BigNumberVarint, &msg)
	simple(t, 12314, Fixed32, 123123, &msg.BigNumberFixed32, &msg)
	simple(t, 12315, Fixed64, 123123, &msg.BigNumberFixed64, &msg)
	simple(t, 12316, String, "hello world", &msg.BigNumberString, &msg)

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
