default: build

BINARY=terraform-provider-tenableio
INSTALL_DIR=~/.terraform.d/plugins/registry.terraform.io/tenable/tenableio/0.1.0/linux_amd64

.PHONY: build install test testacc lint docs clean

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

clean:
	rm -f $(BINARY)
