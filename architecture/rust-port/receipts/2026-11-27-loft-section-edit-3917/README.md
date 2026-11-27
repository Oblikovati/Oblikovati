# Edit an intermediate Loft section

The published `loft-14` scenario creates a midpoint section, reopens Loft1,
and double-clicks the actual projected intermediate point. The editor shows
the saved size of 1. Changing it to 2 doubles the annotated section area and
expands the preview radius beyond both profile radii. The accepted size
survives reopening the edit. Moving the section to 5 mm on the 20 mm guide
shows the position annotation 0.250000 in the viewport.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Eleven steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Eleven steps passed](x11/junit.xml) |

The [input hashes](../2026-11-27-loft-placed-3917/current-inputs.sha256)
identify the published scenario and Part.
