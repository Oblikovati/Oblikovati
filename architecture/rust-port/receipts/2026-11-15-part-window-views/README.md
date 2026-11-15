<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Independent part window diagnostics

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the bindings committed as
`41db02ee6`. The snapshot was verified before and after the runs.

Three filtered document-tabs runs passed 29 steps: restoring distinct views
(10), retaining separate zoom levels (9), and closing the additional window
without closing the part (10). The zoom scenario draws a rectangle and checks
view scale before and after switching. The other scenarios use an empty sketch.
The captures show final states; intermediate states are checked by assertions.

The runs used Xvfb at 1600 × 1200 with Mesa Lavapipe. Input is injected into
the frame loop. These are diagnostics, not acceptance receipts: native build
provenance remains open (#3978). OS-level input and the full feature were not
tested in these runs.
