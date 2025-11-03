
FROM node:22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies and pnpm
RUN apk add --no-cache python3 make g++ bash git && \
    npm install -g pnpm@10.15.0

# Copy workspace configuration files
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./

# Copy all packages (we only need postgrest but workspace deps might be needed)
COPY packages/ ./packages/

# Install dependencies using pnpm with frozen lockfile
RUN pnpm install --frozen-lockfile

# Build mcp-utils first (dependency of postgrest)
WORKDIR /app/packages/mcp-utils
RUN pnpm run build

# Build the postgrest package
WORKDIR /app/packages/mcp-server-postgrest
RUN pnpm run build

# Use a separate runtime environment
FROM node:22-alpine

# Install pnpm in runtime for production dependencies
RUN npm install -g pnpm@10.15.0

# Set the working directory
WORKDIR /app

# Copy the workspace configuration and root package.json
COPY --from=builder /app/pnpm-workspace.yaml /app/package.json ./

# Copy built mcp-utils package (dependency)
COPY --from=builder /app/packages/mcp-utils/package.json ./packages/mcp-utils/
COPY --from=builder /app/packages/mcp-utils/dist ./packages/mcp-utils/dist

# Copy built mcp-server-postgrest package
COPY --from=builder /app/packages/mcp-server-postgrest/package.json ./packages/mcp-server-postgrest/
COPY --from=builder /app/packages/mcp-server-postgrest/dist ./packages/mcp-server-postgrest/dist

# Copy lockfile and install only production dependencies
COPY --from=builder /app/pnpm-lock.yaml ./
RUN pnpm install --prod --frozen-lockfile --ignore-scripts

# Copy the super-gateway binary
COPY super-gateway /app/packages/mcp-server-postgrest/super-gateway

# Set working directory to the server package
WORKDIR /app/packages/mcp-server-postgrest

# Command to run the application
ENTRYPOINT ["/app/packages/mcp-server-postgrest/super-gateway","--transport", "http-stream", "--port","80","--stdio"]