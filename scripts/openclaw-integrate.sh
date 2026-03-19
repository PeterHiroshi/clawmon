#!/usr/bin/env bash
# clawmon OpenClaw integration script
# Usage:
#   openclaw-integrate.sh --mode bare-metal [--openclaw-workspace PATH]
#   openclaw-integrate.sh --mode docker [--container-name NAME]
#
# Integrates clawmon with an OpenClaw installation, either bare metal or Docker.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_DIR="${HOME}/.clawmon"
CONFIG_FILE="${CONFIG_DIR}/config.toml"
DEFAULT_WORKSPACE="${HOME}/.openclaw/workspace-main"

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

# --- Argument parsing ---

MODE=""
OPENCLAW_WS=""
CONTAINER_NAME="openclaw"

usage() {
    cat <<USAGE
Usage: $(basename "$0") --mode <bare-metal|docker> [OPTIONS]

Modes:
  bare-metal    OpenClaw installed directly on this host
  docker        OpenClaw running in a Docker container

Options:
  --openclaw-workspace PATH   Workspace path (bare-metal mode, default: auto-detect)
  --container-name NAME       Docker container name (docker mode, default: openclaw)
  -h, --help                  Show this help

Examples:
  $(basename "$0") --mode bare-metal
  $(basename "$0") --mode bare-metal --openclaw-workspace ~/.openclaw/workspace-main
  $(basename "$0") --mode docker --container-name my-openclaw
USAGE
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --mode)
                MODE="$2"
                shift 2
                ;;
            --openclaw-workspace)
                OPENCLAW_WS="$2"
                shift 2
                ;;
            --container-name)
                CONTAINER_NAME="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    if [ -z "${MODE}" ]; then
        error "Mode is required (--mode bare-metal or --mode docker)"
        usage
        exit 1
    fi

    case "${MODE}" in
        bare-metal|docker) ;;
        *)
            error "Invalid mode: ${MODE} (must be bare-metal or docker)"
            exit 1
            ;;
    esac
}

# --- Binary location ---

find_binary() {
    local name="$1"
    local locations=(
        "${HOME}/.local/bin/${name}"
        "/usr/local/bin/${name}"
        "${SCRIPT_DIR}/${name}"
    )

    for loc in "${locations[@]}"; do
        if [ -x "${loc}" ]; then
            echo "${loc}"
            return 0
        fi
    done

    # Try PATH
    if command -v "${name}" >/dev/null 2>&1; then
        command -v "${name}"
        return 0
    fi

    return 1
}

# --- Workspace detection ---

detect_workspace() {
    # Explicit argument
    if [ -n "${OPENCLAW_WS}" ]; then
        if [ -d "${OPENCLAW_WS}" ]; then
            echo "${OPENCLAW_WS}"
            return 0
        fi
        error "Specified workspace does not exist: ${OPENCLAW_WS}"
        return 1
    fi

    # Environment variable
    if [ -n "${OPENCLAW_WORKSPACE:-}" ] && [ -d "${OPENCLAW_WORKSPACE}" ]; then
        echo "${OPENCLAW_WORKSPACE}"
        return 0
    fi

    # Common paths
    local common_paths=(
        "${HOME}/.openclaw/workspace-main"
        "${HOME}/.openclaw/workspace"
        "/opt/openclaw/workspace-main"
    )

    for path in "${common_paths[@]}"; do
        if [ -d "${path}" ]; then
            echo "${path}"
            return 0
        fi
    done

    error "Could not auto-detect OpenClaw workspace"
    error "Specify with: --openclaw-workspace PATH"
    return 1
}

# --- Bare Metal Mode ---

