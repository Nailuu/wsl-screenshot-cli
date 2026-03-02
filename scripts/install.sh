#!/bin/bash
set -euo pipefail

# Install script for wsl-screenshot-cli
# Usage: curl -fsSL https://raw.githubusercontent.com/Nailuu/wsl-screenshot-cli/main/scripts/install.sh | bash

REPO="Nailuu/wsl-screenshot-cli"
BINARY="wsl-screenshot-cli"
INSTALL_DIR="${HOME}/.local/bin"

info() {
    printf "\033[1;34m==>\033[0m %s\n" "$*"
}

error() {
    printf "\033[1;31merror:\033[0m %s\n" "$*" >&2
    exit 1
}

warn() {
    printf "\033[1;33mwarning:\033[0m %s\n" "$*" >&2
}

success() {
    printf "\033[1;32mdone:\033[0m %s\n" "$*"
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64 | amd64)  echo "amd64" ;;
        aarch64 | arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch. Only amd64 and arm64 are supported." ;;
    esac
}

detect_os() {
    if ! command -v wslinfo &>/dev/null || { ! wslinfo --wsl-version &>/dev/null && ! wslinfo --version &>/dev/null; }; then
        error "WSL2 not detected. This tool only runs inside WSL2."
    fi
    echo "linux"
}

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local response

    if command -v curl &>/dev/null; then
        response=$(curl -fsSL "$url")
    elif command -v wget &>/dev/null; then
        response=$(wget -qO- "$url")
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    local version
    version=$(echo "$response" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        error "Failed to fetch latest version. Check https://github.com/${REPO}/releases"
    fi

    echo "$version"
}

download() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "$dest" "$url"
    else
        error "Neither curl nor wget found."
    fi
}

fix_path() {
    if echo "$PATH" | tr ':' '\n' | grep -qx "${INSTALL_DIR}"; then
        return
    fi

    if [ -f "${HOME}/.bashrc" ] && grep -q '\.local/bin' "${HOME}/.bashrc"; then
        warn "${INSTALL_DIR} is not in your current PATH, but ~/.bashrc already has an entry."
        echo "  Run: source ~/.bashrc"
        return
    fi

    cat >> "${HOME}/.bashrc" << 'PATHEOF'

# Added by wsl-screenshot-cli installer
if [ -d "$HOME/.local/bin" ] && [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    export PATH="$HOME/.local/bin:$PATH"
fi
PATHEOF
    success "Added PATH entry for ~/.local/bin to ~/.bashrc"
    echo "  Run 'source ~/.bashrc' or open a new terminal to apply."
}

add_shell_autostart() {
    if [ -f "${HOME}/.bashrc" ] && grep -q "wsl-screenshot-cli start" "${HOME}/.bashrc"; then
        success "Shell auto-start already configured in ~/.bashrc"
        return
    fi

    cat >> "${HOME}/.bashrc" << 'STARTEOF'

# Auto-start wsl-screenshot-cli (added by installer)
wsl-screenshot-cli start --daemon 2>/dev/null
STARTEOF
    success "Added auto-start to ~/.bashrc"
}

merge_json() {
    local file="$1"
    local start_cmd="wsl-screenshot-cli start --daemon 2>/dev/null; echo 'wsl-screenshot-cli started'"
    local stop_cmd="wsl-screenshot-cli stop 2>/dev/null"

    local existing="{}"
    if [ -f "$file" ]; then
        existing=$(cat "$file")
    fi

    local result
    result=$(echo "$existing" | jq \
        --arg start_cmd "$start_cmd" \
        --arg stop_cmd "$stop_cmd" \
        '
        # Build the new hook entries
        ($start_cmd) as $sc |
        ($stop_cmd) as $ec |
        {matcher: "", hooks: [{type: "command", command: $sc}]} as $start_entry |
        {matcher: "", hooks: [{type: "command", command: $ec}]} as $stop_entry |

        # Merge into existing structure
        .hooks //= {} |
        .hooks.SessionStart //= [] |
        .hooks.SessionEnd //= [] |

        # Only add if not already present
        (if (.hooks.SessionStart | map(.hooks[]?.command) | any(contains("wsl-screenshot-cli")))
         then . else .hooks.SessionStart += [$start_entry] end) |
        (if (.hooks.SessionEnd | map(.hooks[]?.command) | any(contains("wsl-screenshot-cli")))
         then . else .hooks.SessionEnd += [$stop_entry] end)
        '
    ) || return 1

    echo "$result"
}

add_claude_hooks() {
    local settings_dir="${HOME}/.claude"
    local settings_file="${settings_dir}/settings.json"

    # Check for existing hooks
    if [ -f "$settings_file" ] && grep -q "wsl-screenshot-cli" "$settings_file"; then
        success "Claude Code hooks already configured in ${settings_file}"
        return
    fi

    if ! command -v jq &>/dev/null; then
        warn "jq is not available — cannot safely merge settings.json"
        echo ""
        echo "  Add these hooks manually to ~/.claude/settings.json:"
        echo '  https://github.com/Nailuu/wsl-screenshot-cli#claude-code-hooks'
        return
    fi

    mkdir -p "$settings_dir"

    local merged
    if ! merged=$(merge_json "$settings_file"); then
        warn "Failed to merge hooks into ${settings_file}"
        echo "  Add hooks manually: https://github.com/Nailuu/wsl-screenshot-cli#claude-code-hooks"
        return
    fi

    # Atomic write: temp file + mv
    local tmpfile
    tmpfile=$(mktemp "${settings_dir}/settings.json.XXXXXX")
    echo "$merged" > "$tmpfile"
    mv "$tmpfile" "$settings_file"

    success "Added Claude Code hooks to ${settings_file}"
}

setup_menu() {
    if [ "$INTERACTIVE" = "false" ]; then
        echo ""
        info "Non-interactive mode — skipping auto-start setup."
        echo "  To configure later, see: https://github.com/Nailuu/wsl-screenshot-cli#auto-start"
        return
    fi

    echo ""
    info "How would you like to auto-start wsl-screenshot-cli?"
    echo ""
    echo "  1) Add to shell profile (~/.bashrc)"
    echo "  2) Add Claude Code hooks (~/.claude/settings.json)"
    echo "  3) Both"
    echo "  4) Skip (I'll configure manually)"
    echo ""

    local choice
    printf "  Select [1-4]: "
    if ! read -r choice < "$TTY_FD"; then
        echo ""
        warn "Could not read input — skipping auto-start setup."
        return
    fi
    echo ""

    case "$choice" in
        1)
            add_shell_autostart
            ;;
        2)
            add_claude_hooks
            ;;
        3)
            add_shell_autostart
            add_claude_hooks
            ;;
        4)
            info "Skipped. Configure auto-start later:"
            echo "  https://github.com/Nailuu/wsl-screenshot-cli#auto-start"
            ;;
        *)
            warn "Invalid selection '${choice}' — skipping auto-start setup."
            echo "  Configure later: https://github.com/Nailuu/wsl-screenshot-cli#auto-start"
            ;;
    esac
}

