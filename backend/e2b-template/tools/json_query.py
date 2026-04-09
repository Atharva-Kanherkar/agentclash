#!/usr/bin/env python3
"""Query a JSON file with a dotted path expression.

Usage:
    json_query.py <file_path> <path>

Examples:
    json_query.py data.json 'name'
    json_query.py data.json 'results[0].title'
    json_query.py data.json 'users[2].address.city'
"""
import argparse
import json
import re
import sys


def resolve(data, path):
    """Walk a dotted path with optional array indices."""
    for part in re.split(r"\.|\[(\d+)\]", path):
        if part is None or part == "":
            continue
        if part.isdigit():
            data = data[int(part)]
        else:
            data = data[part]
    return data


def main():
    parser = argparse.ArgumentParser(description="Query JSON by dotted path")
    parser.add_argument("file_path", help="Path to the JSON file")
    parser.add_argument("path", help="Dotted path expression (e.g., 'results[0].name')")
    args = parser.parse_args()

    try:
        with open(args.file_path) as f:
            data = json.load(f)
    except FileNotFoundError:
        print(f"Error: file not found: {args.file_path}", file=sys.stderr)
        sys.exit(1)

    try:
        result = resolve(data, args.path)
    except (KeyError, IndexError, TypeError) as exc:
        print(f"Error: path resolution failed: {exc}", file=sys.stderr)
        sys.exit(1)

    if isinstance(result, (dict, list)):
        print(json.dumps(result, indent=2))
    else:
        print(result)


if __name__ == "__main__":
    main()
