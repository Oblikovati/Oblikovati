<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native capture configuration diagnostic

The capture CLI ran `Inspect a part file before opening it` on Linux with Xvfb
and Vulkan Lavapipe. The runner was built offline with Rust 1.99.0, an isolated
Cargo home, and a fresh target from an export of commit `24cd14e9c`. One scenario
and all 12 steps passed. The final image shows the part after cancelling Open.
The scenario uses native frame-loop input, not OS-level mouse or keyboard input.

This is historical diagnostic evidence. Toolchain files, dependency caches,
host files, runtime settings, and concurrent writes were not bounded. The CLI
inventory preflight also ran with the caller environment. Issue #3978 remains
open.
