# Compare Sweep taper in a joined boss

The published `sweep-21` scenario reopens a joined Sweep with a
zero-degree taper. Both signed edits produce certified previews and
leave OK enabled. The desktop preview mesh shows the cap narrowing
at −5° and widening at +5° by the expected amount for the 30 mm
straight path. Cancel closes the edit and restores the saved
zero-degree Join.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [11 steps](wayland/sweep-21/junit.xml) |
| X11 under Xvfb and Lavapipe | [11 steps](x11/sweep-21/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published
scenario and native fixture.
