#!/usr/bin/env python3
"""
Tokenize text using tiktoken-like cl100k_base approximation.
Falls back to simple whitespace split if tiktoken is unavailable.
"""

import sys


def simple_count(text: str) -> int:
    return len(text.split())


def main() -> None:
    data = sys.stdin.read()
    try:
        import tiktoken

        enc = tiktoken.get_encoding("cl100k_base")
        print(len(enc.encode(data)))
    except Exception:
        print(simple_count(data))


if __name__ == "__main__":
    main()
