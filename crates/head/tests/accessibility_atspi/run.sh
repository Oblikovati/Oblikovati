#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
# Run inside dbus-run-session and Xvfb with Openbox and a software Vulkan ICD.
set -euo pipefail
binary=${1:?Expected the path to the Oblikovati GUI binary}
directory=$(cd "$(dirname "$0")" && pwd)
address=$(gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.a11y.Bus.GetAddress | cut -d "'" -f2)
export AT_SPI_BUS_ADDRESS="$address"
/usr/libexec/at-spi2-registryd --use-gnome-session &
registry=$!
openbox >/dev/null 2>&1 &
wm=$!
"$binary" &
app=$!
trap 'kill "$app" "$wm" "$registry" 2>/dev/null || true; wait 2>/dev/null || true' EXIT
sleep 3
gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.freedesktop.DBus.Properties.Set org.a11y.Status IsEnabled '<false>' >/dev/null
gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.freedesktop.DBus.Properties.Set org.a11y.Status IsEnabled '<true>' >/dev/null
/usr/bin/python3 "$directory/client.py"
for ((i=0; i<50; i++)); do
    if ! kill -0 "$app" 2>/dev/null; then wait "$app"; exit; fi
    sleep .1
done
printf 'The GUI did not close after the accessibility action.\n' >&2
exit 1
