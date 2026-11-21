#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# Run the native Main Window Frame scenarios in a private Linux desktop session.
set -euo pipefail

script=$(realpath "$0")
script_dir=$(dirname "$script")

if [[ "${1-}" != --private-session ]]; then
    cd "$script_dir/../../../.."
    # Keep the accessibility setting in the private session, away from user preferences.
    export GSETTINGS_BACKEND=memory
    exec tools/with-xvfb.sh dbus-run-session -- "$script" --private-session
fi

find_registry() {
    local registry
    for registry in /usr/libexec/at-spi2-registryd /usr/lib/at-spi2-core/at-spi2-registryd; do
        if [[ -x "$registry" ]]; then
            printf '%s\n' "$registry"
            return
        fi
    done
    printf '%s\n' 'AT-SPI registry: expected an installed at-spi2-core registry executable' >&2
    return 1
}

accessibility_bus() {
    gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus "$@"
}

wait_for_desktop() {
    local attempt
    for ((attempt=0; attempt<80; attempt++)); do
        if xprop -root _NET_SUPPORTING_WM_CHECK | grep 'window id' >/dev/null &&
           xwininfo -root -tree | grep '"tint2"' >/dev/null; then
            return
        fi
        sleep .1
    done
    printf '%s\n' 'Xvfb desktop: expected the Openbox window manager and tint2 taskbar' >&2
    return 1
}

registry=$(find_registry)
address=$(accessibility_bus --method org.a11y.Bus.GetAddress)
export AT_SPI_BUS_ADDRESS
AT_SPI_BUS_ADDRESS=$(printf '%s\n' "$address" | cut -d "'" -f2)
# AccessKit follows this property. A fresh D-Bus session can start disabled.
accessibility_bus --method org.freedesktop.DBus.Properties.Set org.a11y.Status IsEnabled '<true>' >/dev/null
accessibility_bus --method org.freedesktop.DBus.Properties.Get org.a11y.Status IsEnabled | grep '<true>' >/dev/null

"$registry" --use-gnome-session &
registry_pid=$!
openbox &
manager_pid=$!
tint2 -c "$script_dir/frame_tint2rc" &
taskbar_pid=$!
trap 'kill "$taskbar_pid" "$manager_pid" "$registry_pid" 2>/dev/null || true; wait "$taskbar_pid" "$manager_pid" "$registry_pid" 2>/dev/null || true' EXIT
wait_for_desktop

export OBK_BDD_FEATURE=main-window-frame
cargo test --locked --offline -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture --test-threads=1
