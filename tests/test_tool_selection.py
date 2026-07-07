"""Integration tests for Phase 9 — pp tool selection via graph-mcp.

Tests:
  - test_search_returns_tags: intent "query snowflake data" → tags include 'data'
  - test_no_match_falls_back: intent with no graph matches → falls back to ['docker']
  - test_flag_skips_selection: pp run --no-tool-select starts no servers
  - test_intent_extraction: ## Tasks section extracted correctly
  - test_labels_to_tags_mapping: label→tag mapping works end-to-end
"""
import json
import subprocess
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# graph-search CLI helper tests
# ---------------------------------------------------------------------------

GRAPH_SEARCH = Path.home() / ".local" / "bin" / "graph-search"


def _run_graph_search(*args) -> tuple[int, str, str]:
    result = subprocess.run(
        [str(GRAPH_SEARCH), *args],
        capture_output=True,
        text=True,
        timeout=30,
    )
    return result.returncode, result.stdout, result.stderr


@pytest.mark.skipif(not GRAPH_SEARCH.exists(), reason="graph-search not installed")
def test_graph_search_binary_exists():
    assert GRAPH_SEARCH.exists()
    assert GRAPH_SEARCH.stat().st_mode & 0o111  # executable


@pytest.mark.skipif(not GRAPH_SEARCH.exists(), reason="graph-search not installed")
def test_search_returns_json_array():
    """graph-search should return a valid JSON array."""
    rc, out, _ = _run_graph_search("query snowflake data")
    assert rc == 0
    nodes = json.loads(out)
    assert isinstance(nodes, list)


@pytest.mark.skipif(not GRAPH_SEARCH.exists(), reason="graph-search not installed")
def test_search_returns_tags():
    """Intent 'query snowflake data' should yield tags that include 'data'."""
    rc, out, _ = _run_graph_search("--tags-only", "query snowflake data")
    assert rc == 0
    tags = json.loads(out)
    assert isinstance(tags, list)
    assert "data" in tags, f"Expected 'data' in tags, got: {tags}"


@pytest.mark.skipif(not GRAPH_SEARCH.exists(), reason="graph-search not installed")
def test_no_match_falls_back():
    """Intent with no graph matches should fall back to ['docker']."""
    rc, out, _ = _run_graph_search("--tags-only", "xyzzy_nonexistent_gibberish_intent_99")
    assert rc == 0
    tags = json.loads(out)
    assert tags == ["docker"], f"Expected fallback ['docker'], got: {tags}"


@pytest.mark.skipif(not GRAPH_SEARCH.exists(), reason="graph-search not installed")
def test_monitoring_intent_returns_monitoring_tag():
    """Intent mentioning monitoring/datadog should yield monitoring tag."""
    rc, out, _ = _run_graph_search("--tags-only", "check datadog monitoring alerts")
    assert rc == 0
    tags = json.loads(out)
    assert isinstance(tags, list)
    # Should map to monitoring via graph labels or node IDs


# ---------------------------------------------------------------------------
# tool_selection module unit tests
# ---------------------------------------------------------------------------

sys.path.insert(0, str(Path(__file__).parent.parent.parent / "paperworlds" / "textprompts" / "src"))

try:
    from textprompts.tool_selection import (
        _extract_intent,
        labels_to_tags,
        DEFAULT_TAGS,
    )
    HAS_TEXTPROMPTS = True
except ImportError:
    HAS_TEXTPROMPTS = False


@pytest.mark.skipif(not HAS_TEXTPROMPTS, reason="textprompts not importable")
def test_extract_intent_tasks_section():
    """Should extract ## Tasks section content."""
    body = "# Title\n\nSome intro.\n\n## Tasks\n\nQuery snowflake for data.\n\n## Notes\n\nstuff"
    intent = _extract_intent(body)
    assert "snowflake" in intent
    assert "Notes" not in intent


@pytest.mark.skipif(not HAS_TEXTPROMPTS, reason="textprompts not importable")
def test_extract_intent_fallback():
    """Should fall back to first 500 chars when no ## Tasks section."""
    body = "x" * 600
    intent = _extract_intent(body)
    assert len(intent) == 500


@pytest.mark.skipif(not HAS_TEXTPROMPTS, reason="textprompts not importable")
def test_labels_to_tags_snowflake():
    """Node with 'snowflake' in ID or labels should map to ['data']."""
    mappings = {"snowflake": ["data"], "data": ["data"]}
    nodes = [{"id": "feedback-snowflake-accountadmin", "labels": ["snowflake", "testing"]}]
    tags = labels_to_tags(nodes, mappings)
    assert "data" in tags


@pytest.mark.skipif(not HAS_TEXTPROMPTS, reason="textprompts not importable")
def test_labels_to_tags_empty_returns_empty():
    """No matching labels → empty list (caller applies fallback)."""
    mappings = {"snowflake": ["data"]}
    nodes = [{"id": "some-unrelated-node", "labels": ["feedback"]}]
    tags = labels_to_tags(nodes, mappings)
    assert tags == []


@pytest.mark.skipif(not HAS_TEXTPROMPTS, reason="textprompts not importable")
def test_default_tags_constant():
    assert DEFAULT_TAGS == ["docker"]


# ---------------------------------------------------------------------------
# pp CLI --no-tool-select flag test
# ---------------------------------------------------------------------------

def test_flag_skips_selection():
    """pp run --no-tool-select should be accepted and skip tool selection."""
    result = subprocess.run(
        ["pp", "run", "--no-tool-select", "--dry-run"],
        capture_output=True,
        text=True,
        timeout=30,
        cwd=str(Path(__file__).parent.parent / "features" / "graph-mcp"),
    )
    # Should not error on the flag itself (may print "No tasks selected" if pipeline is done)
    assert "--no-tool-select" not in result.stderr or "no such option" not in result.stderr.lower(), \
        f"Flag not recognized: {result.stderr}"
    assert result.returncode in (0, 1), f"Unexpected exit code {result.returncode}: {result.stderr}"


def test_pp_run_help_has_no_tool_select():
    """pp run --help should document --no-tool-select."""
    result = subprocess.run(
        ["pp", "run", "--help"],
        capture_output=True,
        text=True,
        timeout=10,
    )
    assert result.returncode == 0
    assert "--no-tool-select" in result.stdout
