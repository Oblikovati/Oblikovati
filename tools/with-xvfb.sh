#!/bin/sh
# SPDX-License-Identifier: GPL-2.0-only
set -eu

if [ "$#" -eq 0 ]; then
    printf '%s\n' 'with-xvfb: expected a command and its arguments' >&2
    exit 2
fi
if ! command -v xvfb-run >/dev/null 2>&1; then
    printf '%s\n' 'with-xvfb: xvfb-run is missing; install Xvfb and xauth' >&2
    exit 1
fi
# Native capture compares the complete 1440x900 window with the X11 drawable.
exec xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' "$@"
