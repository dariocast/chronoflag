#!/bin/sh
set -eu
BASE_URL=${BASE_URL:-http://127.0.0.1:8080}
curl -fsS "$BASE_URL/healthz" | grep -q '"status":"ok"'
created=$(curl -fsS -X POST -H 'content-type: application/json' -d '{}' "$BASE_URL/api/v1/instances")
control=$(printf '%s' "$created" | sed -n 's/.*"control_url":"\/c\/\([^"]*\)".*/\1/p')
view=$(printf '%s' "$created" | sed -n 's/.*"view_url":"\/v\/\([^"]*\)".*/\1/p')
test -n "$control" && test -n "$view" && test "$control" != "$view"
snapshot=$(curl -fsS "$BASE_URL/api/v1/control/$control")
clock=$(printf '%s' "$snapshot" | sed -n 's/.*"id":"\([^"]*\)","type":"stopwatch".*/\1/p')
test -n "$clock"
curl -fsS -X POST -H 'content-type: application/json' -H 'Idempotency-Key: smoke-start' -d '{"type":"start","device_id":"smoke"}' "$BASE_URL/api/v1/control/$control/clocks/$clock/commands" | grep -q '"state":"running"'
curl -fsS "$BASE_URL/api/v1/view/$view" | grep -q '"state":"running"'
curl -fsS "$BASE_URL/api/v1/control/$control/export" | grep -q '"schema_version":1'
printf 'smoke: ok\n'
