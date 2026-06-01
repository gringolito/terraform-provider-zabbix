#!/usr/bin/env bash
# Waits for the Zabbix API to be ready, then mints an API token for acceptance tests.
# Writes "export ZABBIX_URL=..." and "export ZABBIX_TOKEN=..." to stdout.
set -euo pipefail

ZABBIX_URL="${ZABBIX_URL:-http://localhost:8080}"
API_URL="${ZABBIX_URL}/api_jsonrpc.php"
MAX_WAIT="${MAX_WAIT:-120}"

api_call() {
  curl -sf -m 10 -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -d "$1"
}

echo "Waiting for Zabbix API at ${API_URL} ..." >&2
deadline=$((SECONDS + MAX_WAIT))
while true; do
  if api_call '{"jsonrpc":"2.0","method":"apiinfo.version","params":[],"id":1}' \
      | grep -q '"result"'; then
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

session=$(api_call '{"jsonrpc":"2.0","method":"user.login","params":{"username":"Admin","password":"zabbix"},"id":1}' \
  | grep -o '"result":"[^"]*"' | cut -d'"' -f4)

if [ -z "$session" ]; then
  echo "ERROR: user.login failed — could not get session token" >&2
  exit 1
fi

token_id=$(api_call "{\"jsonrpc\":\"2.0\",\"method\":\"token.create\",\"params\":{\"name\":\"testacc\",\"userid\":\"1\"},\"auth\":\"${session}\",\"id\":2}" \
  | grep -o '"tokenid":"[^"]*"' | cut -d'"' -f4)

if [ -z "$token_id" ]; then
  echo "ERROR: token.create failed" >&2
  exit 1
fi

token=$(api_call "{\"jsonrpc\":\"2.0\",\"method\":\"token.generate\",\"params\":[\"${token_id}\"],\"auth\":\"${session}\",\"id\":3}" \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$token" ]; then
  echo "ERROR: token.generate failed" >&2
  exit 1
fi

echo "API token minted." >&2

printf 'export ZABBIX_URL=%s\n' "$ZABBIX_URL"
printf 'export ZABBIX_TOKEN=%s\n' "$token"
