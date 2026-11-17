<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Exit after a save failure

A native test on 2026-11-17 created and modified two saved parts. It moved the
active part's original file aside in a temporary directory and put a directory
at its save path. X11 `WM_DELETE_WINDOW` requested exit, and XTEST selected Yes
in the save prompt. The save failed at file replacement. Both documents remained
open and dirty, and the original files remained byte-identical. The reviewed
capture shows both tabs and the save failure in the status area.

The test restored the temporary path, requested exit again, and selected Yes.
The active document saved and closed. Cancel on the remaining document kept it
open and dirty with its original file unchanged. The test passed under Xvfb at
1920 × 1080 with Vulkan Lavapipe. Head Clippy and formatting passed.

This checks one file-replacement failure and retry, not every I/O failure. The
fixture setup used owner operations; the exit and prompt choices used native
X11 events. It is not a pinned BDD scenario, and build provenance remains open
in #3978. The full gate was not run.
