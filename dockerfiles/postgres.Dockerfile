FROM node:22.12-alpine AS builder
COPY src/postgres /app
COPY tsconfig.json /tsconfig.json
WORKDIR /app
RUN --mount=type=cache,target=/root/.npm npm install
RUN --mount=type=cache,target=/root/.npm-production npm ci --ignore-scripts --omit-dev
FROM node:22-alpine AS release
COPY --from=builder /app/dist /app/dist
COPY --from=builder /app/package.json /app/package.json
COPY --from=builder /app/package-lock.json /app/package-lock.json
ENV NODE_ENV=production
WORKDIR /app
RUN npm ci --ignore-scripts --omit-dev

RUN apk add --no-cache git
RUN npm install -g pnpm
RUN pnpm install https://github.com/blaxel-ai/supergateway
ENTRYPOINT ["npx","-y","@blaxel/supergateway","--port","80","--stdio"]