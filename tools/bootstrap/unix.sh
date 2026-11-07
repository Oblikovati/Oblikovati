#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# shellcheck disable=SC2153
# BOOTSTRAP_ROOT is supplied by build.sh before this module is sourced.
# Bash 3.2 is intentional: it is the default shell shipped by macOS.
# shellcheck source=tools/bootstrap/unix_packages.sh
source "$BOOTSTRAP_ROOT/tools/bootstrap/unix_packages.sh"
# shellcheck source=tools/bootstrap/unix_toolchain.sh
source "$BOOTSTRAP_ROOT/tools/bootstrap/unix_toolchain.sh"
# shellcheck source=tools/bootstrap/unix_build.sh
source "$BOOTSTRAP_ROOT/tools/bootstrap/unix_build.sh"

bootstrap_fail() {
    printf 'build.sh: %s\n' "$*" >&2
    return 1
}
bootstrap_has() { command -v "$1" >/dev/null 2>&1; }
bootstrap_usage() {
    printf '%s\n' 'Usage: ./build.sh [build|run [-- GUI_ARGUMENTS...]|--check|--help]' \
        'build installs missing dependencies and calls cargo xtask build.' \
        'run reuses a verified fresh GUI, or builds first; it needs a real desktop display.' \
        '--check reports missing build dependencies without installing or compiling.'
}
bootstrap_cleanup() {
    if [[ -n ${BOOTSTRAP_XVFB_PID:-} ]]; then
        kill "$BOOTSTRAP_XVFB_PID" 2>/dev/null || true
        wait "$BOOTSTRAP_XVFB_PID" 2>/dev/null || true
    fi
    [[ -z ${BOOTSTRAP_TEMP:-} ]] || rm -rf -- "$BOOTSTRAP_TEMP"
}
bootstrap_paths() {
    export PATH="$BOOTSTRAP_ROOT/.tools:${CARGO_HOME:-$HOME/.cargo}/bin:$PATH:/opt/homebrew/bin:/usr/local/bin:/opt/local/bin"
    CARGO_TARGET_DIR=${CARGO_TARGET_DIR:-$BOOTSTRAP_ROOT/target}
    [[ $CARGO_TARGET_DIR = /* ]] || CARGO_TARGET_DIR="$BOOTSTRAP_ROOT/$CARGO_TARGET_DIR"
    CARGO_TARGET_DIR=$(bootstrap_absolute_target "$CARGO_TARGET_DIR") || return
    [[ $CARGO_TARGET_DIR != / && $CARGO_TARGET_DIR != "$BOOTSTRAP_ROOT" ]] || {
        bootstrap_fail "target directory $CARGO_TARGET_DIR: expected a separate build output directory"
        return 1
    }
    export CARGO_TARGET_DIR
    BOOTSTRAP_GUI="$CARGO_TARGET_DIR/debug/oblikovati"
    BOOTSTRAP_STAMP="$BOOTSTRAP_ROOT/.tools/bootstrap-$(printf '%s' "$BOOTSTRAP_GUI" | bootstrap_digest).sha256"
}
bootstrap_absolute_target() {
    local item pieces=() stack=()
    IFS=/ read -r -a pieces <<<"$1"
    for item in "${pieces[@]}"; do
        case $item in
        '' | .) ;;
        ..) [[ ${#stack[@]} = 0 ]] || unset "stack[$((${#stack[@]} - 1))]" ;;
        *) stack+=("$item") ;;
        esac
    done
    local IFS=/
    printf '/%s\n' "${stack[*]-}"
}
bootstrap_platform() {
    BOOTSTRAP_OS=$(uname -s)
    BOOTSTRAP_ARCH=$(uname -m)
    case $BOOTSTRAP_OS in Linux | Darwin) ;; *)
        bootstrap_fail "OS $BOOTSTRAP_OS: expected Linux or macOS; Windows uses build.ps1"
        return 1
        ;;
    esac
    case $BOOTSTRAP_ARCH in x86_64 | amd64) BOOTSTRAP_ARCH=x86_64 ;; aarch64 | arm64) BOOTSTRAP_ARCH=aarch64 ;; *)
        bootstrap_fail "architecture $BOOTSTRAP_ARCH: expected x86_64 or arm64"
        return 1
        ;;
    esac
    [[ -z ${CARGO_BUILD_TARGET:-} ]] || {
        bootstrap_fail "CARGO_BUILD_TARGET=$CARGO_BUILD_TARGET: expected unset for a native desktop build"
        return 1
    }
}
bootstrap_desktop() {
    [[ $BOOTSTRAP_OS != Linux || -n ${DISPLAY:-} ]] && return 0
    bootstrap_fail 'run requires DISPLAY for the X11/XWayland desktop backend; WAYLAND_DISPLAY alone is not sufficient, and a GUI is never launched in a temporary Xvfb'
}
bootstrap_check() {
    bootstrap_native_ready || {
        bootstrap_fail "missing $BOOTSTRAP_OS build tools/libraries; run ./build.sh to install them"
        return 1
    }
    bootstrap_rust_ready || {
        bootstrap_fail 'missing the pinned Rust toolchain/components; run ./build.sh to install them'
        return 1
    }
    bootstrap_cache_ready || {
        bootstrap_fail 'missing the pinned build cache; run ./build.sh to install it'
        return 1
    }
    printf '%s\n' 'Build dependencies are installed.'
}
bootstrap_main() {
    local mode=${1:-build}
    [[ $# = 0 ]] || shift
    case $mode in --help | -h)
        bootstrap_usage
        return 0
        ;;
    build | --check) [[ $# = 0 ]] || {
        bootstrap_usage >&2
        return 2
    } ;; run) [[ ${1:-} != -- ]] || shift ;; *)
        bootstrap_usage >&2
        return 2
        ;;
    esac
    cd -- "$BOOTSTRAP_ROOT" || return
    bootstrap_platform || return
    bootstrap_paths || return
    [[ $mode != --check ]] || {
        bootstrap_check
        return
    }
    if [[ $mode = run ]]; then bootstrap_desktop || return; fi
    if [[ $mode = run ]] && bootstrap_fresh; then
        "$BOOTSTRAP_GUI" "$@"
        return $?
    fi
    BOOTSTRAP_TEMP=$(mktemp -d "${TMPDIR:-/tmp}/oblikovati-build.XXXXXXXX")
    BOOTSTRAP_XVFB_PID=
    trap bootstrap_cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    bootstrap_dependencies || return
    bootstrap_compile || return
    [[ $mode != run ]] || "$BOOTSTRAP_GUI" "$@"
}
