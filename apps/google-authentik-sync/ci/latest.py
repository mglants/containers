#!/usr/bin/env python3
"""Locally maintained release version; no upstream project to poll."""

from pathlib import Path
import sys


def get_latest(channel):
    if channel != "stable":
        raise ValueError("Only the stable channel is supported")
    return (Path(__file__).resolve().parent.parent / "VERSION").read_text().strip()


if __name__ == "__main__":
    print(get_latest(sys.argv[1]))
