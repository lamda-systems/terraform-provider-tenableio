default: build

BINARY=terraform-provider-tenableio
VERSION?=0.1.0
OS_ARCH?=linux_amd64
INSTALL_DIR=~/.terraform.d/plugins/registry.terraform.io/tenable/tenableio/$(VERSION)/$(OS_ARCH)

.PHONY: build install test testacc lint docs clean release-check

build:
	go build -o $(BINARY)

install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY) $(INSTALL_DIR)/

test:
	go test ./... -v -count=1

testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

lint:
	golangci-lint run ./...

docs:
	tfplugindocs generate

release-check:
	goreleaser check
	goreleaser release --snapshot --clean

clean:
	rm -f $(BINARY)
	rm -rf dist/
