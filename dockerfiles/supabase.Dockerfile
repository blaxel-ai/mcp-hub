
FROM node:22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

RUN apk add --no-cache python3 make g++ bash git

# Copy the package.json and package-lock.json files
COPY packages/mcp-server-supabase /app

# Install the dependencies
RUN --mount=type=cache,target=/root/.npm npm install && npm run build

# Use a separate runtime environment
FROM node:22-alpine

# Set the working directory
WORKDIR /app

# Copy built files from the builder stage
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json /app/package-lock.json ./
COPY --from=builder /app/node_modules ./node_modules
# Expose the port the app runs on
# (This line is optional and depends on whether you want to specify a port to be exposed)

# Install only production dependencies
RUN npm ci --production --ignore-scripts

COPY super-gateway ./super-gateway

# Command to run the application
ENTRYPOINT ["./super-gateway","--port","80","--stdio"]