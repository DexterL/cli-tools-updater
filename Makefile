PREFIX ?= $(HOME)/.local/bin
VERSION ?= dev
DIST ?= dist
PACKAGE = upcli_$(VERSION)_darwin_amd64

.PHONY: build install test vet release-darwin-amd64

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/upcli ./cmd/upcli

install: build
	mkdir -p "$(PREFIX)"
	ln -sfn "$(CURDIR)/bin/upcli" "$(PREFIX)/upcli"

test:
	go test ./...

vet:
	go vet ./...

release-darwin-amd64:
	@set -eu; \
	test "$(VERSION)" != dev || { echo "Set VERSION, for example: make release-darwin-amd64 VERSION=v0.1.0" >&2; exit 2; }; \
	stage=$$(mktemp -d); \
	trap 'rm -rf "$$stage"' EXIT; \
	mkdir -p "$(DIST)" "$$stage/$(PACKAGE)"; \
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o "$$stage/$(PACKAGE)/upcli" ./cmd/upcli; \
	cp scripts/install.sh README_INSTALL.md LICENSE "$$stage/$(PACKAGE)/"; \
	tar -czf "$(DIST)/$(PACKAGE).tar.gz" -C "$$stage" "$(PACKAGE)"; \
	(cd "$(DIST)" && shasum -a 256 "$(PACKAGE).tar.gz" > "$(PACKAGE).tar.gz.sha256"); \
	echo "Created $(DIST)/$(PACKAGE).tar.gz"
