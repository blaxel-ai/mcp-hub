#!/bin/sh
PORT=${BL_SERVER_PORT:-8080}
npx -y @blaxel/supergateway --port $PORT --stdio $@