# 006 — Kubernetes/EKS MCP for the fleet

## Goal

Add read-only Kubernetes/EKS visibility to the MCP fleet so Claude Code can
inspect paradex-prod (and other EKS clusters) without paying for Lens Pro.

## Context

- Lens has a built-in MCP server but it's gated behind Plus/Pro/Enterprise
  ($25+/user/month). Not worth it for read-only inspection.
- All our clusters are EKS; kubeconfig uses `aws eks get-token` exec-credential
  plugin, so the AWS CLI needs to be available to whatever runs k8s queries.
- `ghcr.io/containers/kubernetes-mcp-server` works but does NOT ship with
  `aws` CLI — docker runtime would need a custom image. Native process
  runtime sidesteps this.

## Options considered

1. **`containers/kubernetes-mcp-server`** (generic k8s)
   - `--read-only` flag available
   - Uses `~/.kube/config`, inherits host's `aws` CLI when run natively
   - Smaller tool surface, pure kubectl semantics

2. **`awslabs/eks-mcp-server`** (EKS-aware)
   - **Read-only by default** — you opt IN to writes via `--allow-write` and
     `--allow-sensitive-data-access`
   - Knows about EKS clusters, node groups, VPC context, upgrades
   - Larger tool surface — needs aggressive `disabled_tools` list

3. **`awslabs/aws-api-mcp-server`** (full AWS API)
   - No explicit read-only flag — rely on IAM scoping via AWS profile
   - Broadest surface; probably too much context unless scoped carefully

## Recommended path

Start with **(2) `awslabs/eks-mcp-server`** native (process runtime):
- Read-only out of the box (just omit `--allow-write`)
- EKS-aware, which is all our clusters
- One server to reason about, not two

Add `(1) containers/kubernetes-mcp-server` later if we find we want pure
kubectl semantics for non-EKS clusters.

## Implementation steps

1. Add `servers/eks/server.yaml`:
   - `protocol: stdio`
   - `runtime: process` (native — uses host `aws` CLI and `~/.kube/config`)
   - `native_cmd`: whatever `uvx` / `pipx run` / `npx` invocation awslabs
     publishes (verify on their repo)
   - `native_args`: start with the read-only default, no `--allow-write`
   - `env`: `AWS_PROFILE` if needed (pull from local config, not committed)
   - `tags: [k8s, aws, monitoring]`

2. Add to `~/.config/textserve/config.yaml` if AWS profile env var is needed.

3. `mcpf register eks` → verify in `/mcp`.

4. After first run, inventory the tool list and populate `disabled_tools:`
   aggressively — we only care about read ops (describe clusters, list pods,
   get logs, top nodes, etc.). Disable everything that touches state even
   if the server is read-only by default (belt + suspenders).

5. Optional phase 2: dockerize with custom image that includes `aws` CLI
   + mounts `~/.aws` read-only, if we want to isolate the subprocess.

## Open questions to resolve during implementation

- Exact install/invocation command for `awslabs/eks-mcp-server` (uvx? pipx?)
- Does it honor `AWS_PROFILE` or does it need `--profile` flag?
- Does multi-context kubeconfig work cleanly, or does it pin to the current
  context at startup?
- Tool count / token footprint — run `/context` before + after to measure.

## Success criteria

- `mcpf status` shows eks as registered
- Claude Code can list pods in paradex-prod and fetch logs from a named pod
- No write operations exposed
- MCP token count in `/context` is within budget (target < 5k for this server)
