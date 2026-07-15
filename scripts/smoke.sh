#!/bin/sh
set -eu
APP_BASE_URL=${APP_BASE_URL:-${BASE_URL:-http://127.0.0.1:8080}}
LANDING_BASE_URL=${LANDING_BASE_URL:-http://127.0.0.1:8081}

curl -fsS "$LANDING_BASE_URL/healthz" | grep -q '"status":"ok"'
landing=$(curl -fsS "$LANDING_BASE_URL/")
printf '%s' "$landing" | grep -q 'href="https://app.chronoflag.com"'
printf '%s' "$landing" | grep -q 'property="og:image" content="https://chronoflag.com/social-card.png"'
curl -fsS "$LANDING_BASE_URL/robots.txt" | grep -q 'Sitemap: https://chronoflag.com/sitemap.xml'
curl -fsS "$LANDING_BASE_URL/sitemap.xml" | grep -q '<loc>https://chronoflag.com/</loc>'
curl -fsSI "$LANDING_BASE_URL/social-card.png" | grep -qi '^Content-Type: image/png'
curl -fsS "$APP_BASE_URL/healthz" | grep -q '"status":"ok"'
created=$(curl -fsS -X POST -H 'content-type: application/json' -d '{}' "$APP_BASE_URL/api/v1/instances")
control=$(printf '%s' "$created" | sed -n 's/.*"control_url":"\/c\/\([^"]*\)".*/\1/p')
view=$(printf '%s' "$created" | sed -n 's/.*"view_url":"\/v\/\([^"]*\)".*/\1/p')
test -n "$control" && test -n "$view" && test "$control" != "$view"
snapshot=$(curl -fsS "$APP_BASE_URL/api/v1/control/$control")
clock=$(printf '%s' "$snapshot" | sed -n 's/.*"id":"\([^"]*\)","type":"stopwatch".*/\1/p')
test -n "$clock"
curl -fsS -X POST -H 'content-type: application/json' -H 'Idempotency-Key: smoke-start' -d '{"type":"start","device_id":"smoke"}' "$APP_BASE_URL/api/v1/control/$control/clocks/$clock/commands" | grep -q '"state":"running"'
curl -fsS "$APP_BASE_URL/api/v1/view/$view" | grep -q '"state":"running"'
curl -fsS "$APP_BASE_URL/api/v1/control/$control/export" | grep -q '"schema_version":1'
printf 'smoke: ok\n'
