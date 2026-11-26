#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# Exercise a desktop Close shortcut with a private native Wayland compositor.
set -euo pipefail

script=$(realpath "$0")
if [[ "${1-}" != --private-session ]]; then
    cd "$(dirname "$script")/../../../../.."
    exec tools/with-xvfb.sh dbus-run-session -- bash "$script" --private-session
fi

session=$(mktemp -d)
export XDG_RUNTIME_DIR="$session/runtime" XDG_CONFIG_HOME="$session/config"
mkdir -m 700 "$XDG_RUNTIME_DIR" "$XDG_CONFIG_HOME"
config="$XDG_CONFIG_HOME/rc.xml"
cat > "$config" <<'EOF'
<?xml version="1.0"?>
<labwc_config>
  <core><xwayland>false</xwayland></core>
  <keyboard>
    <keybind key="A-F4"><action name="Close" /></keybind>
  </keyboard>
</labwc_config>
EOF

env -u WAYLAND_DEBUG WLR_BACKENDS=x11 WLR_RENDERER=pixman \
    "${LABWC_BIN:-labwc}" --config "$config" &
compositor=$!
cleanup() {
    kill "$compositor" 2>/dev/null || true
    wait "$compositor" 2>/dev/null || true
    rm -rf -- "$session"
}
trap cleanup EXIT

export WAYLAND_DISPLAY=wayland-0 OBK_BDD_LABWC_NESTED_X11=1
for ((attempt=0; attempt<60; attempt++)); do
    [[ ! -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ]] || break
    kill -0 "$compositor"
    sleep .05
done
test -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY"

export OBK_BDD_AREA=application
export OBK_BDD_FEATURE="${OBK_BDD_FEATURE:-main-window-frame}"
export OBK_BDD_SCENARIO="${OBK_BDD_SCENARIO:-Keep the window Close action available to the desktop}"
cargo test --locked --offline -p oblikovati-head --lib application_scenarios_use_the_native_window -- \
    --nocapture --test-threads=1
