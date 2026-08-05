FROM node:23-alpine
# Must be entire project because `prepare` script is run during `npm install` and requires all files.
COPY . /app
WORKDIR /app
# Pinned: pnpm 11 stopped reading the `pnpm.overrides` field from package.json
# (moved to pnpm-workspace.yaml), which turns `--frozen-lockfile` below into a
# hard config-mismatch error against this repo's current lockfile/config shape.
RUN npm install -g pnpm@10.34.5
RUN pnpm install --frozen-lockfile --ignore-scripts
# tsc's type-check across this module's combined SDK surface (aws-sdk,
# openai, twilio, qdrant, mcp) needs more than the default V8 heap; this is a
# pre-existing, unrelated issue (see dependency-verification.yaml) that this
# extra headroom does not fully resolve on a memory-constrained builder.
# Scoped to this one RUN (not ENV) so the heap ceiling never leaks into the
# shipped image: ENV persists into the final layer and would apply to the
# super-gateway entrypoint and every Node MCP subprocess it spawns, not just
# this build step.
RUN NODE_OPTIONS=--max-old-space-size=12288 pnpm run build
COPY super-gateway ./super-gateway
ENTRYPOINT ["./super-gateway","--transport", "http-stream", "--port","80","--stdio"]
