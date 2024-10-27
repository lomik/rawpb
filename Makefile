cover:
	go test -cover -coverprofile coverage.out
	gocovsh

bench:
	go test -bench=. -benchmem
