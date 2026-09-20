#!/bin/sh
# nexit/1 sh bootstrap
#
# NanoKVM exit bootstrap for macOS and Linux (D14). Picks python3 or perl, fetches
# that client from the NanoKVM with the token in the Authorization header, checks
# the body, and runs it. Served fully templated from GET /exit/<slot>/client.sh:
#
#   curl -fsSL[k] -H 'Authorization: Bearer <token>' <base>/client.sh | sh
#
# It embeds no protocol code, only its parameters; the reconnect loop lives in the
# fetched client. __SCHEME__ here is the fetch scheme (http or https).
set -u

SCHEME="__SCHEME__"
HOST="__HOST__"
SLOT="__SLOT__"
TOKEN="__TOKEN__"
VERIFY="__VERIFY__"

fail() { printf 'nexit: %s\n' "$*" >&2; exit 1; }

case "$HOST" in
  __*) fail "this script must be fetched from the NanoKVM, which fills in its parameters" ;;
esac
case "$SCHEME" in
  http|https) ;;
  ws) SCHEME=http ;;
  wss) SCHEME=https ;;
  *) fail "bad scheme '$SCHEME'" ;;
esac
command -v curl >/dev/null 2>&1 || fail "curl is required"
BASE="$SCHEME://$HOST/exit/$SLOT"

# --- pick the runtime ------------------------------------------------------------

python_ok() {
  command -v python3 >/dev/null 2>&1 && python3 -c 'import ssl, socket, selectors' >/dev/null 2>&1
}

pick=
case "$(uname -s 2>/dev/null)" in
  Darwin)
    # Without the Command Line Tools, Apple's python3 shim pops an install dialog.
    if xcode-select -p >/dev/null 2>&1 && python_ok; then pick=py; fi ;;
  *)
    if python_ok; then pick=py; fi ;;
esac
if [ -z "$pick" ]; then
  command -v perl >/dev/null 2>&1 || fail "neither a usable python3 (with ssl, socket, selectors) nor perl was found"
  pick=pl
fi

# --- fetch the client -------------------------------------------------------------

url="$BASE/client.$pick"
body=$(curl -fsS --max-time 30 -H "Authorization: Bearer $TOKEN" "$url")
rc=$?
if [ $rc -ne 0 ] && [ "$SCHEME" = https ] && [ "$VERIFY" != 1 ]; then
  case $rc in
    35|51|58|59|60|77|83|90|91)
      # The NanoKVM normally serves a self-signed certificate, which no CA store
      # can vouch for. Refetch without verification, the way wstunnel reaches the
      # same host; the token in the header is what authenticates the request.
      body=$(curl -fsSk --max-time 30 -H "Authorization: Bearer $TOKEN" "$url")
      rc=$?
      ;;
  esac
fi
[ $rc -eq 0 ] || fail "fetching $url failed (curl exit $rc); a 404 means wrong token or slot, or this source is rate limited"
[ -n "$body" ] || fail "empty response from $url"
if [ "$pick" = py ]; then marker='nexit/1 python client'; else marker='nexit/1 perl client'; fi
printf '%s\n' "$body" | head -n 3 | grep -q "$marker" || fail "unexpected response from $url (not the $marker)"

# --- run it -----------------------------------------------------------------------

if [ "$pick" = py ]; then
  exec python3 -c "$body"
else
  exec perl -e "$body"
fi
