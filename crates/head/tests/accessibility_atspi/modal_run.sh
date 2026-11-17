#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# Run inside dbus-run-session and Xvfb with Openbox and a software Vulkan ICD.
set -euo pipefail
binary=${1:?Expected the path to the compiled head test binary}
directory=$(cd "$(dirname "$0")" && pwd)
address=$(gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.a11y.Bus.GetAddress | cut -d "'" -f2)
export AT_SPI_BUS_ADDRESS="$address"
/usr/libexec/at-spi2-registryd --use-gnome-session &
registry=$!
openbox >/dev/null 2>&1 &
wm=$!
ready_dir=$(mktemp -d)
export OBLIKOVATI_A11Y_READY="$ready_dir/ready"
"$binary" --ignored --exact native::gui_loop_tests::application_close::accessible_dirty_close_live_fixture --nocapture &
app=$!
trap 'kill "$app" "$wm" "$registry" 2>/dev/null || true; wait 2>/dev/null || true; unlink "$OBLIKOVATI_A11Y_READY" 2>/dev/null || true; rmdir "$ready_dir" 2>/dev/null || true' EXIT
for ((i=0; i<600; i++)); do
    if [[ -f "$OBLIKOVATI_A11Y_READY" ]]; then break; fi
    if ! kill -0 "$app" 2>/dev/null; then wait "$app"; exit 1; fi
    sleep .1
done
[[ -f "$OBLIKOVATI_A11Y_READY" ]] || { printf 'Fixture did not become ready.\n' >&2; exit 1; }
gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.freedesktop.DBus.Properties.Set org.a11y.Status IsEnabled '<false>' >/dev/null
gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.freedesktop.DBus.Properties.Set org.a11y.Status IsEnabled '<true>' >/dev/null
/usr/bin/python3 "$directory/modal_client.py"
for ((i=0; i<100; i++)); do
    if ! kill -0 "$app" 2>/dev/null; then wait "$app"; exit; fi
    sleep .1
done
printf 'The dirty-close fixture did not exit after Discard.\n' >&2
exit 1