integrate_bare_metal() {
    printf "\n${BOLD}clawmon - Bare Metal Integration${RESET}\n\n"

    # Find binaries
    local daemon_bin tui_bin workspace
    daemon_bin=$(find_binary "clawmon-daemon") || {
        error "clawmon-daemon not found. Install first:"
        error "  curl -fsSL https://raw.githubusercontent.com/PeterHiroshi/clawmon/main/scripts/install.sh | bash"
        exit 1
    }
    tui_bin=$(find_binary "clawmon-tui") || {
        error "clawmon-tui not found. Install first."
        exit 1
    }

    ok "Found daemon: ${daemon_bin}"
    ok "Found TUI: ${tui_bin}"

    # Detect workspace
    workspace=$(detect_workspace)
    ok "Workspace: ${workspace}"

    # Create/update config
    if [ ! -f "${CONFIG_FILE}" ]; then
        mkdir -p "${CONFIG_DIR}"
        cat > "${CONFIG_FILE}" << TOML
[daemon]
port = 9876
poll_interval = 30

[daemon.workspaces]
paths = [
    "${workspace}",
]

[tui]
daemon_url = "http://127.0.0.1:9876"
refresh_interval = 5
theme = "dark"
TOML
        ok "Created config at ${CONFIG_FILE}"
    else
        info "Config already exists at ${CONFIG_FILE}"
    fi

    # Create systemd user service
    if command -v systemctl >/dev/null 2>&1; then
        local service_dir="${HOME}/.config/systemd/user"
        local service_file="${service_dir}/clawmon-daemon.service"

        mkdir -p "${service_dir}"

        cat > "${service_file}" << SYSTEMD
[Unit]
Description=clawmon daemon - OpenClaw agent monitor
After=network.target

[Service]
ExecStart=${daemon_bin} --workspace ${workspace}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
SYSTEMD

        systemctl --user daemon-reload
        systemctl --user enable --now clawmon-daemon 2>/dev/null || {
            warn "Could not start systemd service (may need 'loginctl enable-linger')"
            warn "Start manually: ${daemon_bin} --workspace ${workspace}"
        }
        ok "Systemd service created: clawmon-daemon"
        info "Manage with: systemctl --user {start|stop|status|restart} clawmon-daemon"
    else
        warn "systemd not available — start daemon manually:"
        warn "  ${daemon_bin} --workspace ${workspace} &"
    fi

    # Add shell alias
    local shell_rc=""
    if [ -f "${HOME}/.bashrc" ]; then
        shell_rc="${HOME}/.bashrc"
    elif [ -f "${HOME}/.zshrc" ]; then
        shell_rc="${HOME}/.zshrc"
    fi

    if [ -n "${shell_rc}" ]; then
        if ! grep -q 'alias clawmon=' "${shell_rc}" 2>/dev/null; then
            printf '\n# clawmon TUI alias\nalias clawmon="%s"\n' "${tui_bin}" >> "${shell_rc}"
            ok "Added alias 'clawmon' to ${shell_rc}"
        else
            info "Alias 'clawmon' already exists in ${shell_rc}"
        fi
    fi

    printf "\n${GREEN}${BOLD}Integration complete!${RESET}\n\n"
    printf "Run ${BOLD}clawmon${RESET} (or ${BOLD}${tui_bin}${RESET}) to open the dashboard.\n\n"
}

# --- Docker Mode ---

