FROM node:22.12-alpine AS builder

COPY src/gitlab /app
COPY tsconfig.json /tsconfig.json

WORKDIR /app

RUN --mount=type=cache,target=/root/.npm npm install

RUN --mount=type=cache,target=/root/.npm-production npm ci --ignore-scripts --omit-dev

FROM node:22.12-alpine AS release

WORKDIR /app

COPY --from=builder /app/dist /app/dist
COPY --from=builder /app/package.json /app/package.json
COPY --from=builder /app/package-lock.json /app/package-lock.json

ENV NODE_ENV=production

# Install only production dependencies
RUN npm ci --production --ignore-scripts

COPY super-gateway ./super-gateway

# Command to run the application
ENTRYPOINT ["./super-gateway","--mode", "http-stream", "--port","80","--stdio"]