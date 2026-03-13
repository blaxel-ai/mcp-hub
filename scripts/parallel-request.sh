#!/bin/sh
set -e

URL="${1:?Usage: $0 <mcp-url>}"
PARALLEL=10

# Initialize session and capture session ID
echo "Initializing MCP session..."
INIT_RESPONSE=$(curl-bl -s -D - "$URL" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "capabilities": {},
      "clientInfo": { "name": "bench", "version": "0.1.0" }
    }
  }')

SESSION_ID=$(echo "$INIT_RESPONSE" | grep -i "^mcp-session-id:" | tr -d '\r' | awk '{print $2}')

if [ -z "$SESSION_ID" ]; then
  echo "Failed to get session ID. Response:"
  echo "$INIT_RESPONSE"
  exit 1
fi

echo "Session ID: $SESSION_ID"
echo "Sending $PARALLEL parallel tool calls..."

start_time=$(date +%s)

for i in $(seq 1 $PARALLEL); do
  (
    RESPONSE=$(curl-bl -s -w "\n---HTTP_CODE:%{http_code} TIME:%{time_total}s---" "$URL" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json, text/event-stream" \
      -H "Mcp-Session-Id: $SESSION_ID" \
      -d "{
        \"jsonrpc\": \"2.0\",
        \"id\": $i,
        \"method\": \"tools/call\",
        \"params\": {
          \"name\": \"web_search_exa\",
          \"arguments\": {
            \"query\": \"hello\",
            \"numResults\": 1
          }
        }
      }")
    echo "[$i] $RESPONSE"
  ) &
done

wait

end_time=$(date +%s)
elapsed=$((end_time - start_time))
echo ""
echo "All $PARALLEL requests completed in ${elapsed}s"
