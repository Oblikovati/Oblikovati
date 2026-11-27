# Save and validate Sweep twist

The published cases use a disposable circular profile on a coaxial
straight path. `sweep-07` applies 90 degrees of twist to a tapered Sweep,
accepts the edit, and checks the saved angle after reopening. `sweep-11`
checks that an invalid expression disables OK and replacing it with
zero restores a valid preview.

| Backend | Reopen with 90 degrees | Correct invalid Twist |
| --- | --- | --- |
| Wayland on COSMIC | [8 steps](wayland/sweep-07/junit.xml) | [7 steps](wayland/sweep-11/junit.xml) |
| X11 under Xvfb and Lavapipe | [8 steps](x11/sweep-07/junit.xml) | [7 steps](x11/sweep-11/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source
and native fixture.
