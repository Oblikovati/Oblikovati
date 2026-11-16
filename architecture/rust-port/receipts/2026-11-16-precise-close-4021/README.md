<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Input close diagnostic (#4021)

The pinned `move-17` scenario failed before the change: six steps passed and
then the Move checkbox was unchecked after closing the toolbar. After the
change, all nine steps passed, including a check of the rendered checkbox in
the Move window, the absent triad, the active Move stage, and the unchanged
sketch recipe and history after Escape. The final capture was reviewed.

Two additional native frame-loop tests passed. They exercise reopening through
the Move checkbox and the Create ribbon, leaving Move, and opening and closing
Precise Input in the Point tool. The GUI state test, strict GUI/head Clippy,
formatting and documentation lint passed. The Go reference has no equivalent
Precise Input toolbar control.

The full pinned Move run passed 11 of 27 scenarios; 16 failed or stopped at
undefined steps. Its report is in `full/`; the pre-fix result is in `red/`.
Each failed scenario stops at its first failure, so later gaps can remain.

The scenario snapshot was `e76cb749ed43c73a91c24854057da7c890041f51`; the
application test tree had local changes on `bbc32148b`. Native runs used
frame-loop input on Linux with Xvfb and Vulkan Lavapipe. OS-level input and
bounded build provenance remain unverified (#3978). The full gate was not run.
