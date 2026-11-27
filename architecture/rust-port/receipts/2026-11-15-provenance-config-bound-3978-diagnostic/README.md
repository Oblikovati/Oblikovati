<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native capture source-config diagnostic

The capture CLI ran `Inspect a part file before opening it` on Linux with Xvfb
and Vulkan Lavapipe. The inventory and runner used a copied registry index and
restricted source configuration from commit `3cc80168a`. One scenario and all
12 steps passed. The final image shows the part after cancelling Open. The
scenario uses native frame-loop input, not OS-level mouse or keyboard input.

The run predates the follow-up lockfile change checks. A separate real Cargo
regression covers that change. This remains historical diagnostic evidence: the
index is not authenticated, and host tools, dependency build scripts, external
files, runtime settings, and concurrent writes are not bounded. Issue #3978
remains open.
