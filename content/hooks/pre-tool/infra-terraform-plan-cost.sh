#!/bin/bash
# Terraform Plan with Cost Estimation
# Runs terraform plan, infracost, and tfsec security scan
# Exit 0 (informational)

set -euo pipefail

# Source shared libraries
source "${CLAUDE_PLUGIN_ROOT}/hooks/lib/json-parser.sh"
source "${CLAUDE_PLUGIN_ROOT}/hooks/lib/config-reader.sh"
source "${CLAUDE_PLUGIN_ROOT}/hooks/lib/agent-detector.sh"
source "${CLAUDE_PLUGIN_ROOT}/hooks/lib/timeout-manager.sh"

# Start timeout tracker (5 minutes)
start_timeout_tracker 300

# Check if terraform plan is enabled
if ! is_hook_enabled "terraform-plan"; then
    exit 0
fi

# Only run for implementer agent or thorough validation
AGENT_TYPE=$(get_agent_type)
VALIDATION_LEVEL=$(get_validation_level)

if [[ "${AGENT_TYPE}" != "implementer" ]] && [[ "${VALIDATION_LEVEL}" != "thorough" ]]; then
    exit 0
fi

# Read hook input from stdin
HOOK_INPUT=$(cat)
FILE_PATH=$(parse_file_path "${HOOK_INPUT}")
TOOL_NAME=$(parse_tool_name "${HOOK_INPUT}")

# Only run on Write/Edit
if [[ "${TOOL_NAME}" != "Write" ]] && [[ "${TOOL_NAME}" != "Edit" ]]; then
    exit 0
fi

# Only check Terraform files
if [[ ! "${FILE_PATH}" =~ \.tf$ ]]; then
    exit 0
fi

PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo ".")

# Find Terraform directory
TF_DIR=$(dirname "${FILE_PATH}")

# Check if terraform is available
if ! command -v terraform >/dev/null 2>&1; then
    exit 0
fi

echo ""
echo "🏗️  Running Terraform Validation..."
echo ""

cd "${TF_DIR}"

# 1. Terraform fmt check
echo "   → Checking format..."
if terraform fmt -check -recursive . 2>&1 | tee /tmp/tf-fmt.log; then
    echo "     ✓ Formatting is correct"
else
    echo "     ⚠️  Formatting issues detected"
    echo "     To fix: terraform fmt -recursive ."
fi

# 2. Terraform validate
if check_remaining_time 60; then
    echo ""
    echo "   → Validating configuration..."

    if terraform init -backend=false 2>&1 >/dev/null; then
        if terraform validate 2>&1 | tee /tmp/tf-validate.log; then
            echo "     ✓ Configuration is valid"
        else
            echo "     ❌ Validation failed"
            cat /tmp/tf-validate.log
        fi
    else
        echo "     ⚠️  Cannot initialize (backend config may be required)"
    fi
fi

# 3. Terraform plan (if state is accessible)
if check_remaining_time 120; then
    echo ""
    echo "   → Running plan..."

    if terraform plan -out=/tmp/tfplan 2>&1 | tee /tmp/tf-plan.log | tail -20; then
        echo "     ✓ Plan generated"

        # Count changes
        ADDS=$(grep -c "will be created" /tmp/tf-plan.log || true)
        CHANGES=$(grep -c "will be updated" /tmp/tf-plan.log || true)
        DESTROYS=$(grep -c "will be destroyed" /tmp/tf-plan.log || true)

        echo ""
        echo "     Plan summary:"
        echo "       + ${ADDS} to add"
        echo "       ~ ${CHANGES} to change"
        echo "       - ${DESTROYS} to destroy"
    else
        echo "     ⚠️  Plan failed (may need backend/credentials)"
    fi
fi

# 4. Cost estimation with Infracost
if command -v infracost >/dev/null 2>&1 && check_remaining_time 60; then
    echo ""
    echo "   → Estimating costs with Infracost..."

    if [[ -f "/tmp/tfplan" ]]; then
        if infracost breakdown --path /tmp/tfplan 2>&1 | tee /tmp/infracost.log | tail -15; then
            echo "     ✓ Cost estimate complete"
        fi
    else
        echo "     ⚠️  Skipping (plan not available)"
    fi
fi

# 5. Security scan with tfsec
if command -v tfsec >/dev/null 2>&1 && check_remaining_time 30; then
    echo ""
    echo "   → Running tfsec security scan..."

    if tfsec . --format json > /tmp/tfsec-results.json 2>&1; then
        CRITICAL=$(jq '[.results[] | select(.severity=="CRITICAL")] | length' /tmp/tfsec-results.json)
        HIGH=$(jq '[.results[] | select(.severity=="HIGH")] | length' /tmp/tfsec-results.json)

        echo "     Security findings:"
        echo "       Critical: ${CRITICAL}"
        echo "       High: ${HIGH}"

        if [[ ${CRITICAL} -gt 0 ]] || [[ ${HIGH} -gt 0 ]]; then
            echo ""
            echo "     Review: cat /tmp/tfsec-results.json"
        fi
    fi
fi

ELAPSED=$(get_elapsed_time)
ELAPSED_FMT=$(format_duration ${ELAPSED})

echo ""
echo "✅ Terraform validation complete (${ELAPSED_FMT})"
echo ""

exit 0
