<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native capture Git-cache diagnostic

The capture CLI ran `Inspect a part file before opening it` on Linux with Xvfb
and Vulkan Lavapipe. Both the inventory and native runner built from fresh Git
databases, including Martensite's locked submodule. The staged databases passed
Git fsck. The build used commit `9c5b14369`. One scenario and all 12 steps
passed. The final image shows the part after cancelling Open. The scenario uses
native frame-loop input, not OS-level mouse or keyboard input.

This run predates the follow-up that disables global and system Git config in
the native build environment. A separate real Cargo regression covers that change.
This is historical diagnostic evidence: host tools, SHA-1 integrity assumptions,
registry index, source configuration, host files, and concurrent writes remain
unbounded. Issue #3978 remains open.
