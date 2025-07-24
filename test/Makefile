test.pb.go.ignore: test.proto
	protoc -I=. --gofast_out=. --gofast_opt=paths=source_relative test.proto
	cp test.pb.go test.pb.go.ignore

all: test.pb.go.ignore