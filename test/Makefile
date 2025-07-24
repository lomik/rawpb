test.pb.go: test.proto
	protoc -I=. --gofast_out=. --gofast_opt=paths=source_relative test.proto

all: test.pb.go