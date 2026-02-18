#!/usr/bin/env bash
#
# Loaf - Levi's Opinionated Agentic Framework Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash
#   ./install.sh [--update] [--target <target>] [--all]
#
set -euo pipefail

# Configuration
REPO_URL="https://github.com/levifig/loaf.git"
INSTALL_DIR="${HOME}/.local/share/loaf"
VERSION="1.0.0"
LOAF_DEBUG="${LOAF_DEBUG:-0}"
LOAF_MARKER_FILE=".loaf-version"

# Colors (256-color)
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'
RED='\033[38;5;196m'
GREEN='\033[38;5;82m'
YELLOW='\033[38;5;220m'
BLUE='\033[38;5;39m'
MAGENTA='\033[38;5;135m'
CYAN='\033[38;5;51m'
PURPLE='\033[38;5;99m'
GRAY='\033[38;5;245m'
WHITE='\033[38;5;255m'
ORANGE='\033[38;5;208m'

# ─────────────────────────────────────────────────────────────────────────────
# UI Components
# ─────────────────────────────────────────────────────────────────────────────

print_header() {
    clear
    echo ""
    echo -e "   \033[38;5;208m█░░\033[0m \033[38;5;214m█▀█\033[0m \033[38;5;220m▄▀█\033[0m \033[38;5;226m█▀▀\033[0m"
    echo -e "   \033[38;5;208m█▄▄\033[0m \033[38;5;214m█▄█\033[0m \033[38;5;220m█▀█\033[0m \033[38;5;226m█▀░\033[0m"
    echo ""
    echo -e "   ${GRAY}Levi's Opinionated Agentic Framework${RESET}  ${GRAY}v${VERSION}${RESET}"
    echo -e "   ${GRAY}\"Why have just a slice when you can get the whole loaf?\"${RESET}"
    echo ""
    echo -e "${GRAY}   ──────────────────────────────────────────────────${RESET}"
    echo ""
}

print_dev_banner() {
    echo -e "  ${ORANGE}${BOLD}◆ DEVELOPMENT MODE${RESET}"
    echo ""
    print_info "Running from local clone - will build from source"
    echo ""
}

print_step() {
    echo -e "  ${PURPLE}${BOLD}[$1]${RESET} $2"
}

print_success() {
    echo -e "  ${GREEN}✓${RESET} $1"
}

print_error() {
    echo -e "  ${RED}✗${RESET} $1" >&2
}

print_info() {
    echo -e "    ${GRAY}$1${RESET}"
}

debug_log() {
    if [[ "${LOAF_DEBUG}" == "1" ]]; then
        print_info "debug: $1"
    fi
}

print_warn() {
    echo -e "  ${YELLOW}⚡${RESET} $1"
}

spinner() {
    local msg="$1"
    shift
    echo -en "    ${GRAY}$msg...${RESET} "
    if "$@" > /dev/null 2>&1; then
        echo -e "${GREEN}done${RESET}"
    else
        echo -e "${RED}failed${RESET}"
        return 1
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Detection
# ─────────────────────────────────────────────────────────────────────────────

declare -a TOOL_KEYS=()
declare -a TOOL_NAMES=()
declare -a TOOL_INSTALLED=()
DETECTION_PATH="${PATH}"

# Config directory resolution
declare -A TOOL_CONFIG_DIRS=()

# Default config directories for all supported targets
# Used when --target overrides auto-detection
declare -A DEFAULT_CONFIG_DIRS=(
    [opencode]="${XDG_CONFIG_HOME:-${HOME}/.config}/opencode"
    [cursor]="${HOME}/.cursor"
    [codex]="${CODEX_HOME:-${HOME}/.codex}"
    [gemini]="${HOME}/.gemini"
)

# Check if running from a local development repo
IS_DEV_MODE=false
LOCAL_REPO_PATH=""

detect_dev_mode() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Check if script is in a git repo with our expected structure
    if [[ -d "${script_dir}/.git" ]] && [[ -f "${script_dir}/package.json" ]] && [[ -d "${script_dir}/src/skills" ]]; then
        # Not running from the install dir itself
        if [[ "${script_dir}" != "${INSTALL_DIR}" ]]; then
            IS_DEV_MODE=true
            LOCAL_REPO_PATH="${script_dir}"
        fi
    fi
}

