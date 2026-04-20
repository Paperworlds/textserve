# Dockerfile for snowflake-mcp
# Source: https://github.com/Snowflake-Labs/mcp-server-snowflake
# Built from upstream source — clone the repo, then: docker build -f servers/snowflake/Dockerfile -t snowflake-mcp .
#
# Last built: v1.3.5 (2026-01-xx)
# To rebuild: see servers/snowflake/build.sh

# Copyright 2025 Snowflake Inc.
# SPDX-License-Identifier: Apache-2.0
FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim

WORKDIR /app

ENV UV_COMPILE_BYTECODE=1
ENV UV_LINK_MODE=copy

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=uv.lock \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    uv sync --locked --no-install-project --no-dev

COPY . /app

RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --locked --no-dev

ENV PATH="/app/.venv/bin:$PATH"

CMD ["mcp-server-snowflake", "--service-config-file", "/app/services/tools_config.yaml", "--transport", "streamable-http", "--endpoint", "/snowflake-mcp"]
