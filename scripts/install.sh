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
    if ! command -v wslinfo &>/dev/null || ! wslinfo --version &>/dev/null; then
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

main() {
    local os arch version version_stripped archive

    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    info "Fetching latest version..."
    version=$(get_latest_version)
    version_stripped="${version#v}"
    info "Latest version: ${version}"

    # Skip download if already installed at latest version
    if command -v "${BINARY}" &>/dev/null; then
        local installed_version
        installed_version=$("${BINARY}" --version 2>&1 | awk '{print $NF}')
        if [ "${installed_version}" = "${version_stripped}" ]; then
            info "Already up to date (${version})."
            exit 0
        fi
        info "Updating from v${installed_version} to ${version}..."
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

    if ! echo "$PATH" | tr ':' '\n' | grep -qx "${INSTALL_DIR}"; then
        warn "${INSTALL_DIR} is not in your PATH."
        echo ""
        echo "Add it to your ~/.bashrc (or ~/.zshrc):"
        echo ""
        echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
        echo ""
        echo "Then reload: source ~/.bashrc"
    fi

    echo ""
    info "Installation complete! Run '${BINARY} --help' to get started."
    echo ""
    echo "To start automatically with every WSL session, add to your ~/.bashrc (or ~/.zshrc):"
    echo ""
    echo "  wsl-screenshot-cli start --daemon"
}

main "$@"
