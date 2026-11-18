#!/bin/sh
# SPDX-License-Identifier: GPL-2.0-only
# Run inside Xvfb; the caller sets the scenario environment.
set -eu
openbox >/dev/null 2>&1 &
wm_pid=$!
trap 'kill "$wm_pid" 2>/dev/null || true' EXIT HUP INT TERM
sleep 1
cargo test --locked --offline -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture --test-threads=1
