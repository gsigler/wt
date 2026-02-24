.PHONY: build test install clean

build:
	go build -o wt .

test:
	go test ./...

install: build
	cp wt $(shell go env GOPATH)/bin/wt
	@echo ""
	@echo "Add this to your shell config (~/.zshrc or ~/.bashrc):"
	@echo '  eval "$$(wt shell-init)"'
	@echo ""
	@echo "Make sure it appears AFTER your PATH setup."

clean:
	rm -f wt
