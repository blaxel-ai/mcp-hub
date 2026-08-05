# AgentMail's own repo (agentmail-to/agentmail-mcp) no longer ships a
# buildable stdio server for us to clone and compile: it was restructured
# into a pnpm monorepo whose server package (packages/server) is AgentMail's
# own hosted, Clerk-authenticated HTTP deployment (mcp.agentmail.to) - it
# opens its own HTTP listener rather than speaking stdio, and needs secrets
# we don't have.
#
# The package upstream publishes for exactly this integration is the
# npm-stdio-bridge (packages/npm-stdio-bridge in that repo), published to npm
# as `agentmail-mcp`: a stdio<->HTTP bridge that authenticates to the hosted
# server with AGENTMAIL_API_KEY. Installing the published package (pinned
# exact version) is simpler and more robust here than cloning/building the
# monorepo: it's what upstream ships for self-hosting the stdio side, and it
# means we track the same compatibility surface upstream's own consumers do.
FROM node:22-alpine

WORKDIR /app

# Pinned exact version: matches packages/npm-stdio-bridge/package.json in
# agentmail-to/agentmail-mcp at the time this was written. The install
# failing is itself proof the package still resolves.
RUN npm install -g agentmail-mcp@1.0.2

COPY super-gateway ./super-gateway

ENTRYPOINT ["./super-gateway","--port","80", "--transport", "http-stream", "--stdio"]
