PLUGIN_NAME := vault-plugin-meilisearch
PLUGIN_DIR := cmd/$(PLUGIN_NAME)
GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)

.PHONY: build test clean fmt dev

build:
	go build -o $(PLUGIN_NAME) ./$(PLUGIN_DIR)

test:
	go test ./... -count=1 -v

fmt:
	go fmt ./...

clean:
	rm -f $(PLUGIN_NAME)

# Dev workflow: build, calculate SHA, register with Vault.
dev: build
	@echo "Plugin built: $(PLUGIN_NAME)"
	@echo "SHA256: $$(shasum -a 256 $(PLUGIN_NAME) | cut -d' ' -f1)"
	@echo ""
	@echo "To register with Vault:"
	@echo "  1. cp $(PLUGIN_NAME) <vault-plugin-dir>/"
	@echo "  2. vault plugin register -sha256=<sha> secret $(PLUGIN_NAME)"
	@echo "  3. vault secrets enable -path=meilisearch $(PLUGIN_NAME)"
