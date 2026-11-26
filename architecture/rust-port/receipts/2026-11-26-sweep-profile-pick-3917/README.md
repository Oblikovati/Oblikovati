# Reselect a cleared Sweep profile

The published `sweep-08` scenario clears the profile in a tapered Sweep
edit. The path remains selected, OK is disabled, and the viewport shows
the profile sketch. A click inside the region restores the profile and the
tapered preview. It also restores the normal sketch visibility.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [9 steps](wayland/sweep-08/junit.xml) |
| X11 under Xvfb and Lavapipe | [9 steps](x11/sweep-08/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source and
native fixture. The [release probe](../../../../crates/martensite/tests/release_probe/sweep_sources.rs)
checks that interior and unambiguous boundary rays select the solved
profile. Exterior and parallel rays leave the selector open.
