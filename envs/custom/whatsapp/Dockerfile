# WhatsApp MCP Server Dockerfile
# Multi-runtime image supporting both Go (WhatsApp bridge) and Python (MCP server)

FROM golang:1.24-bookworm AS go-builder

# Install dependencies for Go build
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Set CGO enabled for sqlite3 support
ENV CGO_ENABLED=1

WORKDIR /app/whatsapp-bridge

# Copy Go module files
COPY whatsapp-bridge/go.mod whatsapp-bridge/go.sum ./

# Download Go dependencies
RUN go mod download

# Copy Go source code
COPY whatsapp-bridge/ ./

# Build the Go WhatsApp bridge
RUN go build -o whatsapp-bridge main.go

# Main runtime image with Python and UV
FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    sqlite3 \
    libsqlite3-dev \
    ffmpeg \
    curl \
    ca-certificates \
    procps \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy the built Go binary from builder stage
COPY --from=go-builder /app/whatsapp-bridge/whatsapp-bridge /usr/local/bin/whatsapp-bridge

# Enable bytecode compilation for Python
ENV UV_COMPILE_BYTECODE=1
# Copy from the cache instead of linking since it's a mounted volume
ENV UV_LINK_MODE=copy

# Create directories for the application structure
RUN mkdir -p /app/whatsapp-bridge/store /app/whatsapp-mcp-server

# Copy Python MCP server requirements first for better caching
COPY whatsapp-mcp-server/pyproject.toml whatsapp-mcp-server/uv.lock* /app/whatsapp-mcp-server/

# Install Python dependencies
RUN --mount=type=cache,target=/root/.cache/uv \
    cd /app/whatsapp-mcp-server && \
    uv sync --frozen --no-install-project --no-dev --no-editable

# Copy the rest of the Python MCP server source code
COPY whatsapp-mcp-server/ /app/whatsapp-mcp-server/

# Install the Python project
RUN --mount=type=cache,target=/root/.cache/uv \
    cd /app/whatsapp-mcp-server && \
    uv sync --frozen --no-dev --no-editable

# Place Python executables in the environment at the front of the path
ENV PATH="/app/whatsapp-mcp-server/.venv/bin:$PATH"

# Create a startup script to run both services
RUN echo '#!/bin/bash' > /app/start.sh && \
    echo 'set -e' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# Create store directory if it does not exist' >> /app/start.sh && \
    echo 'mkdir -p /app/whatsapp-bridge/store' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# Start WhatsApp bridge in the background' >> /app/start.sh && \
    echo 'echo "Starting WhatsApp bridge..."' >> /app/start.sh && \
    echo 'cd /app/whatsapp-bridge' >> /app/start.sh && \
    echo 'whatsapp-bridge &' >> /app/start.sh && \
    echo 'BRIDGE_PID=$!' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# Wait a moment for the bridge to initialize' >> /app/start.sh && \
    echo 'sleep 3' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# Start Python MCP server' >> /app/start.sh && \
    echo 'echo "Starting WhatsApp MCP server..."' >> /app/start.sh && \
    echo 'cd /app/whatsapp-mcp-server' >> /app/start.sh && \
    echo 'exec uv run main.py' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# If we get here, the MCP server has exited, so kill the bridge' >> /app/start.sh && \
    echo 'kill $BRIDGE_PID 2>/dev/null || true' >> /app/start.sh && \
    chmod +x /app/start.sh

# Expose port for MCP server (if needed for external access)
EXPOSE 3000

# Volume for persistent WhatsApp data and database
VOLUME ["/app/whatsapp-bridge/store"]

# Environment variables
ENV PYTHONUNBUFFERED=1
ENV GO111MODULE=on

# Health check to ensure both services are running
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD pgrep -f whatsapp-bridge && pgrep -f "main.py" || exit 1

# Start both services
ENTRYPOINT ["/app/start.sh"]