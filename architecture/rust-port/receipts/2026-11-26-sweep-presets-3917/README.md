# Save and apply a Sweep preset

In the published `sweep-10` scenario, the user enters zero taper and
90 degrees of twist, saves `Follow Path_0 deg_90 deg` through the plus
control, and verifies its settings in the isolated preference directory.
No Preset lets the user change Twist to zero. Selecting the saved entry
restores 90 degrees and keeps both geometry inputs.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [10 steps](wayland/sweep-10/junit.xml) |
| X11 under Xvfb and Lavapipe | [10 steps](x11/sweep-10/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source
and native fixture.
