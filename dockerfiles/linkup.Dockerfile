# Upstream's original Python server (LinkupPlatform/python-mcp-server) was
# archived 2025-10-27. Linkup's maintained successor is a TypeScript rewrite,
# LinkupPlatform/linkup-mcp-server, published to npm as `linkup-mcp-server`
# (its `bin` runs dist/stdio.js directly - stdio mode - and reads
# LINKUP_API_KEY, same as the old server). Installing the published package
# (pinned exact version) replaces the old `uv sync` build from a clone of the
# now-archived repo. Requires Node >=24 per that package's own engines field.
FROM node:24-alpine

WORKDIR /app

RUN npm install -g linkup-mcp-server@3.3.0

COPY super-gateway ./super-gateway

ENTRYPOINT ["./super-gateway","--transport", "http-stream", "--port","80","--stdio"]
