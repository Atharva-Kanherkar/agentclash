#!/usr/bin/env python3
"""Convert a CSV file to JSON.

Usage:
    csv_to_json.py <file_path> [--orient records|columns]
"""
import argparse
import csv
import json
import sys


def main():
    parser = argparse.ArgumentParser(description="Convert CSV to JSON")
    parser.add_argument("file_path", help="Path to the CSV file")
    parser.add_argument(
        "--orient",
        choices=["records", "columns"],
        default="records",
        help="Output orientation (default: records)",
    )
    args = parser.parse_args()

    try:
        with open(args.file_path, newline="") as f:
            reader = csv.DictReader(f)
            rows = list(reader)
    except FileNotFoundError:
        print(f"Error: file not found: {args.file_path}", file=sys.stderr)
        sys.exit(1)

    if args.orient == "records":
        print(json.dumps(rows, indent=2))
    else:
        if not rows:
            print(json.dumps({}))
        else:
            cols = {k: [r[k] for r in rows] for k in rows[0]}
            print(json.dumps(cols, indent=2))


if __name__ == "__main__":
    main()
