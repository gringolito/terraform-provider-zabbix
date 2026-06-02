#!/usr/bin/env bash
# Waits for the Zabbix API to be ready, then mints an API token for acceptance tests.
# Writes "export ZABBIX_URL=..." and "export ZABBIX_TOKEN=..." to stdout.
set -euo pipefail

ZABBIX_URL="${ZABBIX_URL:-http://localhost:8080}"
API_URL="${ZABBIX_URL}/api_jsonrpc.php"
MAX_WAIT="${MAX_WAIT:-120}"

api_call() {
  # Reads a JSON payload from stdin and posts it to the Zabbix API.
  curl -sf -m 10 -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -d @-
}

echo "Waiting for Zabbix API at ${API_URL} ..." >&2
deadline=$((SECONDS + MAX_WAIT))
while true; do
  if jq -n '{"jsonrpc":"2.0","method":"apiinfo.version","params":[],"id":1}' \
      | api_call \
      | jq -e '.result' > /dev/null 2>&1; then
    echo "Zabbix API is ready." >&2
    break
  fi
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "ERROR: Timed out waiting for Zabbix API after ${MAX_WAIT}s" >&2
    exit 1
  fi
  sleep 2
done

echo "Bootstrapping API token ..." >&2

session=$(
  jq -n '{"jsonrpc":"2.0","method":"user.login","params":{"username":"Admin","password":"zabbix"},"id":1}' \
  | api_call \
  | jq -r '.result'
)

if [ -z "$session" ] || [ "$session" = "null" ]; then
  echo "ERROR: user.login failed — could not get session token" >&2
  exit 1
fi

token_id=$(
  jq -n --arg auth "$session" \
    '{"jsonrpc":"2.0","method":"token.create","params":{"name":"testacc","userid":"1"},"auth":$auth,"id":2}' \
  | api_call \
  | jq -r '.result.tokenids[0]'
)

if [ -z "$token_id" ] || [ "$token_id" = "null" ]; then
  echo "ERROR: token.create failed" >&2
  exit 1
fi

token=$(
  jq -n --arg auth "$session" --arg id "$token_id" \
    '{"jsonrpc":"2.0","method":"token.generate","params":[$id],"auth":$auth,"id":3}' \
  | api_call \
  | jq -r '.result[0].token'
)

if [ -z "$token" ] || [ "$token" = "null" ]; then
  echo "ERROR: token.generate failed" >&2
  exit 1
fi

echo "API token minted." >&2

printf 'export ZABBIX_URL=%s\n' "$ZABBIX_URL"
printf 'export ZABBIX_TOKEN=%s\n' "$token"
