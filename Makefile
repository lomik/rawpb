cover:
	go test -cover -coverprofile coverage.out
	gocovsh

bench:
	go test -bench=. -benchmem

link-test:
	ln -f -s rawpb_test.go.ignore rawpb_test.go
	ln -f -s remote_write_test.go.ignore remote_write_test.go
	ln -f -s test.pb.go.ignore test/test.pb.go
	ln -f -s rawpb_all_test.go.ignore rawpb_all_test.go
	ln -f -s writer_test.go.ignore writer_test.go
	go mod tidy

unlink-test:
	rm rawpb_test.go
	rm remote_write_test.go
	rm test/test.pb.go
	rm rawpb_all_test.go
	rm writer_test.go
	go mod tidy

	
