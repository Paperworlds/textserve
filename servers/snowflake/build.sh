#!/usr/bin/env bash
# Build the snowflake-mcp Docker image from upstream source.
# Usage: ./servers/snowflake/build.sh [version]
#
# Clones Snowflake-Labs/mcp-server-snowflake, checks out the given tag (default: latest),
# builds the image, and tags it as snowflake-mcp:latest.

set -euo pipefail

VERSION="${1:-}"
REPO="https://github.com/Snowflake-Labs/mcp-server-snowflake.git"
WORKDIR="$(mktemp -d)"

echo "Cloning $REPO into $WORKDIR..."
git clone --depth=1 ${VERSION:+--branch "$VERSION"} "$REPO" "$WORKDIR"

echo "Building snowflake-mcp Docker image..."
docker build -f "$WORKDIR/docker/server/Dockerfile" -t snowflake-mcp:latest "$WORKDIR"

echo "Done. Image tagged as snowflake-mcp:latest"
rm -rf "$WORKDIR"
