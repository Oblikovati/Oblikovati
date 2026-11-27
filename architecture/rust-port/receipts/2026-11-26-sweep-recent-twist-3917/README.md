# Reuse a recent Sweep twist expression

The published `sweep-12` scenario accepts a typed Twist expression of
`45`, reopens the existing tapered Sweep, and selects `45` from the
Twist menu. The raw expression appears in the draft. Cancel leaves the
saved angle, model generation, and existing solid intact.

The fixture precondition uses the desktop document host to open the
disposable file. The Sweep editing steps use the native controls.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [8 steps](wayland/sweep-12/junit.xml) |
| X11 under Xvfb and Lavapipe | [8 steps](x11/sweep-12/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source and
native fixture.
