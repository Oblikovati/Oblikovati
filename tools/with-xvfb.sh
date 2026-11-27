#!/bin/sh
# SPDX-License-Identifier: GPL-2.0-only
set -eu

managed=0
if [ "${1:-}" = --window-manager ]; then
    managed=1
    shift
fi
if [ "$#" -eq 0 ]; then
    printf '%s\n' 'with-xvfb: expected a command and its arguments' >&2
    exit 2
fi
if ! command -v xvfb-run >/dev/null 2>&1; then
    printf '%s\n' 'with-xvfb: xvfb-run is missing; install Xvfb and xauth' >&2
    exit 1
fi
# Native capture compares the complete 1440x900 window with the X11 drawable.
# The private display has no IBus or Fcitx server from the caller's desktop.
export XMODIFIERS=@im=none
unset WAYLAND_DISPLAY WAYLAND_SOCKET
if [ "$managed" -eq 1 ]; then
    if ! command -v openbox >/dev/null 2>&1 || ! command -v xprop >/dev/null 2>&1; then
        printf '%s\n' 'with-xvfb: --window-manager requires Openbox and xprop' >&2
        exit 1
    fi
    # Native resize and maximize requests are handled by a window manager.
    exec xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' sh -c '
        set -eu
        openbox --sm-disable &
        wm_pid=$!
        trap "kill \$wm_pid 2>/dev/null || :; wait \$wm_pid 2>/dev/null || :" EXIT
        attempts=0
        until xprop -root _NET_SUPPORTING_WM_CHECK 2>/dev/null | grep -q "window id # 0x[1-9a-fA-F]"; do
            attempts=$((attempts + 1))
            if ! kill -0 "$wm_pid" 2>/dev/null || [ "$attempts" -ge 50 ]; then
                printf "%s\n" "with-xvfb: Openbox did not become ready" >&2
                exit 1
            fi
            sleep 0.1
        done
        "$@"
    ' with-xvfb "$@"
fi
exec xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' "$@"
