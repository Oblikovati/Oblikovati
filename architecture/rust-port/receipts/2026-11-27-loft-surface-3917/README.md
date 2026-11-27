# Loft open surface from two circular sections

The published `loft-08` scenario produces a tapered preview between the
parallel-circle sketches. The native check verifies both radii, that the
preview is visible, and that the mesh has no caps. OK creates LoftSrf1 in
Surface Bodies. Saving and reopening retains the same single-face shell and
body identity.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Seven steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Seven steps passed](x11/junit.xml) |

The [input hashes](../2026-11-27-loft-solid-3917/current-inputs.sha256)
identify the published scenario and Part. The
[release probe](../../../../crates/martensite/tests/release_probe/create.rs)
checks that the open shell retains both exact source trims, one wall, and
the analytic lateral area.
