#!/usr/bin/env python3
"""Extract text from a PDF file.

Usage:
    pdf_extract.py <file_path> [--page N]
"""
import argparse
import sys

import pdfplumber


def main():
    parser = argparse.ArgumentParser(description="Extract text from a PDF file")
    parser.add_argument("file_path", help="Path to the PDF file")
    parser.add_argument(
        "--page", type=int, default=None, help="Extract a specific page (1-indexed)"
    )
    args = parser.parse_args()

    try:
        with pdfplumber.open(args.file_path) as pdf:
            if args.page is not None:
                if args.page < 1 or args.page > len(pdf.pages):
                    print(
                        f"Error: page {args.page} out of range (1-{len(pdf.pages)})",
                        file=sys.stderr,
                    )
                    sys.exit(1)
                text = pdf.pages[args.page - 1].extract_text()
                if text:
                    print(text)
            else:
                for page in pdf.pages:
                    text = page.extract_text()
                    if text:
                        print(text)
    except FileNotFoundError:
        print(f"Error: file not found: {args.file_path}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
