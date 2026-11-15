<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Model-browser search diagnostics

On 2026-11-15, four filtered native model-browser scenarios ran at application
commit `b0d839808` with scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f`. The snapshot was verified before
and after the runs.

All 57 steps passed: filtering by a feature name (11), finding an extrusion by
type (17), grouping same-property criteria (16), and reporting no match (13).
The reports and final captures are retained. The grouped and no-match final
captures were visually reviewed; intermediate criteria and prompt states are
checked by assertions.

The runs used Xvfb at 1600 × 1200 with Mesa Lavapipe and frame-loop input.
These are diagnostic results, not acceptance evidence: native build provenance
remains open (#3978). OS-level input and the full model-browser feature were
not tested in these runs.