is_macos() {
    [[ "$(uname -s)" == "Darwin" ]]
}

normalize_detection_path() {
    local candidate_paths=()
    local helper_path=""
    local helper_output=""

    candidate_paths+=("${PATH}")

    if is_macos && [[ -x "/usr/libexec/path_helper" ]]; then
        helper_output="$(/usr/libexec/path_helper -s 2>/dev/null || true)"
        if [[ "${helper_output}" =~ PATH=\"([^\"]*)\" ]]; then
            helper_path="${BASH_REMATCH[1]}"
        fi
    fi

    if [[ -n "${helper_path}" ]]; then
        candidate_paths+=("${helper_path}")
    fi

    if is_macos; then
        candidate_paths+=("/opt/homebrew/bin" "/usr/local/bin" "${HOME}/.local/bin")
        if command -v npm &> /dev/null; then
            local npm_prefix=""
            npm_prefix="$(npm config get prefix 2>/dev/null || true)"
            if [[ -n "${npm_prefix}" ]]; then
                candidate_paths+=("${npm_prefix}/bin")
            fi
        fi
    fi

    local normalized=""
    local seen=":"
    local path_entry=""
    local path_list=""
    local -a path_parts=()
    for path_list in "${candidate_paths[@]}"; do
        IFS=':' read -r -a path_parts <<< "${path_list}"
        for path_entry in "${path_parts[@]}"; do
            if [[ -n "${path_entry}" && "${seen}" != *":${path_entry}:"* ]]; then
                normalized+="${path_entry}:"
                seen+="${path_entry}:"
            fi
        done
    done

    echo "${normalized%:}"
}

has_cmd() {
    local cmd="$1"
    PATH="${DETECTION_PATH}" command -v "${cmd}" >/dev/null 2>&1
}

detect_tools() {
    TOOL_KEYS=()
    TOOL_NAMES=()
    TOOL_INSTALLED=()
    TOOL_CONFIG_DIRS=()

    DETECTION_PATH="${PATH}"
    if is_macos; then
        DETECTION_PATH="$(normalize_detection_path)"
    fi
    debug_log "Detection PATH: ${DETECTION_PATH}"

    # Claude Code - detected separately (just needs marketplace add)
    HAS_CLAUDE_CODE=false
    if has_cmd "claude"; then
        HAS_CLAUDE_CODE=true
        debug_log "Claude Code detected via cli"
    fi

    # OpenCode (always uses XDG)
    local opencode_config="${XDG_CONFIG_HOME:-${HOME}/.config}/opencode"
    if [[ -d "${opencode_config}" ]]; then
        TOOL_KEYS+=("opencode")
        TOOL_NAMES+=("OpenCode")
        TOOL_CONFIG_DIRS[opencode]="${opencode_config}"
        if is_loaf_installed "${opencode_config}" "skill"; then
            TOOL_INSTALLED+=("yes")
        else
            TOOL_INSTALLED+=("no")
        fi
    fi

    # Cursor (uses ~/.cursor/)
    local cursor_config="${HOME}/.cursor"
    local cursor_reason=""
    if has_cmd "cursor"; then
        cursor_reason="cli"
    elif [[ -d "/Applications/Cursor.app" || -d "${HOME}/Applications/Cursor.app" ]]; then
        cursor_reason="app"
    elif [[ -d "${cursor_config}" ]]; then
        cursor_reason="config"
    fi
    if [[ -n "${cursor_reason}" ]]; then
        TOOL_KEYS+=("cursor")
        TOOL_NAMES+=("Cursor")
        TOOL_CONFIG_DIRS[cursor]="${cursor_config}"
        if is_loaf_installed "${cursor_config}" "skills"; then
            TOOL_INSTALLED+=("yes")
        else
            TOOL_INSTALLED+=("no")
        fi
        debug_log "Cursor detected via ${cursor_reason}"
    fi

    # Codex (uses $CODEX_HOME or ~/.codex/)
    local codex_config="${CODEX_HOME:-${HOME}/.codex}"
    codex_config="${codex_config%/}"
    local codex_reason=""
    if has_cmd "codex"; then
        codex_reason="cli"
    elif [[ -d "${codex_config}" || -d "${HOME}/.codex" ]]; then
        codex_reason="config"
    fi
    if [[ -n "${codex_reason}" ]]; then
        TOOL_KEYS+=("codex")
        TOOL_NAMES+=("Codex")
        TOOL_CONFIG_DIRS[codex]="${codex_config}"
        if is_loaf_installed "${codex_config}" "skills"; then
            TOOL_INSTALLED+=("yes")
        else
            TOOL_INSTALLED+=("no")
        fi
        debug_log "Codex detected via ${codex_reason}"
    fi

    # Gemini (uses ~/.gemini/ - no XDG support yet)
    local gemini_config="${HOME}/.gemini"
    local gemini_reason=""
    if has_cmd "gemini"; then
        gemini_reason="cli"
    elif [[ -d "${gemini_config}" ]]; then
        gemini_reason="config"
    fi
    if [[ -n "${gemini_reason}" ]]; then
        TOOL_KEYS+=("gemini")
        TOOL_NAMES+=("Gemini")
        TOOL_CONFIG_DIRS[gemini]="${gemini_config}"
        if is_loaf_installed "${gemini_config}" "skills"; then
            TOOL_INSTALLED+=("yes")
        else
            TOOL_INSTALLED+=("no")
        fi
        debug_log "Gemini detected via ${gemini_reason}"
    fi
}

check_existing_installation() {
    [[ -d "${INSTALL_DIR}" ]] && [[ -f "${INSTALL_DIR}/.version" || -d "${INSTALL_DIR}/opencode" || -d "${INSTALL_DIR}/cursor" || -d "${INSTALL_DIR}/codex" || -d "${INSTALL_DIR}/gemini" ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Requirements
# ─────────────────────────────────────────────────────────────────────────────

check_requirements() {
    local missing=()

    # Git always required
    command -v git &> /dev/null || missing+=("git")

    # Node/npm only required for local dev (building)
    if [[ "$IS_DEV_MODE" == true ]]; then
        command -v node &> /dev/null || missing+=("node")
        command -v npm &> /dev/null || missing+=("npm")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        print_error "Missing required tools: ${missing[*]}"
        echo ""
        print_info "Please install them before continuing."
        exit 1
    fi

    # Check node version only if node is required
    if [[ "$IS_DEV_MODE" == true ]]; then
        local node_version
        node_version=$(node -v | sed 's/v//' | cut -d. -f1)
        if [[ "$node_version" -lt 18 ]]; then
            print_error "Node.js 18+ required (found: $(node -v))"
            exit 1
        fi
        print_success "All requirements met (git, node 18+, npm)"
    else
        print_success "All requirements met (git)"
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Multi-select UI
# ─────────────────────────────────────────────────────────────────────────────

select_targets_interactive() {
    SELECTED_TARGETS=()

    if [[ ${#TOOL_KEYS[@]} -eq 0 ]]; then
        print_info "No other tools detected"
        return
    fi

    local count=${#TOOL_KEYS[@]}

    echo ""
    echo -e "  ${BOLD}Select targets to install:${RESET}"
    echo ""

    # First pass: show all targets faded (pending)
    for ((i = 0; i < count; i++)); do
        local label="${TOOL_NAMES[$i]}"
        local status=""
        if [[ "${TOOL_INSTALLED[$i]}" == "yes" ]]; then
            status=" ${YELLOW}(installed)${RESET}"
        fi
        echo -e "    ${GRAY}○ ${label}${status}${RESET}"
    done

    # Move cursor back up to first target
    echo -en "\033[${count}A"

    # Second pass: prompt for each target
    for ((i = 0; i < count; i++)); do
        local label="${TOOL_NAMES[$i]}"
        local status=""
        if [[ "${TOOL_INSTALLED[$i]}" == "yes" ]]; then
            status=" ${YELLOW}(installed)${RESET}"
        fi

        # Highlight current line with prompt
        echo -en "\033[2K"
        echo -en "    ${WHITE}▶${RESET} ${BOLD}${label}${RESET}${status}  ${GRAY}[y/N]${RESET} "
        read -r response

        # Rewrite line with result
        echo -en "\033[A\033[2K"
        if [[ "$response" =~ ^[Yy] ]]; then
            SELECTED_TARGETS+=("${TOOL_KEYS[$i]}")
            echo -e "    ${GREEN}✓${RESET} ${label}${status}"
        else
            echo -e "    ${GRAY}○ ${label}${status}${RESET}"
        fi
    done

    echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Installation Steps
# ─────────────────────────────────────────────────────────────────────────────

# Build in local dev repo before syncing
build_local() {
    local repo_dir="$1"
    cd "${repo_dir}"

    if [[ ! -d "node_modules" ]] || [[ ! -f "node_modules/.package-lock.json" ]]; then
        spinner "Installing dependencies" npm install --silent
    fi

    spinner "Building distributions" npm run build --silent
    cd - > /dev/null
}

clone_or_update() {
    if [[ "$IS_DEV_MODE" == true ]]; then
        # Running from local development repo
        print_info "Building from: ${LOCAL_REPO_PATH}"

        # Build in the local repo first
        build_local "${LOCAL_REPO_PATH}"

        # Sync only dist/ to cache (not plugins/ - those stay at root for Claude Code)
        rm -rf "${INSTALL_DIR}"
        mkdir -p "${INSTALL_DIR}"
        if command -v rsync &> /dev/null; then
            spinner "Syncing dist to cache" rsync -a "${LOCAL_REPO_PATH}/dist/" "${INSTALL_DIR}/"
        else
            spinner "Copying dist to cache" cp -r "${LOCAL_REPO_PATH}/dist/"* "${INSTALL_DIR}/"
        fi
    else
        # Remote install - clone to temp, copy dist/ to cache
        local temp_dir
        temp_dir=$(mktemp -d)

        if [[ -d "${INSTALL_DIR}" ]]; then
            # Check if update needed by comparing with remote
            spinner "Checking for updates" git clone --depth 1 "${REPO_URL}" "${temp_dir}"

            local remote_hash
            remote_hash=$(git -C "${temp_dir}" rev-parse HEAD)

            if [[ -f "${INSTALL_DIR}/.version" ]] && [[ "$(cat "${INSTALL_DIR}/.version")" == "$remote_hash" ]]; then
                print_info "Already up to date"
                rm -rf "${temp_dir}"
                return 0
            fi
        else
            spinner "Cloning repository" git clone --depth 1 "${REPO_URL}" "${temp_dir}"
        fi

        # Copy only dist/ to cache
        rm -rf "${INSTALL_DIR}"
        mkdir -p "${INSTALL_DIR}"
        cp -r "${temp_dir}/dist/"* "${INSTALL_DIR}/"

        # Store version hash for future update checks
        git -C "${temp_dir}" rev-parse HEAD > "${INSTALL_DIR}/.version"

        rm -rf "${temp_dir}"
    fi
}

show_claude_code_dev_instructions() {
    echo -e "  ${GREEN}✓${RESET} ${BOLD}Claude Code${RESET} detected"
    echo ""
    print_info "For development, test your local marketplace:"
    echo ""
    echo -e "    ${WHITE}/plugin marketplace add ${LOCAL_REPO_PATH}${RESET}"
    echo ""
    print_info "This uses plugins/ built at repo root."
    print_info "For production, users will use:"
    echo ""
    echo -e "    ${GRAY}/plugin marketplace add levifig/loaf${RESET}"
    echo ""
}

show_claude_code_instructions() {
    echo -e "  ${GREEN}✓${RESET} ${BOLD}Claude Code${RESET} detected"
    echo ""
    print_info "Add the marketplace in Claude Code:"
    echo ""
    echo -e "    ${WHITE}/plugin marketplace add levifig/loaf${RESET}"
    echo ""
    print_info "Then browse and install plugins via ${WHITE}/plugin${RESET}"
    echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Copy Helpers
# ─────────────────────────────────────────────────────────────────────────────

# Copy directory contents from source to target
# Uses rsync when available (syncs and removes stale files)
# Falls back to rm + cp otherwise
copy_items() {
    local source_dir="$1"
    local target_dir="$2"

    mkdir -p "${target_dir}"

    if command -v rsync &> /dev/null; then
        rsync -a --delete "${source_dir}/" "${target_dir}/"
    else
        rm -rf "${target_dir:?}"/*
        cp -r "${source_dir}"/* "${target_dir}/" 2>/dev/null || true
    fi
}

is_loaf_installed() {
    local config_dir="$1"
    local skills_dir_name="$2"
    local marker_path="${config_dir}/${LOAF_MARKER_FILE}"

    if [[ -f "${marker_path}" ]]; then
        return 0
    fi

    local skills_dir="${config_dir}/${skills_dir_name}"
    local skill_name=""
    for skill_name in foundations python-development python; do
        if [[ -e "${skills_dir}/${skill_name}" ]]; then
            return 0
        fi
    done

    return 1
}

write_loaf_marker() {
    local config_dir="$1"

    mkdir -p "${config_dir}"
    printf "%s\n" "${VERSION}" > "${config_dir}/${LOAF_MARKER_FILE}"
}

# ─────────────────────────────────────────────────────────────────────────────
# Tool-specific Installers
# ─────────────────────────────────────────────────────────────────────────────

install_opencode() {
    local dist="${INSTALL_DIR}/opencode"
    local config="${TOOL_CONFIG_DIRS[opencode]}"

    mkdir -p "${config}"/{skill,agent,command,plugin}

    if command -v rsync &> /dev/null; then
        rsync -a --delete "${dist}/skill/" "${config}/skill/"
        rsync -a --delete "${dist}/agent/" "${config}/agent/"
        rsync -a --delete "${dist}/command/" "${config}/command/"
        rsync -a --delete "${dist}/plugin/" "${config}/plugin/"
    else
        cp -r "${dist}/skill/"* "${config}/skill/" 2>/dev/null || true
        cp -r "${dist}/agent/"* "${config}/agent/" 2>/dev/null || true
        cp -r "${dist}/command/"* "${config}/command/" 2>/dev/null || true
        cp -r "${dist}/plugin/"* "${config}/plugin/" 2>/dev/null || true
    fi

    write_loaf_marker "${config}"
    print_success "OpenCode installed to ${config}"
}

install_cursor() {
    local cache="${INSTALL_DIR}/cursor"
    local config="${TOOL_CONFIG_DIRS[cursor]}"

    # Remove stale commands from previous Loaf installs
    if [[ -d "${config}/commands" ]]; then
        rm -rf "${config}/commands"
    fi

    # Skills
    if [[ -d "${cache}/skills" ]]; then
        copy_items "${cache}/skills" "${config}/skills"
    fi

    # Agents (subagents)
    if [[ -d "${cache}/agents" ]]; then
        copy_items "${cache}/agents" "${config}/agents"
    fi

    # hooks.json config
    if [[ -f "${cache}/hooks.json" ]]; then
        mkdir -p "${config}"
        cp "${cache}/hooks.json" "${config}/hooks.json"
    fi

    # Hook scripts
    if [[ -d "${cache}/hooks" ]]; then
        copy_items "${cache}/hooks" "${config}/hooks"
    fi

    write_loaf_marker "${config}"
    print_success "Cursor installed to ${config}"
}

install_codex() {
    local cache="${INSTALL_DIR}/codex"
    local config="${TOOL_CONFIG_DIRS[codex]}"

    # Skills only
    if [[ -d "${cache}/skills" ]]; then
        copy_items "${cache}/skills" "${config}/skills"
    fi

    write_loaf_marker "${config}"
    print_success "Codex installed to ${config}/skills"
}

install_gemini() {
    local cache="${INSTALL_DIR}/gemini"
    # Gemini doesn't support XDG yet - default to ~/.gemini/
    local config="${TOOL_CONFIG_DIRS[gemini]:-${HOME}/.gemini}"

    # Skills only
    if [[ -d "${cache}/skills" ]]; then
        copy_items "${cache}/skills" "${config}/skills"
    fi

    write_loaf_marker "${config}"
    print_success "Gemini installed to ${config}/skills"
}

prompt_update() {
    echo ""
    print_warn "Existing installation detected"
    echo ""
    echo -en "  ${WHITE}Update to latest version? [Y/n]${RESET} "
    read -r response
    if [[ "$response" =~ ^[Nn] ]]; then
        return 1
    fi
    return 0
}

show_dev_completion() {
    echo ""
    echo -e "${GRAY}  ──────────────────────────────────────────────────${RESET}"
    echo ""
    echo -e "  ${GREEN}${BOLD}✓ Development build complete!${RESET}"
    echo ""
    echo -e "  ${BOLD}What was built:${RESET}"
    print_info "• Claude Code: plugins/ at repo root"
    print_info "• Others: dist/ synced to ~/.local/share/loaf/"
    echo ""
    echo -e "  ${BOLD}Test your changes:${RESET}"
    print_info "Claude Code:"
    echo -e "    ${WHITE}/plugin marketplace add ${LOCAL_REPO_PATH}${RESET}"
    echo ""
    print_info "Rebuild after changes:"
    echo -e "    ${WHITE}npm run build${RESET}"
    echo ""
}

show_completion() {
    echo ""
    echo -e "${GRAY}  ──────────────────────────────────────────────────${RESET}"
    echo ""
    echo -e "  ${GREEN}${BOLD}✓ Installation complete!${RESET}"
    echo ""
    print_info "Update: curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash"
    echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

main() {
    local specific_target=""
    local install_all=false
    local force_fresh=false
    local upgrade_mode=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --update) shift ;;
            --fresh) force_fresh=true; shift ;;
            --target) specific_target="$2"; shift 2 ;;
            --all) install_all=true; shift ;;
            --upgrade) upgrade_mode=true; shift ;;
            --debug) LOAF_DEBUG=1; shift ;;
            --help|-h)
                echo "Usage: install.sh [options]"
                echo ""
                echo "Options:"
                echo "  --upgrade   Unattended upgrade of all detected targets"
                echo "  --update    Update existing installation"
                echo "  --fresh     Force fresh install (remove existing)"
                echo "  --target X  Install specific target only"
                echo "  --all       Install all detected targets"
                echo "  --help      Show this help"
                echo ""
                echo "Development:"
                echo "  Run from a cloned repo to build from source."
                echo "  Requires: git, node 18+, npm"
                exit 0
                ;;
            *) shift ;;
        esac
    done

    # Detect development mode early
    detect_dev_mode

    print_header

    # Show dev banner if applicable
    if [[ "$IS_DEV_MODE" == true ]]; then
        print_dev_banner
    fi

    # Step 1: Check requirements
    print_step "1" "Checking requirements"
    check_requirements
    echo ""

    # Step 2: Detect tools
    print_step "2" "Detecting AI tools"
    detect_tools
    if [[ ${#TOOL_NAMES[@]} -gt 0 ]]; then
        for ((i = 0; i < ${#TOOL_NAMES[@]}; i++)); do
            local status=""
            if [[ "${TOOL_INSTALLED[$i]}" == "yes" ]]; then
                status=" ${YELLOW}(installed)${RESET}"
            fi
            print_success "${TOOL_NAMES[$i]} detected${status}"
        done
    fi
    echo ""

    # Show Claude Code instructions (dev vs production)
    if [[ "$HAS_CLAUDE_CODE" == true ]]; then
        if [[ "$IS_DEV_MODE" == true ]]; then
            show_claude_code_dev_instructions
        else
            show_claude_code_instructions
        fi
    fi

    # Step 3: Select other targets
    print_step "3" "Other targets"

    if [[ -n "$specific_target" ]]; then
        # Validate against all known targets, not just detected ones
        if [[ -z "${DEFAULT_CONFIG_DIRS[$specific_target]+x}" ]]; then
            print_error "Unknown target: ${specific_target}"
            print_info "Valid targets: ${!DEFAULT_CONFIG_DIRS[*]}"
            exit 1
        fi

        # Ensure config dir is set (may not be if tool wasn't auto-detected)
        if [[ -z "${TOOL_CONFIG_DIRS[$specific_target]+x}" ]]; then
            TOOL_CONFIG_DIRS[$specific_target]="${DEFAULT_CONFIG_DIRS[$specific_target]}"
            print_warn "${specific_target} was not auto-detected; installing to ${DEFAULT_CONFIG_DIRS[$specific_target]}"
        fi

        SELECTED_TARGETS=("$specific_target")
        print_info "Target: $specific_target"
        echo ""
    elif [[ "$upgrade_mode" == true ]]; then
        # Upgrade mode: only select already-installed targets
        SELECTED_TARGETS=()
        for ((i = 0; i < ${#TOOL_KEYS[@]}; i++)); do
            if [[ "${TOOL_INSTALLED[$i]}" == "yes" ]]; then
                SELECTED_TARGETS+=("${TOOL_KEYS[$i]}")
            fi
        done
        if [[ ${#SELECTED_TARGETS[@]} -eq 0 ]]; then
            print_info "No installed targets to upgrade"
        else
            print_info "Upgrading: ${SELECTED_TARGETS[*]}"
        fi
        echo ""
    elif [[ "$install_all" == true ]]; then
        SELECTED_TARGETS=("${TOOL_KEYS[@]}")
        print_info "Installing all detected targets"
        echo ""
    else
        select_targets_interactive
    fi

    # Check for existing installation
    if check_existing_installation; then
        if [[ "$force_fresh" == true ]]; then
            print_info "Removing existing installation..."
            rm -rf "${INSTALL_DIR}"
        elif [[ "$upgrade_mode" == true ]]; then
            print_info "Upgrading existing installation..."
        else
            if ! prompt_update; then
                print_info "Keeping existing installation."
                return 0
            fi
        fi
        echo ""
    fi

    # Step 4: Fetch and build
    if [[ "$IS_DEV_MODE" == true ]]; then
        print_step "4" "Building from source"
    else
        print_step "4" "Fetching loaf"
    fi
    clone_or_update
    echo ""

    # Step 5: Install selected targets
    if [[ ${#SELECTED_TARGETS[@]} -gt 0 ]]; then
        print_step "5" "Installing"
        echo ""
        for target in "${SELECTED_TARGETS[@]}"; do
            case $target in
                opencode) install_opencode ;;
                cursor) install_cursor ;;
                codex) install_codex ;;
                gemini) install_gemini ;;
            esac
        done
    fi

    # Done - show appropriate completion message
    if [[ "$IS_DEV_MODE" == true ]]; then
        show_dev_completion
    else
        show_completion
    fi
}

main "$@"
