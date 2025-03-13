### BUILD IMAGE
FROM node:21-alpine AS build

WORKDIR /blaxel

# Copy dependency files first to leverage Docker cache
COPY . .

# Build the application and clean up afterwards
ARG BUILD_PATH=""
WORKDIR /blaxel${BUILD_PATH}

RUN npm install ws @types/ws uuid
RUN npm install

RUN npm run build

### SYSTEM IMAGE

FROM alpine:3 AS sys

RUN set -xe; \
    mkdir -p /target/etc; \
    mkdir -p /blank; \
    apk --no-cache add \
      ca-certificates \
      tzdata \
    ; \
    update-ca-certificates; \
    ln -sf /usr/share/zoneinfo/Etc/UTC /target/etc/localtime; \
    echo "Etc/UTC" > /target/etc/timezone;

### RUNTIME IMAGE
FROM scratch

COPY --from=sys /target/etc /etc
COPY --from=sys /usr/share/zoneinfo/UTC /usr/share/zoneinfo/UTC
COPY --from=sys /usr/share/zoneinfo/Etc/UTC /usr/share/zoneinfo/Etc/UTC
COPY --from=sys /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=sys /blank /tmp

# # Node binary
COPY --from=build /usr/local/bin/node /usr/bin/node

# # System libraries
COPY --from=build /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1
COPY --from=build /usr/lib/libgcc_s.so.1 /usr/lib/libgcc_s.so.1
COPY --from=build /usr/lib/libstdc++.so.6 /usr/lib/libstdc++.so.6
COPY --from=build /etc/os-release /etc/os-release

# Distribution configuration
ARG DIST_PATH="build"
ARG BUILD_PATH=""

COPY --from=build /blaxel/node_modules /blaxel/node_modules
COPY --from=build /blaxel${BUILD_PATH}/node_modules /blaxel/node_modules
COPY --from=build /blaxel${BUILD_PATH}/package.json /blaxel/package.json
COPY --from=build /blaxel/${DIST_PATH} /blaxel/build

ENTRYPOINT ["/usr/bin/node",  "/blaxel/build/loader.js"]