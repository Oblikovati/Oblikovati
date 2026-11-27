# Reset the Sweep advanced checkbox on reopen

The published `sweep-09` scenario opens a tapered Sweep with a valid
preview, checks Optimize for Single Selection under Advanced Properties,
and accepts the draft. Reopening the feature shows the checkbox
unchecked. The existing profile, path, taper, and solid remain available.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [6 steps](wayland/sweep-09/junit.xml) |
| X11 under Xvfb and Lavapipe | [6 steps](x11/sweep-09/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source
and native fixture.

An initial Wayland run missed the setup's OK click. The first retry
accepted the checked draft and then missed the browser context menu.
The following two serial runs passed all six steps. X11 passed the full
ten-scenario Sweep regression run.
