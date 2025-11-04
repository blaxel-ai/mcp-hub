# Use a separate runtime environment
FROM node:22-alpine

# Set the working directory
WORKDIR /app

# Expose the port the app runs on
# (This line is optional and depends on whether you want to specify a port to be exposed)

RUN apk add git \
  && npm install -g pnpm

RUN pnpm i @hubspot/mcp-server

COPY super-gateway ./super-gateway

# Command to run the application
ENTRYPOINT ["./super-gateway","--transport", "http-stream", "--port","80","--stdio"]