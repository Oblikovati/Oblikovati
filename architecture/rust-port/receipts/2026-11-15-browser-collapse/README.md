<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Model-browser collapse diagnostics

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the test changes committed as
`3490e2196`. The snapshot was verified before and after the runs.

The initial filtered run passed two of four scenarios; two stopped at unbound
steps. After binding those steps and checking the expanded fixture, the same
four scenarios passed all 35 steps. They cover Origin, an extrusion branch,
the whole tree, and Solid Bodies descendants. The assertions now check that
the sketch row is absent when its feature is collapsed and that both feature
and sketch rows disappear and return under Solid Bodies.

The final captures show the expanded states and visible solid; the hidden
states are checked by assertions. The run used Xvfb at 1600 × 1200 with Mesa
Lavapipe and frame-loop input. It is diagnostic, not acceptance evidence:
native build provenance remains open (#3978). OS-level input and the full
model-browser feature were not tested in this run.
