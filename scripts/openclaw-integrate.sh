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

# --- Docker: Detect workspace inside container ---

detect_container_workspace() {
    local container="$1"

    # User-provided override
    if [ -n "${OPENCLAW_WS}" ]; then
        echo "${OPENCLAW_WS}"
        return 0
    fi

    # Auto-detect inside container
    local detected
    detected=$(docker exec "${container}" bash -c '
        if [ -n "${OPENCLAW_WORKSPACE:-}" ] && [ -d "${OPENCLAW_WORKSPACE}" ]; then
            echo "${OPENCLAW_WORKSPACE}"
        elif [ -d "${HOME}/.openclaw/workspace-main" ]; then
            echo "${HOME}/.openclaw/workspace-main"
        elif [ -d "/home/node/.openclaw/workspace-main" ]; then
            echo "/home/node/.openclaw/workspace-main"
        elif [ -d "/root/.openclaw/workspace-main" ]; then
            echo "/root/.openclaw/workspace-main"
        else
            echo ""
        fi
    ' 2>/dev/null || echo "")

    if [ -n "${detected}" ]; then
        echo "${detected}"
        return 0
    fi

    return 1
}

# --- Docker: Check port mapping ---

check_port_mapping() {
    local container="$1"
    local port="$2"

    if docker port "${container}" "${port}" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# --- Docker Mode ---

integrate_docker() {
    printf "\n${BOLD}clawmon - Docker Integration${RESET}\n\n"

    # Step 1: Check prerequisites on host
    if ! command -v docker >/dev/null 2>&1; then
        error "Docker not found. Is it installed?"
        exit 1
    fi

    local daemon_bin
    daemon_bin=$(find_binary "clawmon-daemon") || {
        error "clawmon-daemon not found on host. Install first:"
        error "  curl -fsSL https://raw.githubusercontent.com/PeterHiroshi/clawmon/main/scripts/install.sh | bash"
        exit 1
    }
    ok "Found daemon: ${daemon_bin}"

    # Check container exists and is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
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

    # Step 2: Copy daemon binary into container
    info "Copying daemon binary into container..."
    docker cp "${daemon_bin}" "${CONTAINER_NAME}:/usr/local/bin/clawmon-daemon"
    docker exec "${CONTAINER_NAME}" chmod +x /usr/local/bin/clawmon-daemon
    ok "Daemon binary copied to container"

    # Verify binary works in container
    if docker exec "${CONTAINER_NAME}" /usr/local/bin/clawmon-daemon --version >/dev/null 2>&1; then
        ok "Binary verified inside container"
    else
        warn "Could not verify daemon binary inside container (may need compatible build)"
    fi

    # Step 3: Auto-detect workspace inside container
    local container_workspace
    container_workspace=$(detect_container_workspace "${CONTAINER_NAME}") || {
        warn "Could not auto-detect workspace inside container"
        container_workspace="/home/node/.openclaw/workspace-main"
        warn "Using default: ${container_workspace}"
    }
    ok "Workspace: ${container_workspace}"

    # Step 4: Create config inside container with bind = "0.0.0.0"
    info "Creating config inside container..."
    local container_home
    container_home=$(docker exec "${CONTAINER_NAME}" bash -c 'echo "${HOME}"' 2>/dev/null || echo "/root")
    docker exec "${CONTAINER_NAME}" mkdir -p "${container_home}/.clawmon"
    docker exec "${CONTAINER_NAME}" bash -c "cat > ${container_home}/.clawmon/config.toml << 'TOML'
[daemon]
port = 9876
bind = \"0.0.0.0\"
mode = \"docker\"
poll_interval = 30

[daemon.workspaces]
paths = [\"${container_workspace}\"]

[tui]
daemon_url = \"http://127.0.0.1:9876\"
refresh_interval = 5
theme = \"dark\"
TOML"
    ok "Config created (bind = 0.0.0.0, mode = docker)"

    # Step 5: Start daemon inside container
    info "Starting daemon in container..."
    docker exec "${CONTAINER_NAME}" bash -c 'pkill -f clawmon-daemon 2>/dev/null || true'
    sleep 1
    docker exec -d "${CONTAINER_NAME}" bash -c "nohup /usr/local/bin/clawmon-daemon > /tmp/clawmon-daemon.log 2>&1 &"
    ok "Daemon start command issued"

    # Verify daemon started
    sleep 2
    local daemon_pid
    daemon_pid=$(docker exec "${CONTAINER_NAME}" pgrep -f clawmon-daemon 2>/dev/null || echo "")
    if [ -n "${daemon_pid}" ]; then
        ok "Daemon is running (PID: ${daemon_pid})"
    else
        warn "Daemon may not have started. Check logs:"
        warn "  docker exec ${CONTAINER_NAME} cat /tmp/clawmon-daemon.log"
    fi

    # Verify health endpoint from inside container
    if docker exec "${CONTAINER_NAME}" bash -c 'command -v curl >/dev/null 2>&1 && curl -sf http://127.0.0.1:9876/api/v1/health >/dev/null' 2>/dev/null; then
        ok "Health endpoint responding inside container"
    fi

    # Step 6: Check port exposure
    local port_mapped=false
    if check_port_mapping "${CONTAINER_NAME}" 9876; then
        port_mapped=true
        ok "Port 9876 is mapped from container"
    fi

    # Step 7: Create bootstrap hook for container restart recovery
    info "Creating bootstrap hook..."
    docker exec "${CONTAINER_NAME}" bash -c "cat > /usr/local/bin/clawmon-bootstrap.sh << 'BOOTSTRAP'
#!/bin/bash
# Auto-start clawmon-daemon after container restart
if command -v clawmon-daemon >/dev/null 2>&1; then
    nohup clawmon-daemon > /tmp/clawmon-daemon.log 2>&1 &
fi
BOOTSTRAP"
    docker exec "${CONTAINER_NAME}" chmod +x /usr/local/bin/clawmon-bootstrap.sh
    ok "Bootstrap hook created at /usr/local/bin/clawmon-bootstrap.sh"

    # Step 8: Print summary
    printf "\n${GREEN}${BOLD}clawmon Docker integration complete!${RESET}\n\n"
    if [ -n "${daemon_pid}" ]; then
        printf "  Daemon running in container: ${BOLD}${CONTAINER_NAME}${RESET} (PID: ${daemon_pid})\n"
    else
        printf "  Daemon container: ${BOLD}${CONTAINER_NAME}${RESET}\n"
    fi
    printf "  Listening on: ${BOLD}0.0.0.0:9876${RESET}\n"
    printf "  Workspace: ${BOLD}${container_workspace}${RESET}\n\n"

    if [ "${port_mapped}" = "true" ]; then
        printf "  To open dashboard:\n"
        printf "    ${BOLD}clawmon-tui${RESET}\n\n"
    else
        printf "  ${YELLOW}Port 9876 is not exposed from the container.${RESET}\n\n"
        printf "  ${BOLD}Option A: Recreate container with port mapping (recommended):${RESET}\n"
        printf "    Add to your docker run command: ${BOLD}-p 9876:9876${RESET}\n"
        printf "    Or add to docker-compose.yml:\n"
        printf "      ports:\n"
        printf "        - \"9876:9876\"\n\n"
        printf "  ${BOLD}Option B: Use socat for port forwarding (no container restart):${RESET}\n"
        printf "    On host: ${BOLD}socat TCP-LISTEN:9876,fork,reuseaddr EXEC:\"docker exec -i ${CONTAINER_NAME} socat - TCP:127.0.0.1:9876\"${RESET}\n"
        printf "    (Requires socat on both host and container)\n\n"
        printf "  ${BOLD}Option C: Run TUI inside container:${RESET}\n"
        printf "    ${BOLD}docker exec -it ${CONTAINER_NAME} clawmon-tui${RESET}\n\n"
    fi

    printf "  After container restart:\n"
    printf "    ${BOLD}$(basename "$0") --mode docker --container-name ${CONTAINER_NAME}${RESET}  (re-install)\n"
    printf "    Or add clawmon-bootstrap.sh to container entrypoint\n\n"
}

# --- Main ---

# Allow sourcing for testing without executing
if [ "${CLAWMON_SOURCED:-}" != "1" ]; then
    parse_args "$@"

    case "${MODE}" in
        bare-metal) integrate_bare_metal ;;
        docker)     integrate_docker ;;
    esac
fi
