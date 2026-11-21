<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Second part window diagnostic

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the bindings committed as
`d61773850`. The snapshot was verified before and after the run.

The filtered document-tabs run passed one scenario and 13 steps. It created a
second numbered window for a saved part, checked the title and browser, and
returned through the Switch menu to the original sketch window. The final
capture shows the original empty sketch and both numbered tabs. Intermediate
window and title checks are assertions, not captures.

The run used Xvfb at 1600 × 1200 with Mesa Lavapipe. The runner injects input
into the frame loop. This is diagnostic, not acceptance evidence: native build
provenance remains open (#3978). OS-level input and the full feature were not
tested in this run.
