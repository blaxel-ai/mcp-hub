# Use a separate runtime environment
FROM node:22-alpine

# Set the working directory
WORKDIR /app

# Expose the port the app runs on
# (This line is optional and depends on whether you want to specify a port to be exposed)

RUN apk add git \
  && npm install -g pnpm \
  && pnpm install https://github.com/beamlit/supergateway

RUN pnpm i @hubspot/mcp-server

# Command to run the application
ENTRYPOINT ["npx","-y","@blaxel/supergateway","--port","80","--stdio"]