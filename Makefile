.PHONY: build run test clean

build:
	go build -o agm ./cmd/agm

run: build
	./agm

test:
	go test ./...

clean:
	rm -f agm
