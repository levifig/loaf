#!/usr/bin/env bash
#
# Universal Agent Skills Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/levifig/agent-skills/main/install.sh | bash
#   # OR
#   ./install.sh [--update] [--target <target>]
#
# Options:
#   --update          Update existing installation
#   --target <name>   Install specific target only (claude-code, opencode, cursor, copilot)
#   --all             Install all detected targets (non-interactive)
#   --help            Show this help message
#
set -euo pipefail

# Configuration
REPO_URL="https://github.com/levifig/agent-skills.git"
INSTALL_DIR="${HOME}/.local/share/agent-skills"
VERSION="1.0.0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# Print banner
print_banner() {
    echo -e "${CYAN}"
    echo "  _   _       _                         _"
    echo " | | | |_ __ (_)_   _____ _ __ ___  __ _| |"
    echo " | | | | '_ \| \ \ / / _ \ '__/ __|/ _\` | |"
    echo " | |_| | | | | |\ V /  __/ |  \__ \ (_| | |"
    echo "  \___/|_| |_|_| \_/ \___|_|  |___/\__,_|_|"
    echo ""
    echo "  Agent Skills Installer v${VERSION}"
    echo -e "${NC}"
}

# Check requirements
check_requirements() {
    local missing=()

    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi

    if ! command -v node &> /dev/null; then
        missing+=("node")
    fi

    if ! command -v npm &> /dev/null; then
        missing+=("npm")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Missing required tools: ${missing[*]}"
        echo "Please install them before continuing."
        exit 1
    fi

    local node_version
    node_version=$(node -v | sed 's/v//' | cut -d. -f1)
    if [[ "$node_version" -lt 18 ]]; then
        error "Node.js 18+ required (found: $(node -v))"
        exit 1
    fi
}

# Detect installed AI tools
detect_tools() {
    local tools=()

    # Claude Code
    if command -v claude &> /dev/null; then
        tools+=("claude-code")
    fi

    # OpenCode
    if [[ -d "${HOME}/.config/opencode" ]]; then
        tools+=("opencode")
    fi

    # Cursor
    if [[ -d "${HOME}/.cursor" ]] || [[ -d "/Applications/Cursor.app" ]]; then
        tools+=("cursor")
    fi

    # Copilot (always available - project-level)
    tools+=("copilot")

    echo "${tools[*]}"
}

# Clone or update repository
clone_or_update_repo() {
    if [[ -d "${INSTALL_DIR}" ]]; then
        info "Updating existing installation..."
        cd "${INSTALL_DIR}"
        git fetch origin
        git reset --hard origin/main
        cd - > /dev/null
    else
        info "Cloning repository..."
        mkdir -p "$(dirname "${INSTALL_DIR}")"
        git clone "${REPO_URL}" "${INSTALL_DIR}"
    fi
}

# Build distribution
build_distribution() {
    local target="${1:-all}"

    info "Installing dependencies..."
    cd "${INSTALL_DIR}"
    npm install --silent

    info "Building ${target} distribution..."
    if [[ "$target" == "all" ]]; then
        npm run build
    else
        npm run "build:${target}"
    fi
    cd - > /dev/null
}

# Install to Claude Code
install_claude_code() {
    local dist_dir="${INSTALL_DIR}/dist/claude-code"

    if [[ ! -d "${dist_dir}" ]]; then
        error "Claude Code distribution not built"
        return 1
    fi

    info "Installing Claude Code plugins..."

    # Check if Claude CLI is available
    if ! command -v claude &> /dev/null; then
        warn "Claude CLI not found. Manual installation required:"
        echo "  1. Add marketplace: /plugin marketplace add ${dist_dir}"
        echo "  2. Install plugins: /plugin install orchestration@levifig"
        return 0
    fi

    echo ""
    echo "Claude Code plugins ready at: ${dist_dir}"
    echo ""
    echo "To install, run in Claude Code:"
    echo "  /plugin marketplace add ${dist_dir}"
    echo "  /plugin install orchestration@levifig  # PM coordination"
    echo "  /plugin install foundations@levifig    # Quality gates"
    echo "  /plugin install python@levifig         # Python projects"
    echo ""

    success "Claude Code ready for manual plugin installation"
}

# Install to OpenCode
install_opencode() {
    local dist_dir="${INSTALL_DIR}/dist/opencode"
    local opencode_dir="${HOME}/.config/opencode"

    if [[ ! -d "${dist_dir}" ]]; then
        error "OpenCode distribution not built"
        return 1
    fi

    info "Installing to OpenCode..."

    mkdir -p "${opencode_dir}/skill"
    mkdir -p "${opencode_dir}/agent"
    mkdir -p "${opencode_dir}/command"
    mkdir -p "${opencode_dir}/plugin"

    # Copy skills
    if [[ -d "${dist_dir}/skill" ]]; then
        cp -r "${dist_dir}/skill/"* "${opencode_dir}/skill/"
    fi

    # Copy agents
    if [[ -d "${dist_dir}/agent" ]]; then
        cp -r "${dist_dir}/agent/"* "${opencode_dir}/agent/"
    fi

    # Copy commands
    if [[ -d "${dist_dir}/command" ]]; then
        cp -r "${dist_dir}/command/"* "${opencode_dir}/command/"
    fi

    # Copy plugin (hooks)
    if [[ -d "${dist_dir}/plugin" ]]; then
        cp -r "${dist_dir}/plugin/"* "${opencode_dir}/plugin/"
    fi

    # Create skill-sync config for future updates
    local sync_config="${opencode_dir}/skill-sync.json"
    if [[ ! -f "${sync_config}" ]]; then
        cat > "${sync_config}" << EOF
{
  "sources": [
    {
      "local": "${INSTALL_DIR}",
      "skills": "*",
      "target": "global"
    }
  ]
}
EOF
    fi

    success "OpenCode installation complete"
}

