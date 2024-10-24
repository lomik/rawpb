# rawpb
Raw protobuf message reader

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
            String(1, func(v string) error { // Name
                ts.Labels[len(ts.Labels)-1].Name = v
                return nil
            }),
            String(2, func(v string) error { // Value
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

```bash
> go test -bench=. -benchmem
BenchmarkGogoUnmarshalWriteRequest-8   	     711	   1875505 ns/op	 3815839 B/op	   35980 allocs/op
BenchmarkRawpbParseWriteRequest-8      	    2396	    480921 ns/op	       0 B/op	       0 allocs/op
```
