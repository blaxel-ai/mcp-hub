docker run --rm \
-e BL_SERVER_PORT="80" \
-e REFRESH_TOKEN="asdasd" \
--name gmail \
-p 8080:80 \
ghcr.io/beamlit/mcp-hub/dev/gmail:12dc38fff6ce55eadb06af174f3d10003a2f053c \
"node /blaxel/build/loader.js start gmail"