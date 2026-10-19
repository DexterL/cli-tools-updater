PREFIX ?= $(HOME)/.local/bin

.PHONY: build install test vet

build:
	go build -o bin/upcli ./cmd/upcli

install: build
	mkdir -p "$(PREFIX)"
	ln -sfn "$(CURDIR)/bin/upcli" "$(PREFIX)/upcli"

test:
	go test ./...

vet:
	go vet ./...
