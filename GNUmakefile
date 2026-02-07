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

setup-truenas:
	go build -o setup-truenas ./cmd/setup-truenas

testacc-vm: setup-truenas
	@if [ -z "$$TRUENAS_ISO" ]; then echo "Error: TRUENAS_ISO must be set"; exit 1; fi
	scripts/truenas-vm.sh start
	./setup-truenas -host 127.0.0.1 -port $${TRUENAS_VM_PORT:-8080} -https-port $${TRUENAS_VM_HTTPS_PORT:-8443} -output-file /tmp/truenas-api-key
	TRUENAS_HOST="wss://127.0.0.1:$${TRUENAS_VM_HTTPS_PORT:-8443}" \
	TRUENAS_API_KEY="$$(cat /tmp/truenas-api-key)" \
	TF_ACC=1 go test ./internal/provider/ -v -timeout 10m; \
	rc=$$?; \
	scripts/truenas-vm.sh stop; \
	exit $$rc

.PHONY: build install test testacc fmt lint setup-truenas testacc-vm
