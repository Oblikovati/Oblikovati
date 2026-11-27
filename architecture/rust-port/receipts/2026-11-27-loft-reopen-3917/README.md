# Reopen and cancel a Loft edit

The published `loft-06` scenario creates Loft1 from the two fixture circles.
A native browser double-click opens the saved sections and draws a shaded
preview. Both draft checkboxes are checked before Cancel. Cancellation removes
the preview and preserves the document generation, visible solid, and its body
key. A second double-click shows the preview and both options unchecked.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Eight steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Eight steps passed](x11/junit.xml) |

The [input hashes](../2026-11-27-loft-solid-3917/current-inputs.sha256)
identify the published scenario and native fixture.
