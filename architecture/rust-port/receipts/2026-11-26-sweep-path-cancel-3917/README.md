# Cancel a Sweep edit after clearing its path

The published `sweep-06` scenario creates a tapered Sweep, opens Edit
Feature, and clears the saved path through Properties. The panel retains
one profile and requests a curve or edge. OK becomes unavailable and the
Sweep body leaves the preview scene. Cancel displays the existing solid
again. Reopening the feature shows the original profile, path, and
five-degree taper.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [9 steps](wayland/sweep-06/junit.xml) |
| X11 under Xvfb and Lavapipe | [9 steps](x11/sweep-06/junit.xml) |

The [cleared-path capture](wayland/sweep-path-cleared.png) shows the draft
before Cancel. The [input hashes](current-inputs.sha256) identify the
published scenario source and native fixture.
