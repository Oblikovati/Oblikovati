<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Sketch-preserving document switches

On 2026-11-15, the pinned document-tabs feature at snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` was run with the bindings committed
as `671b5685f`. The source snapshot was verified before and after the runs.

- Visit Home without leaving an active sketch: one scenario, eight steps passed.
- Activate an open sketch from the Switch menu: one scenario, nine steps passed.

The two filtered runs used the native application runner under Xvfb at
1600 × 1200 with Mesa Lavapipe. The captures show the empty sketch viewport,
model browser and sketch controls after returning to each part. The test also
checks the active document and sketch authoring state. It does not check a
non-empty sketch or operating-system input: the runner injects events into
the frame loop.

An initial attempt could not present on the selected GPU and stopped before
the first setup step; the passing runs selected the Lavapipe ICD. These are
diagnostic results, not acceptance receipts: build provenance remains open
under #3978. No full feature or full gate result is claimed here.
