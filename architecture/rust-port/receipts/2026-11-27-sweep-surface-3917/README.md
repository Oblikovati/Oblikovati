# Surface Sweep from explicit viewport selections

The published `sweep-18` scenario uses a circular contour and a straight
path above an existing solid. The test clears the automatic inputs and
picks the profile and path in the viewport. Switching to Surface shows
one curve and reserves Srf1. Acceptance creates SweepSrf1 and lists its
Srf1 body under Surface Bodies. The original solid remains in its folder.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [9 steps](wayland/junit.xml) |
| X11 under Xvfb and Lavapipe | [9 steps](x11/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published
scenario and the native fixture.

The published `sweep-20` scenario starts a Surface Sweep in Guide
orientation. It accepts the viewport profile pick, keeps both Path and
Guide prompts after a browser sketch click, and disables OK. Cancel
closes the draft.

| Backend | Incomplete Guide evidence |
| --- | --- |
| Wayland on COSMIC | [10 steps](wayland/incomplete-guide.xml) |
| X11 under Xvfb and Lavapipe | [10 steps](x11/incomplete-guide.xml) |
