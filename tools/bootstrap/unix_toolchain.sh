#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# shellcheck disable=SC2153
# BOOTSTRAP_ROOT is supplied by build.sh before this module is sourced.
bootstrap_digest() {
    if bootstrap_has sha256sum; then
        sha256sum | awk '{print $1}'
    elif bootstrap_has shasum; then
        shasum -a 256 | awk '{print $1}'
    elif bootstrap_has openssl; then
        openssl dgst -sha256 -r | awk '{print $1}'
    else
        bootstrap_fail 'SHA-256 tools missing: expected sha256sum, shasum, or openssl before bootstrap'
        return 1
    fi
}
bootstrap_file_digest() { bootstrap_digest <"$1"; }
bootstrap_download() {
    local key=$1 output=$2 digest url actual
    digest=$(awk -v key="$key" '$1 == key {print $2}' "$BOOTSTRAP_ROOT/tools/bootstrap/downloads.lock")
    url=$(awk -v key="$key" '$1 == key {print $3}' "$BOOTSTRAP_ROOT/tools/bootstrap/downloads.lock")
    [[ $url = https://* && $digest =~ ^[a-f0-9]{64}$ ]] || {
        bootstrap_fail "download $key: expected a locked HTTPS URL and SHA-256"
        return 1
    }
    curl --fail --location --proto '=https' --proto-redir '=https' --tlsv1.2 --output "$output.part" "$url" || return
    actual=$(bootstrap_file_digest "$output.part")
    [[ $actual = "$digest" ]] || {
        bootstrap_fail "download $key SHA-256 $actual: expected $digest"
        return 1
    }
    mv -- "$output.part" "$output"
}
bootstrap_rust_pin() {
    BOOTSTRAP_RUST=$(sed -n 's/^channel = "\([^"]*\)"$/\1/p' "$BOOTSTRAP_ROOT/rust-toolchain.toml")
    [[ $BOOTSTRAP_RUST =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
        bootstrap_fail "Rust channel $BOOTSTRAP_RUST: expected a pinned semantic version"
        return 1
    }
    BOOTSTRAP_COMPONENTS=()
    while IFS= read -r component; do BOOTSTRAP_COMPONENTS+=("$component"); done < <(
        sed -n 's/^components = \[\(.*\)\]$/\1/p' "$BOOTSTRAP_ROOT/rust-toolchain.toml" | tr ',' '\n' | tr -d '" '
    )
    [[ ${#BOOTSTRAP_COMPONENTS[@]} -gt 0 ]] || bootstrap_fail 'Rust components: expected a nonempty pinned component list'
}
bootstrap_rust_ready() {
    bootstrap_rust_pin || return
    bootstrap_has rustup || return 1
    rustup run "$BOOTSTRAP_RUST" rustc --version >/dev/null 2>&1 || return 1
    local installed component
    installed=$(rustup component list --toolchain "$BOOTSTRAP_RUST" --installed) || return
    for component in "${BOOTSTRAP_COMPONENTS[@]}"; do
        [[ $component != *-preview ]] || component=${component%-preview}
        printf '%s\n' "$installed" | grep -Eq "^${component}(-|$)" || return 1
    done
}
bootstrap_rust_host() {
    if [[ $BOOTSTRAP_OS = Darwin ]]; then
        printf '%s-apple-darwin\n' "$BOOTSTRAP_ARCH"
        return
    fi
    if ldd --version 2>&1 | grep -qi musl; then
        printf '%s-unknown-linux-musl\n' "$BOOTSTRAP_ARCH"
    else printf '%s-unknown-linux-gnu\n' "$BOOTSTRAP_ARCH"; fi
}
bootstrap_install_rust() {
    bootstrap_rust_ready && return 0
    if ! bootstrap_has rustup; then
        bootstrap_download "rustup-$(bootstrap_rust_host)" "$BOOTSTRAP_TEMP/rustup-init" || return
        chmod +x "$BOOTSTRAP_TEMP/rustup-init"
        "$BOOTSTRAP_TEMP/rustup-init" -y --no-modify-path --default-toolchain none || return
        hash -r
    fi
    local arguments=(toolchain install "$BOOTSTRAP_RUST" --profile minimal) component
    for component in "${BOOTSTRAP_COMPONENTS[@]}"; do arguments+=(--component "$component"); done
    rustup "${arguments[@]}" || return
    bootstrap_rust_ready || bootstrap_fail "Rust $BOOTSTRAP_RUST: installation did not provide all pinned components"
}
bootstrap_cache_ready() {
    [[ ${BUILD_CACHE:-kache} != off ]] || return 0
    local version
    version=$(sed -n 's/^const KACHE_VERSION: &str = "\([^"]*\)";$/\1/p' "$BOOTSTRAP_ROOT/tools/build/src/kache.rs")
    [[ -n $version ]] && bootstrap_has kache && [[ $(kache --version) = "kache $version" ]]
}
bootstrap_prepare_cache() {
    case ${BUILD_CACHE:-kache} in
    off)
        export RUSTC_WRAPPER=
        return 0
        ;;
    kache) ;;
    *)
        bootstrap_fail "BUILD_CACHE=$BUILD_CACHE: expected kache or off"
        return 1
        ;;
    esac
    if ! bootstrap_cache_ready; then
        RUSTC_WRAPPER='' cargo xtask cache-tools || return
        hash -r
    fi
    bootstrap_cache_ready || {
        bootstrap_fail 'cache-tools did not install the pinned kache executable'
        return 1
    }
    export RUSTC_WRAPPER=kache KACHE_CONFIG="$BOOTSTRAP_ROOT/.kache.toml"
    local common
    common=$(git rev-parse --path-format=absolute --git-common-dir) || return
    export KACHE_CACHE_DIR="${KACHE_CACHE_DIR:-$common/oblikovati-kache}"
}
bootstrap_dependencies() {
    bootstrap_native_ready || bootstrap_install_native || return
    bootstrap_native_ready || {
        bootstrap_fail "$BOOTSTRAP_OS dependencies remain missing after installation; check package-manager output"
        return 1
    }
    bootstrap_install_rust || return
    export RUSTUP_TOOLCHAIN="$BOOTSTRAP_RUST"
    bootstrap_prepare_cache
}
