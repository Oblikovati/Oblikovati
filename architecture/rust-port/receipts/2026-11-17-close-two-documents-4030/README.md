<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Closing two modified documents

On 2026-11-17, a native test based on `f65061aab` created two saved parts and
modified each through the owner API. X11 `WM_DELETE_WINDOW` requested application
close. The first capture shows the prompt for the active second part. XTEST
clicked No. The second capture shows the prompt for the remaining first part;
XTEST clicked Cancel.

The test checked that the discarded document closed, the remaining document
stayed open with unsaved edits, and both saved files retained their original
bytes. Both captures were inspected. The test passed with Xvfb at 1920 × 1080
and Vulkan Lavapipe. Head Clippy and formatting checks passed.

This covers one multi-document close sequence. It is not a pinned BDD scenario
or evidence for every save, failure, or prompt-frequency path. Native build
provenance remains open in #3978. The full gate was not run.
