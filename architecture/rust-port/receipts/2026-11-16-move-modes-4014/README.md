<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move coordinate-mode diagnostics (#4014)

On the pinned scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`,
`move-13` passed six steps and `move-16` passed ten steps. The native tests
opened the mode and reference controls, checked their choices and units, and
checked the triad and unchanged ellipse in the reference case. The mode capture
was reviewed and shows Distance and Angle with millimetre and degree units.

These are frame-loop diagnostics from an uncommitted test tree on Linux with
Xvfb and Vulkan Lavapipe. They do not establish OS-level input or bounded build
provenance (#3978). The last full Move run predates these changes.
