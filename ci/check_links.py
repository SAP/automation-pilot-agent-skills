#!/usr/bin/env python3
"""Check local Markdown links, heading anchors, and HTML images in public guidance."""
import argparse
import html
import re
import unicodedata
from pathlib import Path
from urllib.parse import unquote, urlsplit


def prose(text):
    """Exclude fenced examples while preserving headings and inline formatting."""
    lines = []
    fence = None
    for line in text.splitlines():
        match = re.match(r"^ {0,3}(`{3,}|~{3,})", line)
        if match:
            marker = match.group(1)
            if fence is None:
                fence = marker
            elif marker[0] == fence[0] and len(marker) >= len(fence):
                fence = None
            continue
        if fence is None:
            lines.append(line)
    return "\n".join(lines)


def anchors(text):
    used = set()
    for match in re.finditer(r"^ {0,3}#{1,6}\s+(.+?)\s*#*\s*$", prose(text), re.M):
        heading = re.sub(r"<[^>]*>", "", match.group(1))
        heading = re.sub(r"\[([^]]+)\]\([^)]*\)", r"\1", heading)
        heading = html.unescape(heading).lower()
        slug = "".join(c for c in heading if c in "_- " or unicodedata.category(c)[0] not in "PS").replace(" ", "-")
        anchor, suffix = slug, 0
        while anchor in used:
            suffix += 1
            anchor = f"{slug}-{suffix}"
        used.add(anchor)
    return used


def check(root):
    sources = [root / name for name in ("README.md", "CONTRIBUTING.md", "ci/README.md", "ci/evals/skills-e2e/README.md")]
    sources += sorted((root / "skills").rglob("*.md"))
    errors, count, cached_anchors = [], 0, {}
    for source in sources:
        if not source.is_file():
            errors.append(f"{source.relative_to(root)}: missing guidance file")
            continue
        text = re.sub(r"(`+).*?\1", "", prose(source.read_text(encoding="utf-8")))
        targets = re.findall(r'!?\[[^]\n]*\]\(\s*(<[^>]+>|[^\s)]+)(?:\s+"[^"]*")?\s*\)', text)
        targets += re.findall(r'<img\b[^>]*\bsrc="([^"]+)"', text)
        for target in targets:
            target = html.unescape(target.strip("<>"))
            url = urlsplit(target)
            if url.scheme or url.netloc:
                continue
            count += 1
            destination = (source.parent / unquote(url.path)).resolve() if url.path else source
            if not destination.exists():
                errors.append(f"{source.relative_to(root)}: missing target {target}")
            elif url.fragment and destination.suffix == ".md":
                if destination not in cached_anchors:
                    cached_anchors[destination] = anchors(destination.read_text(encoding="utf-8"))
                if unquote(url.fragment) not in cached_anchors[destination]:
                    errors.append(f"{source.relative_to(root)}: missing anchor {target}")
    return count, errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    count, errors = check(args.root.resolve())
    if errors:
        parser.exit(1, "\n".join(errors) + "\n")
    print(f"Verified {count} local links and anchors in README, CONTRIBUTING, skills, and CI guidance")


if __name__ == "__main__":
    main()
