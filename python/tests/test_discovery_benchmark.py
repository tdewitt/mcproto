"""Tests for discovery benchmark helpers."""

import json
import os
import sys
import unittest

SCRIPT_DIR = os.path.join(os.getcwd(), "python", "scripts")
sys.path.insert(0, SCRIPT_DIR)

import discovery_benchmark


class _FakeEncoder:
    """Simple encoder stub for token counting."""

    def encode(self, text):
        """Return a naive token list."""
        return text.split()


class _FakeTool:
    """Minimal tool stub for payload construction."""

    def __init__(self, name, description, bsr_ref):
        """Initialize the stub tool."""
        self.name = name
        self.description = description
        self.bsr_ref = bsr_ref


class DiscoveryBenchmarkTest(unittest.TestCase):
    """Validates benchmark payload helpers."""

    def test_build_proto_listing_payload(self):
        """Includes name, description, and bsr_ref fields."""
        tools = [_FakeTool("tool_one", "desc", "buf.build/acme/a.Msg:main")]

        payload = discovery_benchmark.build_proto_listing_payload(tools)
        data = json.loads(payload)

        self.assertEqual(data["tools"][0]["name"], "tool_one")
        self.assertEqual(data["tools"][0]["description"], "desc")
        self.assertEqual(
            data["tools"][0]["bsr_ref"],
            "buf.build/acme/a.Msg:main",
        )

    def test_build_legacy_listing_payload(self):
        """Includes inline inputSchema content per tool."""
        tools = [_FakeTool("tool_one", "desc", "buf.build/acme/a.Msg:main")]
        schema_map = {"buf.build/acme/a.Msg:main": {"type": "object"}}

        payload = discovery_benchmark.build_legacy_listing_payload(
            tools, schema_map
        )
        data = json.loads(payload)

        self.assertEqual(data["tools"][0]["inputSchema"], {"type": "object"})

    def test_token_count_uses_encoder(self):
        """Uses the provided encoder when counting tokens."""
        encoder = _FakeEncoder()
        tokens = discovery_benchmark.token_count("a b c", encoder)

        self.assertEqual(tokens, 3)


if __name__ == "__main__":
    unittest.main()
