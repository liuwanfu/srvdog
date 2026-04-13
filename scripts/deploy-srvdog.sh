#!/usr/bin/env bash
set -euo pipefail

HOST="${SRVDOG_DEPLOY_HOST:-107.174.48.241}"
PORT="${SRVDOG_DEPLOY_PORT:-45678}"
USER_NAME="${SRVDOG_DEPLOY_USER:-root}"
IDENTITY_FILE="${SRVDOG_IDENTITY_FILE:-}"
REMOTE_WORKDIR="${SRVDOG_REMOTE_WORKDIR:-/root/tmp/srvdog2}"
REMOTE_BINARY_PATH="${SRVDOG_REMOTE_BINARY_PATH:-/opt/srvdog/srvdog}"
REMOTE_SERVICE_NAME="${SRVDOG_REMOTE_SERVICE_NAME:-srvdog}"
GO_IMAGE="${SRVDOG_GO_IMAGE:-golang:1.25}"
VERIFY_ATTEMPTS="${SRVDOG_VERIFY_ATTEMPTS:-10}"
VERIFY_DELAY_SECONDS="${SRVDOG_VERIFY_DELAY_SECONDS:-1}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if ! command -v ssh >/dev/null 2>&1; then
  echo "ssh was not found in PATH." >&2
  exit 1
fi

if ! command -v rsync >/dev/null 2>&1; then
  echo "rsync was not found in PATH." >&2
  exit 1
fi

SSH_ARGS=(
  -p "${PORT}"
  -o IdentitiesOnly=yes
)

if [[ -n "${IDENTITY_FILE}" ]]; then
  SSH_ARGS=(-i "${IDENTITY_FILE}" "${SSH_ARGS[@]}")
fi

REMOTE_TARGET="${USER_NAME}@${HOST}"

echo "Syncing repository to ${REMOTE_TARGET}:${REMOTE_WORKDIR}"
rsync -az --delete \
  --exclude ".git" \
  --exclude ".superpowers" \
  --exclude ".DS_Store" \
  --exclude "srvdog-linux-amd64" \
  -e "ssh ${SSH_ARGS[*]}" \
  "${REPO_ROOT}/" "${REMOTE_TARGET}:${REMOTE_WORKDIR}/"

echo "Running remote test, build, install, and restart"
ssh "${SSH_ARGS[@]}" "${REMOTE_TARGET}" \
  "REMOTE_WORKDIR='${REMOTE_WORKDIR}' REMOTE_BINARY_PATH='${REMOTE_BINARY_PATH}' REMOTE_SERVICE_NAME='${REMOTE_SERVICE_NAME}' GO_IMAGE='${GO_IMAGE}' VERIFY_ATTEMPTS='${VERIFY_ATTEMPTS}' VERIFY_DELAY_SECONDS='${VERIFY_DELAY_SECONDS}' bash -s" <<'REMOTE'
set -euo pipefail

wait_for_http_endpoint() {
  local url="$1"
  local attempts="${2:-10}"
  local delay_seconds="${3:-1}"
  local attempt output

  for ((attempt = 1; attempt <= attempts; attempt += 1)); do
    if output="$(curl -fsS "${url}" 2>&1 >/dev/null)"; then
      return 0
    fi
    if (( attempt == attempts )); then
      echo "endpoint did not become ready: ${url}" >&2
      echo "${output}" >&2
      return 1
    fi
    echo "endpoint not ready yet (${attempt}/${attempts}): ${url}" >&2
    sleep "${delay_seconds}"
  done
}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker was not found on the remote host." >&2
  exit 1
fi

cd "${REMOTE_WORKDIR}"

docker run --rm \
  -v "${PWD}:/src" \
  -w /src \
  "${GO_IMAGE}" \
  go test ./...

docker run --rm \
  -e CGO_ENABLED=0 \
  -e GOOS=linux \
  -e GOARCH=amd64 \
  -v "${PWD}:/src" \
  -w /src \
  "${GO_IMAGE}" \
  go build -buildvcs=false -o srvdog-linux-amd64 ./cmd/srvdog

timestamp="$(date +%Y%m%d-%H%M%S)"
backup_path="${REMOTE_BINARY_PATH}.bak-${timestamp}"

if [[ -f "${REMOTE_BINARY_PATH}" ]]; then
  cp "${REMOTE_BINARY_PATH}" "${backup_path}"
  echo "Backed up existing binary to ${backup_path}"
fi

install -m 755 srvdog-linux-amd64 "${REMOTE_BINARY_PATH}"
systemctl restart "${REMOTE_SERVICE_NAME}"
systemctl is-active "${REMOTE_SERVICE_NAME}"

echo "Verifying local-only endpoints"
wait_for_http_endpoint "http://127.0.0.1:8090/" "${VERIFY_ATTEMPTS}" "${VERIFY_DELAY_SECONDS}"
wait_for_http_endpoint "http://127.0.0.1:8090/api/clash/status" "${VERIFY_ATTEMPTS}" "${VERIFY_DELAY_SECONDS}"

echo "Deploy complete"
echo "Service: ${REMOTE_SERVICE_NAME}"
echo "Binary: ${REMOTE_BINARY_PATH}"
REMOTE
