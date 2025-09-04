
FROM node:22-alpine

# Set the working directory inside the container
WORKDIR /app

RUN apk add --no-cache python3 make g++ bash git

# Copy the package.json and package-lock.json files
COPY . .

RUN npm i -g pnpm && pnpm install && pnpm run build

WORKDIR /app/packages/mcp-server-postgrest

# Install the dependencies
RUN pnpm install && pnpm run build

COPY super-gateway ./super-gateway

# Command to run the application
ENTRYPOINT ["./super-gateway","--port","80","--stdio"]