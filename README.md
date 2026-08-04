# rawpb

[![Go Reference](https://pkg.go.dev/badge/github.com/lomik/rawpb.svg)](https://pkg.go.dev/github.com/lomik/rawpb)

rawpb is a tiny Go library for reading and writing protobuf messages directly, without generated code.

A lightweight Go library for reading and writing raw protobuf messages without generated code.

## Features

- Parse raw protobuf without generated code
- Write protobuf messages
- High‑performance zero‑allocation parsing

## Limitations

- Groups (wire types 3 and 4, `SGROUP`/`EGROUP`) are not supported. They were
  deprecated in proto3 but still appear in some proto2 messages; encountering
  them returns `ErrorWrongWireType`.

## Installation

go get github.com/lomik/rawpb

## Usage
Using the library with an example of a Prometheus remote write message.

```golang
var ts prompb.TimeSeries

r := New(
    Begin(func() error {
        // begin WriteRequest
        return nil
    }),
    End(func() error {
        // end WriteRequest
        return nil
    }),
    Message(1, New( // TimeSeries
        Begin(func() error {
            // begin TimeSeries, reset
            ts.Labels = ts.Labels[:0]
            ts.Samples = ts.Samples[:0]
            return nil
        }),
        End(func() error {
            // do something with single TimeSeries
            return nil
        }),
        Message(1, New( // Labels
            Begin(func() error {
                // append new Label
                ts.Labels = append(ts.Labels, prompb.Label{})
                return nil
            }),
            End(func() error { return nil }),
            UnsafeString(1, func(v string) error { // Name
                ts.Labels[len(ts.Labels)-1].Name = v
                return nil
            }),
            UnsafeString(2, func(v string) error { // Value
                ts.Labels[len(ts.Labels)-1].Value = v
                return nil
            }),
        )),
        Message(2, New( // Samples
            Begin(func() error {
                // append new Sample
                ts.Samples = append(ts.Samples, prompb.Sample{})
                return nil
            }),
            End(func() error { return nil }),
            Double(1, func(v float64) error { // Value
                ts.Samples[len(ts.Samples)-1].Value = v
                return nil
            }),
            Int64(2, func(v int64) error { // Timestamp
                ts.Samples[len(ts.Samples)-1].Timestamp = v
                return nil
            }),
        )),
    )),
)

r.Parse(raw)
```

## Pull decoder

The callback API above builds a decoder from a schema tree; every field
callback is stored in the parser and invoked during `Parse`. If you prefer
straight-line code, `Decoder` exposes the same wire-format engine as a
pull-based iterator:

```golang
var d rawpb.Decoder
d.Reset(raw)
for d.Next() {
    if d.Num() != 1 { continue }          // top-level TimeSeries

    ts.Labels = ts.Labels[:0]
    ts.Samples = ts.Samples[:0]
    tsD := d.Submessage()

    for tsD.Next() {
        switch tsD.Num() {
        case 1: // Label
            ts.Labels = append(ts.Labels, prompb.Label{})
            lab := tsD.Submessage()
            for lab.Next() {
                switch lab.Num() {
                case 1: ts.Labels[len(ts.Labels)-1].Name = lab.UnsafeString()
                case 2: ts.Labels[len(ts.Labels)-1].Value = lab.UnsafeString()
                }
            }
        case 2: // Sample
            ts.Samples = append(ts.Samples, prompb.Sample{})
            sam := tsD.Submessage()
            for sam.Next() {
                switch sam.Num() {
                case 1: ts.Samples[len(ts.Samples)-1].Value = sam.Double()
                case 2: ts.Samples[len(ts.Samples)-1].Timestamp = sam.Int64()
                }
            }
        }
    }
}
if err := d.Err(); err != nil { return err }
```

`Decoder` is a value type — declare it locally and it stays on the stack.
`Submessage()` also returns a value, so nested loops need no allocation.
Errors are sticky: check `d.Err()` after the loop. Wire-type mismatches
(e.g. `Uint64()` on a length-delimited field) set `ErrorWrongWireType` and
terminate iteration.

Repeated scalar fields — packed (proto3 default) or unpacked — use the
same accessor. Calling a scalar accessor (`Int32`, `Double`, `Fixed64`,
...) on a length-delimited field auto-unpacks the payload: the call
returns the first value, and each subsequent `Next()` yields the next
packed value under the same field number. If the caller doesn't touch a
LEN field with a scalar accessor, `Next()` moves straight to the next
tag on its next call.

```golang
for d.Next() {
    switch d.Num() {
    case 3: xs = append(xs, d.Int32())     // packed OR unpacked
    case 5: ys = append(ys, d.Double())
    case 7: name = d.UnsafeString()        // LEN as string, no unpacking
    }
}
```

An empty packed field yields one iteration with a zero value (real-world
encoders normally omit empty packed fields, so this rarely surfaces).

Benchmarks on the same Prometheus `WriteRequest` fixture:

```
BenchmarkGogoUnmarshalWriteRequest   2465019 ns/op   3815806 B/op   35980 allocs/op
BenchmarkRawpbParseWriteRequest       765138 ns/op         0 B/op       0 allocs/op
BenchmarkRawpbDecoderWriteRequest     657991 ns/op         0 B/op       0 allocs/op
```

## Write

```golang
rawpb.Write(out, func(w *Writer) error {
    w.String(12, "test string")
    w.Float(16, 123.456)

    // submessage
    w.Message(17, func(w *Writer) error {
        w.String(1, "sub message")
        w.Enum(2, int32(test.EnumType_ENUM_TYPE_VALUE2))

        w.Message(3, func(w *Writer) error {
            w.Uint32(28, 42)
            return nil
        })
        return nil
    })
})
```

```bash
> go test -bench=. -benchmem
BenchmarkGogoUnmarshalWriteRequest-8   	     711	   1875505 ns/op	 3815839 B/op	   35980 allocs/op
BenchmarkRawpbParseWriteRequest-8      	    2396	    480921 ns/op	       0 B/op	       0 allocs/op
```
