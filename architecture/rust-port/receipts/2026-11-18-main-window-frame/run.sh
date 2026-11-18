#!/bin/sh
# SPDX-License-Identifier: GPL-2.0-only
# Run inside a D-Bus session and Xvfb; the caller sets the scenario environment.
set -eu
receipt_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
address=$(gdbus call --session --dest org.a11y.Bus --object-path /org/a11y/bus --method org.a11y.Bus.GetAddress | cut -d "'" -f2)
export AT_SPI_BUS_ADDRESS="$address"
/usr/libexec/at-spi2-registryd --use-gnome-session >/dev/null 2>&1 &
registry_pid=$!
openbox >/dev/null 2>&1 &
wm_pid=$!
tint2 -c "$receipt_dir/tint2rc" >/dev/null 2>&1 &
bar_pid=$!
trap 'kill "$bar_pid" "$wm_pid" "$registry_pid" 2>/dev/null || true' EXIT HUP INT TERM
sleep 2
cargo test --locked --offline -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture --test-threads=1
