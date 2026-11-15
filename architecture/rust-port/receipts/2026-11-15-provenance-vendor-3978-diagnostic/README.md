<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native capture registry-cache diagnostic

The capture CLI ran `Inspect a part file before opening it` on Linux with Xvfb
and Vulkan Lavapipe. The inventory and runner builds used fresh Cargo source
caches extracted from archives whose SHA-256 values matched the selected commit's
lockfile. The build used commit `769b95254`. One scenario and all 12 steps
passed. The final image shows the part after cancelling Open. The scenario uses
native frame-loop input, not OS-level mouse or keyboard input.

This is historical diagnostic evidence. Git dependency caches, the registry
index, toolchain, host and runtime inputs, and concurrent writes were not
bounded. Issue #3978 remains open.
