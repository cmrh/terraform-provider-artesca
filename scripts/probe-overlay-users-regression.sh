#!/usr/bin/env bash
#
# Probe the management-API overlay-view endpoint for the `users` regression.
#
# Given an ARTESCA cluster and an OIDC user, this script:
#   1. Obtains an OIDC token
#   2. Reads the overlay before creating an account
#   3. Creates a probe account
#   4. Reads the overlay again after creation
#   5. Deletes the probe account
#
# It exits 0 (PASS) if the created account appears in `users` on the second
# read, and 1 (FAIL) if `users` stays null / empty of that account.
#
# Point ~/.scality-artesca.env at an older ARTESCA to compare behavior across
# versions.
#
# Requires:
#   ARTESCA_MANAGEMENT_ENDPOINT, ARTESCA_OIDC_URL, ARTESCA_USERNAME,
#   ARTESCA_PASSWORD in the environment (or in the env file the caller sources).
#
# Optional:
#   ARTESCA_OIDC_REALM   (default: artesca)
#   ARTESCA_CLIENT_ID    (default: zenko-ui)
#   ARTESCA_INSECURE_SKIP_VERIFY (any value → -k passed to curl)

set -euo pipefail

for v in ARTESCA_MANAGEMENT_ENDPOINT ARTESCA_OIDC_URL ARTESCA_USERNAME ARTESCA_PASSWORD; do
  if [ -z "${!v:-}" ]; then
    echo "ERROR: $v not set. Source your env file first." >&2
    exit 2
  fi
done

REALM=${ARTESCA_OIDC_REALM:-artesca}
CLIENT_ID=${ARTESCA_CLIENT_ID:-zenko-ui}
CURL="curl -sS"
if [ -n "${ARTESCA_INSECURE_SKIP_VERIFY:-}" ]; then
  CURL="$CURL -k"
fi

echo "==> Obtaining OIDC token from $ARTESCA_OIDC_URL"
TOKEN=$($CURL \
  -d "grant_type=password" -d "client_id=$CLIENT_ID" \
  -d "username=$ARTESCA_USERNAME" -d "password=$ARTESCA_PASSWORD" \
  "$ARTESCA_OIDC_URL/auth/realms/$REALM/protocol/openid-connect/token" \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")

INSTANCE_ID=$(printf '%s' "$TOKEN" | cut -d. -f2 | python3 -c "
import sys, base64, json
raw = sys.stdin.read().strip(); raw += '=' * (-len(raw) % 4)
ids = json.loads(base64.urlsafe_b64decode(raw)).get('instanceIds', [])
print(ids[0] if ids else '')")

if [ -z "$INSTANCE_ID" ]; then
  echo "ERROR: could not extract instanceIds claim from token" >&2
  exit 2
fi

API="$ARTESCA_MANAGEMENT_ENDPOINT/api/v1"
PROBE_NAME="overlay-probe-$(date +%s)"
AUTH_HEADER="X-Authentication-Token: $TOKEN"

# --- describe_users: print a compact summary of the overlay's users field ---
describe_users() {
  local label=$1
  local body
  body=$($CURL -H "$AUTH_HEADER" "$API/config/overlay/view/$INSTANCE_ID")
  echo "$body" | python3 -c "
import sys, json
label = '$label'
d = json.load(sys.stdin)
u = d.get('users')
if u is None:
    print(f'  {label}: users = null')
elif isinstance(u, list):
    names = [x.get('accountName') or x.get('userName') for x in u]
    print(f'  {label}: users = list len={len(u)} names={names}')
else:
    print(f'  {label}: users = {type(u).__name__} {u!r}')
"
}

echo "==> Before create"
describe_users "before"

echo
echo "==> Create probe account: $PROBE_NAME"
CREATE_STATUS=$($CURL -H "$AUTH_HEADER" -H "Content-Type: application/json" \
  -o /tmp/probe-create.json -w '%{http_code}' \
  -d "{\"userName\":\"$PROBE_NAME\",\"email\":\"$PROBE_NAME@probe.invalid\"}" \
  "$API/config/$INSTANCE_ID/user")
echo "  create status: $CREATE_STATUS"
if [ "$CREATE_STATUS" != "201" ] && [ "$CREATE_STATUS" != "200" ]; then
  echo "  create failed:"
  cat /tmp/probe-create.json
  exit 2
fi

# --- probe: check overlay immediately and after a short wait ---
echo
echo "==> Immediately after create"
describe_users "immediate"

echo
echo "==> After 5s wait"
sleep 5
describe_users "t+5s"

# Determine pass/fail from the t+5s state
FINAL_USERS=$($CURL -H "$AUTH_HEADER" "$API/config/overlay/view/$INSTANCE_ID" \
  | python3 -c "
import sys, json
d = json.load(sys.stdin)
u = d.get('users')
if u is None:
    print('NULL')
elif isinstance(u, list):
    names = [x.get('accountName') or x.get('userName') for x in u]
    print('FOUND' if '$PROBE_NAME' in names else 'MISSING')
else:
    print(f'UNEXPECTED: {type(u).__name__}')")

echo
echo "==> Cleanup: delete probe account"
DEL_STATUS=$($CURL -X DELETE -H "$AUTH_HEADER" -o /dev/null -w '%{http_code}' \
  "$API/config/$INSTANCE_ID/user?accountName=$PROBE_NAME&roleName=scality-internal/storage-manager-role")
echo "  delete status: $DEL_STATUS"

echo
case "$FINAL_USERS" in
  FOUND)
    echo "==> RESULT: PASS — overlay's users array contains the probe account"
    exit 0
    ;;
  NULL|MISSING)
    echo "==> RESULT: FAIL — overlay returned $FINAL_USERS for users field"
    echo "    The account was successfully created (status $CREATE_STATUS) but does not appear"
    echo "    in the /config/overlay/view/{uuid} response. Terraform Read paths that rely"
    echo "    on this endpoint (artesca_account, data.artesca_account, data.artesca_accounts)"
    echo "    will report the account as absent, causing spurious drift and test failures."
    exit 1
    ;;
  *)
    echo "==> RESULT: UNEXPECTED — $FINAL_USERS"
    exit 2
    ;;
esac
