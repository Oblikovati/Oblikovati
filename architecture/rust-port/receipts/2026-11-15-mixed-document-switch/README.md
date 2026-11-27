<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Drawing and sketch switch diagnostic

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the bindings committed as
`50e87f9af`. The snapshot was verified before and after the run.

The filtered document-tabs run passed one scenario and 14 steps. It switched
between a drawing and an actively edited part sketch, checked each browser and
ribbon, and confirmed that the drawing resources branch stayed collapsed. The
final capture shows the empty sketch workspace after returning to the part;
the intermediate drawing state is checked by assertions, not captured here.

The run used Xvfb at 1600 × 1200 with Mesa Lavapipe. The runner injects input
into the frame loop. This is diagnostic, not acceptance evidence: native build
provenance remains open (#3978). OS-level switching and the full feature were
not tested in this run.
