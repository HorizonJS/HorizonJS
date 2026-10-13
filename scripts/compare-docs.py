#!/usr/bin/env python3
"""Compare native docs with saved Java output, ignoring HTML serialization whitespace.

Development-only check; the build tool itself does not require Python.
"""
import argparse
import difflib
import hashlib
import html
import json
import re
from html.parser import HTMLParser
from pathlib import Path


class Page(HTMLParser):
    def __init__(self, content):
        super().__init__(convert_charrefs=True)
        self.tokens = []
        self.feed(content)

    def handle_starttag(self, tag, attrs):
        self.tokens.append(["start", tag, sorted(attrs)])

    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)
        if tag not in {"br", "hr", "img", "input", "meta", "link"}:
            self.handle_endtag(tag)

    def handle_endtag(self, tag):
        if tag not in {"br", "hr", "img", "input", "meta", "link"}:
            self.tokens.append(["end", tag])

    def handle_data(self, data):
        text = re.sub(r"\s+", " ", data).strip()
        if text:
            self.tokens.append(["text", text])


def pages(root):
    return {
        path.name: Page(path.read_text(encoding="utf-8")).tokens
        for path in sorted((root / "output").glob("*.html"))
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("reference", type=Path)
    parser.add_argument("generated", type=Path)
    parser.add_argument("--golden", type=Path)
    args = parser.parse_args()
    reference, generated = pages(args.reference), pages(args.generated)
    failures = 0
    for name in sorted(reference.keys() | generated.keys()):
        old, new = reference.get(name), generated.get(name)
        if old != new:
            failures += 1
            print(f"DIFFERENT: {name}")
            old_lines = [json.dumps(token, ensure_ascii=False) for token in (old or [])]
            new_lines = [json.dumps(token, ensure_ascii=False) for token in (new or [])]
            diff = list(difflib.unified_diff(old_lines, new_lines, fromfile="java", tofile="go", n=1))
            print("\n".join(diff[:45]))
    if args.golden:
        hashes = {}
        for path in sorted((args.reference / "output").glob("*.html")):
            text = path.read_text(encoding="utf-8")
            snapshot = {
                "ids": re.findall(r'\bid="([^"]*)"', text),
                "links": re.findall(r'<a\b[^>]*?\bhref="([^"]*)"', text),
                "rows": re.findall(r'<tr class="([^"]*)"', text),
                "text": " ".join(
                    normalized for part in re.split(r"<[^>]*>", text)
                    if (normalized := " ".join(html.unescape(part).split()))
                ),
            }
            hashes[path.name] = hashlib.sha256(json.dumps(snapshot, ensure_ascii=False, separators=(",", ":")).encode()).hexdigest()
        args.golden.write_text(json.dumps(hashes, indent=2) + "\n", encoding="utf-8")
    print(f"Compared {len(reference)} class pages; {failures} differ")
    return bool(failures)


if __name__ == "__main__":
    raise SystemExit(main())
