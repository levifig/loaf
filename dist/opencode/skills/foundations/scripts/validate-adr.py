#!/usr/bin/env python3
"""Validate the maintained ADR Markdown/frontmatter contract.

Usage: validate-adr.py <adr-file.md> [<adr-file.md> ...]
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


VALID_STATUSES = {"Proposed", "Accepted", "Rejected", "Deprecated", "Superseded"}
DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
ID_RE = re.compile(r"^ADR-\d{3}$")
FILENAME_RE = re.compile(r"^(ADR-\d{3})-[a-z0-9][a-z0-9-]*\.md$")
REQUIRED_SECTIONS = ("Context", "Decision", "Consequences", "Alternatives Considered")


def frontmatter(content: str) -> tuple[dict[str, str], list[str]]:
    errors: list[str] = []
    if not content.startswith("---\n"):
        return {}, ["missing YAML frontmatter"]
    end = content.find("\n---\n", 4)
    if end == -1:
        return {}, ["unterminated YAML frontmatter"]

    values: dict[str, str] = {}
    sequence_key: str | None = None
    for line in content[4:end].splitlines():
        if not line or line.startswith("#"):
            continue
        match = re.match(r"^([a-z_]+):\s*(.*?)\s*(?:#.*)?$", line)
        if match:
            key, value = match.groups()
            values[key] = value.strip().strip('"\'')
            sequence_key = key if not values[key] else None
            continue
        item = re.match(r"^\s+-\s+(.*?)\s*(?:#.*)?$", line)
        if item and sequence_key:
            value = item.group(1).strip().strip('"\'')
            values[sequence_key] = ", ".join(filter(None, (values[sequence_key], value)))
            continue
        if line.startswith((" ", "\t")):
            continue
        errors.append(f"unsupported frontmatter line: {line}")
    return values, errors


def has_section(content: str, heading: str) -> bool:
    return re.search(rf"^##\s+{re.escape(heading)}\s*$", content, re.MULTILINE) is not None


def validate_adr(filepath: Path) -> list[str]:
    errors: list[str] = []
    content = filepath.read_text(encoding="utf-8")
    values, frontmatter_errors = frontmatter(content)
    errors.extend(frontmatter_errors)

    filename_match = FILENAME_RE.match(filepath.name)
    if not filename_match:
        errors.append("filename must match ADR-NNN-lowercase-slug.md")

    record_id = values.get("id", "")
    if not ID_RE.match(record_id):
        errors.append("frontmatter id must match ADR-NNN")
    elif filename_match and filename_match.group(1) != record_id:
        errors.append("frontmatter id must match filename id")

    title = values.get("title", "")
    if not title:
        errors.append("frontmatter title is required")
    expected_heading = f"# {record_id}: {title}" if record_id and title else ""
    if expected_heading and expected_heading not in content:
        errors.append("H1 must match frontmatter id and title")

    status = values.get("status", "")
    if status not in VALID_STATUSES:
        errors.append(f"status must be one of: {', '.join(sorted(VALID_STATUSES))}")

    for field in ("date", "revised", "accepted_date", "rejected_date", "deprecated_date"):
        value = values.get(field)
        if value and not DATE_RE.match(value):
            errors.append(f"{field} must use YYYY-MM-DD")
    if "date" not in values:
        errors.append("frontmatter date is required")

    for section in REQUIRED_SECTIONS:
        if not has_section(content, section):
            errors.append(f"missing section: ## {section}")

    if status == "Rejected":
        if not values.get("rejected_date"):
            errors.append("Rejected status requires rejected_date")
        if not has_section(content, "Rejected"):
            errors.append("Rejected status requires ## Rejected")
    if status == "Deprecated":
        if not values.get("deprecated_date"):
            errors.append("Deprecated status requires deprecated_date")
        if not has_section(content, "Deprecated"):
            errors.append("Deprecated status requires ## Deprecated")
    if status == "Superseded":
        replacement = values.get("superseded_by", "")
        if not replacement or replacement == "null" or "ADR-" not in replacement:
            errors.append("Superseded status requires superseded_by ADR reference(s)")
        if not has_section(content, "Superseded"):
            errors.append("Superseded status requires ## Superseded")
    if values.get("revised") and not has_section(content, "Revisions"):
        errors.append("revised frontmatter requires ## Revisions")

    return errors


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: validate-adr.py <adr-file.md> [<adr-file.md> ...]")
        return 2

    failed = False
    for raw_path in sys.argv[1:]:
        filepath = Path(raw_path)
        if not filepath.is_file():
            print(f"{filepath}: file not found")
            failed = True
            continue
        errors = validate_adr(filepath)
        if errors:
            failed = True
            for error in errors:
                print(f"{filepath}: {error}")
        else:
            print(f"{filepath}: ok")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
