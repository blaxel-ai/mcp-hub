FROM python:3.12-slim-bookworm

WORKDIR /app

# Install system dependencies and pipx
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/* \
    && python -m pip install --upgrade pip \
    && python -m pip install pipx \
    && python -m pipx ensurepath

# Ensure pipx is in PATH
ENV PATH="/root/.local/bin:$PATH"

# Copy the super-gateway
COPY super-gateway ./super-gateway

# Test that pipx can run nia-mcp-server (this will install it if needed)
RUN pipx run --no-cache nia-mcp-server --help || true

ENTRYPOINT ["./super-gateway","--port","80","--stdio"]