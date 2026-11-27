# Loft with a straight Center Line

The published `loft-12` scenario selects both circles, changes the path mode,
and clicks Sketch3's line in the viewport. The real shaded solid preview
appears after the pick. OK commits Loft1 with one solid and records the
authored line identity. The native test saves and reopens the Part, then
checks the same line identity and body key.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Eight steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Eight steps passed](x11/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published scenario
and Part. The [center-line checks](../../../../crates/martensite/tests/unit/loft_center_line.rs)
accept either direction of the same segment and refuse displaced or
degenerate curves.
