test.pb.go: test.proto
	protoc -I=. --go_out=. --go_opt=paths=source_relative test.proto

all: test.pb.go