integrate_docker() {
    printf "\n${BOLD}clawmon - Docker Integration${RESET}\n\n"

    # Check docker is available
    if ! command -v docker >/dev/null 2>&1; then
        error "Docker not found. Is it installed?"
        exit 1
    fi

    # Find binaries on host
    local daemon_bin tui_bin
    daemon_bin=$(find_binary "clawmon-daemon") || {
        error "clawmon-daemon not found on host. Install first:"
        error "  curl -fsSL https://raw.githubusercontent.com/PeterHiroshi/clawmon/main/scripts/install.sh | bash"
        exit 1
    }
    tui_bin=$(find_binary "clawmon-tui") || {
        error "clawmon-tui not found on host. Install first."
        exit 1
    }

    ok "Found daemon: ${daemon_bin}"
    ok "Found TUI: ${tui_bin}"

    # Check container exists and is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        # Check if container exists but is stopped
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            error "Container '${CONTAINER_NAME}' exists but is not running"
            error "Start it first: docker start ${CONTAINER_NAME}"
        else
            error "Container '${CONTAINER_NAME}' not found"
            error "List running containers: docker ps --format '{{.Names}}'"
        fi
        exit 1
    fi

    ok "Found running container: ${CONTAINER_NAME}"

    # Copy binaries into container
    info "Copying binaries into container..."
    docker cp "${daemon_bin}" "${CONTAINER_NAME}:/usr/local/bin/clawmon-daemon"
    docker cp "${tui_bin}" "${CONTAINER_NAME}:/usr/local/bin/clawmon-tui"
    docker exec "${CONTAINER_NAME}" chmod +x /usr/local/bin/clawmon-daemon /usr/local/bin/clawmon-tui
    ok "Binaries copied to container"

    # Detect workspace inside container
    local container_workspace
    container_workspace=$(docker exec "${CONTAINER_NAME}" bash -c '
        if [ -n "${OPENCLAW_WORKSPACE:-}" ] && [ -d "${OPENCLAW_WORKSPACE}" ]; then
            echo "${OPENCLAW_WORKSPACE}"
        elif [ -d "${HOME}/.openclaw/workspace-main" ]; then
            echo "${HOME}/.openclaw/workspace-main"
        elif [ -d "/root/.openclaw/workspace-main" ]; then
            echo "/root/.openclaw/workspace-main"
        else
            echo ""
        fi
    ' 2>/dev/null || echo "")

    if [ -z "${container_workspace}" ]; then
        warn "Could not auto-detect workspace inside container"
        container_workspace="/root/.openclaw/workspace-main"
        warn "Using default: ${container_workspace}"
    else
        ok "Detected workspace in container: ${container_workspace}"
    fi

    # Create config inside container
    info "Creating config inside container..."
    docker exec "${CONTAINER_NAME}" mkdir -p /root/.clawmon
    docker exec "${CONTAINER_NAME}" bash -c "cat > /root/.clawmon/config.toml << 'TOML'
[daemon]
port = 9876
poll_interval = 30

[daemon.workspaces]
paths = [\"${container_workspace}\"]

[tui]
daemon_url = \"http://127.0.0.1:9876\"
refresh_interval = 5
theme = \"dark\"
TOML"
    ok "Config created in container"

    # Start daemon in container (background, no systemd)
    info "Starting daemon in container..."
    docker exec "${CONTAINER_NAME}" bash -c 'pkill clawmon-daemon 2>/dev/null || true'
    docker exec -d "${CONTAINER_NAME}" /usr/local/bin/clawmon-daemon --workspace "${container_workspace}"
    ok "Daemon started in container"

    # Verify daemon is running
    sleep 1
    if docker exec "${CONTAINER_NAME}" pgrep -f clawmon-daemon >/dev/null 2>&1; then
        ok "Daemon is running"
    else
        warn "Daemon may not have started — check container logs"
    fi

    printf "\n${GREEN}${BOLD}Integration complete!${RESET}\n\n"
    printf "Access the dashboard:\n\n"
    printf "  ${BOLD}Option A (recommended):${RESET} Run TUI on host with port forwarding\n"
    printf "    If container port 9876 is exposed:\n"
    printf "      ${BOLD}${tui_bin}${RESET}\n\n"
    printf "  ${BOLD}Option B:${RESET} Run TUI inside container\n"
    printf "      ${BOLD}docker exec -it ${CONTAINER_NAME} clawmon-tui${RESET}\n\n"
    printf "Note: If the container resets, re-run this script to reinstall.\n"
    printf "Quick re-install: ${BOLD}$(basename "$0") --mode docker --container-name ${CONTAINER_NAME}${RESET}\n\n"
}

# --- Main ---

parse_args "$@"

case "${MODE}" in
    bare-metal) integrate_bare_metal ;;
    docker)     integrate_docker ;;
esac
