FROM ghcr.io/astral-sh/uv:python3.12-bookworm AS render-requirements

WORKDIR /app
COPY pyproject.toml /app/pyproject.toml
COPY uv.lock /app/uv.lock

COPY README.m[d] /app/README.md
COPY LICENS[E] /app/LICENSE
COPY src /app/src
RUN uv sync
RUN uv add 'uvicorn[standard]'
RUN uv export --no-hashes --format requirements-txt > requirements.txt

FROM python:3.12-bookworm AS build

RUN apt update && apt install -y curl

WORKDIR /blaxel

# Install the application dependencies.
COPY pyproject.toml /blaxel/pyproject.toml
COPY uv.lock /blaxel/uv.lock

COPY README.m[d] /blaxel/README.md
COPY LICENS[E] /blaxel/LICENSE
COPY src /blaxel/src
COPY --from=render-requirements /app/requirements.txt requirements.txt
RUN pip3 install -r requirements.txt --no-cache-dir
RUN /usr/sbin/ldconfig /usr/local/lib

FROM scratch
COPY --from=build /usr/local/lib /usr/local/lib
COPY --from=build /usr/local/bin/python3 /usr/bin/python3
COPY --from=build /lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/libc.so.6
COPY --from=build /lib/x86_64-linux-gnu/libm.so.6 /lib/x86_64-linux-gnu/libm.so.6
COPY --from=build /usr/lib/x86_64-linux-gnu/libz.so.1 /usr/lib/x86_64-linux-gnu/libz.so.1
COPY --from=build /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=build /etc/ld.so.cache /etc/ld.so.cache
COPY --from=build /usr/local/lib/python3.12 /usr/local/lib/python3.12
COPY --from=build /usr/lib/x86_64-linux-gnu/librt.so.1 /usr/lib/x86_64-linux-gnu/librt.so.1
COPY --from=build /usr/lib/x86_64-linux-gnu/libgcc_s.so.1 /usr/lib/x86_64-linux-gnu/libgcc_s.so.1
COPY --from=build /usr/lib/x86_64-linux-gnu/libssl.so.3 /usr/lib/x86_64-linux-gnu/libssl.so.3
COPY --from=build /usr/lib/x86_64-linux-gnu/libpthread.so.0 /usr/lib/x86_64-linux-gnu/libpthread.so.0
COPY --from=build /usr/lib/x86_64-linux-gnu/libcrypto.so.3 /usr/lib/x86_64-linux-gnu/libcrypto.so.3
COPY --from=build /usr/lib/x86_64-linux-gnu/libffi.so.8 /usr/lib/x86_64-linux-gnu/libffi.so.8
COPY  --from=build /blaxel /blaxel

ENTRYPOINT ["/usr/bin/python3", "-m", "blaxel.src.discord_mcp"]