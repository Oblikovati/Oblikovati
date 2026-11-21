<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Quick access availability check

On 2026-11-16, the unsupported Revolve shortcut regression failed before the
change and passed after it. The pinned quick-access scenario “Disable the add
option for an existing shortcut” then passed all 10 steps. It checks that the
supported Extrude command can still be added, removed, and added again.

The scenario used snapshot `e76cb749ed43c73a91c24854057da7c890041f51`, verified
before and after the run. The application was based on `40ff7af28` with this
change uncommitted and the unrelated `README.md` modification present. The
runner used frame-loop input and Vulkan Lavapipe under Xvfb. This is not a
verified acceptance receipt because build provenance remains open (#3978).
