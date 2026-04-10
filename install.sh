#!/usr/bin/env bash
# install.sh — build mcpf and symlink to ~/.local/bin/mcpf
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${HOME}/.local/bin"
TARGET="${TARGET_DIR}/mcpf"

cd "${REPO_ROOT}"
go build -o bin/mcpf ./cmd/mcpf

mkdir -p "${TARGET_DIR}"
ln -sf "${REPO_ROOT}/bin/mcpf" "${TARGET}"
echo "Installed: mcpf -> ${TARGET}"
