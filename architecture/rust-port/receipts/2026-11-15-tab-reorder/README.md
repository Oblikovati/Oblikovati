<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Document tab reordering diagnostic

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the bindings committed as
`f1fe6b370`. The snapshot was verified before and after the run.

The filtered document-tabs run passed two scenarios and 17 steps: reordering
an active drawing tab left and back, and moving an inactive part tab left.
The checks measure tab positions, active document identity, and the drawing or
part browser. The final captures show the drawing and the reordered part; the
intermediate positions are recorded as test assertions rather than captures.

The run used Xvfb at 1600 × 1200 with Mesa Lavapipe. The runner injects input
into the frame loop. These results are diagnostic, not acceptance receipts:
native build provenance remains open (#3978). The full feature and OS-level
drag behavior were not tested in this run.
