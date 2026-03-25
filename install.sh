#!/usr/bin/env bash
#
# Loaf - An Opinionated Agentic Framework
# Bootstrap Installer
#
# For remote install:
#   curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash
#
# For local development:
#   ./install.sh
#
# Flags are passed through to `loaf install`:
#   ./install.sh --to cursor        # Install to specific target
#   ./install.sh --to all           # Install to all detected targets
#   ./install.sh --upgrade          # Update already-installed targets
#   ./install.sh                    # Interactive selection (default)
#
set -euo pipefail

REPO_URL="https://github.com/levifig/loaf.git"
INSTALL_DIR="${HOME}/.local/share/loaf"

# Colors
BOLD='\033[1m'
RESET='\033[0m'
GREEN='\033[38;5;82m'
YELLOW='\033[38;5;220m'
GRAY='\033[38;5;245m'
RED='\033[38;5;196m'

print_header() {
    echo ""
    echo -e "   \033[38;5;208m█░░\033[0m \033[38;5;214m█▀█\033[0m \033[38;5;220m▄▀█\033[0m \033[38;5;226m█▀▀\033[0m"
    echo -e "   \033[38;5;208m█▄▄\033[0m \033[38;5;214m█▄█\033[0m \033[38;5;220m█▀█\033[0m \033[38;5;226m█▀░\033[0m"
    echo ""
    echo -e "   ${GRAY}An Opinionated Agentic Framework${RESET}"
    echo ""
}

ok()   { echo -e "  ${GREEN}✓${RESET} $1"; }
warn() { echo -e "  ${YELLOW}⚡${RESET} $1"; }
err()  { echo -e "  ${RED}✗${RESET} $1" >&2; }
info() { echo -e "    ${GRAY}$1${RESET}"; }

# ─────────────────────────────────────────────────────────────────────────────
# Dev mode: running from a local clone
# ─────────────────────────────────────────────────────────────────────────────

detect_dev_mode() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    if [[ -d "${script_dir}/.git" ]] && [[ -f "${script_dir}/package.json" ]] && [[ -d "${script_dir}/content/skills" ]]; then
        echo "${script_dir}"
        return 0
    fi
    echo ""
    return 1
}

# ─────────────────────────────────────────────────────────────────────────────
# Requirements
# ─────────────────────────────────────────────────────────────────────────────

check_requirements() {
    local missing=()
    command -v git &>/dev/null  || missing+=("git")
    command -v node &>/dev/null || missing+=("node (22+)")
    command -v npm &>/dev/null  || missing+=("npm")

    if [[ ${#missing[@]} -gt 0 ]]; then
        err "Missing: ${missing[*]}"
        exit 1
    fi

    local node_version
    node_version=$(node -v | sed 's/v//' | cut -d. -f1)
    if [[ "$node_version" -lt 22 ]]; then
        err "Node.js 22+ required (found: $(node -v))"
        exit 1
    fi

    ok "Requirements met (git, node $(node -v), npm)"
}

# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

main() {
    # Capture all args to pass through to loaf install
    local install_args=("$@")

    print_header

    # Step 1: Requirements
    check_requirements
    echo ""

    # Step 2: Get the code
    local repo_dir=""
    repo_dir="$(detect_dev_mode)" || true

    if [[ -n "${repo_dir}" ]]; then
        ok "Development mode: ${repo_dir}"
        echo ""
    else
        if [[ -d "${INSTALL_DIR}" ]]; then
            info "Updating existing installation..."
            git -C "${INSTALL_DIR}" pull --ff-only 2>/dev/null || {
                warn "Pull failed, re-cloning..."
                rm -rf "${INSTALL_DIR}"
                git clone --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
            }
        else
            info "Cloning loaf..."
            git clone --depth 1 "${REPO_URL}" "${INSTALL_DIR}"
        fi
        repo_dir="${INSTALL_DIR}"
        ok "Source ready"
        echo ""
    fi

    # Step 3: Build
    cd "${repo_dir}"

    # Always install deps to catch lockfile changes after git pull
    info "Installing dependencies..."
    npm install --silent 2>/dev/null

    info "Building CLI..."
    npx tsup --silent 2>/dev/null
    ok "CLI built"

    info "Building targets..."
    node dist-cli/index.js build 2>/dev/null
    ok "All targets built"
    echo ""

    # Step 4: Install to detected tools (pass through any flags)
    if [[ ${#install_args[@]} -gt 0 ]]; then
        node dist-cli/index.js install "${install_args[@]}"
    else
        node dist-cli/index.js install --to all
    fi
    echo ""

    # Claude Code instructions
    if command -v claude &>/dev/null; then
        ok "${BOLD}Claude Code${RESET} detected"
        echo ""
        if [[ "${repo_dir}" == "${INSTALL_DIR}" ]]; then
            info "Add marketplace: ${BOLD}/plugin marketplace add levifig/loaf${RESET}"
        else
            info "Dev mode: ${BOLD}/plugin marketplace add ${repo_dir}${RESET}"
        fi
        echo ""
    fi

    echo -e "  ${GREEN}${BOLD}✓ Done!${RESET}"
    echo ""
}

main "$@"
