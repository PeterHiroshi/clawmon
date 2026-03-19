#!/usr/bin/env bash
# clawmon installer
# Usage: curl -fsSL https://raw.githubusercontent.com/PeterHiroshi/clawmon/main/scripts/install.sh | bash
#
# Installs clawmon-daemon and clawmon-tui from the latest GitHub release.
# Supports: linux/darwin, amd64/arm64

set -euo pipefail

REPO="PeterHiroshi/clawmon"
GITHUB_API="https://api.github.com"
CONFIG_DIR="${HOME}/.clawmon"
CONFIG_FILE="${CONFIG_DIR}/config.toml"

# --- Color output ---

use_color() {
    if [ "${NO_COLOR:-}" != "" ] || [ "${TERM:-}" = "dumb" ]; then
        return 1
    fi
    return 0
}

if use_color; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    RESET=''
fi

info()  { printf "${BLUE}[*]${RESET} %s\n" "$1"; }
ok()    { printf "${GREEN}[+]${RESET} %s\n" "$1"; }
warn()  { printf "${YELLOW}[!]${RESET} %s\n" "$1"; }
error() { printf "${RED}[-]${RESET} %s\n" "$1" >&2; }

# --- Detection ---

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "${os}" in
        linux)  echo "linux" ;;
        darwin) echo "darwin" ;;
        *)      error "Unsupported OS: ${os}"; exit 1 ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              error "Unsupported architecture: ${arch}"; exit 1 ;;
    esac
}

# --- HTTP fetch ---

fetch() {
    local url="$1"
    local output="${2:-}"

    if command -v curl >/dev/null 2>&1; then
        if [ -n "${output}" ]; then
            curl -fsSL -o "${output}" "${url}"
        else
            curl -fsSL "${url}"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [ -n "${output}" ]; then
            wget -qO "${output}" "${url}"
        else
            wget -qO- "${url}"
        fi
    else
        error "Neither curl nor wget found. Please install one."
        exit 1
    fi
}

# --- Install directory ---

get_install_dir() {
    if [ "$(id -u)" -eq 0 ]; then
        echo "/usr/local/bin"
    else
        echo "${HOME}/.local/bin"
    fi
}

ensure_install_dir() {
    local dir="$1"
    if [ ! -d "${dir}" ]; then
        info "Creating install directory: ${dir}"
        mkdir -p "${dir}"
    fi
}

ensure_in_path() {
    local dir="$1"
    case ":${PATH}:" in
        *":${dir}:"*) return 0 ;;
    esac
    warn "${dir} is not in your PATH"
    warn "Add this to your shell profile:"
    printf "  ${BOLD}export PATH=\"%s:\$PATH\"${RESET}\n" "${dir}"
}

# --- Latest release ---

get_latest_version() {
    local response
    response=$(fetch "${GITHUB_API}/repos/${REPO}/releases/latest" 2>/dev/null || true)

    if [ -z "${response}" ]; then
        error "Failed to fetch latest release from GitHub"
        exit 1
    fi

    # Parse tag_name from JSON (portable, no jq dependency)
    echo "${response}" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
}

# --- Main ---

