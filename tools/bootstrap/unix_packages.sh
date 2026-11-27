#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
bootstrap_native_ready() {
    local executable
    for executable in git curl tar cc c++ ar make cmake ninja pkg-config; do bootstrap_has "$executable" || return 1; done
    if [[ $BOOTSTRAP_OS = Darwin ]]; then
        xcode-select -p >/dev/null 2>&1 && xcrun --find clang++ >/dev/null 2>&1 || return 1
        pkg-config --exists glfw3 vulkan || return 1
    else
        pkg-config --exists glfw3 vulkan x11 xrandr xcursor xinerama xi || return 1
        [[ -n ${DISPLAY:-} ]] || bootstrap_has Xvfb || return 1
    fi
}
bootstrap_manager() {
    local candidate
    for candidate in apt-get apt dnf microdnf yum zypper pacman apk xbps-install eopkg emerge rpm-ostree nix nix-env; do
        if bootstrap_has "$candidate"; then
            printf '%s\n' "$candidate"
            return
        fi
    done
    bootstrap_fail 'no package manager found: expected apt, dnf, yum, zypper, pacman, apk, xbps, eopkg, emerge, rpm-ostree, or Nix'
}
bootstrap_privileged() {
    if [[ $(id -u) = 0 ]]; then
        "$@"
        return
    fi
    bootstrap_has sudo || {
        bootstrap_fail "install command $1: expected root rights or sudo; install the dependencies as administrator and rerun"
        return 1
    }
    sudo -- "$@"
}
bootstrap_linux_packages() {
    case $1 in
    apt-get | apt) BOOTSTRAP_PACKAGES=(build-essential cmake ninja-build pkg-config git curl ca-certificates tar libglfw3-dev libvulkan-dev libxkbcommon-dev libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev xvfb) ;;
    dnf | microdnf | yum | rpm-ostree) BOOTSTRAP_PACKAGES=(gcc gcc-c++ make cmake ninja-build pkgconf-pkg-config git curl ca-certificates tar glfw-devel vulkan-loader-devel libxkbcommon-devel vulkan-headers libX11-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel xorg-x11-server-Xvfb) ;;
    zypper) BOOTSTRAP_PACKAGES=(gcc gcc-c++ make cmake ninja pkg-config git curl ca-certificates tar glfw-devel libvulkan-devel vulkan-headers libX11-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel xorg-x11-server-Xvfb) ;;
    pacman) BOOTSTRAP_PACKAGES=(base-devel cmake ninja pkgconf git curl ca-certificates tar glfw vulkan-headers vulkan-icd-loader libxkbcommon libx11 libxrandr libxcursor libxinerama libxi xorg-server-xvfb) ;;
    apk) BOOTSTRAP_PACKAGES=(build-base cmake ninja pkgconf git curl ca-certificates tar glfw-dev vulkan-loader-dev libxkbcommon-dev vulkan-headers libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev xorg-server-xvfb) ;;
    xbps-install) BOOTSTRAP_PACKAGES=(base-devel cmake ninja pkg-config git curl ca-certificates tar glfw-devel Vulkan-Headers Vulkan-Loader-devel libxkbcommon-devel libX11-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel xorg-server-xvfb) ;;
    eopkg) BOOTSTRAP_PACKAGES=(-c system.devel cmake ninja pkg-config git curl ca-certs tar glfw-devel vulkan-devel libxkbcommon-devel libx11-devel libxrandr-devel libxcursor-devel libxinerama-devel libxi-devel xorg-server-xvfb) ;;
    emerge) BOOTSTRAP_PACKAGES=(sys-devel/gcc dev-build/make dev-build/cmake dev-build/ninja dev-build/pkgconf dev-vcs/git net-misc/curl app-misc/ca-certificates app-arch/tar media-libs/glfw media-libs/vulkan-loader dev-util/vulkan-headers x11-libs/libxkbcommon x11-libs/libX11 x11-libs/libXrandr x11-libs/libXcursor x11-libs/libXinerama x11-libs/libXi x11-base/xorg-server) ;;
    nix | nix-env) BOOTSTRAP_PACKAGES=(gcc gnumake cmake ninja pkg-config git curl cacert gnutar glfw vulkan-loader vulkan-headers libxkbcommon xorg.libX11 xorg.libXrandr xorg.libXcursor xorg.libXinerama xorg.libXi xorg.xorgserver) ;;
    *)
        bootstrap_fail "package manager $1: expected a supported Linux manager"
        return 1
        ;;
    esac
}
bootstrap_install_linux() {
    local manager
    manager=$(bootstrap_manager) || return
    bootstrap_linux_packages "$manager" || return
    case $manager in
    apt-get | apt) bootstrap_privileged "$manager" update && bootstrap_privileged "$manager" install -y --no-install-recommends "${BOOTSTRAP_PACKAGES[@]}" ;;
    dnf | microdnf | yum) bootstrap_privileged "$manager" install -y "${BOOTSTRAP_PACKAGES[@]}" ;;
    zypper) bootstrap_privileged zypper --non-interactive install "${BOOTSTRAP_PACKAGES[@]}" ;;
    pacman) bootstrap_privileged pacman -S --needed --noconfirm "${BOOTSTRAP_PACKAGES[@]}" ;;
    apk) bootstrap_privileged apk add --no-cache "${BOOTSTRAP_PACKAGES[@]}" ;;
    xbps-install) bootstrap_privileged xbps-install -Sy "${BOOTSTRAP_PACKAGES[@]}" ;;
    eopkg) bootstrap_privileged eopkg install -y "${BOOTSTRAP_PACKAGES[@]}" ;;
    emerge) bootstrap_privileged emerge --noreplace "${BOOTSTRAP_PACKAGES[@]}" ;;
    rpm-ostree) bootstrap_privileged rpm-ostree install "${BOOTSTRAP_PACKAGES[@]}" && bootstrap_fail 'rpm-ostree changes require a reboot; restart the OS and rerun build.sh' ;;
    nix | nix-env) bootstrap_install_nix "$manager" ;;
    esac
}
bootstrap_install_nix() {
    local manager=$1 package arguments=()
    for package in "${BOOTSTRAP_PACKAGES[@]}"; do
        if [[ $manager = nix ]]; then arguments+=("nixpkgs#$package"); else arguments+=("nixpkgs.$package"); fi
    done
    if [[ $manager = nix ]]; then
        nix --extra-experimental-features 'nix-command flakes' profile install "${arguments[@]}"
    else nix-env -iA "${arguments[@]}"; fi
    export PATH="$HOME/.nix-profile/bin:$PATH"
    export PKG_CONFIG_PATH="$HOME/.nix-profile/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
}
bootstrap_macos_manager() {
    if bootstrap_has brew; then
        printf '%s\n' brew
        return
    fi
    if bootstrap_has port; then
        printf '%s\n' port
        return
    fi
    bootstrap_download homebrew "$BOOTSTRAP_TEMP/homebrew-install.sh" || return
    NONINTERACTIVE=1 /bin/bash "$BOOTSTRAP_TEMP/homebrew-install.sh" >&2 || return
    bootstrap_has brew || {
        bootstrap_fail 'Homebrew installer did not provide brew in /opt/homebrew/bin or /usr/local/bin'
        return 1
    }
    printf '%s\n' brew
}
bootstrap_install_macos() {
    if ! xcode-select -p >/dev/null 2>&1; then
        xcode-select --install || return
        bootstrap_fail 'Apple Command Line Tools installation was started; complete the OS installer and rerun build.sh'
        return 1
    fi
    local manager prefix
    manager=$(bootstrap_macos_manager) || return
    if [[ $manager = brew ]]; then
        brew install git curl cmake ninja pkg-config glfw vulkan-loader vulkan-headers libxkbcommon molten-vk || return
        prefix=$(brew --prefix) || return
    else
        bootstrap_privileged port install git curl cmake ninja pkgconfig glfw vulkan-loader Vulkan-Headers MoltenVK || return
        prefix=/opt/local
    fi
    export PKG_CONFIG_PATH="$prefix/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
}
bootstrap_install_native() {
    if [[ $BOOTSTRAP_OS = Darwin ]]; then bootstrap_install_macos; else bootstrap_install_linux; fi
}
