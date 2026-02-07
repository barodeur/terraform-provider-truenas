default: build

build:
	go build -o terraform-provider-truenas

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/barodeur/truenas/0.1.0/linux_amd64
	cp terraform-provider-truenas ~/.terraform.d/plugins/registry.terraform.io/barodeur/truenas/0.1.0/linux_amd64/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

.PHONY: build install test testacc fmt lint
