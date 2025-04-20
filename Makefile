# This Makefile serves as a helper to run the MCP hub and the GitHub MCP server.
ARGS:= $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
REGISTRY:= ghcr.io/beamlit/hub

import:
	go run main.go import -c hub -m $(ARGS) --debug

import-ukc:
	go run main.go import -c hub -m $(ARGS) --debug --ukc

run:
	go run main.go start -m $(ARGS) --debug

catalog:
	go run main.go catalog -m $(ARGS) --debug --skip-build

test:
	cd hack/test_client \
	&& cp src/configs/config.$(ARGS).ts src/config.ts \
	&& pnpm run test

%:
	@:
