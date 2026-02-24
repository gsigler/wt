.PHONY: build test install clean

build:
	go build -o wt .

test:
	go test ./...

install: build
	cp wt $(shell go env GOPATH)/bin/wt

clean:
	rm -f wt
