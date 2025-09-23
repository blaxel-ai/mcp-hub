FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS uv
# Install the project into /app
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Then, add the rest of the project source code and install it
# Installing separately from its dependencies allows optimal layer caching
ADD . /app

RUN uv sync --frozen --no-dev

# Place executables in the environment at the front of the path
ENV PATH="/app/.venv/bin:$PATH"

COPY super-gateway ./super-gateway

ENTRYPOINT ["./super-gateway","--mode", "http-stream", "--port","80","--stdio"]