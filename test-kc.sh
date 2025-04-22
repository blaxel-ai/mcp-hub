kraft cloud instance rm test-kc

kraft cloud instance create \
--start \
--name test-kc \
--memory 1024 \
--entrypoint "node /blaxel/build/loader.js start gmail" \
--env BL_CLOUD=true \
--env BL_DEBUG_TELEMETRY=false \
--env BL_ENV=dev \
--env BL_NAME=gmail \
--env BL_SERVER_PORT=80 \
--env LOG_LEVEL=DEBUG \
--env PYTHONHTTPSVERIFY=1 \
--env PYTHONPATH=/blaxel \
--env REFRESH_TOKEN=asdsad \
--env BL_WORKSPACE=main \
--env BL_TYPE=function \
blaxel/mcp-hub/dev/gmail:latest

kraft cloud instance logs test-kc