# Install to Cursor
install_cursor() {
    local dist_dir="${INSTALL_DIR}/dist/cursor"
    local cursor_dir="${HOME}/.cursor"

    if [[ ! -d "${dist_dir}" ]]; then
        error "Cursor distribution not built"
        return 1
    fi

    if [[ ! -d "${cursor_dir}" ]]; then
        warn "Cursor config directory not found at ${cursor_dir}"
        echo "Copy manually: cp -r ${dist_dir}/.cursor/rules ~/.cursor/"
        return 0
    fi

    info "Installing to Cursor..."

    mkdir -p "${cursor_dir}/rules"
    cp -r "${dist_dir}/.cursor/rules/"* "${cursor_dir}/rules/"

    success "Cursor rules installed to ${cursor_dir}/rules/"
}

# Install to Copilot (instructions only)
install_copilot() {
    local dist_dir="${INSTALL_DIR}/dist/copilot"

    if [[ ! -d "${dist_dir}" ]]; then
        error "Copilot distribution not built"
        return 1
    fi

    echo ""
    echo "Copilot instructions generated at: ${dist_dir}/.github/"
    echo ""
    echo "To use in a project, copy to your repository:"
    echo "  cp ${dist_dir}/.github/copilot-instructions.md <your-repo>/.github/"
    echo ""

    success "Copilot instructions ready"
}

# Interactive target selection
select_targets() {
    local detected
    detected=$(detect_tools)

    echo ""
    echo "Detected AI tools:"

    local options=()
    local i=1
    for tool in $detected; do
        case $tool in
            claude-code) echo "  ${i}) Claude Code (CLI detected)" ;;
            opencode) echo "  ${i}) OpenCode (config found)" ;;
            cursor) echo "  ${i}) Cursor (app/config found)" ;;
            copilot) echo "  ${i}) Copilot (always available)" ;;
        esac
        options+=("$tool")
        ((i++))
    done

    echo ""
    echo "Enter numbers to install (space-separated), 'all', or 'q' to quit:"
    read -r selection

    if [[ "$selection" == "q" ]]; then
        echo "Installation cancelled."
        exit 0
    fi

    if [[ "$selection" == "all" ]]; then
        SELECTED_TARGETS=("${options[@]}")
        return
    fi

    SELECTED_TARGETS=()
    for num in $selection; do
        if [[ "$num" =~ ^[0-9]+$ ]] && [[ "$num" -le "${#options[@]}" ]] && [[ "$num" -gt 0 ]]; then
            SELECTED_TARGETS+=("${options[$((num-1))]}")
        fi
    done
}

# Main installation flow
main() {
    local update_mode=false
    local specific_target=""
    local install_all=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --update)
                update_mode=true
                shift
                ;;
            --target)
                specific_target="$2"
                shift 2
                ;;
            --all)
                install_all=true
                shift
                ;;
            --help|-h)
                echo "Usage: install.sh [--update] [--target <target>] [--all]"
                echo ""
                echo "Options:"
                echo "  --update          Update existing installation"
                echo "  --target <name>   Install specific target (claude-code, opencode, cursor, copilot)"
                echo "  --all             Install all detected targets non-interactively"
                echo "  --help            Show this help"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    print_banner
    check_requirements

    # Clone or update repository
    clone_or_update_repo

    # Determine which targets to install
    if [[ -n "$specific_target" ]]; then
        SELECTED_TARGETS=("$specific_target")
    elif [[ "$install_all" == true ]]; then
        IFS=' ' read -ra SELECTED_TARGETS <<< "$(detect_tools)"
    else
        select_targets
    fi

    if [[ ${#SELECTED_TARGETS[@]} -eq 0 ]]; then
        warn "No targets selected"
        exit 0
    fi

    # Build distributions
    for target in "${SELECTED_TARGETS[@]}"; do
        build_distribution "$target"
    done

    # Install to each target
    echo ""
    for target in "${SELECTED_TARGETS[@]}"; do
        case $target in
            claude-code) install_claude_code ;;
            opencode) install_opencode ;;
            cursor) install_cursor ;;
            copilot) install_copilot ;;
        esac
    done

    echo ""
    success "Installation complete!"
    echo ""
    echo "To update later, run:"
    echo "  ${INSTALL_DIR}/install.sh --update"
    echo ""
}

# Run main
main "$@"
