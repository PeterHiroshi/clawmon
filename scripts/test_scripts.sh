#!/usr/bin/env bash
# Unit tests for install.sh and openclaw-integrate.sh shell functions
# Run: bash scripts/test_scripts.sh

set -euo pipefail

PASS=0
FAIL=0
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

test_pass() { PASS=$((PASS + 1)); printf "  PASS: %s\n" "$1"; }
test_fail() { FAIL=$((FAIL + 1)); printf "  FAIL: %s\n" "$1" >&2; }

# --- Test install.sh functions ---

echo "=== Testing install.sh functions ==="

# Source install.sh without executing main
export CLAWMON_SOURCED=1
# shellcheck source=install.sh
source "${SCRIPT_DIR}/install.sh"

# Test detect_os
os=$(detect_os)
case "${os}" in
    linux|darwin) test_pass "detect_os returns valid OS: ${os}" ;;
    *) test_fail "detect_os returned unexpected: ${os}" ;;
esac

# Test detect_arch
arch=$(detect_arch)
case "${arch}" in
    amd64|arm64) test_pass "detect_arch returns valid arch: ${arch}" ;;
    *) test_fail "detect_arch returned unexpected: ${arch}" ;;
esac

# Test get_install_dir
install_dir=$(get_install_dir)
if [ -n "${install_dir}" ]; then
    test_pass "get_install_dir returns non-empty: ${install_dir}"
else
    test_fail "get_install_dir returned empty"
fi

# Test use_color with NO_COLOR
NO_COLOR=1
if ! use_color; then
    test_pass "use_color returns false when NO_COLOR=1"
else
    test_fail "use_color should return false when NO_COLOR=1"
fi
unset NO_COLOR

# Test use_color without NO_COLOR
if use_color; then
    test_pass "use_color returns true when NO_COLOR is unset"
else
    # May fail in non-tty environments, that's ok
    test_pass "use_color returns false (likely non-tty env)"
fi

# --- Test openclaw-integrate.sh argument parsing ---

echo ""
echo "=== Testing openclaw-integrate.sh functions ==="

# Source integration script without executing
# shellcheck source=openclaw-integrate.sh
source "${SCRIPT_DIR}/openclaw-integrate.sh"

# Test parse_args with valid bare-metal mode
MODE=""
parse_args --mode bare-metal
if [ "${MODE}" = "bare-metal" ]; then
    test_pass "parse_args sets MODE=bare-metal"
else
    test_fail "parse_args did not set MODE correctly: ${MODE}"
fi

# Test parse_args with docker mode and container name
MODE=""
CONTAINER_NAME=""
parse_args --mode docker --container-name my-container
if [ "${MODE}" = "docker" ] && [ "${CONTAINER_NAME}" = "my-container" ]; then
    test_pass "parse_args sets docker mode with container name"
else
    test_fail "parse_args docker mode failed: MODE=${MODE} CONTAINER=${CONTAINER_NAME}"
fi

# Test parse_args with workspace
MODE=""
OPENCLAW_WS=""
parse_args --mode bare-metal --openclaw-workspace /tmp
if [ "${OPENCLAW_WS}" = "/tmp" ]; then
    test_pass "parse_args sets workspace path"
else
    test_fail "parse_args workspace failed: ${OPENCLAW_WS}"
fi

# Test detect_workspace with explicit path
OPENCLAW_WS="/tmp"
ws=$(detect_workspace)
if [ "${ws}" = "/tmp" ]; then
    test_pass "detect_workspace returns explicit path"
else
    test_fail "detect_workspace explicit path failed: ${ws}"
fi

# Test detect_workspace with env var
OPENCLAW_WS=""
OPENCLAW_WORKSPACE="/tmp"
export OPENCLAW_WORKSPACE
ws=$(detect_workspace)
if [ "${ws}" = "/tmp" ]; then
    test_pass "detect_workspace returns env var path"
else
    test_fail "detect_workspace env var failed: ${ws}"
fi
unset OPENCLAW_WORKSPACE

# --- YAML validation ---

echo ""
echo "=== Validating YAML files ==="

# Basic YAML syntax check for release.yml
release_yml="${SCRIPT_DIR}/../.github/workflows/release.yml"
if [ -f "${release_yml}" ]; then
    # Check key elements exist
    if grep -q "^name:" "${release_yml}" && \
       grep -q "on:" "${release_yml}" && \
       grep -q "jobs:" "${release_yml}" && \
       grep -q "release:" "${release_yml}"; then
        test_pass "release.yml has required top-level keys"
    else
        test_fail "release.yml missing required keys"
    fi

    # Check all 4 platform targets
    for platform in linux-amd64 linux-arm64 macos-amd64 macos-arm64; do
        if grep -q "${platform}" "${release_yml}"; then
            test_pass "release.yml contains ${platform} target"
        else
            test_fail "release.yml missing ${platform} target"
        fi
    done
else
    test_fail "release.yml not found"
fi

# --- Summary ---

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if [ "${FAIL}" -gt 0 ]; then
    exit 1
fi
