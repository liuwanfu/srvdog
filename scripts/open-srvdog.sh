#!/usr/bin/env bash
set -euo pipefail

HOST="107.174.48.241"
PORT="45678"
USER_NAME="root"
LOCAL_HOST="127.0.0.1"
LOCAL_PORT="8090"
REMOTE_HOST="127.0.0.1"
REMOTE_PORT="8090"
URL="http://127.0.0.1:8090"
IDENTITY_FILE="${SRVDOG_IDENTITY_FILE:-}"

if ! command -v ssh >/dev/null 2>&1; then
  echo "ssh was not found in PATH." >&2
  exit 1
fi

if ! command -v open >/dev/null 2>&1; then
  echo "open was not found. This launcher is intended for macOS." >&2
  exit 1
fi

if lsof -nP -iTCP:"${LOCAL_PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Local port ${LOCAL_PORT} is already in use. Close the conflicting process or change the launcher config." >&2
  exit 1
fi

SSH_ARGS=(
  -N
  -o IdentitiesOnly=yes
  -L "${LOCAL_HOST}:${LOCAL_PORT}:${REMOTE_HOST}:${REMOTE_PORT}"
  -p "${PORT}"
  "${USER_NAME}@${HOST}"
)

if [[ -n "${IDENTITY_FILE}" ]]; then
  SSH_ARGS=(-i "${IDENTITY_FILE}" "${SSH_ARGS[@]}")
fi

echo "Opening srvdog tunnel on ${URL}"
echo "Target: ${USER_NAME}@${HOST}:${PORT} -> ${REMOTE_HOST}:${REMOTE_PORT}"
echo "Close this terminal to stop the tunnel."

(
  sleep 2
  if ! open "${URL}" >/dev/null 2>&1; then
    echo "Failed to open browser automatically. Open ${URL} manually."
  fi
) &

exec ssh "${SSH_ARGS[@]}"
