package rawpb

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/prometheus/prometheus/prompb"
)

func BenchmarkSlice(b *testing.B) {
	a := make([]field, 100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if a[42].bytes != nil {
			panic("fail")
		}
	}
}

func BenchmarkSlicePointer(b *testing.B) {
	a := make([]*field, 100)
	for i := 0; i < 100; i++ {
		a[i] = &field{}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if a[42].bytes != nil {
			panic("fail")
		}
	}
}

func BenchmarkMap(b *testing.B) {
	a := make(map[int]field)
	for i := 0; i < 100; i++ {
		a[i] = field{}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if a[42].bytes != nil {
			panic("fail")
		}
	}
}

func BenchmarkMapPointer(b *testing.B) {
	a := make(map[int]*field)
	for i := 0; i < 100; i++ {
		a[i] = &field{}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if a[42].bytes != nil {
			panic("fail")
		}
	}
}

func BenchmarkParse(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	r := New(
		Begin(func() error { return nil }),
		End(func() error { return nil }),
		Message(1, New(
			Begin(func() error { return nil }),
			End(func() error { return nil }),
			Message(1, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				Bytes(1, func(v []byte) error {
					return nil
				}),
				Bytes(2, func(v []byte) error {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Parse(raw); err != nil {
			panic(err)
		}
	}
}

func BenchmarkProtoUnmarshal(b *testing.B) {
	var req prompb.WriteRequest
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := proto.Unmarshal(raw, &req); err != nil {
			panic(err)
		}
	}
}
