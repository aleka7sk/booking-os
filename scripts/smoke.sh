#!/usr/bin/env sh
set -eu

BASE_URL="${1:-http://localhost:8080}"
EMAIL="${BOOKING_OS_ADMIN_EMAIL:-owner@booking.local}"
PASSWORD="${BOOKING_OS_ADMIN_PASSWORD:-demo1234}"
COOKIE_FILE="${TMPDIR:-/tmp}/booking-os-smoke-cookie.txt"

cleanup() { rm -f "$COOKIE_FILE"; }
trap cleanup EXIT

curl -fsS "$BASE_URL/api/health" >/dev/null
curl -fsS -c "$COOKIE_FILE" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  "$BASE_URL/api/auth/login" >/dev/null
curl -fsS -b "$COOKIE_FILE" "$BASE_URL/api/dashboard" >/dev/null
curl -fsS "$BASE_URL/api/public/profile" >/dev/null
curl -fsS "$BASE_URL/api/public/offerings" >/dev/null

printf '%s\n' "Booking OS smoke test passed: $BASE_URL"
