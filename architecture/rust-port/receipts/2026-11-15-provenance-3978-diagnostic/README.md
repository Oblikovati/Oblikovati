<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native capture export diagnostic

The capture CLI ran `Inspect a part file before opening it` on Linux with Xvfb
and Vulkan Lavapipe. The runner was built from an export of commit `956d9f4fd`
with a fresh temporary target. One scenario and all 12 steps passed. The final
capture shows the active part after the Open dialog was cancelled. The test uses
native frame-loop input; it does not establish OS-level mouse or keyboard input.

This is historical diagnostic evidence. Compiler settings, dependencies, host
files and concurrent writes were not bounded. Issue #3978 remains open.
