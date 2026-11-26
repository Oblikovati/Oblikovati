# Surface Sweep with automatically selected inputs

The fixture contains a circular contour and a straight path above an
existing solid. In this [acceptance variant](automatic-inputs.feature),
Sweep acquires the unique visible inputs when it opens. Switching to
Surface shows one curve and reserves Srf1. Acceptance creates SweepSrf1
under Features and Srf1 under Surface Bodies, with the original solid
still listed separately.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [9 steps](wayland/junit.xml) |
| X11 under Xvfb and Lavapipe | [9 steps](x11/junit.xml) |

The published `sweep-18` scenario also requires explicit profile and path
picking. That acquisition step still needs native verification. The
[input hashes](current-inputs.sha256) identify the published scenario
and the unmodified native fixture.