main() {
    local os arch version version_stripped archive
    local IS_FRESH_INSTALL=true

    # TTY detection: handles direct run, curl|bash, and headless/CI
    INTERACTIVE=false
    TTY_FD="/dev/null"
    if [ -t 0 ]; then
        # Direct terminal run — stdin is a tty
        INTERACTIVE=true
        TTY_FD="/dev/stdin"
    elif [ -e /dev/tty ]; then
        # curl|bash or piped — try /dev/tty
        INTERACTIVE=true
        TTY_FD="/dev/tty"
    fi

    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    info "Fetching latest version..."
    version=$(get_latest_version)
    version_stripped="${version#v}"
    info "Latest version: ${version}"

    # Skip download if already installed at latest version
    if command -v "${BINARY}" &>/dev/null; then
        IS_FRESH_INSTALL=false
        local installed_version
        installed_version=$("${BINARY}" --version 2>&1 | awk '{print $NF}')
        if [ "${installed_version}" = "${version_stripped}" ]; then
            info "Already up to date (${version})."
            exit 0
        fi
        info "Updating from v${installed_version} to ${version}..."
    fi

    # Install jq if not present (needed for safe JSON merging)
    if ! command -v jq &>/dev/null; then
        info "Installing jq (required for configuration)..."
        if sudo -n apt-get update -qq 2>/dev/null && sudo -n apt-get install -y -qq jq 2>/dev/null; then
            success "Installed jq"
        else
            warn "Could not install jq automatically (sudo may require a password)."
            warn "Install it manually: sudo apt-get install jq"
        fi
    fi

    # Archive name matches GoReleaser template
    archive="${BINARY}_${version_stripped}_${os}_${arch}.tar.gz"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    local base_url="https://github.com/${REPO}/releases/download/${version}"

    info "Downloading ${archive}..."
    download "${base_url}/${archive}" "${tmpdir}/${archive}"

    info "Downloading checksums..."
    download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"

    info "Verifying checksum..."
    local expected actual
    expected=$(grep "${archive}" "${tmpdir}/checksums.txt" | awk '{print $1}')

    if [ -z "$expected" ]; then
        error "Could not find checksum for ${archive}"
    fi

    actual=$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')

    if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch! Expected: ${expected}, Got: ${actual}"
    fi
    info "Checksum verified."

    info "Extracting ${BINARY}..."
    tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}" "${BINARY}"

    mkdir -p "${INSTALL_DIR}"
    mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"

    info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

    if "${INSTALL_DIR}/${BINARY}" --version &>/dev/null; then
        local installed_version
        installed_version=$("${INSTALL_DIR}/${BINARY}" --version 2>&1)
        info "Verified: ${installed_version}"
    fi

    fix_path

    echo ""
    info "Installation complete! Run '${BINARY} --help' to get started."

    if [ "$IS_FRESH_INSTALL" = "true" ]; then
        setup_menu
    fi
}

main "$@"
