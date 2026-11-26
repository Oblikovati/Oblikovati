# Loft solid from two circular sections

The published `loft-03` scenario uses a disposable copy of the parallel-circle
Part. The test opens Loft from the ribbon and clicks both circular edges in the
viewport. It checks their order in Sections, the visible tapered solid, and the
preview radii of 10 mm and 5 mm at the two source planes. Before accepting,
the browser has zero bodies. OK commits Loft1 as the owner of one solid with
two caps and a wall. Saving and reopening the Part retains the body identity.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Six steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Six steps passed](x11/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published scenario, SVG,
and native fixture. Both runs used the existing Martensite v0.14.2 pin.
