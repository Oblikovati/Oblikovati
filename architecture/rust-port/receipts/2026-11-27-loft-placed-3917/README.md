# Place an intermediate Loft section

The published `loft-13` scenario creates and reopens a center-line Loft from
the native 20 mm guide and two circular sections. A click near the guide
midpoint creates Point1 and opens Section Dimensions. The viewport shows the
intermediate position and analytic area. After both accepts, the same Loft1
body and section settings survive saving and reopening.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Eight steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Eight steps passed](x11/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published scenario and
Part. The [Martensite release probe](../../../../crates/martensite/tests/release_probe/create.rs)
checks the valid solid, exact wall history, and volume change with a larger
intermediate section.
