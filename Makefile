# This Makefile serves as a helper to run the MCP hub and the GitHub MCP server.
ARGS:= $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
REGISTRY:= ghcr.io/blaxel-ai/hub

import:
	go run main.go import -c hub -m $(ARGS) --debug

run: build-super-gateway
	@if [ ! -f bin/super-gateway ]; then \
		echo "bin/super-gateway not found, building it..."; \
		$(MAKE) build-super-gateway; \
	fi
	go run main.go start -m $(ARGS) --debug

catalog:
	go run main.go catalog -m $(ARGS) --debug --skip-build

test:
	cd tests \
	&& cp src/configs/config.$(ARGS).ts src/config.ts \
	&& pnpm run test

build-super-gateway:
	@echo "Building super-gateway for linux/amd64..."
	@cd super-gateway && \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-trimpath \
		-o ../bin/super-gateway \
		.
	@echo "Binary built: bin/super-gateway"
	@echo "Binary size: $$(du -h bin/super-gateway | cut -f1)"

%:
	@:

list:
	@echo "Available MCP servers:"
	@echo "======================"
	@for file in hub/*.yaml; do \
		basename=$$(basename $$file .yaml); \
		echo "  $$basename"; \
	done | sort
	@echo ""
	@echo "Usage: make <command> <mcp-name>"
	@echo "Example: make run github"

mr_develop:
	$(eval BRANCH_NAME := $(shell git rev-parse --abbrev-ref HEAD))
	gh pr create --base develop --head $(BRANCH_NAME) --title "$(BRANCH_NAME)" --body "Merge request from $(BRANCH_NAME) to develop"