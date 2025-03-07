# This Makefile serves as a helper to run the MCP hub and the GitHub MCP server.
ARGS:= $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
REGISTRY:= ghcr.io/beamlit/hub
GITHUB_PERSONAL_ACCESS_TOKEN ?= "ghp_2555555555555555555555555555555555555555"

build-mcps:
	go run main.go import -c hub

run-github-mcp: build-mcps
	docker run -p 8000:8000 -e GITHUB_PERSONAL_ACCESS_TOKEN=$(GITHUB_PERSONAL_ACCESS_TOKEN) github-smithery-reference-servers:main --baseUrl "http://0.0.0.0:8000"

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
