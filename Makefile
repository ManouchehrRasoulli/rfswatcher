unit_test:
	go test -v ./...

build:
	go build -o=./bin/rfswatcher.out ./cmd/main.go