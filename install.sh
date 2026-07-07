#!/usr/bin/env bash
# install.sh — build textserve and symlink to ~/.local/bin/textserve
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${HOME}/.local/bin"
TARGET="${TARGET_DIR}/textserve"

cd "${REPO_ROOT}"
go build -o bin/textserve ./cmd/textserve

mkdir -p "${TARGET_DIR}"
ln -sf "${REPO_ROOT}/bin/textserve" "${TARGET}"
echo "Installed: textserve -> ${TARGET}"
