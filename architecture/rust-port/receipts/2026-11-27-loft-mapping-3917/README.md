# Loft Automatic Mapping

The published `loft-04` scenario opens the parallel-circle Part and selects
both circular sections in the viewport. The test opens Transition, checks
that Automatic Mapping is active, and toggles it twice. The Point Set list
offers Click to add in manual mode. Restoring Automatic Mapping hides the
entry and restores acceptance for the current solid preview.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Six steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Six steps passed](x11/junit.xml) |

The [input hashes](../2026-11-27-loft-solid-3917/current-inputs.sha256)
identify the published scenario and native fixture.
