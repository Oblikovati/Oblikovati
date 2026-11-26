# Sweep Orientation

The published scenarios use disposable copies of the circular-profile Sweep
fixture. `sweep-04` changes the selector to Dropdown, reads its three
choices, and verifies that Guide requires a reference before OK. `sweep-05`
returns to a valid Fixed preview, verifies the absence of the angle fields
and the default unchecked advanced option, and commits the Fixed edit to
the existing feature and body.

| Backend | Guide requires a reference | Accept Fixed |
| --- | --- | --- |
| Wayland on COSMIC | [8 steps](wayland/sweep-04/junit.xml) | [11 steps](wayland/sweep-05/junit.xml) |
| X11 under Xvfb and Lavapipe | [8 steps](x11/sweep-04/junit.xml) | [11 steps](x11/sweep-05/junit.xml) |

The [Guide capture](wayland/guide-required.png) shows the incomplete draft
with both original sources retained and OK disabled. The
[input hashes](current-inputs.sha256) identify the published scenario source
and native fixture.
