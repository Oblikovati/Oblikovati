# Cancel an incomplete Surface Loft point set

The published `loft-10` scenario creates and reopens LoftSrf1. Clearing
Automatic Mapping shows Set 1 with both source sketch rows. Click to add
creates Set 2 with two starred map-point rows and OK remains disabled.
Cancel restores the visible surface and leaves its body key and document
generation intact.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Nine steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Nine steps passed](x11/junit.xml) |

The [input hashes](../2026-11-27-loft-solid-3917/current-inputs.sha256)
identify the published scenario and Part.
