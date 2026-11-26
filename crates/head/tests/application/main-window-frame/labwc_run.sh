#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# Run native Wayland frame scenarios with a private Labwc session.
set -euo pipefail

script=$(realpath "$0")
if [[ "${1-}" != --private-session ]]; then
    fixtures=$(dirname "$script")
    cd "$fixtures/../../../../.."
    bus_options=()
    if [[ "${OBK_BDD_LABWC_TASKBAR:-0}" == 1 ]]; then
        bus_options+=(--config-file="$fixtures/waybar-session.conf")
    fi
    exec tools/with-xvfb.sh dbus-run-session "${bus_options[@]}" -- bash "$script" --private-session
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
EOF

if [[ "${OBK_BDD_LABWC_TASKBAR:-0}" == 1 ]]; then
    cat >> "$config" <<'EOF'
  <windowRules>
    <windowRule identifier="org.oblikovati.Oblikovati" serverDecoration="no">
      <action name="MoveTo" x="0" y="0" />
      <action name="ResizeTo" width="960" height="680" />
    </windowRule>
  </windowRules>
EOF
fi
printf '%s\n' '</labwc_config>' >> "$config"

env -u WAYLAND_DEBUG WLR_BACKENDS=x11 WLR_RENDERER=pixman \
    "${LABWC_BIN:-labwc}" --config "$config" &
compositor=$!
cleanup() {
    if [[ -n "${panel:-}" ]]; then
        kill "$panel" 2>/dev/null || true
        wait "$panel" 2>/dev/null || true
    fi
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

if [[ "${OBK_BDD_LABWC_TASKBAR:-0}" == 1 ]]; then
    fixtures=$(dirname "$script")
    env GDK_BACKEND=wayland GTK_USE_PORTAL=0 GIO_USE_VFS=local NO_AT_BRIDGE=1 \
        XDG_CURRENT_DESKTOP=labwc "${WAYBAR_BIN:-waybar}" \
        --config "$fixtures/waybar-taskbar.json" --style "$fixtures/waybar-taskbar.css" &
    panel=$!
fi

export OBK_BDD_AREA=application
export OBK_BDD_FEATURE="${OBK_BDD_FEATURE:-main-window-frame}"
export OBK_BDD_SCENARIO="${OBK_BDD_SCENARIO:-Keep the window Close action available to the desktop}"
cargo test --locked --offline -p oblikovati-head --lib application_scenarios_use_the_native_window -- \
    --nocapture --test-threads=1
