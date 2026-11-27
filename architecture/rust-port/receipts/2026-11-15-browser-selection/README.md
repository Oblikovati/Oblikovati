<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Model-browser selection diagnostics

On 2026-11-15, four filtered native model-browser scenarios ran with scenario
snapshot `05317bbb2d0809a532b29b7ff6f726a5722b865f`. The snapshot was verified
before and after the runs. The selection, Find in Window, and origin-plane runs
used application commit `ae733c5a1`. The extrusion-menu rerun used the binding
committed as `b0ee7721e`.

The runs passed 37 steps: feature selection and clearing (9), the extrusion
context menu (10), Find in Window (8), and origin-plane selection and clearing
(10). The menu initially stopped at an unbound wording variant; the rerun binds
that wording to the existing assertion for both menu entries.

The final captures and reports are retained. The Find in Window and plane
captures were visually reviewed; intermediate highlight and menu states are
checked by assertions. The runs used Xvfb at 1600 × 1200 with Mesa Lavapipe and
frame-loop input. They are diagnostic, not acceptance evidence: build
provenance remains open (#3978). OS-level input and the full model-browser
feature were not tested in these runs.
