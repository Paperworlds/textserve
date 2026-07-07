"""MCP server that exposes Docker AI (Gordon) as a tool."""
import subprocess

from mcp.server.fastmcp import FastMCP

mcp = FastMCP("gordon", log_level="ERROR")


@mcp.tool()
def ask(question: str, working_dir: str | None = None) -> str:
    """Ask Docker AI (Gordon) a question about Docker, containers, Compose, or related tooling.

    Args:
        question: The question to ask Gordon.
        working_dir: Optional directory to pass via -C (gives Gordon project context).
    """
    cmd = ["docker", "ai"]
    if working_dir:
        cmd += ["-C", working_dir]
    cmd.append(question)
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    output = result.stdout.strip()
    if result.returncode != 0 and not output:
        output = result.stderr.strip()
    return output or "(no output)"


def main() -> None:
    mcp.run(transport="stdio")
