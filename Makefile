# This Makefile serves as a helper to run the MCP hub and the GitHub MCP server.

GITHUB_PERSONAL_ACCESS_TOKEN ?= "ghp_2555555555555555555555555555555555555555"

build-mcps:
	go run main.go import -c hub.yaml

run-github-mcp: build-mcps
	docker run -p 8000:8000 -e GITHUB_PERSONAL_ACCESS_TOKEN=$(GITHUB_PERSONAL_ACCESS_TOKEN) github-smithery-reference-servers:main --baseUrl "http://0.0.0.0:8000"

