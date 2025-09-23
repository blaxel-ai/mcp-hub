FROM node:23-alpine
# Must be entire project because `prepare` script is run during `npm install` and requires all files.
COPY . /app
WORKDIR /app
RUN npm install -g pnpm
RUN pnpm install --frozen-lockfile --ignore-scripts
RUN pnpm run build
COPY super-gateway ./super-gateway
ENTRYPOINT ["./super-gateway","--mode", "http-stream", "--port","80","--stdio"]
