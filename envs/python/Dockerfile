FROM ghcr.io/astral-sh/uv:python3.12-bookworm

WORKDIR /blaxel

RUN apt update && apt install -y curl gnupg git

# Enable bytecode compilation
ENV UV_COMPILE_BYTECODE=1
# Copy from the cache instead of linking since it's a mounted volume
ENV UV_LINK_MODE=copy
# Install the project's dependencies using the lockfile and settings
RUN --mount=type=cache,target=/root/.cache/uv \
  --mount=type=bind,source=uv.lock,target=uv.lock \
  --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
  uv sync --frozen --no-install-project --no-dev --no-editable

COPY . /blaxel

RUN --mount=type=cache,target=/root/.cache/uv \
  uv sync --frozen --no-dev --no-editable
RUN uv add 'uvicorn[standard]'

# Place executables in the environment at the front of the path
ENV PATH="/blaxel/.venv/bin:$PATH"

ENTRYPOINT [${ENTRYPOINT}]