#!/bin/bash
# Generate a new council file with correct format
# Usage: new-council.sh <topic> <session-ref> <participant1> <participant2> ...
# Example: new-council.sh database-schema-choice 20251204-143000-auth-system architect dba-specialist ops-lead security

set -e

TOPIC="${1:?Usage: new-council.sh <topic> <session-filename> <participant1> <participant2> ...}"
SESSION="${2:?Session ref required}"
shift 2
PARTICIPANTS=("$@")

# Validate minimum participants (5) and odd count
if [[ ${#PARTICIPANTS[@]} -lt 5 ]]; then
    echo "Error: Council requires at least 5 participants (got ${#PARTICIPANTS[@]})" >&2
    exit 1
fi

if [[ $((${#PARTICIPANTS[@]} % 2)) -eq 0 ]]; then
    echo "Error: Council requires an odd number of participants (got ${#PARTICIPANTS[@]})" >&2
    exit 1
fi

# Generate timestamps
TIMESTAMP=$(date -u +"%Y%m%d-%H%M%S")
ISO_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Validate topic format (kebab-case)
if [[ ! "$TOPIC" =~ ^[a-z0-9-]+$ ]]; then
    echo "Error: Topic must be kebab-case (lowercase letters, numbers, hyphens)" >&2
    exit 1
fi

# Validate session exists in native SQLite state when loaf is available.
if command -v loaf >/dev/null 2>&1 && ! loaf session show "$SESSION" >/dev/null 2>&1; then
    echo "Error: Session not found in SQLite state: $SESSION" >&2
    exit 1
fi

# Build filename
FILENAME="${TIMESTAMP}-${TOPIC}.md"
FILEPATH=".agents/councils/${FILENAME}"

# Check if file already exists
if [[ -f "$FILEPATH" ]]; then
    echo "Error: Council file already exists: $FILEPATH" >&2
    exit 1
fi

# Build composition YAML
COMPOSITION_YAML=""
for p in "${PARTICIPANTS[@]}"; do
    COMPOSITION_YAML="${COMPOSITION_YAML}    - ${p}
"
done

# Generate council file (shape must match council/templates/council.md)
cat > "$FILEPATH" << EOF
---
council:
  topic: "${TOPIC//-/ }"
  created: "${ISO_TIMESTAMP}"
  status: draft
  composition:
${COMPOSITION_YAML}  session_reference: "${SESSION}"
---

# Council: ${TOPIC//-/ }

## Decision Question

[Clear, specific question being decided]

## Options

### Option 1: [Name]
Brief description and trade-offs.

### Option 2: [Name]
Brief description and trade-offs.

## Context

Why this council was convened.
What problem or decision needed multiple perspectives.

## Agent Perspectives

$(for p in "${PARTICIPANTS[@]}"; do
echo "### ${p}

**Position**: [Option X]

**Pros**:
-

**Cons**:
-

**Concerns**:
-

"
done)

## Synthesis

**Consensus Points**:
-

**Key Disagreements**:
-

**Recommendation**: [Option X]

## Decision

[To be filled after user approval]

---

## Deliberation Log

### ${ISO_TIMESTAMP} - Council Convened
Agents: ${PARTICIPANTS[*]}
Composition selected by orchestrator.
EOF

echo "Created: $FILEPATH"
echo "Participants: ${PARTICIPANTS[*]}"