main() {
    printf "\n${BOLD}clawmon installer${RESET}\n\n"

    local os arch install_dir version tarball_name download_url tmpdir
    os=$(detect_os)
    arch=$(detect_arch)
    install_dir=$(get_install_dir)

    info "Detected: ${os}/${arch}"
    info "Install directory: ${install_dir}"

    # Get latest version
    info "Fetching latest release..."
    version=$(get_latest_version)

    if [ -z "${version}" ]; then
        error "Could not determine latest version"
        exit 1
    fi

    ok "Latest version: ${version}"

    # Build download URL
    tarball_name="clawmon-${version}-${os}-${arch}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/${version}/${tarball_name}"

    # Create temp directory
    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' EXIT

    # Download
    info "Downloading ${tarball_name}..."
    fetch "${download_url}" "${tmpdir}/${tarball_name}"
    ok "Downloaded successfully"

    # Extract
    info "Extracting..."
    tar -xzf "${tmpdir}/${tarball_name}" -C "${tmpdir}"

    # Find binaries in extracted archive
    local extract_dir="${tmpdir}/clawmon-${version}-${os}-${arch}"
    if [ ! -d "${extract_dir}" ]; then
        # Try without version prefix
        extract_dir=$(find "${tmpdir}" -maxdepth 1 -type d -name "clawmon-*" | head -1)
    fi

    if [ -z "${extract_dir}" ] || [ ! -d "${extract_dir}" ]; then
        error "Could not find extracted files"
        exit 1
    fi

    # Backup existing installation
    ensure_install_dir "${install_dir}"
    for binary in clawmon-daemon clawmon-tui; do
        if [ -f "${install_dir}/${binary}" ]; then
            warn "Backing up existing ${binary} to ${binary}.bak"
            mv "${install_dir}/${binary}" "${install_dir}/${binary}.bak"
        fi
    done

    # Install binaries
    cp "${extract_dir}/clawmon-daemon" "${install_dir}/clawmon-daemon"
    cp "${extract_dir}/clawmon-tui" "${install_dir}/clawmon-tui"
    chmod +x "${install_dir}/clawmon-daemon" "${install_dir}/clawmon-tui"
    ok "Installed binaries to ${install_dir}"

    # Verify
    if "${install_dir}/clawmon-daemon" --version >/dev/null 2>&1 && \
       "${install_dir}/clawmon-tui" -version >/dev/null 2>&1; then
        ok "Binaries verified"
    else
        warn "Could not verify binaries — they may not run on this platform"
    fi

    # Create default config
    if [ ! -f "${CONFIG_FILE}" ]; then
        mkdir -p "${CONFIG_DIR}"
        if [ -f "${extract_dir}/config.example.toml" ]; then
            cp "${extract_dir}/config.example.toml" "${CONFIG_FILE}"
            ok "Created default config at ${CONFIG_FILE}"
        fi
    else
        info "Config already exists at ${CONFIG_FILE} (not overwriting)"
    fi

    # Check PATH
    ensure_in_path "${install_dir}"

    # Print quick start
    printf "\n${GREEN}${BOLD}Installation complete!${RESET}\n\n"
    printf "Quick start:\n"
    printf "  ${BOLD}1.${RESET} Start the daemon:  ${BOLD}clawmon-daemon${RESET}\n"
    printf "  ${BOLD}2.${RESET} Open the TUI:      ${BOLD}clawmon-tui${RESET}\n"
    printf "\n"
    printf "Config: ${CONFIG_FILE}\n"
    printf "Docs:   https://github.com/${REPO}\n\n"

    # Offer systemd service
    if command -v systemctl >/dev/null 2>&1 && [ "$(id -u)" -ne 0 ]; then
        printf "Would you like to create a systemd user service for auto-start? [y/N] "
        read -r answer </dev/tty 2>/dev/null || answer="n"
        if [ "${answer}" = "y" ] || [ "${answer}" = "Y" ]; then
            create_systemd_service "${install_dir}"
        fi
    fi
}

create_systemd_service() {
    local install_dir="$1"
    local service_dir="${HOME}/.config/systemd/user"
    local service_file="${service_dir}/clawmon-daemon.service"

    mkdir -p "${service_dir}"

    cat > "${service_file}" << SYSTEMD
[Unit]
Description=clawmon daemon - OpenClaw agent monitor
After=network.target

[Service]
ExecStart=${install_dir}/clawmon-daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
SYSTEMD

    systemctl --user daemon-reload
    systemctl --user enable clawmon-daemon
    systemctl --user start clawmon-daemon

    ok "Systemd service created and started"
    info "Manage with: systemctl --user {start|stop|status} clawmon-daemon"
}

main "$@"
