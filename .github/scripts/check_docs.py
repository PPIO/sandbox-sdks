#!/usr/bin/env python3
"""Check customer-facing local links, native version examples and license copies."""
from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlsplit
import xml.etree.ElementTree as ET


def anchors(text):
    text = re.sub(r"```.*?```", "", text, flags=re.S)
    found = set(re.findall(r'(?:id|name)=[\"\']([^\"\']+)[\"\']', text))
    counts = {}
    for heading in re.findall(r"^#{1,6}\s+(.+?)\s*#*\s*$", text, flags=re.M):
        heading = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", heading)
        slug = re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-")
        count = counts.get(slug, 0)
        counts[slug] = count + 1
        found.add(slug + (f"-{count}" if count else ""))
    return found


def check(root):
    errors = []
    docs = sorted(set(root.glob("*.md")) | set(root.glob("*/README.md"))
                  | set(root.glob("sdk-java/docs/*.md")) | set(root.glob("docs/*.md")))
    for doc in docs:
        text = re.sub(r"```.*?```", "", doc.read_text(), flags=re.S)
        for match in re.finditer(r'\[[^\]\n]*\]\(([^\s)]+)(?:\s+"[^"\n]*")?\)', text):
            link = urlsplit(match[1].strip("<>"))
            if link.scheme or link.netloc:
                continue
            base = root if link.path.startswith("/") else doc.parent
            target = base / unquote(link.path.lstrip("/")) if link.path else doc
            if not target.exists():
                errors.append(f"{doc.relative_to(root)}: missing link target {match[1]}")
            elif link.fragment and target.suffix == ".md":
                if unquote(link.fragment) not in anchors(target.read_text()):
                    errors.append(f"{doc.relative_to(root)}: missing Markdown anchor {match[1]}")

    java = root / "sdk-java"
    ns = {"m": "http://maven.apache.org/POM/4.0.0"}
    pom = ET.parse(java / "pom.xml").getroot()
    version = pom.findtext("m:version", namespaces=ns)
    group = pom.findtext("m:groupId", namespaces=ns)
    artifact = pom.findtext("m:artifactId", namespaces=ns)
    readme = (java / "README.md").read_text()
    examples = re.findall(r"<version>([^<]+)</version>", readme)
    examples += re.findall(re.escape(f"{group}:{artifact}:") + r"([^'\"\s]+)", readme)
    if not examples or any(example != version for example in examples):
        errors.append("sdk-java/README.md: dependency examples differ from pom.xml")

    go = root / "sdk-go"
    version_match = re.search(r'SDKVersion\s*=\s*"([^"]+)"', (go / "core/connection_config.go").read_text())
    module = re.search(r"^module\s+(\S+)", (go / "go.mod").read_text(), re.M).group(1)
    go_readme = (go / "README.md").read_text()
    if (not version_match or not re.search(r"(?m)^go get " + re.escape(module) + r"$", go_readme)
            or re.search(re.escape(module) + r"@v[0-9]", go_readme)):
        errors.append("sdk-go/README.md: expected versionless installation command for the declared module")

    license_text = (root / "LICENSE").read_bytes()
    for path in [java / "LICENSE", go / "LICENSE",
                 java / "src/main/resources/META-INF/LICENSE",
                 java / "src/main/javadoc/doc-files/LICENSE"]:
        if not path.is_file() or path.read_bytes() != license_text:
            errors.append(f"{path.relative_to(root)}: license differs from root LICENSE")
    return len(docs), errors


def main():
    root = Path(__file__).resolve().parents[2]
    count, errors = check(root)
    for error in errors:
        print(error)
    print(f"Checked {count} customer documents, native versions and license copies; {len(errors)} findings")
    return bool(errors)


if __name__ == "__main__":
    sys.exit(main())
