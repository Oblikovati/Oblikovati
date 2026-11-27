<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Centered rectangle height cancellation

On 2026-11-16, `rectangle-center-16` passed all four steps with frame-loop
input and Martensite v0.13.1. The pinned scenario snapshot
`e76cb749ed43c73a91c24854057da7c890041f51` verified before and after the run.
The application was based on `37b33a3ff` with the two step bindings and an
unrelated `README.md` modification uncommitted. Vulkan Lavapipe ran under Xvfb.

The screenshot shows the reopened editor with 6 mm after the proposed 8 mm
edit was cancelled. This is not verified acceptance: native build provenance
remains open (#3978), and this run did not use OS-level input.
