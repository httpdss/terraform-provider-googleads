BINARY=terraform-provider-googleads
VERSION?=0.1.0
OS_ARCH?=linux_amd64

.PHONY: fmt test build install-local docs

fmt:
	gofmt -w .

test:
	go test ./...

build:
	go build -o bin/$(BINARY) .
	go build -o bin/googleads-auth ./cmd/googleads-auth

install-local: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/local/googleads/$(VERSION)/$(OS_ARCH)
	cp bin/$(BINARY) ~/.terraform.d/plugins/registry.terraform.io/local/googleads/$(VERSION)/$(OS_ARCH)/$(BINARY)_v$(VERSION)

docs:
	@echo "Documentation is handwritten in README.md and docs/."
