<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Full document-tabs diagnostic

On 2026-11-15, the native application runner used scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f` and the test bindings committed as
`b99ab16bb`. The snapshot was verified before and after the run. The full
document-tabs feature passed 38 scenarios and 334 steps.

An earlier run on the same snapshot stopped five scenarios at undefined
interactions. The later run binds those interactions and prepares the
left-arrow fixture so its action uses the single click specified by the source.
The checked behavior includes detached drawings and tab overflow.

The capture for “Reveal earlier tabs without changing the active document”
shows a visible part tab highlighted while the presentation workspace remains
active and its tab is off-screen. This visible defect is tracked in #3992 and
was not detected by the scenario assertions. The floating drawing capture shows
an in-application window; it does not establish a separate operating-system
window.

The run used Xvfb at 1600 × 1200 with Mesa Lavapipe. Input is injected into
the frame loop. This is diagnostic, not acceptance evidence: native build
provenance remains open (#3978), and OS-level input was not tested in this run.
