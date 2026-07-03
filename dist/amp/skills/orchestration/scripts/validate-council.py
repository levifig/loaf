#!/usr/bin/env python3
"""
Validate council file format and required fields.
Usage: validate-council.py <council-file>
Exit codes: 0 = valid, 1 = invalid
"""

import sys
import re
import yaml
from pathlib import Path
from datetime import datetime


def parse_frontmatter(content: str) -> tuple[dict, str]:
    """Extract YAML frontmatter from markdown content."""
    if not content.startswith('---'):
        return {}, content

    parts = content.split('---', 2)
    if len(parts) < 3:
        return {}, content

    try:
        frontmatter = yaml.safe_load(parts[1])
        body = parts[2]
        return frontmatter or {}, body
    except yaml.YAMLError as e:
        print(f"Error parsing YAML frontmatter: {e}")
        return {}, content


def validate_filename(filepath: Path) -> list[str]:
    """Validate council filename format."""
    errors = []
    pattern = r'^\d{8}-\d{6}-[a-z0-9-]+\.md$'

    if not re.match(pattern, filepath.name):
        errors.append(f"Filename must match YYYYMMDD-HHMMSS-<topic>.md, got: {filepath.name}")

    return errors


def validate_frontmatter(fm: dict) -> list[str]:
    """Validate required frontmatter fields."""
    errors = []

    # Check council block exists
    council = fm.get('council', {})
    if not council:
        errors.append("Missing 'council' block in frontmatter")
        return errors

    # Required council fields — must match council/templates/council.md
    required_fields = ['topic', 'created', 'status', 'composition']
    for field in required_fields:
        if field not in council:
            errors.append(f"Missing required field: council.{field}")

    # Status follows the canonical council lifecycle (SPEC-049)
    valid_statuses = ['draft', 'done', 'archived']
    if council.get('status') and council['status'] not in valid_statuses:
        errors.append(f"Invalid council.status: {council['status']} (must be one of: {', '.join(valid_statuses)})")

    # Validate ISO 8601 created timestamp
    created = council.get('created', '')
    if created:
        try:
            datetime.fromisoformat(created.replace('Z', '+00:00'))
        except ValueError:
            errors.append(f"Invalid ISO 8601 created timestamp: {created}")

    # Validate composition
    composition = council.get('composition', [])
    if composition:
        if len(composition) < 5:
            errors.append(f"Council requires at least 5 agents in composition (got {len(composition)})")
        if len(composition) % 2 == 0:
            errors.append(f"Council requires an ODD number of agents in composition (got {len(composition)})")

    # session_reference and linear_issue are optional free-form references

    return errors


def validate_sections(body: str) -> list[str]:
    """Validate required markdown sections."""
    errors = []
    required_sections = ['## Decision Question', '## Context', '## Agent Perspectives', '## Synthesis', '## Decision']

    # Exact heading match so '## Decision Question' does not satisfy '## Decision'
    headings = {line.strip() for line in body.splitlines() if line.strip().startswith('##')}
    for section in required_sections:
        if section not in headings:
            errors.append(f"Missing required section: {section}")

    return errors


def main():
    if len(sys.argv) != 2:
        print("Usage: validate-council.py <council-file>")
        sys.exit(1)

    filepath = Path(sys.argv[1])

    if not filepath.exists():
        print(f"Error: File not found: {filepath}")
        sys.exit(1)

    content = filepath.read_text()
    frontmatter, body = parse_frontmatter(content)

    errors = []

    errors.extend(validate_filename(filepath))
    errors.extend(validate_frontmatter(frontmatter))
    errors.extend(validate_sections(body))

    # Output results
    if errors:
        print("VALIDATION FAILED")
        print("\nErrors:")
        for e in errors:
            print(f"  - {e}")
        sys.exit(1)
    else:
        print("VALIDATION PASSED")
        sys.exit(0)


if __name__ == '__main__':
    main()
