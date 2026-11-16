<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Quick access command choices

On 2026-11-16 the full pinned quick-access feature passed 12 scenarios and
128 steps with frame-loop input. The scenario snapshot
`e76cb749ed43c73a91c24854057da7c890041f51` verified before and after the run.
The application was based on `7de7c6988` with the menu change and an unrelated
`README.md` modification uncommitted. Vulkan Lavapipe ran under Xvfb.

The image shows the four unavailable commands as disabled and unchecked. A
separate native test checked they cannot be added. These entries do not claim
that their commands are implemented. The full run is not verified BDD
acceptance while native build provenance remains open (#3978).
